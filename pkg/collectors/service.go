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
	descServiceLabelsName          = "kube_service_labels"
	descServiceLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descServiceLabelsDefaultLabels = []string{"namespace", "service"}

	serviceMetricFamilies = []metrics.MetricFamily{
		{
			"kube_service_info",
			"Information about service.",
			func(obj interface{}) []*metrics.Metric {
				sPointer := obj.(*v1.Service)
				s := *sPointer

				return []*metrics.Metric{
					newServiceMetric(
						sPointer,
						"kube_service_info",
						[]string{"cluster_ip", "external_name", "load_balancer_ip"},
						[]string{s.Spec.ClusterIP, s.Spec.ExternalName, s.Spec.LoadBalancerIP},
						1,
					),
				}
			},
		},
		{
			"kube_service_created",
			"Unix creation timestamp",
			func(obj interface{}) []*metrics.Metric {
				sPointer := obj.(*v1.Service)
				s := *sPointer

				if !s.CreationTimestamp.IsZero() {
					return []*metrics.Metric{
						newServiceMetric(
							sPointer,
							"kube_service_created",
							nil,
							nil,
							float64(s.CreationTimestamp.Unix()),
						),
					}
				}
				return nil
			},
		},
		{
			"kube_service_spec_type",
			"Type about service.",
			func(obj interface{}) []*metrics.Metric {
				sPointer := obj.(*v1.Service)
				s := *sPointer

				return []*metrics.Metric{
					newServiceMetric(
						sPointer,
						"kube_service_spec_type",
						[]string{"type"},
						[]string{string(s.Spec.Type)},
						1,
					),
				}
			},
		},
		{
			descServiceLabelsName,
			descServiceLabelsHelp,
			func(obj interface{}) []*metrics.Metric {
				sPointer := obj.(*v1.Service)
				s := *sPointer

				labelKeys, labelValues := kubeLabelsToPrometheusLabels(s.Labels)
				return []*metrics.Metric{
					newServiceMetric(
						sPointer,
						descServiceLabelsName,
						labelKeys,
						labelValues,
						1,
					),
				}
			},
		},
		// Defined, but not used anywhere. See
		// https://github.com/kubernetes/kube-state-metrics/pull/571#pullrequestreview-176215628.
		// {
		// 	"kube_service_external_name",
		// 	"Service external name",
		// 	// []string{"type"},
		// },
		// {
		// 	"kube_service_load_balancer_ip",
		// 	"Load balancer IP of service",
		// 	// []string{"load_balancer_ip"},
		// },
		{
			"kube_service_spec_external_ip",
			"Service external ips. One series for each ip",
			func(obj interface{}) []*metrics.Metric {
				sPointer := obj.(*v1.Service)
				s := *sPointer

				metrics := []*metrics.Metric{}

				if len(s.Spec.ExternalIPs) > 0 {
					for _, externalIP := range s.Spec.ExternalIPs {
						m := newServiceMetric(
							sPointer,
							"kube_service_spec_external_ip",
							[]string{"external_ip"},
							[]string{externalIP},
							1,
						)

						metrics = append(metrics, m)
					}
				}

				return metrics
			},
		},
		{
			"kube_service_status_load_balancer_ingress",
			"Service load balancer ingress status",
			func(obj interface{}) []*metrics.Metric {
				sPointer := obj.(*v1.Service)
				s := *sPointer

				metrics := []*metrics.Metric{}

				if len(s.Status.LoadBalancer.Ingress) > 0 {
					for _, ingress := range s.Status.LoadBalancer.Ingress {
						m := newServiceMetric(
							sPointer,
							"kube_service_status_load_balancer_ingress",
							[]string{"ip", "hostname"},
							[]string{ingress.IP, ingress.Hostname},
							1,
						)

						metrics = append(metrics, m)
					}
				}

				return metrics
			},
		},
	}
)

func newServiceMetric(s *v1.Service, name string, lk []string, lv []string, v float64) *metrics.Metric {
	lk = append(descServiceLabelsDefaultLabels, lk...)
	lv = append([]string{s.Namespace, s.Name}, lv...)

	m, err := metrics.NewMetric(name, lk, lv, v)
	if err != nil {
		// TODO: Move this panic into metrics.NewMetric
		panic(err)
	}

	return m
}

func createServiceListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Services(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Services(ns).Watch(opts)
		},
	}
}
