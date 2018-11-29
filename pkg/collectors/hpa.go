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

package collectors

import (
	"k8s.io/kube-state-metrics/pkg/metrics"

	autoscaling "k8s.io/api/autoscaling/v2beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descHorizontalPodAutoscalerLabelsName          = "kube_hpa_labels"
	descHorizontalPodAutoscalerLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descHorizontalPodAutoscalerLabelsDefaultLabels = []string{"namespace", "hpa"}

	hpaMetricFamilies = []metrics.FamilyGenerator{
		metrics.FamilyGenerator{
			Name: "kube_hpa_metadata_generation",
			Type: metrics.MetricTypeGauge,
			Help: "The generation observed by the HorizontalPodAutoscaler controller.",
			GenerateFunc: wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_hpa_metadata_generation",
					Value: float64(a.ObjectMeta.Generation),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_hpa_spec_max_replicas",
			Type: metrics.MetricTypeGauge,
			Help: "Upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas.",
			GenerateFunc: wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_hpa_spec_max_replicas",
					Value: float64(a.Spec.MaxReplicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_hpa_spec_min_replicas",
			Type: metrics.MetricTypeGauge,
			Help: "Lower limit for the number of pods that can be set by the autoscaler, default 1.",
			GenerateFunc: wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_hpa_spec_min_replicas",
					Value: float64(*a.Spec.MinReplicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_hpa_status_current_replicas",
			Type: metrics.MetricTypeGauge,
			Help: "Current number of replicas of pods managed by this autoscaler.",
			GenerateFunc: wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_hpa_status_current_replicas",
					Value: float64(a.Status.CurrentReplicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_hpa_status_desired_replicas",
			Type: metrics.MetricTypeGauge,
			Help: "Desired number of replicas of pods managed by this autoscaler.",
			GenerateFunc: wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_hpa_status_desired_replicas",
					Value: float64(a.Status.DesiredReplicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: descHorizontalPodAutoscalerLabelsName,
			Type: metrics.MetricTypeGauge,
			Help: descHorizontalPodAutoscalerLabelsHelp,
			GenerateFunc: wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) metrics.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(a.Labels)
				return metrics.Family{&metrics.Metric{
					Name:        descHorizontalPodAutoscalerLabelsName,
					LabelKeys:   labelKeys,
					LabelValues: labelValues,
					Value:       1,
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_hpa_status_condition",
			Type: metrics.MetricTypeGauge,
			Help: "The condition of this autoscaler.",
			GenerateFunc: wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) metrics.Family {
				f := metrics.Family{}

				for _, c := range a.Status.Conditions {
					metrics := addConditionMetrics(c.Status)

					for _, m := range metrics {
						metric := m
						metric.Name = "kube_hpa_status_condition"
						metric.LabelKeys = []string{"condition", "status"}
						metric.LabelValues = append(metric.LabelValues, string(c.Type))
						f = append(f, metric)
					}
				}

				return f
			}),
		},
	}
)

func wrapHPAFunc(f func(*autoscaling.HorizontalPodAutoscaler) metrics.Family) func(interface{}) metrics.Family {
	return func(obj interface{}) metrics.Family {
		hpa := obj.(*autoscaling.HorizontalPodAutoscaler)

		metricFamily := f(hpa)

		for _, m := range metricFamily {
			m.LabelKeys = append(descHorizontalPodAutoscalerLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{hpa.Namespace, hpa.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createHPAListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AutoscalingV2beta1().HorizontalPodAutoscalers(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AutoscalingV2beta1().HorizontalPodAutoscalers(ns).Watch(opts)
		},
	}
}
