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
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	internalstore "k8s.io/kube-state-metrics/internal/store"
	ksmtypes "k8s.io/kube-state-metrics/pkg/builder/types"
	"k8s.io/kube-state-metrics/pkg/options"
)

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
func (b *Builder) WithMetrics(r *prometheus.Registry) {
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

// WithVPAClient sets the vpaClient property of a Builder so that the verticalpodautoscaler collector can query VPA objects.
func (b *Builder) WithVPAClient(c vpaclientset.Interface) {
	b.internal.WithVPAClient(c)
}

// WithAllowDenyList configures the allow or denylisted metric to be exposed
// by the store build by the Builder.
func (b *Builder) WithAllowDenyList(l ksmtypes.AllowDenyLister) {
	b.internal.WithAllowDenyList(l)
}

// WithGenerateStoreFunc configures a constom generate store function
func (b *Builder) WithGenerateStoreFunc(f ksmtypes.BuildStoreFunc) {
	b.internal.WithGenerateStoreFunc(f)
}

// DefaultGenerateStoreFunc returns default buildStore function
func (b *Builder) DefaultGenerateStoreFunc() ksmtypes.BuildStoreFunc {
	return b.internal.DefaultGenerateStoreFunc()
}

// Build initializes and registers all enabled stores.
func (b *Builder) Build() []cache.Store {
	return b.internal.Build()
}
