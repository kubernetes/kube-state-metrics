/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

package store

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/customresource"
)

// buildListWatchFunc creates a lister/watcher bound to the builder generation context.
type buildListWatchFunc func(ctx context.Context, kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher

// ListWatchContext returns ctx or context.Background when ctx is nil.
func ListWatchContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

// listWatchFromBuildContext binds b.ctx into legacy BuildStoresFunc list-watch callbacks.
func listWatchFromBuildContext(b *Builder, f buildListWatchFunc) func(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return func(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
		return f(ListWatchContext(b.ctx), kubeClient, ns, fieldSelector)
	}
}

// customResourceListWatchFromBuildContext binds listCtx while preserving RegistryFactory.ListWatch.
func customResourceListWatchFromBuildContext(ctx context.Context, factory customresource.RegistryFactory) func(customResourceClient any, ns string, fieldSelector string) cache.ListerWatcher {
	return func(customResourceClient any, ns string, fieldSelector string) cache.ListerWatcher {
		ctx = ListWatchContext(ctx)
		if ctxFactory, ok := factory.(customresource.ContextListWatchFactory); ok {
			return ctxFactory.ListWatchWithContext(ctx, customResourceClient, ns, fieldSelector)
		}
		return factory.ListWatch(customResourceClient, ns, fieldSelector)
	}
}

// DynamicNamespacedListWatch lists/watches a dynamic resource in one namespace.
func DynamicNamespacedListWatch(ctx context.Context, client dynamic.NamespaceableResourceInterface, ns string, fieldSelector string) cache.ListerWatcher {
	ctx = ListWatchContext(ctx)
	api := client.Namespace(ns)
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return api.List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return api.Watch(ctx, options)
		},
	}
}
