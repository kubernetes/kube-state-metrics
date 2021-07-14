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
	descServiceAnnotationsName     = "kube_service_annotations"
	descServiceAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descServiceLabelsName          = "kube_service_labels"
	descServiceLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descServiceLabelsDefaultLabels = []string{"namespace", "service"}
)

func serviceMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			"kube_service_info",
			"Information about service.",
			metric.Gauge,
			"",
			wrapSvcFunc(func(s *v1.Service) *metric.Family {
				m := metric.Metric{
					LabelKeys:   []string{"cluster_ip", "external_name", "load_balancer_ip"},
					LabelValues: []string{s.Spec.ClusterIP, s.Spec.ExternalName, s.Spec.LoadBalancerIP},
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_service_created",
			"Unix creation timestamp",
			metric.Gauge,
			"",
			wrapSvcFunc(func(s *v1.Service) *metric.Family {
				if !s.CreationTimestamp.IsZero() {
					m := metric.Metric{
						LabelKeys:   nil,
						LabelValues: nil,
						Value:       float64(s.CreationTimestamp.Unix()),
					}
					return &metric.Family{Metrics: []*metric.Metric{&m}}
				}
				return &metric.Family{Metrics: []*metric.Metric{}}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_service_spec_type",
			"Type about service.",
			metric.Gauge,
			"",
			wrapSvcFunc(func(s *v1.Service) *metric.Family {
				m := metric.Metric{

					LabelKeys:   []string{"type"},
					LabelValues: []string{string(s.Spec.Type)},
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		),
		*generator.NewFamilyGenerator(
			descServiceAnnotationsName,
			descServiceAnnotationsHelp,
			metric.Gauge,
			"",
			wrapSvcFunc(func(s *v1.Service) *metric.Family {
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", s.Annotations, allowAnnotationsList)
				m := metric.Metric{
					LabelKeys:   annotationKeys,
					LabelValues: annotationValues,
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		),
		*generator.NewFamilyGenerator(
			descServiceLabelsName,
			descServiceLabelsHelp,
			metric.Gauge,
			"",
			wrapSvcFunc(func(s *v1.Service) *metric.Family {
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", s.Labels, allowLabelsList)
				m := metric.Metric{
					LabelKeys:   labelKeys,
					LabelValues: labelValues,
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_service_spec_external_ip",
			"Service external ips. One series for each ip",
			metric.Gauge,
			"",
			wrapSvcFunc(func(s *v1.Service) *metric.Family {
				if len(s.Spec.ExternalIPs) == 0 {
					return &metric.Family{
						Metrics: []*metric.Metric{},
					}
				}

				ms := make([]*metric.Metric, len(s.Spec.ExternalIPs))

				for i, externalIP := range s.Spec.ExternalIPs {
					ms[i] = &metric.Metric{
						LabelKeys:   []string{"external_ip"},
						LabelValues: []string{externalIP},
						Value:       1,
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_service_status_load_balancer_ingress",
			"Service load balancer ingress status",
			metric.Gauge,
			"",
			wrapSvcFunc(func(s *v1.Service) *metric.Family {
				if len(s.Status.LoadBalancer.Ingress) == 0 {
					return &metric.Family{
						Metrics: []*metric.Metric{},
					}
				}

				ms := make([]*metric.Metric, len(s.Status.LoadBalancer.Ingress))

				for i, ingress := range s.Status.LoadBalancer.Ingress {
					ms[i] = &metric.Metric{
						LabelKeys:   []string{"ip", "hostname"},
						LabelValues: []string{ingress.IP, ingress.Hostname},
						Value:       1,
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
}

func wrapSvcFunc(f func(*v1.Service) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		svc := obj.(*v1.Service)

		metricFamily := f(svc)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descServiceLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{svc.Namespace, svc.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createServiceListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Services(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Services(ns).Watch(context.TODO(), opts)
		},
	}
}
