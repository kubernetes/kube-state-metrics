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
	descValidatingAdmissionPolicyBindingDefaultLabels = []string{"namespace", "validatingadmissionpolicybinding"}

	validatingAdmissionPolicyBindingMetricFamilies = []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_validatingadmissionpolicybinding_info",
			"Information about the ValidatingAdmissionPolicyBinding.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapValidatingAdmissionPolicyBindingFunc(func(vapb *admissionregistrationv1.ValidatingAdmissionPolicyBinding) *metric.Family {
				var policyName, paramName, paramNamespace, paramNotFoundAction string
				policyName = vapb.Spec.PolicyName
				if vapb.Spec.ParamRef != nil {
					paramName = vapb.Spec.ParamRef.Name
					paramNamespace = vapb.Spec.ParamRef.Namespace
					if vapb.Spec.ParamRef.ParameterNotFoundAction != nil {
						paramNotFoundAction = string(*vapb.Spec.ParamRef.ParameterNotFoundAction)
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
			"kube_validatingadmissionpolicybinding_created",
			"Unix creation timestamp.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapValidatingAdmissionPolicyBindingFunc(func(vapb *admissionregistrationv1.ValidatingAdmissionPolicyBinding) *metric.Family {
				ms := []*metric.Metric{}

				if !vapb.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(vapb.CreationTimestamp.Unix()),
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_validatingadmissionpolicybinding_validation_action",
			"Validation actions for the ValidatingAdmissionPolicyBinding.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapValidatingAdmissionPolicyBindingFunc(func(vapb *admissionregistrationv1.ValidatingAdmissionPolicyBinding) *metric.Family {
				ms := make([]*metric.Metric, 0, len(vapb.Spec.ValidationActions))
				for _, action := range vapb.Spec.ValidationActions {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"action"},
						LabelValues: []string{string(action)},
						Value:       1,
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
)

func createValidatingAdmissionPolicyBindingListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AdmissionregistrationV1().ValidatingAdmissionPolicyBindings().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AdmissionregistrationV1().ValidatingAdmissionPolicyBindings().Watch(context.TODO(), opts)
		},
	}
}

func wrapValidatingAdmissionPolicyBindingFunc(f func(*admissionregistrationv1.ValidatingAdmissionPolicyBinding) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		vapb := obj.(*admissionregistrationv1.ValidatingAdmissionPolicyBinding)

		metricFamily := f(vapb)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descValidatingAdmissionPolicyBindingDefaultLabels, []string{vapb.Namespace, vapb.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}
