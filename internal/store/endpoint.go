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
	"strconv"

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
	descEndpointAnnotationsName     = "kube_endpoint_annotations"
	descEndpointAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descEndpointLabelsName          = "kube_endpoint_labels"
	descEndpointLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descEndpointLabelsDefaultLabels = []string{"namespace", "endpoint"}
)

func endpointMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			"kube_endpoint_info",
			"Information about endpoint.",
			metric.Gauge,
			"",
			wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: 1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_endpoint_created",
			"Unix creation timestamp",
			metric.Gauge,
			"",
			wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				ms := []*metric.Metric{}

				if !e.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{

						Value: float64(e.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			descEndpointAnnotationsName,
			descEndpointAnnotationsHelp,
			metric.Gauge,
			"",
			wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", e.Annotations, allowAnnotationsList)
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
			descEndpointLabelsName,
			descEndpointLabelsHelp,
			metric.Gauge,
			"",
			wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", e.Labels, allowLabelsList)
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
			"kube_endpoint_address_available_count",
			"Number of addresses available in endpoint.",
			metric.Gauge,
			"",
			wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				var available int
				for _, s := range e.Subsets {
					available += len(s.Addresses) * len(s.Ports)
				}

				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(available),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_endpoint_address_not_ready_count",
			"Number of addresses not ready in endpoint",
			metric.Gauge,
			"",
			wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				var notReady int
				for _, s := range e.Subsets {
					notReady += len(s.NotReadyAddresses) * len(s.Ports)
				}
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(notReady),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_endpoint_address_available",
			"Addresses available in endpoint",
			metric.Gauge,
			"",
			wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				var addresses string
				var ports string
				ms := []*metric.Metric{}
				for _, s := range e.Subsets {
					for _, address := range s.Addresses {
						addresses += address.IP + "|"
					}
					// Trim last "|" character from addresses
					if len(addresses) > 0 {
						addresses = addresses[:len(addresses)-1]
					}

					for _, port := range s.Ports {
						ports += strconv.FormatInt(int64(port.Port), 10) + "|"
					}
					// Trim last "|" character from ports
					if len(ports) > 0 {
						ports = ports[:len(ports)-1]
					}

					// not add to ms list if addresses not belong to available
					if addresses != "" {
						ms = append(ms, &metric.Metric{
							LabelKeys:   []string{"addresses", "ports"},
							LabelValues: []string{addresses, ports},
							Value:       1,
						})
					}
					addresses = ""
					ports = ""
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_endpoint_address_not_ready",
			"Addresses not ready in endpoint",
			metric.Gauge,
			"",
			wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				var notReadyAddresses string
				var ports string
				ms := []*metric.Metric{}
				for _, s := range e.Subsets {
					for _, notReadyAddress := range s.NotReadyAddresses {
						notReadyAddresses += notReadyAddress.IP + "|"
					}
					// Trim last "|" character from notReadyAddresses
					if len(notReadyAddresses) > 0 {
						notReadyAddresses = notReadyAddresses[:len(notReadyAddresses)-1]
					}

					for _, port := range s.Ports {
						ports += strconv.FormatInt(int64(port.Port), 10) + "|"
					}
					// Trim last "|" character from ports
					if len(ports) > 0 {
						ports = ports[:len(ports)-1]
					}

					// not add to ms list if addresses not belong to not ready
					if notReadyAddresses != "" {
						ms = append(ms, &metric.Metric{
							LabelKeys:   []string{"addresses", "ports"},
							LabelValues: []string{notReadyAddresses, ports},
							Value:       1,
						})
					}
					notReadyAddresses = ""
					ports = ""
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
}

func wrapEndpointFunc(f func(*v1.Endpoints) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		endpoint := obj.(*v1.Endpoints)

		metricFamily := f(endpoint)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descEndpointLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{endpoint.Namespace, endpoint.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createEndpointsListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Endpoints(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Endpoints(ns).Watch(context.TODO(), opts)
		},
	}
}
