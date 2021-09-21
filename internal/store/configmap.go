/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descConfigMapLabelsDefaultLabels = []string{"namespace", "configmap"}
)

func configMapMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			"kube_configmap_annotations",
			"Kubernetes annotations converted to Prometheus labels.",
			metric.Gauge,
			"",
			wrapConfigMapFunc(func(c *v1.ConfigMap) *metric.Family {
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", c.Annotations, allowAnnotationsList)
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
		*generator.NewFamilyGenerator(
			"kube_configmap_labels",
			"Kubernetes labels converted to Prometheus labels.",
			metric.Gauge,
			"",
			wrapConfigMapFunc(func(c *v1.ConfigMap) *metric.Family {
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", c.Labels, allowLabelsList)
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
		*generator.NewFamilyGenerator(
			"kube_configmap_info",
			"Information about configmap.",
			metric.Gauge,
			"",
			wrapConfigMapFunc(func(c *v1.ConfigMap) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       1,
					}},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_configmap_created",
			"Unix creation timestamp",
			metric.Gauge,
			"",
			wrapConfigMapFunc(func(c *v1.ConfigMap) *metric.Family {
				ms := []*metric.Metric{}

				if !c.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(c.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_configmap_metadata_resource_version",
			"Resource version representing a specific version of the configmap.",
			metric.Gauge,
			"",
			wrapConfigMapFunc(func(c *v1.ConfigMap) *metric.Family {
				return &metric.Family{
					Metrics: resourceVersionMetric(c.ObjectMeta.ResourceVersion),
				}
			}),
		),
	}
}

func createConfigMapListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().ConfigMaps(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().ConfigMaps(ns).Watch(context.TODO(), opts)
		},
	}
}

func wrapConfigMapFunc(f func(*v1.ConfigMap) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		configMap := obj.(*v1.ConfigMap)

		metricFamily := f(configMap)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descConfigMapLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{configMap.Namespace, configMap.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}
