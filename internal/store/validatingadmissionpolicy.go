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
	descValidatingAdmissionPolicyDefaultLabels = []string{"namespace", "validatingadmissionpolicy"}

	validatingAdmissionPolicyMetricFamilies = []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_validatingadmissionpolicy_info",
			"Information about the ValidatingAdmissionPolicy.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapValidatingAdmissionPolicyFunc(func(vap *admissionregistrationv1.ValidatingAdmissionPolicy) *metric.Family {
				var paramAPIVersion, paramKind, failurePolicy string
				if vap.Spec.ParamKind != nil {
					paramAPIVersion = vap.Spec.ParamKind.APIVersion
					paramKind = vap.Spec.ParamKind.Kind
				}
				if vap.Spec.FailurePolicy != nil {
					failurePolicy = string(*vap.Spec.FailurePolicy)
				}

				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"param_api_version", "param_kind", "failure_policy"},
							LabelValues: []string{paramAPIVersion, paramKind, failurePolicy},
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_validatingadmissionpolicy_created",
			"Unix creation timestamp.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapValidatingAdmissionPolicyFunc(func(vap *admissionregistrationv1.ValidatingAdmissionPolicy) *metric.Family {
				ms := []*metric.Metric{}

				if !vap.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(vap.CreationTimestamp.Unix()),
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
)

func createValidatingAdmissionPolicyListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AdmissionregistrationV1().ValidatingAdmissionPolicies().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AdmissionregistrationV1().ValidatingAdmissionPolicies().Watch(context.TODO(), opts)
		},
	}
}

func wrapValidatingAdmissionPolicyFunc(f func(*admissionregistrationv1.ValidatingAdmissionPolicy) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		vap := obj.(*admissionregistrationv1.ValidatingAdmissionPolicy)

		metricFamily := f(vap)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descValidatingAdmissionPolicyDefaultLabels, []string{vap.Namespace, vap.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}
