/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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

package customresourcestate

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/internal/store"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

// customResourceMetrics is an implementation of the customresource.RegistryFactory
// interface which provides metrics for custom resources defined in a configuration file.
type customResourceMetrics struct {
	MetricNamePrefix string
	GroupVersionKind schema.GroupVersionKind
	ResourceName     string
	Families         []compiledFamily
}

var _ customresource.RegistryFactory = &customResourceMetrics{}
var _ customresource.ContextListWatchFactory = customResourceMetrics{}

// NewCustomResourceMetrics creates a customresource.RegistryFactory from a configuration object.
func NewCustomResourceMetrics(resource Resource) (customresource.RegistryFactory, error) {
	compiled, err := compile(resource)
	if err != nil {
		return nil, err
	}
	gvk := schema.GroupVersionKind(resource.GroupVersionKind)
	return &customResourceMetrics{
		MetricNamePrefix: resource.GetMetricNamePrefix(),
		GroupVersionKind: gvk,
		Families:         compiled,
		ResourceName:     resource.GetResourceName(),
	}, nil
}

func (s customResourceMetrics) Name() string {
	return s.ResourceName
}

func (s customResourceMetrics) CreateClient(cfg *rest.Config) (any, error) {
	c, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return c.Resource(schema.GroupVersionResource{
		Group:    s.GroupVersionKind.Group,
		Version:  s.GroupVersionKind.Version,
		Resource: s.ResourceName,
	}), nil
}

func (s customResourceMetrics) MetricFamilyGenerators() (result []generator.FamilyGenerator) {
	klog.InfoS("Custom resource state added metrics", "familyNames", s.names())
	for _, f := range s.Families {
		result = append(result, famGen(f))
	}

	return result
}

func (s customResourceMetrics) ExpectedType() any {
	u := unstructured.Unstructured{}
	u.SetGroupVersionKind(s.GroupVersionKind)
	return &u
}

func (s customResourceMetrics) ListWatchWithContext(ctx context.Context, customResourceClient any, ns string, fieldSelector string) cache.ListerWatcher {
	client := customResourceClient.(dynamic.NamespaceableResourceInterface)
	return store.DynamicNamespacedListWatch(ctx, client, ns, fieldSelector)
}

func (s customResourceMetrics) ListWatch(customResourceClient any, ns string, fieldSelector string) cache.ListerWatcher {
	return s.ListWatchWithContext(context.Background(), customResourceClient, ns, fieldSelector)
}

func (s customResourceMetrics) names() (names []string) {
	for _, family := range s.Families {
		names = append(names, family.Name)
	}
	return names
}
