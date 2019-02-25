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

package collector

import (
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/metric"
)

var (
	descNetworkPolicyLabelsName          = "kube_networkpolicy_labels"
	descNetworkPolicyLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descNetworkPolicyLabelsDefaultLabels = []string{"namespace", "networkpolicy"}

	networkPolicyMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_networkpolicy_info",
			Type: metric.MetricTypeGauge,
			Help: "Information about network policy.",
			GenerateFunc: wrapNetworkPolicyFunc(func(policy *networking.NetworkPolicy) *metric.Family {
				ms := []*metric.Metric{
					{
						Value: 1,
					},
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: descNetworkPolicyLabelsName,
			Type: metric.MetricTypeGauge,
			Help: descNetworkPolicyLabelsHelp,
			GenerateFunc: wrapNetworkPolicyFunc(func(policy *networking.NetworkPolicy) *metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(policy.Labels)
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
			Name: "kube_networkpolicy_created",
			Type: metric.MetricTypeGauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapNetworkPolicyFunc(func(policy *networking.NetworkPolicy) *metric.Family {
				ms := []*metric.Metric{}

				if !policy.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(policy.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_networkpolicy_metadata_resource_version",
			Type: metric.MetricTypeGauge,
			Help: "Resource version representing a specific version of networkpolicy.",
			GenerateFunc: wrapNetworkPolicyFunc(func(policy *networking.NetworkPolicy) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"resource_version"},
							LabelValues: []string{string(policy.ObjectMeta.ResourceVersion)},
							Value:       1,
						},
					}}
			}),
		},
	}
)

func wrapNetworkPolicyFunc(f func(policy *networking.NetworkPolicy) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		policy := obj.(*networking.NetworkPolicy)
		metricFamily := f(policy)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descNetworkPolicyLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{policy.Namespace, policy.Name}, m.LabelValues...)
		}
		return metricFamily
	}
}

func createNetworkPolicyListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.NetworkingV1().NetworkPolicies(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.NetworkingV1().NetworkPolicies(ns).Watch(opts)
		},
	}
}
