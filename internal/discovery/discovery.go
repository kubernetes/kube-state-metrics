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
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sdiscovery "k8s.io/client-go/discovery"
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
	err := r.startCRDDiscovery(ctx, config)
	if err != nil {
		return err
	}
	err = r.startAPIServiceDiscovery(ctx, config)
	if err != nil {
		return err
	}
	return nil
}

func (r *CRDiscoverer) runInformer(ctx context.Context, informer cache.SharedIndexInformer, extractor extractor) error {
	stopper := make(chan struct{})
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			sourceID := extractor.SourceID(obj)
			resources := extractor.ExtractGVKs(obj)
			r.UpdateSource(sourceID, resources)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			sourceID := extractor.SourceID(newObj)
			resources := extractor.ExtractGVKs(newObj)
			r.UpdateSource(sourceID, resources)
		},
		DeleteFunc: func(obj interface{}) {
			sourceID := extractor.SourceID(obj)
			r.DeleteSource(sourceID)
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

func (r *CRDiscoverer) startCRDDiscovery(ctx context.Context, config *rest.Config) error {
	client := dynamic.NewForConfigOrDie(config)
	factory := dynamicinformer.NewFilteredDynamicInformer(client, schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}, "", 0, nil, nil)

	extractor := &crdExtractor{}

	return r.runInformer(ctx, factory.Informer(), extractor)
}

func (r *CRDiscoverer) startAPIServiceDiscovery(ctx context.Context, config *rest.Config) error {
	client := dynamic.NewForConfigOrDie(config)
	factory := dynamicinformer.NewFilteredDynamicInformer(client, schema.GroupVersionResource{
		Group:    "apiregistration.k8s.io",
		Version:  "v1",
		Resource: "apiservices",
	}, "", 0, nil, nil)

	discoveryClient := k8sdiscovery.NewDiscoveryClientForConfigOrDie(config)
	extractor := &apiServiceExtractor{
		discoveryClient: discoveryClient,
	}

	return r.runInformer(ctx, factory.Informer(), extractor)
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
		// Update metric handler with the new configs.
		m.BuildWriters(ctx)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				klog.InfoS("context cancelled")
				t.Stop()
				return
			case <-t.C:
				// Check if cache has been updated and reset the flag.
				if r.CheckAndResetUpdated() {
					generateMetrics()
					klog.InfoS("discovery finished, cache updated")
				}
			}
		}
	}()
}
