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
	"time"

	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestDeviceTaintRuleStore(t *testing.T) {
	const metadata = `
		# HELP kube_devicetaintrule_created Unix creation timestamp
		# TYPE kube_devicetaintrule_created gauge
		# HELP kube_devicetaintrule_info Information about device taint rule.
		# TYPE kube_devicetaintrule_info gauge
		# HELP kube_devicetaintrule_taint_time_added_timestamp_seconds Unix timestamp at which the taint was added or its effect was last modified.
		# TYPE kube_devicetaintrule_taint_time_added_timestamp_seconds gauge
		# HELP kube_devicetaintrule_metadata_generation Sequence number representing a specific generation of the desired state.
		# TYPE kube_devicetaintrule_metadata_generation gauge
		# HELP kube_devicetaintrule_status_condition The condition of a device taint rule.
		# TYPE kube_devicetaintrule_status_condition gauge
	`
	metricNames := []string{
		"kube_devicetaintrule_created",
		"kube_devicetaintrule_info",
		"kube_devicetaintrule_taint_time_added_timestamp_seconds",
		"kube_devicetaintrule_metadata_generation",
		"kube_devicetaintrule_status_condition",
	}

	cases := []generateMetricsTestCase{
		{
			Obj: &resourcev1.DeviceTaintRule{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "rule-1",
					Generation:        2,
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Spec: resourcev1.DeviceTaintRuleSpec{
					DeviceSelector: &resourcev1.DeviceTaintSelector{
						Driver: ptr.To("driver-1"),
						Pool:   ptr.To("pool-1"),
						Device: ptr.To("device-1"),
					},
					Taint: resourcev1.DeviceTaint{
						Key:       "example.com/unhealthy",
						Value:     "ecc",
						Effect:    resourcev1.DeviceTaintEffectNoExecute,
						TimeAdded: &metav1.Time{Time: time.Unix(1600000000, 0)},
					},
				},
				Status: resourcev1.DeviceTaintRuleStatus{
					Conditions: []metav1.Condition{
						{
							Type:    resourcev1.DeviceTaintConditionEvictionInProgress,
							Status:  metav1.ConditionTrue,
							Reason:  "PodsPendingEviction",
							Message: "2 pods need to be evicted",
						},
					},
				},
			},
			Want: metadata + `
				kube_devicetaintrule_created{devicetaintrule="rule-1"} 1.5e+09
				kube_devicetaintrule_info{device="device-1",device_selector="filtered",devicetaintrule="rule-1",driver="driver-1",pool="pool-1",taint_effect="NoExecute",taint_key="example.com/unhealthy",taint_value="ecc"} 1
				kube_devicetaintrule_taint_time_added_timestamp_seconds{devicetaintrule="rule-1"} 1.6e+09
				kube_devicetaintrule_metadata_generation{devicetaintrule="rule-1"} 2
				kube_devicetaintrule_status_condition{condition="EvictionInProgress",devicetaintrule="rule-1",status="false"} 0
				kube_devicetaintrule_status_condition{condition="EvictionInProgress",devicetaintrule="rule-1",status="true"} 1
				kube_devicetaintrule_status_condition{condition="EvictionInProgress",devicetaintrule="rule-1",status="unknown"} 0
			`,
			MetricNames: metricNames,
		},
		{
			Obj: &resourcev1.DeviceTaintRule{
				ObjectMeta: metav1.ObjectMeta{Name: "rule-minimal"},
				Spec: resourcev1.DeviceTaintRuleSpec{
					Taint: resourcev1.DeviceTaint{
						Key:    "example.com/maintenance",
						Effect: resourcev1.DeviceTaintEffectNone,
					},
				},
			},
			Want: metadata + `
				kube_devicetaintrule_info{device="",device_selector="none",devicetaintrule="rule-minimal",driver="",pool="",taint_effect="None",taint_key="example.com/maintenance",taint_value=""} 1
				kube_devicetaintrule_metadata_generation{devicetaintrule="rule-minimal"} 0
			`,
			MetricNames: metricNames,
		},
		{
			Obj: &resourcev1.DeviceTaintRule{
				ObjectMeta: metav1.ObjectMeta{Name: "rule-all-devices"},
				Spec: resourcev1.DeviceTaintRuleSpec{
					DeviceSelector: &resourcev1.DeviceTaintSelector{},
					Taint: resourcev1.DeviceTaint{
						Key:    "example.com/drain",
						Effect: resourcev1.DeviceTaintEffectNoExecute,
					},
				},
			},
			Want: `
				# HELP kube_devicetaintrule_info Information about device taint rule.
				# TYPE kube_devicetaintrule_info gauge
				kube_devicetaintrule_info{device="",device_selector="all",devicetaintrule="rule-all-devices",driver="",pool="",taint_effect="NoExecute",taint_key="example.com/drain",taint_value=""} 1
			`,
			MetricNames: []string{"kube_devicetaintrule_info"},
		},
		{
			AllowAnnotationsList: []string{"team"},
			AllowLabelsList:      []string{"app"},
			Obj: &resourcev1.DeviceTaintRule{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "rule-labels",
					Annotations: map[string]string{"team": "gpu", "ignored": "x"},
					Labels:      map[string]string{"app": "trainer", "ignored": "x"},
				},
			},
			Want: `
				# HELP kube_devicetaintrule_annotations Kubernetes annotations converted to Prometheus labels.
				# TYPE kube_devicetaintrule_annotations gauge
				# HELP kube_devicetaintrule_labels Kubernetes labels converted to Prometheus labels.
				# TYPE kube_devicetaintrule_labels gauge
				kube_devicetaintrule_annotations{annotation_team="gpu",devicetaintrule="rule-labels"} 1
				kube_devicetaintrule_labels{devicetaintrule="rule-labels",label_app="trainer"} 1
			`,
			MetricNames: []string{"kube_devicetaintrule_annotations", "kube_devicetaintrule_labels"},
		},
	}

	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(deviceTaintRuleMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(deviceTaintRuleMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %dth run:\n%v", i, err)
		}
	}
}
