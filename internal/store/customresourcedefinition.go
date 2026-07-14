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

	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

var (
	descCustomResourceDefinitionAnnotationsName     = "kube_customresourcedefinition_annotations"
	descCustomResourceDefinitionAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descCustomResourceDefinitionLabelsName          = "kube_customresourcedefinition_labels"
	descCustomResourceDefinitionLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descCustomResourceDefinitionLabelsDefaultLabels = []string{"customresourcedefinition"}
)

func customResourceDefinitionMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_customresourcedefinition_info",
			"Information about a CustomResourceDefinition.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapCustomResourceDefinitionFunc(func(crd *apiextensionsv1.CustomResourceDefinition) *metric.Family {
				m := metric.Metric{
					LabelKeys:   []string{"group", "kind", "scope"},
					LabelValues: []string{crd.Spec.Group, crd.Spec.Names.Kind, string(crd.Spec.Scope)},
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_customresourcedefinition_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapCustomResourceDefinitionFunc(func(crd *apiextensionsv1.CustomResourceDefinition) *metric.Family {
				ms := []*metric.Metric{}
				if !crd.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(crd.CreationTimestamp.Unix()),
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descCustomResourceDefinitionAnnotationsName,
			descCustomResourceDefinitionAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapCustomResourceDefinitionFunc(func(crd *apiextensionsv1.CustomResourceDefinition) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", crd.Annotations, allowAnnotationsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   annotationKeys,
							LabelValues: annotationValues,
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descCustomResourceDefinitionLabelsName,
			descCustomResourceDefinitionLabelsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapCustomResourceDefinitionFunc(func(crd *apiextensionsv1.CustomResourceDefinition) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", crd.Labels, allowLabelsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
							Value:       1,
						},
					},
				}
			}),
		),
	}
}

func wrapCustomResourceDefinitionFunc(f func(*apiextensionsv1.CustomResourceDefinition) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		crd := obj.(*apiextensionsv1.CustomResourceDefinition)

		metricFamily := f(crd)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descCustomResourceDefinitionLabelsDefaultLabels, []string{crd.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createCustomResourceDefinitionListWatch(apiextensionsClient apiextensionsclientset.Interface, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return apiextensionsClient.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return apiextensionsClient.ApiextensionsV1().CustomResourceDefinitions().Watch(context.TODO(), opts)
		},
	}
}
