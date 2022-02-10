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
	"context"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descNetworkPolicyAnnotationsName     = "kube_networkpolicy_annotations"
	descNetworkPolicyAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descNetworkPolicyLabelsName          = "kube_networkpolicy_labels"
	descNetworkPolicyLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descNetworkPolicyLabelsDefaultLabels = []string{"namespace", "networkpolicy"}
)

func networkPolicyMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			"kube_networkpolicy_created",
			"Unix creation timestamp of network policy",
			metric.Gauge,
			"",
			wrapNetworkPolicyFunc(func(n *networkingv1.NetworkPolicy) *metric.Family {
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
		),
		*generator.NewFamilyGenerator(
			descNetworkPolicyAnnotationsName,
			descNetworkPolicyAnnotationsHelp,
			metric.Gauge,
			"",
			wrapNetworkPolicyFunc(func(n *networkingv1.NetworkPolicy) *metric.Family {
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", n.Annotations, allowAnnotationsList)
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
			descNetworkPolicyLabelsName,
			descNetworkPolicyLabelsHelp,
			metric.Gauge,
			"",
			wrapNetworkPolicyFunc(func(n *networkingv1.NetworkPolicy) *metric.Family {
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", n.Labels, allowLabelsList)
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
			"kube_networkpolicy_spec_ingress_rules",
			"Number of ingress rules on the networkpolicy",
			metric.Gauge,
			"",
			wrapNetworkPolicyFunc(func(n *networkingv1.NetworkPolicy) *metric.Family {
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
		),
		*generator.NewFamilyGenerator(
			"kube_networkpolicy_spec_egress_rules",
			"Number of egress rules on the networkpolicy",
			metric.Gauge,
			"",
			wrapNetworkPolicyFunc(func(n *networkingv1.NetworkPolicy) *metric.Family {
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
		),
	}
}

func wrapNetworkPolicyFunc(f func(*networkingv1.NetworkPolicy) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		networkPolicy := obj.(*networkingv1.NetworkPolicy)

		metricFamily := f(networkPolicy)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descNetworkPolicyLabelsDefaultLabels, []string{networkPolicy.Namespace, networkPolicy.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createNetworkPolicyListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.NetworkingV1().NetworkPolicies(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.NetworkingV1().NetworkPolicies(ns).Watch(context.TODO(), opts)
		},
	}
}
