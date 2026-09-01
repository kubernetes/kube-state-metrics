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

func TestValidatingAdmissionPolicyStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)
	failurePolicyFail := admissionregistrationv1.Fail
	failurePolicyIgnore := admissionregistrationv1.Ignore

	cases := []generateMetricsTestCase{
		{
			Obj: &admissionregistrationv1.ValidatingAdmissionPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "validatingadmissionpolicy1",
					Namespace:       "ns1",
					ResourceVersion: "123456",
				},
				Spec: admissionregistrationv1.ValidatingAdmissionPolicySpec{
					ParamKind: &admissionregistrationv1.ParamKind{
						APIVersion: "rules.example.com/v1",
						Kind:       "Rule",
					},
					FailurePolicy: &failurePolicyFail,
				},
			},
			Want: `
				# HELP kube_validatingadmissionpolicy_info Information about the ValidatingAdmissionPolicy.
				# TYPE kube_validatingadmissionpolicy_info gauge
				kube_validatingadmissionpolicy_info{failure_policy="Fail",namespace="ns1",param_api_version="rules.example.com/v1",param_kind="Rule",validatingadmissionpolicy="validatingadmissionpolicy1"} 1
				`,
			MetricNames: []string{"kube_validatingadmissionpolicy_info"},
		},
		{
			Obj: &admissionregistrationv1.ValidatingAdmissionPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "validatingadmissionpolicy2",
					Namespace:         "ns2",
					CreationTimestamp: metav1StartTime,
					ResourceVersion:   "abcdef",
				},
				Spec: admissionregistrationv1.ValidatingAdmissionPolicySpec{
					FailurePolicy: &failurePolicyIgnore,
				},
			},
			Want: `
				# HELP kube_validatingadmissionpolicy_created Unix creation timestamp.
				# HELP kube_validatingadmissionpolicy_info Information about the ValidatingAdmissionPolicy.
				# TYPE kube_validatingadmissionpolicy_created gauge
				# TYPE kube_validatingadmissionpolicy_info gauge
				kube_validatingadmissionpolicy_created{namespace="ns2",validatingadmissionpolicy="validatingadmissionpolicy2"} 1.501569018e+09
				kube_validatingadmissionpolicy_info{failure_policy="Ignore",namespace="ns2",param_api_version="",param_kind="",validatingadmissionpolicy="validatingadmissionpolicy2"} 1
				`,
			MetricNames: []string{"kube_validatingadmissionpolicy_created", "kube_validatingadmissionpolicy_info"},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(validatingAdmissionPolicyMetricFamilies)
		c.Headers = generator.ExtractMetricFamilyHeaders(validatingAdmissionPolicyMetricFamilies)
		c.FamilyGens = validatingAdmissionPolicyMetricFamilies
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
