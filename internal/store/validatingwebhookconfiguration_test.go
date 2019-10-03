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
	"testing"

	admissionregistration "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kube-state-metrics/pkg/metric"
)

func TestValidatingWebhookConfigurationStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)

	cases := []generateMetricsTestCase{
		{
			Obj: &admissionregistration.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "validatingwebhookconfiguration1",
					Namespace:       "ns1",
					ResourceVersion: "123456",
				},
			},
			Want: `
				# HELP kube_validatingwebhookconfiguration_info Information about the ValidatingWebhookConfiguration.
				# HELP kube_validatingwebhookconfiguration_metadata_resource_version Resource version representing a specific version of the ValidatingWebhookConfiguration.
				# TYPE kube_validatingwebhookconfiguration_info gauge
				# TYPE kube_validatingwebhookconfiguration_metadata_resource_version gauge
				kube_validatingwebhookconfiguration_info{validatingwebhookconfiguration="validatingwebhookconfiguration1",namespace="ns1"} 1
				kube_validatingwebhookconfiguration_metadata_resource_version{validatingwebhookconfiguration="validatingwebhookconfiguration1",namespace="ns1",resource_version="123456"} 1
				`,
			MetricNames: []string{"kube_validatingwebhookconfiguration_info", "kube_validatingwebhookconfiguration_metadata_resource_version"},
		},
		{
			Obj: &admissionregistration.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "validatingwebhookconfiguration2",
					Namespace:         "ns2",
					CreationTimestamp: metav1StartTime,
					ResourceVersion:   "abcdef",
				},
			},
			Want: `
			# HELP kube_validatingwebhookconfiguration_created Unix creation timestamp.
			# HELP kube_validatingwebhookconfiguration_info Information about the ValidatingWebhookConfiguration.
			# HELP kube_validatingwebhookconfiguration_metadata_resource_version Resource version representing a specific version of the ValidatingWebhookConfiguration.
			# TYPE kube_validatingwebhookconfiguration_created gauge
			# TYPE kube_validatingwebhookconfiguration_info gauge
			# TYPE kube_validatingwebhookconfiguration_metadata_resource_version gauge
			kube_validatingwebhookconfiguration_created{validatingwebhookconfiguration="validatingwebhookconfiguration2",namespace="ns2"} 1.501569018e+09
			kube_validatingwebhookconfiguration_info{validatingwebhookconfiguration="validatingwebhookconfiguration2",namespace="ns2"} 1
			kube_validatingwebhookconfiguration_metadata_resource_version{validatingwebhookconfiguration="validatingwebhookconfiguration2",namespace="ns2",resource_version="abcdef"} 1
			`,
			MetricNames: []string{"kube_validatingwebhookconfiguration_created", "kube_validatingwebhookconfiguration_info", "kube_validatingwebhookconfiguration_metadata_resource_version"},
		},
	}
	for i, c := range cases {
		c.Func = metric.ComposeMetricGenFuncs(validatingWebhookConfigurationMetricFamilies)
		c.Headers = metric.ExtractMetricFamilyHeaders(validatingWebhookConfigurationMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
