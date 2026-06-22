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
	descMutatingAdmissionPolicyDefaultLabels = []string{"namespace", "mutatingadmissionpolicy"}

	mutatingAdmissionPolicyMetricFamilies = []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_mutatingadmissionpolicy_info",
			"Information about the MutatingAdmissionPolicy.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapMutatingAdmissionPolicyFunc(func(mapObj *admissionregistrationv1.MutatingAdmissionPolicy) *metric.Family {
				var paramAPIVersion, paramKind, failurePolicy, reininvocationPolicy string
				if mapObj.Spec.ParamKind != nil {
					paramAPIVersion = mapObj.Spec.ParamKind.APIVersion
					paramKind = mapObj.Spec.ParamKind.Kind
				}
				if mapObj.Spec.FailurePolicy != nil {
					failurePolicy = string(*mapObj.Spec.FailurePolicy)
				}
				reininvocationPolicy = string(mapObj.Spec.ReinvocationPolicy)

				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"param_api_version", "param_kind", "failure_policy", "reinvocation_policy"},
							LabelValues: []string{paramAPIVersion, paramKind, failurePolicy, reininvocationPolicy},
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_mutatingadmissionpolicy_created",
			"Unix creation timestamp.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapMutatingAdmissionPolicyFunc(func(mapObj *admissionregistrationv1.MutatingAdmissionPolicy) *metric.Family {
				ms := []*metric.Metric{}

				if !mapObj.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(mapObj.CreationTimestamp.Unix()),
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
)

func createMutatingAdmissionPolicyListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AdmissionregistrationV1().MutatingAdmissionPolicies().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AdmissionregistrationV1().MutatingAdmissionPolicies().Watch(context.TODO(), opts)
		},
	}
}

func wrapMutatingAdmissionPolicyFunc(f func(*admissionregistrationv1.MutatingAdmissionPolicy) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		mapObj := obj.(*admissionregistrationv1.MutatingAdmissionPolicy)

		metricFamily := f(mapObj)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descMutatingAdmissionPolicyDefaultLabels, []string{mapObj.Namespace, mapObj.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}
