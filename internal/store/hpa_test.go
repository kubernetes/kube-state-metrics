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

	v12 "k8s.io/api/core/v1"

	autoscaling "k8s.io/api/autoscaling/v2beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/pkg/metric"
)

var (
	hpa1MinReplicas int32 = 2
)

func TestHPAStore(t *testing.T) {
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
					Annotations: map[string]string{
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
							Status: v12.ConditionTrue,
						},
					},
				},
			},
			Want: `
                kube_hpa_labels{hpa="hpa1",label_app="foobar",namespace="ns1"} 1
				kube_hpa_metadata_generation{hpa="hpa1",namespace="ns1"} 2
				kube_hpa_spec_max_replicas{hpa="hpa1",namespace="ns1"} 4
				kube_hpa_spec_min_replicas{hpa="hpa1",namespace="ns1"} 2
                kube_hpa_status_condition{condition="false",hpa="hpa1",namespace="ns1",status="AbleToScale"} 0
                kube_hpa_status_condition{condition="true",hpa="hpa1",namespace="ns1",status="AbleToScale"} 1
                kube_hpa_status_condition{condition="unknown",hpa="hpa1",namespace="ns1",status="AbleToScale"} 0
				kube_hpa_status_current_replicas{hpa="hpa1",namespace="ns1"} 2
				kube_hpa_status_desired_replicas{hpa="hpa1",namespace="ns1"} 2
				kube_hpa_annotations{hpa="hpa1",namespace="ns1",annotation_app="foobar"} 1
			`,
			MetricNames: []string{
				"kube_hpa_metadata_generation",
				"kube_hpa_spec_max_replicas",
				"kube_hpa_spec_min_replicas",
				"kube_hpa_status_current_replicas",
				"kube_hpa_status_desired_replicas",
				"kube_hpa_status_condition",
				"kube_hpa_labels",
				"kube_hpa_annotations",
			},
		},
	}
	for i, c := range cases {
		c.Func = metric.ComposeMetricGenFuncs(hpaMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
