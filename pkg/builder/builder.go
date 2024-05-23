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

package builder

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	internalstore "k8s.io/kube-state-metrics/v2/internal/store"
	ksmtypes "k8s.io/kube-state-metrics/v2/pkg/builder/types"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

// Make sure the public Builder implements the public BuilderInterface.
// New internal Builder methods should be added to the public BuilderInterface.
var _ ksmtypes.BuilderInterface = &Builder{}

// Builder helps to build store. It follows the builder pattern
// (https://en.wikipedia.org/wiki/Builder_pattern).
type Builder struct {
	internal ksmtypes.BuilderInterface
}

// NewBuilder returns a new builder.
func NewBuilder() *Builder {
	b := &Builder{
		internal: internalstore.NewBuilder(),
	}
	return b
}

// WithMetrics sets the metrics property of a Builder.
func (b *Builder) WithMetrics(r prometheus.Registerer) {
	b.internal.WithMetrics(r)
}

// WithEnabledResources sets the enabledResources property of a Builder.
func (b *Builder) WithEnabledResources(c []string) error {
	return b.internal.WithEnabledResources(c)
}

// WithNamespaces sets the namespaces property of a Builder.
func (b *Builder) WithNamespaces(n options.NamespaceList) {
	b.internal.WithNamespaces(n)
}

// WithFieldSelectorFilter sets the fieldSelector property of a Builder.
func (b *Builder) WithFieldSelectorFilter(fieldSelectorFilter string) {
	b.internal.WithFieldSelectorFilter(fieldSelectorFilter)
}

// WithSharding sets the shard and totalShards property of a Builder.
func (b *Builder) WithSharding(shard int32, totalShards int) {
	b.internal.WithSharding(shard, totalShards)
}

// WithContext sets the ctx property of a Builder.
func (b *Builder) WithContext(ctx context.Context) {
	b.internal.WithContext(ctx)
}

// WithKubeClient sets the kubeClient property of a Builder.
func (b *Builder) WithKubeClient(c clientset.Interface) {
	b.internal.WithKubeClient(c)
}

// WithCustomResourceClients sets the customResourceClients property of a Builder.
func (b *Builder) WithCustomResourceClients(cs map[string]interface{}) {
	b.internal.WithCustomResourceClients(cs)
}

// WithUsingAPIServerCache configures whether using APIServer cache or not.
func (b *Builder) WithUsingAPIServerCache(u bool) {
	b.internal.WithUsingAPIServerCache(u)
}

// WithFamilyGeneratorFilter configures the family generator filter which decides which
// metrics are to be exposed by the store build by the Builder.
func (b *Builder) WithFamilyGeneratorFilter(l generator.FamilyGeneratorFilter) {
	b.internal.WithFamilyGeneratorFilter(l)
}

// WithAllowAnnotations configures which annotations can be returned for metrics
func (b *Builder) WithAllowAnnotations(annotations map[string][]string) error {
	return b.internal.WithAllowAnnotations(annotations)
}

// WithAllowLabels configures which labels can be returned for metrics
func (b *Builder) WithAllowLabels(l map[string][]string) error {
	return b.internal.WithAllowLabels(l)
}

// WithGenerateStoresFunc configures a custom generate store function
func (b *Builder) WithGenerateStoresFunc(f ksmtypes.BuildStoresFunc) {
	b.internal.WithGenerateStoresFunc(f)
}

// DefaultGenerateStoresFunc returns default buildStore function
func (b *Builder) DefaultGenerateStoresFunc() ksmtypes.BuildStoresFunc {
	return b.internal.DefaultGenerateStoresFunc()
}

// DefaultGenerateCustomResourceStoresFunc returns default buildStores function
func (b *Builder) DefaultGenerateCustomResourceStoresFunc() ksmtypes.BuildCustomResourceStoresFunc {
	return b.internal.DefaultGenerateCustomResourceStoresFunc()
}

// WithCustomResourceStoreFactories returns configures a custom resource stores factory
func (b *Builder) WithCustomResourceStoreFactories(fs ...customresource.RegistryFactory) {
	b.internal.WithCustomResourceStoreFactories(fs...)
}

// Build initializes and registers all enabled stores.
// Returns metric writers.
func (b *Builder) Build() metricsstore.MetricsWriterList {
	return b.internal.Build()
}

// BuildStores initializes and registers all enabled stores.
// Returns metric stores.
func (b *Builder) BuildStores() [][]cache.Store {
	return b.internal.BuildStores()
}

// WithGenerateCustomResourceStoresFunc configures a custom generate custom resource store function
func (b *Builder) WithGenerateCustomResourceStoresFunc(f ksmtypes.BuildCustomResourceStoresFunc) {
	b.internal.WithGenerateCustomResourceStoresFunc(f)
}
