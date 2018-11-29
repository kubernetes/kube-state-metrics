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

package collectors

import (
	"k8s.io/kube-state-metrics/pkg/metrics"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descConfigMapLabelsDefaultLabels = []string{"namespace", "configmap"}

	configMapMetricFamilies = []metrics.FamilyGenerator{
		metrics.FamilyGenerator{
			Name: "kube_configmap_info",
			Type: metrics.MetricTypeGauge,
			Help: "Information about configmap.",
			GenerateFunc: wrapConfigMapFunc(func(c *v1.ConfigMap) metrics.Family {
				return metrics.Family{
					&metrics.Metric{
						Name:        "kube_configmap_info",
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       1,
					},
				}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_configmap_created",
			Type: metrics.MetricTypeGauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapConfigMapFunc(func(c *v1.ConfigMap) metrics.Family {
				f := metrics.Family{}

				if !c.CreationTimestamp.IsZero() {
					f = append(f, &metrics.Metric{
						Name:        "kube_configmap_created",
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(c.CreationTimestamp.Unix()),
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_configmap_metadata_resource_version",
			Type: metrics.MetricTypeGauge,
			Help: "Resource version representing a specific version of the configmap.",
			GenerateFunc: wrapConfigMapFunc(func(c *v1.ConfigMap) metrics.Family {
				return metrics.Family{
					&metrics.Metric{
						Name:        "kube_configmap_metadata_resource_version",
						LabelKeys:   []string{"resource_version"},
						LabelValues: []string{string(c.ObjectMeta.ResourceVersion)},
						Value:       1,
					},
				}
			}),
		},
	}
)

func createConfigMapListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().ConfigMaps(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().ConfigMaps(ns).Watch(opts)
		},
	}
}

func wrapConfigMapFunc(f func(*v1.ConfigMap) metrics.Family) func(interface{}) metrics.Family {
	return func(obj interface{}) metrics.Family {
		configMap := obj.(*v1.ConfigMap)

		metricFamily := f(configMap)

		for _, m := range metricFamily {
			m.LabelKeys = append(descConfigMapLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{configMap.Namespace, configMap.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}
