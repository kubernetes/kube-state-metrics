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
	"context"

	autoscaling "k8s.io/api/autoscaling/v2beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

type metricTargetType int

const (
	value metricTargetType = iota
	utilization
	average

	metricTargetTypeCount // Used as a length argument to arrays
)

func (m metricTargetType) String() string {
	return [...]string{"value", "utilization", "average"}[m]
}

var (
	descHorizontalPodAutoscalerAnnotationsName     = "kube_horizontalpodautoscaler_annotations"
	descHorizontalPodAutoscalerAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descHorizontalPodAutoscalerLabelsName          = "kube_horizontalpodautoscaler_labels"
	descHorizontalPodAutoscalerLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descHorizontalPodAutoscalerLabelsDefaultLabels = []string{"namespace", "horizontalpodautoscaler"}

	targetMetricLabels = []string{"metric_name", "metric_target_type"}
)

func hpaMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			"kube_horizontalpodautoscaler_metadata_generation",
			"The generation observed by the HorizontalPodAutoscaler controller.",
			metric.Gauge,
			"",
			wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(a.ObjectMeta.Generation),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_horizontalpodautoscaler_spec_max_replicas",
			"Upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas.",
			metric.Gauge,
			"",
			wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(a.Spec.MaxReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_horizontalpodautoscaler_spec_min_replicas",
			"Lower limit for the number of pods that can be set by the autoscaler, default 1.",
			metric.Gauge,
			"",
			wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(*a.Spec.MinReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_horizontalpodautoscaler_spec_target_metric",
			"The metric specifications used by this autoscaler when calculating the desired replica count.",
			metric.Gauge,
			"",
			wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
				ms := make([]*metric.Metric, 0, len(a.Spec.Metrics))
				for _, m := range a.Spec.Metrics {
					var metricName string

					var v [metricTargetTypeCount]int64
					var ok [metricTargetTypeCount]bool

					switch m.Type {
					case autoscaling.ObjectMetricSourceType:
						metricName = m.Object.Metric.Name

						v[value], ok[value] = m.Object.Target.Value.AsInt64()
						if m.Object.Target.AverageValue != nil {
							v[average], ok[average] = m.Object.Target.AverageValue.AsInt64()
						}
					case autoscaling.PodsMetricSourceType:
						metricName = m.Pods.Metric.Name

						v[average], ok[average] = m.Pods.Target.AverageValue.AsInt64()
					case autoscaling.ResourceMetricSourceType:
						metricName = string(m.Resource.Name)

						if ok[utilization] = (m.Resource.Target.AverageUtilization != nil); ok[utilization] {
							v[utilization] = int64(*m.Resource.Target.AverageUtilization)
						}

						if m.Resource.Target.AverageValue != nil {
							v[average], ok[average] = m.Resource.Target.AverageValue.AsInt64()
						}
					case autoscaling.ExternalMetricSourceType:
						metricName = m.External.Metric.Name

						if m.External.Target.Value != nil {
							v[value], ok[value] = m.External.Target.Value.AsInt64()
						}
						if m.External.Target.AverageValue != nil {
							v[average], ok[average] = m.External.Target.AverageValue.AsInt64()
						}
					default:
						// Skip unsupported metric type
						continue
					}

					for i := range ok {
						if ok[i] {
							ms = append(ms, &metric.Metric{
								LabelKeys:   targetMetricLabels,
								LabelValues: []string{metricName, metricTargetType(i).String()},
								Value:       float64(v[i]),
							})
						}
					}
				}
				return &metric.Family{Metrics: ms}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_horizontalpodautoscaler_status_current_replicas",
			"Current number of replicas of pods managed by this autoscaler.",
			metric.Gauge,
			"",
			wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(a.Status.CurrentReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_horizontalpodautoscaler_status_desired_replicas",
			"Desired number of replicas of pods managed by this autoscaler.",
			metric.Gauge,
			"",
			wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(a.Status.DesiredReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			descHorizontalPodAutoscalerAnnotationsName,
			descHorizontalPodAutoscalerAnnotationsHelp,
			metric.Gauge,
			"",
			wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", a.Annotations, allowAnnotationsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   annotationKeys,
							LabelValues: annotationValues,
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			descHorizontalPodAutoscalerLabelsName,
			descHorizontalPodAutoscalerLabelsHelp,
			metric.Gauge,
			"",
			wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", a.Labels, allowLabelsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_horizontalpodautoscaler_status_condition",
			"The condition of this autoscaler.",
			metric.Gauge,
			"",
			wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
				ms := make([]*metric.Metric, 0, len(a.Status.Conditions)*len(conditionStatuses))

				for _, c := range a.Status.Conditions {
					metrics := addConditionMetrics(c.Status)

					for _, m := range metrics {
						metric := m
						metric.LabelKeys = []string{"condition", "status"}
						metric.LabelValues = append([]string{string(c.Type)}, metric.LabelValues...)
						ms = append(ms, metric)
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
}

func wrapHPAFunc(f func(*autoscaling.HorizontalPodAutoscaler) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		hpa := obj.(*autoscaling.HorizontalPodAutoscaler)

		metricFamily := f(hpa)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descHorizontalPodAutoscalerLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{hpa.Namespace, hpa.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createHPAListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AutoscalingV2beta2().HorizontalPodAutoscalers(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AutoscalingV2beta2().HorizontalPodAutoscalers(ns).Watch(context.TODO(), opts)
		},
	}
}
