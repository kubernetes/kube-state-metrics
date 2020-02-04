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

	generator "k8s.io/kube-state-metrics/pkg/metric_generator"
)

var (
	hpa1MinReplicas int32 = 2
)

func TestHPAStore(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_horizontalpodautoscaler_labels Kubernetes labels converted to Prometheus labels.
		# HELP kube_horizontalpodautoscaler_metadata_generation The generation observed by the HorizontalPodAutoscaler controller.
		# HELP kube_horizontalpodautoscaler_spec_max_replicas Upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas.
		# HELP kube_horizontalpodautoscaler_spec_min_replicas Lower limit for the number of pods that can be set by the autoscaler, default 1.
		# HELP kube_horizontalpodautoscaler_spec_target_metric The metric specifications used by this autoscaler when calculating the desired replica count.
		# HELP kube_horizontalpodautoscaler_status_condition The condition of this autoscaler.
		# HELP kube_horizontalpodautoscaler_status_current_replicas Current number of replicas of pods managed by this autoscaler.
		# HELP kube_horizontalpodautoscaler_status_desired_replicas Desired number of replicas of pods managed by this autoscaler.
		# TYPE kube_horizontalpodautoscaler_labels gauge
		# TYPE kube_horizontalpodautoscaler_metadata_generation gauge
		# TYPE kube_horizontalpodautoscaler_spec_max_replicas gauge
		# TYPE kube_horizontalpodautoscaler_spec_min_replicas gauge
		# TYPE kube_horizontalpodautoscaler_spec_target_metric gauge
		# TYPE kube_horizontalpodautoscaler_status_condition gauge
		# TYPE kube_horizontalpodautoscaler_status_current_replicas gauge
		# TYPE kube_horizontalpodautoscaler_status_desired_replicas gauge
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
						{
							Type: "Resource",
							Resource: &autoscaling.ResourceMetricStatus{
								Name:                      "memory",
								CurrentAverageUtilization: new(int32),
								CurrentAverageValue:       resource.MustParse("26335914666m"),
							},
						},
					},
				},
			},
			Want: metadata + `
				kube_horizontalpodautoscaler_labels{horizontalpodautoscaler="hpa1",label_app="foobar",namespace="ns1"} 1
				kube_horizontalpodautoscaler_metadata_generation{horizontalpodautoscaler="hpa1",namespace="ns1"} 2
				kube_horizontalpodautoscaler_spec_max_replicas{horizontalpodautoscaler="hpa1",namespace="ns1"} 4
				kube_horizontalpodautoscaler_spec_min_replicas{horizontalpodautoscaler="hpa1",namespace="ns1"} 2
				kube_horizontalpodautoscaler_spec_target_metric{horizontalpodautoscaler="hpa1",metric_name="cpu",metric_target_type="utilization",namespace="ns1"} 80
				kube_horizontalpodautoscaler_spec_target_metric{horizontalpodautoscaler="hpa1",metric_name="events",metric_target_type="average",namespace="ns1"} 30
				kube_horizontalpodautoscaler_spec_target_metric{horizontalpodautoscaler="hpa1",metric_name="hits",metric_target_type="average",namespace="ns1"} 12
				kube_horizontalpodautoscaler_spec_target_metric{horizontalpodautoscaler="hpa1",metric_name="hits",metric_target_type="value",namespace="ns1"} 10
				kube_horizontalpodautoscaler_spec_target_metric{horizontalpodautoscaler="hpa1",metric_name="memory",metric_target_type="average",namespace="ns1"} 819200
				kube_horizontalpodautoscaler_spec_target_metric{horizontalpodautoscaler="hpa1",metric_name="memory",metric_target_type="utilization",namespace="ns1"} 80
				kube_horizontalpodautoscaler_spec_target_metric{horizontalpodautoscaler="hpa1",metric_name="sqs_jobs",metric_target_type="value",namespace="ns1"} 30
				kube_horizontalpodautoscaler_spec_target_metric{horizontalpodautoscaler="hpa1",metric_name="transactions_processed",metric_target_type="average",namespace="ns1"} 33
				kube_horizontalpodautoscaler_status_condition{condition="AbleToScale",horizontalpodautoscaler="hpa1",namespace="ns1",status="false"} 0
				kube_horizontalpodautoscaler_status_condition{condition="AbleToScale",horizontalpodautoscaler="hpa1",namespace="ns1",status="true"} 1
				kube_horizontalpodautoscaler_status_condition{condition="AbleToScale",horizontalpodautoscaler="hpa1",namespace="ns1",status="unknown"} 0
				kube_horizontalpodautoscaler_status_current_replicas{horizontalpodautoscaler="hpa1",namespace="ns1"} 2
				kube_horizontalpodautoscaler_status_desired_replicas{horizontalpodautoscaler="hpa1",namespace="ns1"} 2
			`,
			MetricNames: []string{
				"kube_horizontalpodautoscaler_metadata_generation",
				"kube_horizontalpodautoscaler_spec_max_replicas",
				"kube_horizontalpodautoscaler_spec_min_replicas",
				"kube_horizontalpodautoscaler_spec_target_metric",
				"kube_horizontalpodautoscaler_status_current_replicas",
				"kube_horizontalpodautoscaler_status_desired_replicas",
				"kube_horizontalpodautoscaler_status_condition",
				"kube_horizontalpodautoscaler_labels",
			},
		},
		{
			// Verify populating base metric.
			Obj: &autoscaling.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 2,
					Name:       "hpa2",
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
							Type: autoscaling.ResourceMetricSourceType,
							Resource: &autoscaling.ResourceMetricSource{
								Name:                     "memory",
								TargetAverageUtilization: int32ptr(75),
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
							Type: autoscaling.ExternalMetricSourceType,
							External: &autoscaling.ExternalMetricSource{
								MetricName:  "traefik_backend_requests_per_second",
								TargetValue: resourcePtr(resource.MustParse("100")),
							},
						},
						{
							Type: autoscaling.ExternalMetricSourceType,
							External: &autoscaling.ExternalMetricSource{
								MetricName:  "traefik_backend_errors_per_second",
								TargetValue: resourcePtr(resource.MustParse("100")),
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
								Name:                      "memory",
								CurrentAverageUtilization: int32ptr(28),
								CurrentAverageValue:       resource.MustParse("847775744"),
							},
						},
						{
							Type: "Resource",
							Resource: &autoscaling.ResourceMetricStatus{
								Name:                      "cpu",
								CurrentAverageUtilization: int32ptr(6),
								CurrentAverageValue:       resource.MustParse("62m"),
							},
						},
						{
							Type: "External",
							External: &autoscaling.ExternalMetricStatus{
								MetricName:          "traefik_backend_requests_per_second",
								CurrentValue:        resource.MustParse("0"),
								CurrentAverageValue: resourcePtr(resource.MustParse("2900m")),
							},
						},
						{
							Type: "External",
							External: &autoscaling.ExternalMetricStatus{
								MetricName:   "traefik_backend_errors_per_second",
								CurrentValue: resource.MustParse("0"),
							},
						},
					},
				},
			},
			Want: metadata + `
				kube_horizontalpodautoscaler_labels{horizontalpodautoscaler="hpa2",label_app="foobar",namespace="ns1"} 1
				kube_horizontalpodautoscaler_metadata_generation{horizontalpodautoscaler="hpa2",namespace="ns1"} 2
				kube_horizontalpodautoscaler_spec_max_replicas{horizontalpodautoscaler="hpa2",namespace="ns1"} 4
				kube_horizontalpodautoscaler_spec_min_replicas{horizontalpodautoscaler="hpa2",namespace="ns1"} 2
				kube_horizontalpodautoscaler_spec_target_metric{horizontalpodautoscaler="hpa2",metric_name="cpu",metric_target_type="utilization",namespace="ns1"} 80
				kube_horizontalpodautoscaler_spec_target_metric{horizontalpodautoscaler="hpa2",metric_name="memory",metric_target_type="utilization",namespace="ns1"} 75
				kube_horizontalpodautoscaler_spec_target_metric{horizontalpodautoscaler="hpa2",metric_name="traefik_backend_errors_per_second",metric_target_type="value",namespace="ns1"} 100
				kube_horizontalpodautoscaler_spec_target_metric{horizontalpodautoscaler="hpa2",metric_name="traefik_backend_requests_per_second",metric_target_type="value",namespace="ns1"} 100
				kube_horizontalpodautoscaler_status_condition{condition="AbleToScale",horizontalpodautoscaler="hpa2",namespace="ns1",status="false"} 0
				kube_horizontalpodautoscaler_status_condition{condition="AbleToScale",horizontalpodautoscaler="hpa2",namespace="ns1",status="true"} 1
				kube_horizontalpodautoscaler_status_condition{condition="AbleToScale",horizontalpodautoscaler="hpa2",namespace="ns1",status="unknown"} 0
				kube_horizontalpodautoscaler_status_current_replicas{horizontalpodautoscaler="hpa2",namespace="ns1"} 2
				kube_horizontalpodautoscaler_status_desired_replicas{horizontalpodautoscaler="hpa2",namespace="ns1"} 2
			`,
			MetricNames: []string{
				"kube_horizontalpodautoscaler_metadata_generation",
				"kube_horizontalpodautoscaler_spec_max_replicas",
				"kube_horizontalpodautoscaler_spec_min_replicas",
				"kube_horizontalpodautoscaler_spec_target_metric",
				"kube_horizontalpodautoscaler_status_current_replicas",
				"kube_horizontalpodautoscaler_status_desired_replicas",
				"kube_horizontalpodautoscaler_status_condition",
				"kube_horizontalpodautoscaler_labels",
			},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(hpaMetricFamilies)
		c.Headers = generator.ExtractMetricFamilyHeaders(hpaMetricFamilies)
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
