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
	descSecretLabelsName          = "kube_secret_labels"
	descSecretLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descSecretLabelsDefaultLabels = []string{"namespace", "secret"}

	secretMetricFamilies = []metrics.FamilyGenerator{
		metrics.FamilyGenerator{
			Name: "kube_secret_info",
			Type: metrics.MetricTypeGauge,
			Help: "Information about secret.",
			GenerateFunc: wrapSecretFunc(func(s *v1.Secret) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_secret_info",
					Value: 1,
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_secret_type",
			Type: metrics.MetricTypeGauge,
			Help: "Type about secret.",
			GenerateFunc: wrapSecretFunc(func(s *v1.Secret) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:        "kube_secret_type",
					LabelKeys:   []string{"type"},
					LabelValues: []string{string(s.Type)},
					Value:       1,
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: descSecretLabelsName,
			Type: metrics.MetricTypeGauge,
			Help: descSecretLabelsHelp,
			GenerateFunc: wrapSecretFunc(func(s *v1.Secret) metrics.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(s.Labels)
				return metrics.Family{&metrics.Metric{
					Name:        descSecretLabelsName,
					LabelKeys:   labelKeys,
					LabelValues: labelValues,
					Value:       1,
				}}

			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_secret_created",
			Type: metrics.MetricTypeGauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapSecretFunc(func(s *v1.Secret) metrics.Family {
				f := metrics.Family{}

				if !s.CreationTimestamp.IsZero() {
					f = append(f, &metrics.Metric{
						Name:  "kube_secret_created",
						Value: float64(s.CreationTimestamp.Unix()),
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_secret_metadata_resource_version",
			Type: metrics.MetricTypeGauge,
			Help: "Resource version representing a specific version of secret.",
			GenerateFunc: wrapSecretFunc(func(s *v1.Secret) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:        "kube_secret_metadata_resource_version",
					LabelKeys:   []string{"resource_version"},
					LabelValues: []string{string(s.ObjectMeta.ResourceVersion)},
					Value:       1,
				}}
			}),
		},
	}
)

func wrapSecretFunc(f func(*v1.Secret) metrics.Family) func(interface{}) metrics.Family {
	return func(obj interface{}) metrics.Family {
		secret := obj.(*v1.Secret)

		metricFamily := f(secret)

		for _, m := range metricFamily {
			m.LabelKeys = append(descSecretLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{secret.Namespace, secret.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createSecretListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Secrets(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Secrets(ns).Watch(opts)
		},
	}
}
