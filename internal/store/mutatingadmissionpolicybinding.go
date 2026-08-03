/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
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
	descMutatingAdmissionPolicyBindingDefaultLabels = []string{"namespace", "mutatingadmissionpolicybinding"}

	mutatingAdmissionPolicyBindingMetricFamilies = []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_mutatingadmissionpolicybinding_info",
			"Information about the MutatingAdmissionPolicyBinding.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapMutatingAdmissionPolicyBindingFunc(func(mapb *admissionregistrationv1.MutatingAdmissionPolicyBinding) *metric.Family {
				var policyName, paramName, paramNamespace, paramNotFoundAction string
				policyName = mapb.Spec.PolicyName
				if mapb.Spec.ParamRef != nil {
					paramName = mapb.Spec.ParamRef.Name
					paramNamespace = mapb.Spec.ParamRef.Namespace
					if mapb.Spec.ParamRef.ParameterNotFoundAction != nil {
						paramNotFoundAction = string(*mapb.Spec.ParamRef.ParameterNotFoundAction)
					}
				}

				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"policy_name", "param_name", "param_namespace", "param_not_found_action"},
							LabelValues: []string{policyName, paramName, paramNamespace, paramNotFoundAction},
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_mutatingadmissionpolicybinding_created",
			"Unix creation timestamp.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapMutatingAdmissionPolicyBindingFunc(func(mapb *admissionregistrationv1.MutatingAdmissionPolicyBinding) *metric.Family {
				ms := []*metric.Metric{}

				if !mapb.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(mapb.CreationTimestamp.Unix()),
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
)

func createMutatingAdmissionPolicyBindingListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AdmissionregistrationV1().MutatingAdmissionPolicyBindings().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AdmissionregistrationV1().MutatingAdmissionPolicyBindings().Watch(context.TODO(), opts)
		},
	}
}

func wrapMutatingAdmissionPolicyBindingFunc(f func(*admissionregistrationv1.MutatingAdmissionPolicyBinding) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		mapb := obj.(*admissionregistrationv1.MutatingAdmissionPolicyBinding)

		metricFamily := f(mapb)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descMutatingAdmissionPolicyBindingDefaultLabels, []string{mapb.Namespace, mapb.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}
