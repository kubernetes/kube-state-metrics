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

package customresource

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

// RegistryFactory is a registry interface for a CustomResourceStore.
// Users who want to extend the kube-state-metrics to support Custom Resource metrics should
// implement this interface.
type RegistryFactory interface {
	// Name returns the name of custom resource.
	//
	// Example:
	//
	// func (f *FooFactory) Name() string {
	//	return "foos"
	// }
	Name() string

	// CreateClient creates a new custom resource client for the given config.
	//
	// Example:
	//
	// func (f *FooFactory) CreateClient(cfg *rest.Config) (interface{}, error) {
	// 	return clientset.NewForConfig(cfg)
	// }
	CreateClient(cfg *rest.Config) (interface{}, error)

	// MetricFamilyGenerators returns the metric family generators to generate metric families with a
	// Kubernetes custom resource object.
	//
	// Example:
	//
	// func (f *FooFactory) MetricFamilyGenerators() []generator.FamilyGenerator {
	//	return []generator.FamilyGenerator{
	//		*generator.NewFamilyGeneratorWithStability(
	//			"kube_foo_spec_replicas",
	//			"Number of desired replicas for a foo.",
	//			metric.Gauge,
	//			basemetrics.ALPHA,
	//			"",
	//			wrapFooFunc(func(f *samplev1alpha1.Foo) *metric.Family {
	//				return &metric.Family{
	//					Metrics: []*metric.Metric{
	//						{
	//							Value: float64(*f.Spec.Replicas),
	//						},
	//					},
	//				}
	//			}),
	//		),
	//		*generator.NewFamilyGeneratorWithStability(
	//			"kube_foo_status_replicas_available",
	//			"The number of available replicas per foo.",
	//			metric.Gauge,
	//			basemetrics.ALPHA,
	//			"",
	//			wrapFooFunc(func(f *samplev1alpha1.Foo) *metric.Family {
	//				return &metric.Family{
	//					Metrics: []*metric.Metric{
	//						{
	//							Value: float64(f.Status.AvailableReplicas),
	//						},
	//					},
	//				}
	//			}),
	//		),
	//	}
	// }
	MetricFamilyGenerators() []generator.FamilyGenerator

	// ExpectedType returns a pointer to an empty custom resource object.
	//
	// Example:
	//
	// func (f *FooFactory) ExpectedType() interface{} {
	//	return &samplev1alpha1.Foo{}
	// }
	ExpectedType() interface{}

	// ListWatch constructs a cache.ListerWatcher of the custom resource object.
	//
	// Example:
	//
	// func (f *FooFactory) ListWatch(customResourceClient interface{}, ns string, fieldSelector string) cache.ListerWatcher {
	// 	client := customResourceClient.(*clientset.Clientset)
	//	return &cache.ListWatch{
	//		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
	//			opts.FieldSelector = fieldSelector
	//			return client.SamplecontrollerV1alpha1().Foos(ns).List(context.Background(), opts)
	//		},
	//		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
	//			opts.FieldSelector = fieldSelector
	//			return client.SamplecontrollerV1alpha1().Foos(ns).Watch(context.Background(), opts)
	//		},
	//	}
	// }
	ListWatch(customResourceClient interface{}, ns string, fieldSelector string) cache.ListerWatcher
}
