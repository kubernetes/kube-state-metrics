/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

package types

import (
	"context"

	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"

	"github.com/prometheus/client_golang/prometheus"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

// BuilderInterface represent all methods that a Builder should implements
type BuilderInterface interface {
	WithMetrics(r prometheus.Registerer)
	WithEnabledResources(c []string) error
	WithNamespaces(n options.NamespaceList)
	WithFieldSelectorFilter(fieldSelectors string)
	WithSharding(shard int32, totalShards int)
	WithContext(ctx context.Context)
	WithKubeClient(c clientset.Interface)
	WithCustomResourceClients(cs map[string]interface{})
	WithUsingAPIServerCache(u bool)
	WithFamilyGeneratorFilter(l generator.FamilyGeneratorFilter)
	WithAllowAnnotations(a map[string][]string) error
	WithAllowLabels(l map[string][]string) error
	WithGenerateStoresFunc(f BuildStoresFunc)
	DefaultGenerateStoresFunc() BuildStoresFunc
	DefaultGenerateCustomResourceStoresFunc() BuildCustomResourceStoresFunc
	WithCustomResourceStoreFactories(fs ...customresource.RegistryFactory)
	Build() metricsstore.MetricsWriterList
	BuildStores() [][]cache.Store
	WithGenerateCustomResourceStoresFunc(f BuildCustomResourceStoresFunc)
}

// BuildStoresFunc function signature that is used to return a list of cache.Store
type BuildStoresFunc func(metricFamilies []generator.FamilyGenerator,
	expectedType interface{},
	listWatchFunc func(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher,
	useAPIServerCache bool, limit int64,
) []cache.Store

// BuildCustomResourceStoresFunc function signature that is used to return a list of custom resource cache.Store
type BuildCustomResourceStoresFunc func(resourceName string,
	metricFamilies []generator.FamilyGenerator,
	expectedType interface{},
	listWatchFunc func(customResourceClient interface{}, ns string, fieldSelector string) cache.ListerWatcher,
	useAPIServerCache bool, limit int64,
) []cache.Store

// AllowDenyLister interface for AllowDeny lister that can allow or exclude metrics by there names
type AllowDenyLister interface {
	IsIncluded(string) bool
	IsExcluded(string) bool
}
