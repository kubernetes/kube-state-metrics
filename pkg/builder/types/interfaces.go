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
	"errors"
	"time"

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
	WithCustomResourceClients(cs map[string]any)
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

// CustomResourceReplacer replaces the discovered custom resource set while preserving built-in names.
type CustomResourceReplacer interface {
	ReplaceEnabledCustomResources(c []string) error
}

// ErrStoreSyncTimeout is returned when the configured store sync timeout elapses
// before every reflector has completed its initial list. It is distinct from a
// deadline on the generation context: only this error permits installing the
// generation that timed out.
var ErrStoreSyncTimeout = errors.New("store sync timed out")

// ErrReflectorStopped is returned when a reflector stops before its initial list.
// The generation may still be installed; reflectors that are still running keep
// filling their stores.
var ErrReflectorStopped = errors.New("reflector stopped before initial sync")

// StoreSyncBuilder waits for reflector stores to sync after Build().
type StoreSyncBuilder interface {
	// WaitForStoresSync blocks until every reflector from the latest Build has
	// completed its initial list. A nil error means every reflector listed.
	// ErrStoreSyncTimeout and ErrReflectorStopped are the only non-nil results
	// that still describe a generation eligible for installation. A canceled or
	// expired ctx is returned as ctx.Err() and must not be installed.
	WaitForStoresSync(ctx context.Context, timeout time.Duration) error
}

// BuildStoresFunc function signature that is used to return a list of cache.Store
type BuildStoresFunc func(metricFamilies []generator.FamilyGenerator,
	expectedType any,
	listWatchFunc func(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher,
	useAPIServerCache bool, limit int64,
) []cache.Store

// BuildCustomResourceStoresFunc function signature that is used to return a list of custom resource cache.Store
type BuildCustomResourceStoresFunc func(resourceName string,
	metricFamilies []generator.FamilyGenerator,
	expectedType any,
	listWatchFunc func(customResourceClient any, ns string, fieldSelector string) cache.ListerWatcher,
	useAPIServerCache bool, limit int64,
) []cache.Store

// AllowDenyLister interface for AllowDeny lister that can allow or exclude metrics by there names
type AllowDenyLister interface {
	IsIncluded(string) bool
	IsExcluded(string) bool
}
