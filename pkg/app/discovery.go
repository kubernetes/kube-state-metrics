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

package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/internal/discovery"
	"k8s.io/kube-state-metrics/v2/internal/store"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	"k8s.io/kube-state-metrics/v2/pkg/metricshandler"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

// Interval is the time interval between two cache sync checks.
const Interval = 3 * time.Second

// CRDiscoverer provides a cache of the collected GVKs, along with helper utilities.
type CRDiscoverer struct {
	// m is a mutex to protect the cache.
	m sync.RWMutex
	// Map is a cache of the collected GVKs.
	Map map[string]map[string][]discovery.KindPlural
	// ShouldUpdate is a flag that indicates whether the cache was updated.
	WasUpdated bool
	// CRDsAddEventsCounter tracks the number of times that the CRD informer triggered the "add" event.
	CRDsAddEventsCounter prometheus.Counter
	// CRDsDeleteEventsCounter tracks the number of times that the CRD informer triggered the "remove" event.
	CRDsDeleteEventsCounter prometheus.Counter
	// CRDsCacheCountGauge tracks the net amount of CRDs affecting the cache at this point.
	CRDsCacheCountGauge prometheus.Gauge
}

// SafeRead executes the given function while holding a read lock.
func (r *CRDiscoverer) SafeRead(f func()) {
	r.m.RLock()
	defer r.m.RUnlock()
	f()
}

// SafeWrite executes the given function while holding a write lock.
func (r *CRDiscoverer) SafeWrite(f func()) {
	r.m.Lock()
	defer r.m.Unlock()
	f()
}

// AppendToMap appends the given GVKs to the cache.
func (r *CRDiscoverer) AppendToMap(gvkps ...discovery.GroupVersionKindPlural) {
	if r.Map == nil {
		r.Map = map[string]map[string][]discovery.KindPlural{}
	}
	for _, gvkp := range gvkps {
		if _, ok := r.Map[gvkp.Group]; !ok {
			r.Map[gvkp.Group] = map[string][]discovery.KindPlural{}
		}
		if _, ok := r.Map[gvkp.Group][gvkp.Version]; !ok {
			r.Map[gvkp.Group][gvkp.Version] = []discovery.KindPlural{}
		}
		r.Map[gvkp.Group][gvkp.Version] = append(r.Map[gvkp.Group][gvkp.Version], discovery.KindPlural{Kind: gvkp.Kind, Plural: gvkp.Plural})
	}
}

// RemoveFromMap removes the given GVKs from the cache.
func (r *CRDiscoverer) RemoveFromMap(gvkps ...discovery.GroupVersionKindPlural) {
	for _, gvkp := range gvkps {
		if _, ok := r.Map[gvkp.Group]; !ok {
			continue
		}
		if _, ok := r.Map[gvkp.Group][gvkp.Version]; !ok {
			continue
		}
		for i, el := range r.Map[gvkp.Group][gvkp.Version] {
			if el.Kind == gvkp.Kind {
				if len(r.Map[gvkp.Group][gvkp.Version]) == 1 {
					delete(r.Map[gvkp.Group], gvkp.Version)
					if len(r.Map[gvkp.Group]) == 0 {
						delete(r.Map, gvkp.Group)
					}
					break
				}
				r.Map[gvkp.Group][gvkp.Version] = append(r.Map[gvkp.Group][gvkp.Version][:i], r.Map[gvkp.Group][gvkp.Version][i+1:]...)
				break
			}
		}
	}
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
		AddFunc: func(obj interface{}) {
			objSpec := obj.(*unstructured.Unstructured).Object["spec"].(map[string]interface{})
			for _, version := range objSpec["versions"].([]interface{}) {
				g := objSpec["group"].(string)
				v := version.(map[string]interface{})["name"].(string)
				k := objSpec["names"].(map[string]interface{})["kind"].(string)
				p := objSpec["names"].(map[string]interface{})["plural"].(string)
				gotGVKP := discovery.GroupVersionKindPlural{
					GroupVersionKind: schema.GroupVersionKind{
						Group:   g,
						Version: v,
						Kind:    k,
					},
					Plural: p,
				}
				r.AppendToMap(gotGVKP)
				r.SafeWrite(func() {
					r.WasUpdated = true
				})
			}
			r.SafeWrite(func() {
				r.CRDsAddEventsCounter.Inc()
				r.CRDsCacheCountGauge.Inc()
			})
		},
		DeleteFunc: func(obj interface{}) {
			objSpec := obj.(*unstructured.Unstructured).Object["spec"].(map[string]interface{})
			for _, version := range objSpec["versions"].([]interface{}) {
				g := objSpec["group"].(string)
				v := version.(map[string]interface{})["name"].(string)
				k := objSpec["names"].(map[string]interface{})["kind"].(string)
				p := objSpec["names"].(map[string]interface{})["plural"].(string)
				gotGVKP := discovery.GroupVersionKindPlural{
					GroupVersionKind: schema.GroupVersionKind{
						Group:   g,
						Version: v,
						Kind:    k,
					},
					Plural: p,
				}
				r.RemoveFromMap(gotGVKP)
				r.SafeWrite(func() {
					r.WasUpdated = true
				})
			}
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
		for range ctx.Done() {
			klog.InfoS("context cancelled, stopping discovery")
			close(stopper)
			return
		}
	}()
	go informer.Run(stopper)
	return nil
}

// ResolveGVKToGVKPs resolves the variable VKs to a GVK list, based on the current cache.
func (r *CRDiscoverer) ResolveGVKToGVKPs(gvk schema.GroupVersionKind) (resolvedGVKPs []discovery.GroupVersionKindPlural, err error) { // nolint:revive
	g := gvk.Group
	v := gvk.Version
	k := gvk.Kind
	if g == "" || g == "*" {
		return nil, fmt.Errorf("group is required in the defined GVK %v", gvk)
	}
	hasVersion := v != "" && v != "*"
	hasKind := k != "" && k != "*"
	// No need to resolve, return.
	if hasVersion && hasKind {
		var p string
		for _, el := range r.Map[g][v] {
			if el.Kind == k {
				p = el.Plural
				break
			}
		}
		return []discovery.GroupVersionKindPlural{
			{
				GroupVersionKind: schema.GroupVersionKind{
					Group:   g,
					Version: v,
					Kind:    k,
				},
				Plural: p,
			},
		}, nil
	}
	if hasVersion && !hasKind {
		kinds := r.Map[g][v]
		for _, el := range kinds {
			resolvedGVKPs = append(resolvedGVKPs, discovery.GroupVersionKindPlural{
				GroupVersionKind: schema.GroupVersionKind{
					Group:   g,
					Version: v,
					Kind:    el.Kind,
				},
				Plural: el.Plural,
			})
		}
	}
	if !hasVersion && hasKind {
		versions := r.Map[g]
		for version, kinds := range versions {
			for _, el := range kinds {
				if el.Kind == k {
					resolvedGVKPs = append(resolvedGVKPs, discovery.GroupVersionKindPlural{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   g,
							Version: version,
							Kind:    k,
						},
						Plural: el.Plural,
					})
				}
			}
		}
	}
	if !hasVersion && !hasKind {
		versions := r.Map[g]
		for version, kinds := range versions {
			for _, el := range kinds {
				resolvedGVKPs = append(resolvedGVKPs, discovery.GroupVersionKindPlural{
					GroupVersionKind: schema.GroupVersionKind{
						Group:   g,
						Version: version,
						Kind:    el.Kind,
					},
					Plural: el.Plural,
				})
			}
		}
	}
	return
}

// PollForCacheUpdates polls the cache for updates and updates the stores accordingly.
func (r *CRDiscoverer) PollForCacheUpdates(
	ctx context.Context,
	opts *options.Options,
	storeBuilder *store.Builder,
	m *metricshandler.MetricsHandler,
	factoryGenerator func() ([]customresource.RegistryFactory, error),
) {
	// The interval at which we will check the cache for updates.
	t := time.NewTicker(Interval)
	// Track previous context to allow refreshing cache.
	olderContext, olderCancel := context.WithCancel(ctx)
	// Prevent context leak (kill the last metric handler instance).
	defer olderCancel()
	generateMetrics := func() {
		// Get families for discovered factories.
		customFactories, err := factoryGenerator()
		if err != nil {
			klog.ErrorS(err, "failed to update custom resource stores")
		}
		// Update the list of enabled custom resources.
		var enabledCustomResources []string
		for _, factory := range customFactories {
			gvrString := customresource.GVRFromType(factory.Name(), factory.ExpectedType()).String()
			enabledCustomResources = append(enabledCustomResources, gvrString)
		}
		// Create clients for discovered factories.
		discoveredCustomResourceClients, err := customresource.CreateCustomResourceClients(opts.Apiserver, opts.Kubeconfig, customFactories...)
		if err != nil {
			klog.ErrorS(err, "failed to update custom resource stores")
		}
		// Update the store builder with the new clients.
		storeBuilder.WithCustomResourceClients(discoveredCustomResourceClients)
		// Inject families' constructors to the existing set of stores.
		storeBuilder.WithCustomResourceStoreFactories(customFactories...)
		// Update the store builder with the new custom resources.
		if err := storeBuilder.WithEnabledResources(enabledCustomResources); err != nil {
			klog.ErrorS(err, "failed to update custom resource stores")
		}
		// Configure the generation function for the custom resource stores.
		storeBuilder.WithGenerateCustomResourceStoresFunc(storeBuilder.DefaultGenerateCustomResourceStoresFunc())
		// Reset the flag, if there were no errors. Else, we'll try again on the next tick.
		// Keep retrying if there were errors.
		r.SafeWrite(func() {
			r.WasUpdated = false
		})
		// Run the metrics handler with updated configs.
		olderContext, olderCancel = context.WithCancel(ctx)
		go func() {
			// Blocks indefinitely until the unbuffered context is cancelled to serve metrics for that duration.
			err = m.Run(olderContext)
			if err != nil {
				// Check if context was cancelled.
				select {
				case <-olderContext.Done():
					// Context cancelled, don't really need to log this though.
				default:
					klog.ErrorS(err, "failed to run metrics handler")
				}
			}
		}()
	}
	go func() {
		for range t.C {
			select {
			case <-ctx.Done():
				klog.InfoS("context cancelled")
				t.Stop()
				return
			default:
				// Check if cache has been updated.
				shouldGenerateMetrics := false
				r.SafeRead(func() {
					shouldGenerateMetrics = r.WasUpdated
				})
				if shouldGenerateMetrics {
					olderCancel()
					generateMetrics()
					klog.InfoS("discovery finished, cache updated")
				}
			}
		}
	}()
}
