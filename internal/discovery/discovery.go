/*
Copyright 2023 The Kubernetes Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package discovery provides a discovery and resolution logic for GVKs.
package discovery

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	ksmtypes "k8s.io/kube-state-metrics/v2/pkg/builder/types"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	"k8s.io/kube-state-metrics/v2/pkg/metricshandler"
	"k8s.io/kube-state-metrics/v2/pkg/options"
	"k8s.io/kube-state-metrics/v2/pkg/util"
)

// Interval is the time interval between two cache sync checks.
const Interval = 3 * time.Second

// maxRetryBackoff caps the delay between retries of a failed store rebuild.
const maxRetryBackoff = 5 * time.Minute

// retryBackoff returns the delay before retrying a store rebuild that has
// failed the given number of consecutive times.
func retryBackoff(failures int) time.Duration {
	backoff := Interval
	for i := 1; i < failures && backoff < maxRetryBackoff; i++ {
		backoff *= 2
	}
	return min(backoff, maxRetryBackoff)
}

// extractGVKPs returns the GVKPs defined by the given CRD, skipping any version
// that the API server does not serve.
func extractGVKPs(obj any) []groupVersionKindPlural {
	if d, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj = d.Obj
	}
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		klog.ErrorS(nil, "expected *unstructured.Unstructured", "got", fmt.Sprintf("%T", obj))
		return nil
	}
	// Every field read below is required on apiextensions.k8s.io/v1, so a CRD the
	// API server accepted has them. They are still checked rather than asserted:
	// this runs on the informer goroutine, where a failed assertion is an
	// unrecovered panic that takes the process down, and the cost of being wrong
	// about the shape of one object should be that object being skipped.
	objSpec, ok := u.Object["spec"].(map[string]any)
	if !ok {
		klog.ErrorS(nil, "CRD has no spec object", "crd", u.GetName())
		return nil
	}
	g, ok := objSpec["group"].(string)
	if !ok {
		klog.ErrorS(nil, "CRD spec has no group", "crd", u.GetName())
		return nil
	}
	names, ok := objSpec["names"].(map[string]any)
	if !ok {
		klog.ErrorS(nil, "CRD spec has no names object", "crd", u.GetName())
		return nil
	}
	k, ok := names["kind"].(string)
	if !ok {
		klog.ErrorS(nil, "CRD spec has no kind", "crd", u.GetName())
		return nil
	}
	p, ok := names["plural"].(string)
	if !ok {
		klog.ErrorS(nil, "CRD spec has no plural", "crd", u.GetName())
		return nil
	}
	versions, ok := objSpec["versions"].([]any)
	if !ok {
		klog.ErrorS(nil, "CRD spec has no versions list", "crd", u.GetName())
		return nil
	}
	// Non-nil empty means every version was successfully read as unserved.
	// Nil (below) means a version-level parse failure, so callers must not
	// treat the result as "all versions removed".
	gvkps := make([]groupVersionKindPlural, 0, len(versions))
	parseFailed := false
	for _, version := range versions {
		versionSpec, ok := version.(map[string]any)
		if !ok {
			klog.ErrorS(nil, "CRD version is not an object", "crd", u.GetName())
			parseFailed = true
			continue
		}
		v, ok := versionSpec["name"].(string)
		if !ok {
			klog.ErrorS(nil, "CRD version has no name", "crd", u.GetName())
			parseFailed = true
			continue
		}
		// Versions that are not served by the API server cannot be listed or
		// watched, so any reflector started for them would fail indefinitely.
		// `served` is a required field on apiextensions.k8s.io/v1, so treat it as
		// served when absent, but skip the version when it is present and not a
		// bool: the point of this check is to avoid starting a reflector that can
		// never succeed, and a value we cannot read is not a reason to start one.
		if rawServed, present := versionSpec["served"]; present {
			served, ok := rawServed.(bool)
			if !ok {
				klog.ErrorS(nil, "CRD version has a non-boolean served field", "crd", u.GetName(), "version", v)
				parseFailed = true
				continue
			}
			if !served {
				klog.V(1).InfoS("skipping CRD version that is not served by the API server", "group", g, "version", v, "kind", k)
				continue
			}
		}
		gvkps = append(gvkps, groupVersionKindPlural{
			Group:   g,
			Version: v,
			Kind:    k,
			Plural:  p,
		})
	}
	if len(gvkps) == 0 && parseFailed {
		return nil
	}
	return gvkps
}

// StartDiscovery starts the discovery process, fetching all the objects that can be listed from the apiserver, every `Interval` seconds.
// resolveGVK needs to be called after StartDiscovery to generate factories.
func (r *CRDiscoverer) StartDiscovery(ctx context.Context, config *rest.Config) error {
	client := dynamic.NewForConfigOrDie(config)
	factory := dynamicinformer.NewFilteredDynamicInformer(client, schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}, "", 0, nil, nil)
	informer := factory.Informer()
	stopper := make(chan struct{})
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			gvkps := extractGVKPs(obj)
			r.SafeWrite(func() {
				// Only flag a store rebuild when the cache gains a new GVK/plural.
				// Re-adds of an already-known CRD are no-ops for metrics writers.
				if r.AppendToMap(gvkps...) {
					r.WasUpdated = true
				}
			})
			r.SafeWrite(func() {
				r.CRDsAddEventsCounter.Inc()
				r.CRDsCacheCountGauge.Inc()
			})
		},
		UpdateFunc: func(oldObj, newObj any) {
			oldGVKPs := extractGVKPs(oldObj)
			newGVKPs := extractGVKPs(newObj)
			// CRD status/metadata churn is common and does not change the
			// discoverable GVK/plural set. Applying Remove+Append for those
			// events previously closed active reflector stop channels and
			// forced full metrics rebuilds for unchanged stores.
			r.SafeWrite(func() {
				r.applyCRDUpdate(oldGVKPs, newGVKPs)
				r.CRDsUpdateEventsCounter.Inc()
			})
		},
		DeleteFunc: func(obj any) {
			gvkps := extractGVKPs(obj)
			r.SafeWrite(func() {
				if r.RemoveFromMap(gvkps...) {
					r.WasUpdated = true
				}
			})
			r.SafeWrite(func() {
				r.CRDsDeleteEventsCounter.Inc()
				r.CRDsCacheCountGauge.Dec()
			})
		},
	})
	if err != nil {
		return err
	}
	// Respect context cancellation.
	go func() {
		<-ctx.Done()
		klog.InfoS("context cancelled, stopping discovery")
		close(stopper)
	}()
	go informer.Run(stopper)
	return nil
}

// ResolveGVKToGVKPs resolves the variable VKs to a GVK list, based on the current cache.
func (r *CRDiscoverer) ResolveGVKToGVKPs(gvk schema.GroupVersionKind) (resolvedGVKPs []groupVersionKindPlural, err error) { // nolint:revive
	g := gvk.Group
	v := gvk.Version
	k := gvk.Kind
	if g == "" || g == "*" {
		return nil, fmt.Errorf("group is required in the defined GVK %v", gvk)
	}
	hasVersion := v != "" && v != "*"
	hasKind := k != "" && k != "*"

	// The cache is written by the CRD informer's event handlers, so reading it
	// has to hold the lock. The whole body takes the lock once rather than
	// locking each read: the warning bookkeeping writes to the cache as well,
	// and sync.RWMutex is not reentrant, so a nested Safe* call from inside a
	// held lock would deadlock. The logging is deliberately left outside.
	warnMissing := false
	r.SafeWrite(func() {
		// No need to resolve, return.
		if hasVersion && hasKind {
			for _, el := range r.Map[g][v] {
				if el.Kind == k {
					r.clearMissingGVKWarningLocked(gvk)
					resolvedGVKPs = []groupVersionKindPlural{
						{
							Group:   g,
							Version: v,
							Kind:    k,
							Plural:  el.Plural,
						},
					}
					return
				}
			}
			warnMissing = r.markMissingGVKWarnedLocked(gvk)
			return
		}
		if hasVersion && !hasKind {
			kinds := r.Map[g][v]
			for _, el := range kinds {
				resolvedGVKPs = append(resolvedGVKPs, groupVersionKindPlural{
					Group:   g,
					Version: v,
					Kind:    el.Kind,
					Plural:  el.Plural,
				})
			}
		}
		if !hasVersion && hasKind {
			versions := r.Map[g]
			for version, kinds := range versions {
				for _, el := range kinds {
					if el.Kind == k {
						resolvedGVKPs = append(resolvedGVKPs, groupVersionKindPlural{
							Group:   g,
							Version: version,
							Kind:    k,
							Plural:  el.Plural,
						})
					}
				}
			}
		}
		if !hasVersion && !hasKind {
			versions := r.Map[g]
			for version, kinds := range versions {
				for _, el := range kinds {
					resolvedGVKPs = append(resolvedGVKPs, groupVersionKindPlural{
						Group:   g,
						Version: version,
						Kind:    el.Kind,
						Plural:  el.Plural,
					})
				}
			}
		}
	})

	if warnMissing {
		klog.InfoS("Configured custom resource was not found in the cluster, no metrics will be generated for it until a CRD serving this version is installed", "gvk", gvk)
	}
	return resolvedGVKPs, nil
}

// storeConfigurer applies a configuration to the store builder and rebuilds the
// metrics writers. *metricshandler.MetricsHandler implements it.
type storeConfigurer interface {
	ConfigureStore(ctx context.Context, configure func(ksmtypes.BuilderInterface) error) error
}

// cacheReconciler rebuilds the custom resource stores when the discovery cache
// changes in a way that affects them. Only the polling goroutine uses it, so its
// state needs no lock.
type cacheReconciler struct {
	discoverer       *CRDiscoverer
	store            storeConfigurer
	factoryGenerator func() ([]customresource.RegistryFactory, error)
	createClients    func(factories ...customresource.RegistryFactory) (map[string]any, error)

	// The enabled CR set and the stop channels of their GVKs at the last
	// rebuild, compared on each tick to skip no-op rebuilds.
	lastAppliedEnabled   []string
	lastAppliedStopChans map[string]chan struct{}

	// Failed rebuilds are retried with exponential backoff so a persistent error
	// (e.g. an invalid custom resource state config) does not retry and log on
	// every tick. A new discovery revision resets the backoff.
	consecutiveFailures int
	failedRevision      uint64
	nextRetry           time.Time
}

// PollForCacheUpdates polls the cache for updates and updates the stores accordingly.
func (r *CRDiscoverer) PollForCacheUpdates(
	ctx context.Context,
	opts *options.Options,
	m *metricshandler.MetricsHandler,
	factoryGenerator func() ([]customresource.RegistryFactory, error),
) {
	// The interval at which we will check the cache for updates.
	t := time.NewTicker(Interval)
	c := &cacheReconciler{
		discoverer:       r,
		store:            m,
		factoryGenerator: factoryGenerator,
		createClients: func(factories ...customresource.RegistryFactory) (map[string]any, error) {
			return util.CreateCustomResourceClients(opts.Apiserver, opts.Kubeconfig, factories...)
		},
	}
	go func() {
		for range t.C {
			select {
			case <-ctx.Done():
				klog.InfoS("context cancelled")
				t.Stop()
				return
			default:
				c.reconcile(ctx)
			}
		}
	}()
}

// reconcile rebuilds the custom resource stores if the cache was updated, and
// retries a failed rebuild with backoff.
func (c *cacheReconciler) reconcile(ctx context.Context) {
	r := c.discoverer
	// Check if cache has been updated.
	shouldGenerateMetrics := false
	var observedRevision uint64
	var observedStopChans map[string]chan struct{}
	r.SafeRead(func() {
		shouldGenerateMetrics = r.WasUpdated
		observedRevision = r.cacheRevision
		// Take the stop channels before the rebuild hands them to the reflectors.
		// A channel replaced after this point then differs from the applied one
		// on the next tick, instead of being recorded as applied while the
		// reflectors still hold the closed one.
		if shouldGenerateMetrics {
			observedStopChans = maps.Clone(r.GVKToReflectorStopChanMap)
		}
	})
	if !shouldGenerateMetrics {
		return
	}
	if c.consecutiveFailures > 0 && observedRevision == c.failedRevision && time.Now().Before(c.nextRetry) {
		return
	}
	if observedRevision != c.failedRevision {
		c.consecutiveFailures = 0
	}
	applied, err := c.generateMetrics(ctx, observedRevision, observedStopChans)
	if err != nil {
		c.consecutiveFailures++
		c.failedRevision = observedRevision
		backoff := retryBackoff(c.consecutiveFailures)
		c.nextRetry = time.Now().Add(backoff)
		klog.ErrorS(err, "failed to update custom resource stores", "retryIn", backoff, "consecutiveFailures", c.consecutiveFailures)
		return
	}
	c.consecutiveFailures = 0
	if applied {
		klog.InfoS("discovery finished, cache updated")
	}
}

// generateMetrics rebuilds the custom resource stores for the factories that
// the configuration currently resolves to, unless neither the resolved
// resources nor the stop channels of their GVKs changed since the last rebuild.
func (c *cacheReconciler) generateMetrics(ctx context.Context, observedRevision uint64, observedStopChans map[string]chan struct{}) (applied bool, err error) {
	r := c.discoverer
	// Get families for discovered factories.
	customFactories, err := c.factoryGenerator()
	if err != nil {
		// Preserve WasUpdated so a later tick retries.
		return false, err
	}
	// Update the list of enabled custom resources, and collect the stop channels
	// that their reflectors will be started with.
	var enabledCustomResources []string
	stopChans := make(map[string]chan struct{}, len(customFactories))
	for _, factory := range customFactories {
		expectedType := factory.ExpectedType()
		gvr, err := util.GVRFromType(factory.Name(), expectedType)
		if err != nil {
			klog.ErrorS(err, "failed to update custom resource stores")
		}
		var gvrString string
		if gvr != nil {
			gvrString = gvr.String()
		} else {
			gvrString = factory.Name()
		}
		enabledCustomResources = append(enabledCustomResources, gvrString)
		// Key the channels the same way startReflector looks them up.
		if u, ok := expectedType.(*unstructured.Unstructured); ok {
			gvk := u.GroupVersionKind().String()
			stopChans[gvk] = observedStopChans[gvk]
		}
	}
	sort.Strings(enabledCustomResources)
	// Rebuild only when either changed since the last rebuild:
	//   - the set of enabled custom resources, or
	//   - the stop channel of a resolved GVK. A stop channel is closed only when
	//     its GVK leaves the cache or its plural changes, and a closed channel
	//     never comes back, so a different channel means that the reflectors of
	//     that GVK have been stopped, even if the CRD has been re-added with the
	//     same GVR since.
	//
	// Changes to CRDs that the configuration does not resolve to change neither.
	if slices.Equal(enabledCustomResources, c.lastAppliedEnabled) && maps.Equal(stopChans, c.lastAppliedStopChans) {
		r.SafeWrite(func() {
			if r.cacheRevision == observedRevision {
				r.WasUpdated = false
			}
		})
		klog.V(2).InfoS("discovery cache changed but enabled custom resources are unchanged; skipping store rebuild")
		return false, nil
	}
	// Create clients for discovered factories. Keep this outside ConfigureStore:
	// client construction is slow and must not hold the metrics handler lock.
	discoveredCustomResourceClients, err := c.createClients(customFactories...)
	if err != nil {
		// Preserve WasUpdated so a later tick retries.
		return false, err
	}
	// Apply builder config under the same lock Build() uses, then rebuild.
	// Mutating the shared builder directly would race ConfigureSharding / BuildWriters.
	if err := c.store.ConfigureStore(ctx, func(b ksmtypes.BuilderInterface) error {
		replacer, ok := b.(ksmtypes.CustomResourceReplacer)
		if !ok {
			return fmt.Errorf("builder does not support replacing custom resources")
		}
		b.WithCustomResourceClients(discoveredCustomResourceClients)
		b.WithCustomResourceStoreFactories(customFactories...)
		if err := replacer.ReplaceEnabledCustomResources(enabledCustomResources); err != nil {
			return err
		}
		b.WithGenerateCustomResourceStoresFunc(b.DefaultGenerateCustomResourceStoresFunc())
		return nil
	}); err != nil {
		// Preserve WasUpdated so a later tick retries.
		return false, err
	}
	c.lastAppliedEnabled = slices.Clone(enabledCustomResources)
	c.lastAppliedStopChans = stopChans
	// Clear only the revision we applied. If discovery changed while clients
	// were being created, leave the flag set for another reconciliation.
	r.SafeWrite(func() {
		if r.cacheRevision == observedRevision {
			r.WasUpdated = false
		}
	})
	return true, nil
}
