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

func TestValidatingAdmissionPolicyBindingStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)
	paramNotFoundActionDeny := admissionregistrationv1.DenyAction

	cases := []generateMetricsTestCase{
		{
			Obj: &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "validatingadmissionpolicybinding1",
					Namespace:       "ns1",
					ResourceVersion: "123456",
				},
				Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
					PolicyName: "validatingadmissionpolicy1",
					ParamRef: &admissionregistrationv1.ParamRef{
						Name:                    "param1",
						Namespace:               "ns1",
						ParameterNotFoundAction: &paramNotFoundActionDeny,
					},
					ValidationActions: []admissionregistrationv1.ValidationAction{
						admissionregistrationv1.Deny,
						admissionregistrationv1.Warn,
					},
				},
			},
			Want: `
				# HELP kube_validatingadmissionpolicybinding_info Information about the ValidatingAdmissionPolicyBinding.
				# HELP kube_validatingadmissionpolicybinding_validation_action Validation actions for the ValidatingAdmissionPolicyBinding.
				# TYPE kube_validatingadmissionpolicybinding_info gauge
				# TYPE kube_validatingadmissionpolicybinding_validation_action gauge
				kube_validatingadmissionpolicybinding_info{namespace="ns1",param_name="param1",param_namespace="ns1",param_not_found_action="Deny",policy_name="validatingadmissionpolicy1",validatingadmissionpolicybinding="validatingadmissionpolicybinding1"} 1
				kube_validatingadmissionpolicybinding_validation_action{action="Deny",namespace="ns1",validatingadmissionpolicybinding="validatingadmissionpolicybinding1"} 1
				kube_validatingadmissionpolicybinding_validation_action{action="Warn",namespace="ns1",validatingadmissionpolicybinding="validatingadmissionpolicybinding1"} 1
				`,
			MetricNames: []string{"kube_validatingadmissionpolicybinding_info", "kube_validatingadmissionpolicybinding_validation_action"},
		},
		{
			Obj: &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "validatingadmissionpolicybinding2",
					Namespace:         "ns2",
					CreationTimestamp: metav1StartTime,
					ResourceVersion:   "abcdef",
				},
				Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
					PolicyName: "validatingadmissionpolicy2",
				},
			},
			Want: `
				# HELP kube_validatingadmissionpolicybinding_created Unix creation timestamp.
				# HELP kube_validatingadmissionpolicybinding_info Information about the ValidatingAdmissionPolicyBinding.
				# TYPE kube_validatingadmissionpolicybinding_created gauge
				# TYPE kube_validatingadmissionpolicybinding_info gauge
				kube_validatingadmissionpolicybinding_created{namespace="ns2",validatingadmissionpolicybinding="validatingadmissionpolicybinding2"} 1.501569018e+09
				kube_validatingadmissionpolicybinding_info{namespace="ns2",param_name="",param_namespace="",param_not_found_action="",policy_name="validatingadmissionpolicy2",validatingadmissionpolicybinding="validatingadmissionpolicybinding2"} 1
				`,
			MetricNames: []string{"kube_validatingadmissionpolicybinding_created", "kube_validatingadmissionpolicybinding_info"},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(validatingAdmissionPolicyBindingMetricFamilies)
		c.Headers = generator.ExtractMetricFamilyHeaders(validatingAdmissionPolicyBindingMetricFamilies)
		c.FamilyGens = validatingAdmissionPolicyBindingMetricFamilies
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
