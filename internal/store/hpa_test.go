/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

	autoscaling "k8s.io/api/autoscaling/v2beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kube-state-metrics/pkg/metric"
)

var (
	hpa1MinReplicas int32 = 2
)

func TestHPAStore(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_hpa_metadata_generation The generation observed by the HorizontalPodAutoscaler controller.
		# TYPE kube_hpa_metadata_generation gauge
		# HELP kube_hpa_spec_max_replicas Upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas.
		# TYPE kube_hpa_spec_max_replicas gauge
		# HELP kube_hpa_spec_min_replicas Lower limit for the number of pods that can be set by the autoscaler, default 1.
		# TYPE kube_hpa_spec_min_replicas gauge
		# HELP kube_hpa_status_current_replicas Current number of replicas of pods managed by this autoscaler.
		# TYPE kube_hpa_status_current_replicas gauge
		# HELP kube_hpa_status_desired_replicas Desired number of replicas of pods managed by this autoscaler.
		# TYPE kube_hpa_status_desired_replicas gauge
		# HELP kube_hpa_status_condition The condition of this autoscaler.
		# TYPE kube_hpa_status_condition gauge
		# HELP kube_hpa_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_hpa_labels gauge
	`
	cases := []generateMetricsTestCase{
		{
			// Verify populating base metric.
			Obj: &autoscaling.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 2,
					Name:       "hpa1",
					Namespace:  "ns1",
					Labels: map[string]string{
						"app": "foobar",
					},
				},
				Spec: autoscaling.HorizontalPodAutoscalerSpec{
					MaxReplicas: 4,
					MinReplicas: &hpa1MinReplicas,
					ScaleTargetRef: autoscaling.CrossVersionObjectReference{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "deployment1",
					},
				},
				Status: autoscaling.HorizontalPodAutoscalerStatus{
					CurrentReplicas: 2,
					DesiredReplicas: 2,
					Conditions: []autoscaling.HorizontalPodAutoscalerCondition{
						{
							Type:   autoscaling.AbleToScale,
							Status: v1.ConditionTrue,
						},
					},
				},
			},
			Want: metadata + `
				kube_hpa_labels{hpa="hpa1",label_app="foobar",namespace="ns1"} 1
				kube_hpa_metadata_generation{hpa="hpa1",namespace="ns1"} 2
				kube_hpa_spec_max_replicas{hpa="hpa1",namespace="ns1"} 4
				kube_hpa_spec_min_replicas{hpa="hpa1",namespace="ns1"} 2
				kube_hpa_status_condition{condition="AbleToScale",hpa="hpa1",namespace="ns1",status="false"} 0
				kube_hpa_status_condition{condition="AbleToScale",hpa="hpa1",namespace="ns1",status="true"} 1
				kube_hpa_status_condition{condition="AbleToScale",hpa="hpa1",namespace="ns1",status="unknown"} 0
				kube_hpa_status_current_replicas{hpa="hpa1",namespace="ns1"} 2
				kube_hpa_status_desired_replicas{hpa="hpa1",namespace="ns1"} 2
			`,
			MetricNames: []string{
				"kube_hpa_metadata_generation",
				"kube_hpa_spec_max_replicas",
				"kube_hpa_spec_min_replicas",
				"kube_hpa_status_current_replicas",
				"kube_hpa_status_desired_replicas",
				"kube_hpa_status_condition",
				"kube_hpa_labels",
			},
		},
	}
	for i, c := range cases {
		c.Func = metric.ComposeMetricGenFuncs(hpaMetricFamilies)
		c.Headers = metric.ExtractMetricFamilyHeaders(hpaMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
