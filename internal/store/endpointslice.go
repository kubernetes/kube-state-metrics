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
	"strconv"

	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descEndpointSliceAnnotationsName     = "kube_endpointslice_annotations"
	descEndpointSliceAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descEndpointSliceLabelsName          = "kube_endpointslice_labels"
	descEndpointSliceLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descEndpointSliceLabelsDefaultLabels = []string{"endpointslice"}
)

func endpointSliceMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_endpointslice_info",
			"Information about endpointslice.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapEndpointSliceFunc(func(s *discoveryv1.EndpointSlice) *metric.Family {

				m := metric.Metric{
					LabelKeys:   []string{"addresstype"},
					LabelValues: []string{string(s.AddressType)},
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_endpointslice_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapEndpointSliceFunc(func(s *discoveryv1.EndpointSlice) *metric.Family {
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
			"kube_endpointslice_endpoints",
			"Endpoints attached to the endpointslice.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapEndpointSliceFunc(func(e *discoveryv1.EndpointSlice) *metric.Family {
				m := []*metric.Metric{}
				for _, ep := range e.Endpoints {
					var (
						labelKeys,
						labelValues []string
					)

					if ep.Conditions.Ready != nil {
						labelKeys = append(labelKeys, "ready")
						labelValues = append(labelValues, strconv.FormatBool(*ep.Conditions.Ready))
					}
					if ep.Conditions.Serving != nil {
						labelKeys = append(labelKeys, "serving")
						labelValues = append(labelValues, strconv.FormatBool(*ep.Conditions.Serving))
					}
					if ep.Conditions.Terminating != nil {
						labelKeys = append(labelKeys, "terminating")
						labelValues = append(labelValues, strconv.FormatBool(*ep.Conditions.Terminating))
					}

					if ep.Hostname != nil {
						labelKeys = append(labelKeys, "hostname")
						labelValues = append(labelValues, *ep.Hostname)
					}

					if ep.TargetRef != nil {
						if ep.TargetRef.Kind != "" {
							labelKeys = append(labelKeys, "targetref_kind")
							labelValues = append(labelValues, ep.TargetRef.Kind)
						}
						if ep.TargetRef.Name != "" {
							labelKeys = append(labelKeys, "targetref_name")
							labelValues = append(labelValues, ep.TargetRef.Name)
						}
						if ep.TargetRef.Namespace != "" {
							labelKeys = append(labelKeys, "targetref_namespace")
							labelValues = append(labelValues, ep.TargetRef.Namespace)
						}
					}

					if ep.NodeName != nil {
						labelKeys = append(labelKeys, "endpoint_nodename")
						labelValues = append(labelValues, *ep.NodeName)
					}

					if ep.Zone != nil {
						labelKeys = append(labelKeys, "endpoint_zone")
						labelValues = append(labelValues, *ep.Zone)
					}
					labelKeys = append(labelKeys, "address")
					for _, address := range ep.Addresses {
						newlabelValues := make([]string, len(labelValues))
						copy(newlabelValues, labelValues)
						m = append(m, &metric.Metric{
							LabelKeys:   labelKeys,
							LabelValues: append(newlabelValues, address),
							Value:       1,
						})
					}
				}
				return &metric.Family{
					Metrics: m,
				}
			}),
		),

		*generator.NewFamilyGeneratorWithStability(
			"kube_endpointslice_ports",
			"Ports attached to the endpointslice.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapEndpointSliceFunc(func(e *discoveryv1.EndpointSlice) *metric.Family {
				m := []*metric.Metric{}
				for _, port := range e.Ports {
					m = append(m, &metric.Metric{
						LabelValues: []string{*port.Name, string(*port.Protocol), strconv.FormatInt(int64(*port.Port), 10)},
						LabelKeys:   []string{"port_name", "port_protocol", "port_number"},
						Value:       1,
					})
				}
				return &metric.Family{
					Metrics: m,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descEndpointSliceAnnotationsName,
			descEndpointSliceAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapEndpointSliceFunc(func(s *discoveryv1.EndpointSlice) *metric.Family {
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
			descEndpointSliceLabelsName,
			descEndpointSliceLabelsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapEndpointSliceFunc(func(s *discoveryv1.EndpointSlice) *metric.Family {
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

func wrapEndpointSliceFunc(f func(*discoveryv1.EndpointSlice) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		endpointSlice := obj.(*discoveryv1.EndpointSlice)

		metricFamily := f(endpointSlice)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descEndpointSliceLabelsDefaultLabels, []string{endpointSlice.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createEndpointSliceListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.DiscoveryV1().EndpointSlices(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.DiscoveryV1().EndpointSlices(ns).Watch(context.TODO(), opts)
		},
	}
}
