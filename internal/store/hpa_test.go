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
	"k8s.io/apimachinery/pkg/api/resource"
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
		# HELP kube_hpa_spec_target_metric The metric specifications used by this autoscaler when calculating the desired replica count.
		# TYPE kube_hpa_spec_target_metric gauge
		# HELP kube_hpa_status_current_replicas Current number of replicas of pods managed by this autoscaler.
		# TYPE kube_hpa_status_current_replicas gauge
		# HELP kube_hpa_status_desired_replicas Desired number of replicas of pods managed by this autoscaler.
		# TYPE kube_hpa_status_desired_replicas gauge
        # HELP kube_hpa_status_condition The condition of this autoscaler.
        # TYPE kube_hpa_status_condition gauge
        # HELP kube_hpa_labels Kubernetes labels converted to Prometheus labels.
        # TYPE kube_hpa_labels gauge
        # HELP kube_hpa_status_current_metrics_average_value Average metric value observed by the autoscaler.
        # TYPE kube_hpa_status_current_metrics_average_value gauge
        # HELP kube_hpa_status_current_metrics_average_utilization Average metric utilization observed by the autoscaler.
        # TYPE kube_hpa_status_current_metrics_average_utilization gauge
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
					Metrics: []autoscaling.MetricSpec{
						{
							Type: autoscaling.ObjectMetricSourceType,
							Object: &autoscaling.ObjectMetricSource{
								MetricName:   "hits",
								TargetValue:  resource.MustParse("10"),
								AverageValue: resourcePtr(resource.MustParse("12")),
							},
						},
						{
							Type: autoscaling.PodsMetricSourceType,
							Pods: &autoscaling.PodsMetricSource{
								MetricName:         "transactions_processed",
								TargetAverageValue: resource.MustParse("33"),
							},
						},
						{
							Type: autoscaling.ResourceMetricSourceType,
							Resource: &autoscaling.ResourceMetricSource{
								Name:                     "cpu",
								TargetAverageUtilization: int32ptr(80),
							},
						},
						{
							Type: autoscaling.ResourceMetricSourceType,
							Resource: &autoscaling.ResourceMetricSource{
								Name:                     "memory",
								TargetAverageUtilization: int32ptr(80),
								TargetAverageValue:       resourcePtr(resource.MustParse("800Ki")),
							},
						},
						// No targets, this metric should be ignored
						{
							Type: autoscaling.ResourceMetricSourceType,
							Resource: &autoscaling.ResourceMetricSource{
								Name: "disk",
							},
						},
						{
							Type: autoscaling.ExternalMetricSourceType,
							External: &autoscaling.ExternalMetricSource{
								MetricName:  "sqs_jobs",
								TargetValue: resourcePtr(resource.MustParse("30")),
							},
						},
						{
							Type: autoscaling.ExternalMetricSourceType,
							External: &autoscaling.ExternalMetricSource{
								MetricName:         "events",
								TargetAverageValue: resourcePtr(resource.MustParse("30")),
							},
						},
					},
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
							Reason: "reason",
						},
					},
					CurrentMetrics: []autoscaling.MetricStatus{
						{
							Type: "Resource",
							Resource: &autoscaling.ResourceMetricStatus{
								Name:                      "cpu",
								CurrentAverageUtilization: new(int32),
								CurrentAverageValue:       resource.MustParse("7m"),
							},
						},
					},
				},
			},
			Want: metadata + `
				kube_hpa_labels{hpa="hpa1",label_app="foobar",namespace="ns1"} 1
				kube_hpa_metadata_generation{hpa="hpa1",namespace="ns1"} 2
				kube_hpa_spec_max_replicas{hpa="hpa1",namespace="ns1"} 4
				kube_hpa_spec_min_replicas{hpa="hpa1",namespace="ns1"} 2
				kube_hpa_spec_target_metric{hpa="hpa1",namespace="ns1",metric_name="hits",metric_target_type="value"} 10
				kube_hpa_spec_target_metric{hpa="hpa1",namespace="ns1",metric_name="hits",metric_target_type="average"} 12
				kube_hpa_spec_target_metric{hpa="hpa1",namespace="ns1",metric_name="transactions_processed",metric_target_type="average"} 33
				kube_hpa_spec_target_metric{hpa="hpa1",namespace="ns1",metric_name="cpu",metric_target_type="utilization"} 80
				kube_hpa_spec_target_metric{hpa="hpa1",namespace="ns1",metric_name="memory",metric_target_type="utilization"} 80
				kube_hpa_spec_target_metric{hpa="hpa1",namespace="ns1",metric_name="memory",metric_target_type="average"} 819200
				kube_hpa_spec_target_metric{hpa="hpa1",namespace="ns1",metric_name="sqs_jobs",metric_target_type="value"} 30
				kube_hpa_spec_target_metric{hpa="hpa1",namespace="ns1",metric_name="events",metric_target_type="average"} 30
				kube_hpa_status_condition{condition="AbleToScale",hpa="hpa1",namespace="ns1",status="false"} 0
				kube_hpa_status_condition{condition="AbleToScale",hpa="hpa1",namespace="ns1",status="true"} 1
				kube_hpa_status_condition{condition="AbleToScale",hpa="hpa1",namespace="ns1",status="unknown"} 0
				kube_hpa_status_current_replicas{hpa="hpa1",namespace="ns1"} 2
				kube_hpa_status_desired_replicas{hpa="hpa1",namespace="ns1"} 2
				kube_hpa_status_current_metrics_average_value{hpa="hpa1",namespace="ns1"} 0.007
				kube_hpa_status_current_metrics_average_utilization{hpa="hpa1",namespace="ns1"} 0
			`,
			MetricNames: []string{
				"kube_hpa_metadata_generation",
				"kube_hpa_spec_max_replicas",
				"kube_hpa_spec_min_replicas",
				"kube_hpa_spec_target_metric",
				"kube_hpa_status_current_replicas",
				"kube_hpa_status_desired_replicas",
				"kube_hpa_status_condition",
				"kube_hpa_labels",
				"kube_hpa_status_current_metrics_average_value",
				"kube_hpa_status_current_metrics_average_utilization",
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

func int32ptr(value int32) *int32 {
	return &value
}

func resourcePtr(quantity resource.Quantity) *resource.Quantity {
	return &quantity
}
