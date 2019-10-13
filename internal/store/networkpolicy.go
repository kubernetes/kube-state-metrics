/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/pkg/metric"
	generator "k8s.io/kube-state-metrics/pkg/metric_generator"
)

var (
	descNetworkPolicyLabelsDefaultLabels = []string{"namespace", "networkpolicy"}

	networkpolicyMetricFamilies = []generator.FamilyGenerator{
		{
			Name: "kube_networkpolicy_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp of network policy",
			GenerateFunc: wrapNetworkPolicyFunc(func(n *networkingv1.NetworkPolicy) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(n.CreationTimestamp.Unix()),
						},
					},
				}
			}),
		},
		{
			Name: "kube_networkpolicy_labels",
			Type: metric.Gauge,
			Help: "Kubernetes labels converted to Prometheus labels",
			GenerateFunc: wrapNetworkPolicyFunc(func(n *networkingv1.NetworkPolicy) *metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(n.Labels)
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
		},
		{
			Name: "kube_networkpolicy_spec_ingress_rules",
			Type: metric.Gauge,
			Help: "Number of ingress rules on the networkpolicy",
			GenerateFunc: wrapNetworkPolicyFunc(func(n *networkingv1.NetworkPolicy) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(len(n.Spec.Ingress)),
						},
					},
				}
			}),
		},
		{
			Name: "kube_networkpolicy_spec_egress_rules",
			Type: metric.Gauge,
			Help: "Number of egress rules on the networkpolicy",
			GenerateFunc: wrapNetworkPolicyFunc(func(n *networkingv1.NetworkPolicy) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(len(n.Spec.Egress)),
						},
					},
				}
			}),
		},
	}
)

func wrapNetworkPolicyFunc(f func(*networkingv1.NetworkPolicy) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		networkPolicy := obj.(*networkingv1.NetworkPolicy)

		metricFamily := f(networkPolicy)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descNetworkPolicyLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{networkPolicy.Namespace, networkPolicy.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createNetworkPolicyListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.NetworkingV1().NetworkPolicies(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.NetworkingV1().NetworkPolicies(ns).Watch(opts)
		},
	}
}
