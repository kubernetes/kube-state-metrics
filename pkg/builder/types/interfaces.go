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

	"github.com/prometheus/client_golang/prometheus"
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

// BuilderInterface represent all methods that a Builder should implements
type BuilderInterface interface {
	WithMetrics(r prometheus.Registerer)
	WithEnabledResources(c []string) error
	WithNamespaces(n options.NamespaceList)
	WithSharding(shard int32, totalShards int)
	WithContext(ctx context.Context)
	WithKubeClient(c clientset.Interface)
	WithVPAClient(c vpaclientset.Interface)
	WithAllowDenyList(l AllowDenyLister)
	WithGenerateStoreFunc(f BuildStoreFunc)
	WithAllowLabels(l map[string][]string)
	DefaultGenerateStoreFunc() BuildStoreFunc
	Build() []cache.Store
}

// BuildStoreFunc function signature that is use to returns a cache.Store
type BuildStoreFunc func(metricFamilies []generator.FamilyGenerator,
	expectedType interface{},
	listWatchFunc func(kubeClient clientset.Interface, ns string) cache.ListerWatcher,
) cache.Store

// AllowDenyLister interface for AllowDeny lister that can allow or exclude metrics by there names
type AllowDenyLister interface {
	IsIncluded(string) bool
	IsExcluded(string) bool
}
