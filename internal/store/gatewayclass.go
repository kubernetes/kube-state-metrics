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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayapiclientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

var (
	descGatewayClassAnnotationsName     = "kube_gatewayclass_annotations"
	descGatewayClassAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descGatewayClassLabelsName          = "kube_gatewayclass_labels" //nolint:gosec
	descGatewayClassLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descGatewayClassLabelsDefaultLabels = []string{"gatewayclass"}
)

func gatewayClassMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_gatewayclass_info",
			"Information about gatewayclass.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapGatewayClassFunc(func(s *gatewayapiv1.GatewayClass) *metric.Family {

				m := metric.Metric{
					LabelKeys:   []string{"controller"},
					LabelValues: []string{string(s.Spec.ControllerName)},
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_gatewayclass_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapGatewayClassFunc(func(s *gatewayapiv1.GatewayClass) *metric.Family {
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
			descGatewayClassAnnotationsName,
			descGatewayClassAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapGatewayClassFunc(func(s *gatewayapiv1.GatewayClass) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
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
			descGatewayClassLabelsName,
			descGatewayClassLabelsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapGatewayClassFunc(func(s *gatewayapiv1.GatewayClass) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
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

func wrapGatewayClassFunc(f func(*gatewayapiv1.GatewayClass) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		gatewayClass := obj.(*gatewayapiv1.GatewayClass)

		metricFamily := f(gatewayClass)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descGatewayClassLabelsDefaultLabels, []string{gatewayClass.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createGatewayClassListWatch(kubeClient gatewayapiclientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.GatewayV1().GatewayClasses().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.GatewayV1().GatewayClasses().Watch(context.TODO(), opts)
		},
	}
}
