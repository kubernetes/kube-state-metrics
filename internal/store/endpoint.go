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
	basemetrics "k8s.io/component-base/metrics"

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
		*generator.NewFamilyGeneratorWithStability(
			"kube_endpoint_info",
			"Information about endpoint.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapEndpointFunc(func(_ *v1.Endpoints) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: 1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_endpoint_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.STABLE,
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
		*generator.NewFamilyGeneratorWithStability(
			descEndpointAnnotationsName,
			descEndpointAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
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
		*generator.NewFamilyGeneratorWithStability(
			descEndpointLabelsName,
			descEndpointLabelsHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
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
		*generator.NewFamilyGeneratorWithStability(
			"kube_endpoint_address",
			"Information about Endpoint available and non available addresses.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				ms := []*metric.Metric{}
				labelKeys := []string{"port_protocol", "port_number", "port_name", "ip", "ready"}

				for _, s := range e.Subsets {
					for _, port := range s.Ports {
						for _, available := range s.Addresses {
							labelValues := []string{string(port.Protocol), strconv.FormatInt(int64(port.Port), 10), port.Name}

							ms = append(ms, &metric.Metric{
								LabelValues: append(labelValues, available.IP, "true"),
								LabelKeys:   labelKeys,
								Value:       1,
							})
						}
						for _, notReadyAddresses := range s.NotReadyAddresses {
							labelValues := []string{string(port.Protocol), strconv.FormatInt(int64(port.Port), 10), port.Name}

							ms = append(ms, &metric.Metric{
								LabelValues: append(labelValues, notReadyAddresses.IP, "false"),
								LabelKeys:   labelKeys,
								Value:       1,
							})
						}
					}
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_endpoint_ports",
			"Information about the Endpoint ports.",
			metric.Gauge,
			basemetrics.STABLE,
			"v2.14.0",
			wrapEndpointFunc(func(e *v1.Endpoints) *metric.Family {
				ms := []*metric.Metric{}
				for _, s := range e.Subsets {
					for _, port := range s.Ports {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{port.Name, string(port.Protocol), strconv.FormatInt(int64(port.Port), 10)},
							LabelKeys:   []string{"port_name", "port_protocol", "port_number"},
							Value:       1,
						})
					}
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
			m.LabelKeys, m.LabelValues = mergeKeyValues(descEndpointLabelsDefaultLabels, []string{endpoint.Namespace, endpoint.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createEndpointsListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().Endpoints(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().Endpoints(ns).Watch(context.TODO(), opts)
		},
	}
}
