/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descNamespaceAnnotationsName     = "kube_namespace_annotations"
	descNamespaceAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descNamespaceLabelsName          = "kube_namespace_labels"
	descNamespaceLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descNamespaceLabelsDefaultLabels = []string{"namespace"}
)

func namespaceMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_namespace_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapNamespaceFunc(func(n *v1.Namespace) *metric.Family {
				ms := []*metric.Metric{}
				if !n.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(n.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descNamespaceAnnotationsName,
			descNamespaceAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapNamespaceFunc(func(n *v1.Namespace) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}

				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", n.Annotations, allowAnnotationsList)
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
			descNamespaceLabelsName,
			descNamespaceLabelsHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapNamespaceFunc(func(n *v1.Namespace) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", n.Labels, allowLabelsList)
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
		*generator.NewFamilyGeneratorWithStability(
			"kube_namespace_status_phase",
			"kubernetes namespace status phase.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapNamespaceFunc(func(n *v1.Namespace) *metric.Family {
				ms := []*metric.Metric{
					{
						LabelValues: []string{string(v1.NamespaceActive)},
						Value:       boolFloat64(n.Status.Phase == v1.NamespaceActive),
					},
					{
						LabelValues: []string{string(v1.NamespaceTerminating)},
						Value:       boolFloat64(n.Status.Phase == v1.NamespaceTerminating),
					},
				}

				for _, metric := range ms {
					metric.LabelKeys = []string{"phase"}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_namespace_status_condition",
			"The condition of a namespace.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapNamespaceFunc(func(n *v1.Namespace) *metric.Family {
				ms := make([]*metric.Metric, len(n.Status.Conditions)*len(conditionStatuses))
				for i, c := range n.Status.Conditions {
					conditionMetrics := addConditionMetrics(c.Status)

					for j, m := range conditionMetrics {
						metric := m

						metric.LabelKeys = []string{"condition", "status"}
						metric.LabelValues = append([]string{string(c.Type)}, metric.LabelValues...)

						ms[i*len(conditionStatuses)+j] = metric
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
}

func wrapNamespaceFunc(f func(*v1.Namespace) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		namespace := obj.(*v1.Namespace)

		metricFamily := f(namespace)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descNamespaceLabelsDefaultLabels, []string{namespace.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createNamespaceListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcherWithContext {
	return &cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Namespaces().List(context.TODO(), opts)
		},
		WatchFuncWithContext: func(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Namespaces().Watch(context.TODO(), opts)
		},
	}
}
