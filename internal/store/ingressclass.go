/*
Copyright 2022 The Kubernetes Authors All rights reserved.
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

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descIngressClassAnnotationsName     = "kube_ingressclass_annotations"
	descIngressClassAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descIngressClassLabelsName          = "kube_ingressclass_labels"
	descIngressClassLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descIngressClassLabelsDefaultLabels = []string{"ingressclass"}
)

func ingressClassMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_ingressclass_info",
			"Information about ingressclass.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapIngressClassFunc(func(s *networkingv1.IngressClass) *metric.Family {

				m := metric.Metric{
					LabelKeys:   []string{"controller"},
					LabelValues: []string{s.Spec.Controller},
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_ingressclass_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapIngressClassFunc(func(s *networkingv1.IngressClass) *metric.Family {
				ms := []*metric.Metric{}
				if !s.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(s.CreationTimestamp.Unix()),
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descIngressClassAnnotationsName,
			descIngressClassAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapIngressClassFunc(func(s *networkingv1.IngressClass) *metric.Family {
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", s.Annotations, allowAnnotationsList)
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
			descIngressClassLabelsName,
			descIngressClassLabelsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapIngressClassFunc(func(s *networkingv1.IngressClass) *metric.Family {
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", s.Labels, allowLabelsList)
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

func wrapIngressClassFunc(f func(*networkingv1.IngressClass) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		ingressClass := obj.(*networkingv1.IngressClass)

		metricFamily := f(ingressClass)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descIngressClassLabelsDefaultLabels, []string{ingressClass.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createIngressClassListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.NetworkingV1().IngressClasses().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.NetworkingV1().IngressClasses().Watch(context.TODO(), opts)
		},
	}
}
