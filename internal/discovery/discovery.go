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
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/internal/store"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	"k8s.io/kube-state-metrics/v2/pkg/metricshandler"
	"k8s.io/kube-state-metrics/v2/pkg/options"
	"k8s.io/kube-state-metrics/v2/pkg/util"
)

// Interval is the time interval between two cache sync checks.
const Interval = 3 * time.Second

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
				gotGVKP := groupVersionKindPlural{
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
				gotGVKP := groupVersionKindPlural{
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
func (r *CRDiscoverer) ResolveGVKToGVKPs(gvk schema.GroupVersionKind) (resolvedGVKPs []groupVersionKindPlural, err error) { // nolint:revive
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
		for _, el := range r.Map[g][v] {
			if el.Kind == k {
				return []groupVersionKindPlural{
					{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   g,
							Version: v,
							Kind:    k,
						},
						Plural: el.Plural,
					},
				}, nil
			}
		}
	}
	if hasVersion && !hasKind {
		kinds := r.Map[g][v]
		for _, el := range kinds {
			resolvedGVKPs = append(resolvedGVKPs, groupVersionKindPlural{
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
					resolvedGVKPs = append(resolvedGVKPs, groupVersionKindPlural{
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
				resolvedGVKPs = append(resolvedGVKPs, groupVersionKindPlural{
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
	generateMetrics := func() {
		// Get families for discovered factories.
		customFactories, err := factoryGenerator()
		if err != nil {
			klog.ErrorS(err, "failed to update custom resource stores")
		}
		// Update the list of enabled custom resources.
		var enabledCustomResources []string
		for _, factory := range customFactories {
			gvr, err := util.GVRFromType(factory.Name(), factory.ExpectedType())
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
		}
		// Create clients for discovered factories.
		discoveredCustomResourceClients, err := util.CreateCustomResourceClients(opts.Apiserver, opts.Kubeconfig, customFactories...)
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
		// Update metric handler with the new configs.
		m.BuildWriters(ctx)
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
					generateMetrics()
					klog.InfoS("discovery finished, cache updated")
				}
			}
		}
	}()
}
