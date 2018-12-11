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
	descEndpointLabelsName          = "kube_endpoint_labels"
	descEndpointLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descEndpointLabelsDefaultLabels = []string{"namespace", "endpoint"}

	endpointMetricFamilies = []metrics.FamilyGenerator{
		metrics.FamilyGenerator{
			Name: "kube_endpoint_info",
			Type: metrics.MetricTypeGauge,
			Help: "Information about endpoint.",
			GenerateFunc: wrapEndpointFunc(func(e *v1.Endpoints) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_endpoint_info",
					Value: 1,
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_endpoint_created",
			Type: metrics.MetricTypeGauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapEndpointFunc(func(e *v1.Endpoints) metrics.Family {
				f := metrics.Family{}

				if !e.CreationTimestamp.IsZero() {
					f = append(f, &metrics.Metric{
						Name:  "kube_endpoint_created",
						Value: float64(e.CreationTimestamp.Unix()),
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: descEndpointLabelsName,
			Type: metrics.MetricTypeGauge,
			Help: descEndpointLabelsHelp,
			GenerateFunc: wrapEndpointFunc(func(e *v1.Endpoints) metrics.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(e.Labels)
				return metrics.Family{&metrics.Metric{
					Name:        descEndpointLabelsName,
					LabelKeys:   labelKeys,
					LabelValues: labelValues,
					Value:       1,
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_endpoint_address_available",
			Type: metrics.MetricTypeGauge,
			Help: "Number of addresses available in endpoint.",
			GenerateFunc: wrapEndpointFunc(func(e *v1.Endpoints) metrics.Family {
				var available int
				for _, s := range e.Subsets {
					available += len(s.Addresses) * len(s.Ports)
				}

				return metrics.Family{&metrics.Metric{
					Name:  "kube_endpoint_address_available",
					Value: float64(available),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_endpoint_address_not_ready",
			Type: metrics.MetricTypeGauge,
			Help: "Number of addresses not ready in endpoint",
			GenerateFunc: wrapEndpointFunc(func(e *v1.Endpoints) metrics.Family {
				var notReady int
				for _, s := range e.Subsets {
					notReady += len(s.NotReadyAddresses) * len(s.Ports)
				}
				return metrics.Family{&metrics.Metric{
					Name:  "kube_endpoint_address_not_ready",
					Value: float64(notReady),
				}}
			}),
		},
	}
)

func wrapEndpointFunc(f func(*v1.Endpoints) metrics.Family) func(interface{}) metrics.Family {
	return func(obj interface{}) metrics.Family {
		endpoint := obj.(*v1.Endpoints)

		metricFamily := f(endpoint)

		for _, m := range metricFamily {
			m.LabelKeys = append(descEndpointLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{endpoint.Namespace, endpoint.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createEndpointsListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Endpoints(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Endpoints(ns).Watch(opts)
		},
	}
}
