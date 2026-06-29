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
	"testing"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestMutatingAdmissionPolicyBindingStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)
	paramNotFoundActionDeny := admissionregistrationv1.DenyAction

	cases := []generateMetricsTestCase{
		{
			Obj: &admissionregistrationv1.MutatingAdmissionPolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "mutatingadmissionpolicybinding1",
					Namespace:       "ns1",
					ResourceVersion: "123456",
				},
				Spec: admissionregistrationv1.MutatingAdmissionPolicyBindingSpec{
					PolicyName: "mutatingadmissionpolicy1",
					ParamRef: &admissionregistrationv1.ParamRef{
						Name:                    "param1",
						Namespace:               "ns1",
						ParameterNotFoundAction: &paramNotFoundActionDeny,
					},
				},
			},
			Want: `
				# HELP kube_mutatingadmissionpolicybinding_info Information about the MutatingAdmissionPolicyBinding.
				# TYPE kube_mutatingadmissionpolicybinding_info gauge
				kube_mutatingadmissionpolicybinding_info{namespace="ns1",param_name="param1",param_namespace="ns1",param_not_found_action="Deny",policy_name="mutatingadmissionpolicy1",mutatingadmissionpolicybinding="mutatingadmissionpolicybinding1"} 1
				`,
			MetricNames: []string{"kube_mutatingadmissionpolicybinding_info"},
		},
		{
			Obj: &admissionregistrationv1.MutatingAdmissionPolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "mutatingadmissionpolicybinding2",
					Namespace:         "ns2",
					CreationTimestamp: metav1StartTime,
					ResourceVersion:   "abcdef",
				},
				Spec: admissionregistrationv1.MutatingAdmissionPolicyBindingSpec{
					PolicyName: "mutatingadmissionpolicy2",
				},
			},
			Want: `
				# HELP kube_mutatingadmissionpolicybinding_created Unix creation timestamp.
				# HELP kube_mutatingadmissionpolicybinding_info Information about the MutatingAdmissionPolicyBinding.
				# TYPE kube_mutatingadmissionpolicybinding_created gauge
				# TYPE kube_mutatingadmissionpolicybinding_info gauge
				kube_mutatingadmissionpolicybinding_created{namespace="ns2",mutatingadmissionpolicybinding="mutatingadmissionpolicybinding2"} 1.501569018e+09
				kube_mutatingadmissionpolicybinding_info{namespace="ns2",param_name="",param_namespace="",param_not_found_action="",policy_name="mutatingadmissionpolicy2",mutatingadmissionpolicybinding="mutatingadmissionpolicybinding2"} 1
				`,
			MetricNames: []string{"kube_mutatingadmissionpolicybinding_created", "kube_mutatingadmissionpolicybinding_info"},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(mutatingAdmissionPolicyBindingMetricFamilies)
		c.Headers = generator.ExtractMetricFamilyHeaders(mutatingAdmissionPolicyBindingMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
