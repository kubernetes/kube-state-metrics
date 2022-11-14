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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descCustomResourceLabelsName = "kube_%s_labels"
	descCustomResourceLabelsHelp = "Kubernetes labels converted to Prometheus labels."
)

func customResourceMetricFamilies(customResourceName string, _, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			fmt.Sprintf(descCustomResourceLabelsName, customResourceName),
			descCustomResourceLabelsHelp,
			metric.Gauge,
			"",
			wrapCustomResourceFunc(func(obj metav1.Object) *metric.Family {
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", obj.GetLabels(), allowLabelsList)
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

func wrapCustomResourceFunc(f func(metav1.Object) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		metaObject := obj.(metav1.Object)

		metricFamily := f(metaObject)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}
