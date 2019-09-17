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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	autoscaling "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/pkg/constant"
	"k8s.io/kube-state-metrics/pkg/metric"
)

var (
	descVerticalPodAutoscalerLabelsName          = "kube_verticalpodautoscaler_labels"
	descVerticalPodAutoscalerLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descVerticalPodAutoscalerLabelsDefaultLabels = []string{"namespace", "verticalpodautoscaler", "target_api_version", "target_kind", "target_name"}

	vpaMetricFamilies = []metric.FamilyGenerator{
		{
			Name: descVerticalPodAutoscalerLabelsName,
			Type: metric.Gauge,
			Help: descVerticalPodAutoscalerLabelsHelp,
			GenerateFunc: wrapVPAFunc(func(a *autoscaling.VerticalPodAutoscaler) *metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(a.Labels)
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
		},
		{
			Name: "kube_verticalpodautoscaler_spec_updatepolicy_updatemode",
			Type: metric.Gauge,
			Help: "Update mode of the VerticalPodAutoscaler.",
			GenerateFunc: wrapVPAFunc(func(a *autoscaling.VerticalPodAutoscaler) *metric.Family {
				ms := []*metric.Metric{}

				if a.Spec.UpdatePolicy == nil || a.Spec.UpdatePolicy.UpdateMode == nil {
					return &metric.Family{
						Metrics: ms,
					}
				}

				for _, mode := range []autoscaling.UpdateMode{
					autoscaling.UpdateModeOff,
					autoscaling.UpdateModeInitial,
					autoscaling.UpdateModeRecreate,
					autoscaling.UpdateModeAuto,
				} {
					var v float64
					if *a.Spec.UpdatePolicy.UpdateMode == mode {
						v = 1
					} else {
						v = 0
					}
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"update_mode"},
						LabelValues: []string{string(mode)},
						Value:       v,
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_minallowed",
			Type: metric.Gauge,
			Help: "Minimum resources the VerticalPodAutoscaler can set for containers matching the name.",
			GenerateFunc: wrapVPAFunc(func(a *autoscaling.VerticalPodAutoscaler) *metric.Family {
				ms := []*metric.Metric{}
				if a.Spec.ResourcePolicy == nil || a.Spec.ResourcePolicy.ContainerPolicies == nil {
					return &metric.Family{
						Metrics: ms,
					}
				}

				for _, c := range a.Spec.ResourcePolicy.ContainerPolicies {
					ms = append(ms, vpaResourcesToMetrics(c.ContainerName, c.MinAllowed)...)

				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_maxallowed",
			Type: metric.Gauge,
			Help: "Maximum resources the VerticalPodAutoscaler can set for containers matching the name.",
			GenerateFunc: wrapVPAFunc(func(a *autoscaling.VerticalPodAutoscaler) *metric.Family {
				ms := []*metric.Metric{}
				if a.Spec.ResourcePolicy == nil || a.Spec.ResourcePolicy.ContainerPolicies == nil {
					return &metric.Family{
						Metrics: ms,
					}
				}

				for _, c := range a.Spec.ResourcePolicy.ContainerPolicies {
					ms = append(ms, vpaResourcesToMetrics(c.ContainerName, c.MaxAllowed)...)
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound",
			Type: metric.Gauge,
			Help: "Minimum resources the container can use before the VerticalPodAutoscaler updater evicts it.",
			GenerateFunc: wrapVPAFunc(func(a *autoscaling.VerticalPodAutoscaler) *metric.Family {
				ms := []*metric.Metric{}
				if a.Status.Recommendation == nil || a.Status.Recommendation.ContainerRecommendations == nil {
					return &metric.Family{
						Metrics: ms,
					}
				}

				for _, c := range a.Status.Recommendation.ContainerRecommendations {
					ms = append(ms, vpaResourcesToMetrics(c.ContainerName, c.LowerBound)...)
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_verticalpodautoscaler_status_recommendation_containerrecommendations_upperbound",
			Type: metric.Gauge,
			Help: "Maximum resources the container can use before the VerticalPodAutoscaler updater evicts it.",
			GenerateFunc: wrapVPAFunc(func(a *autoscaling.VerticalPodAutoscaler) *metric.Family {
				ms := []*metric.Metric{}
				if a.Status.Recommendation == nil || a.Status.Recommendation.ContainerRecommendations == nil {
					return &metric.Family{
						Metrics: ms,
					}
				}

				for _, c := range a.Status.Recommendation.ContainerRecommendations {
					ms = append(ms, vpaResourcesToMetrics(c.ContainerName, c.UpperBound)...)
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_verticalpodautoscaler_status_recommendation_containerrecommendations_target",
			Type: metric.Gauge,
			Help: "Target resources the VerticalPodAutoscaler recommends for the container.",
			GenerateFunc: wrapVPAFunc(func(a *autoscaling.VerticalPodAutoscaler) *metric.Family {
				ms := []*metric.Metric{}
				if a.Status.Recommendation == nil || a.Status.Recommendation.ContainerRecommendations == nil {
					return &metric.Family{
						Metrics: ms,
					}
				}
				for _, c := range a.Status.Recommendation.ContainerRecommendations {
					ms = append(ms, vpaResourcesToMetrics(c.ContainerName, c.Target)...)
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_verticalpodautoscaler_status_recommendation_containerrecommendations_uncappedtarget",
			Type: metric.Gauge,
			Help: "Target resources the VerticalPodAutoscaler recommends for the container ignoring bounds.",
			GenerateFunc: wrapVPAFunc(func(a *autoscaling.VerticalPodAutoscaler) *metric.Family {
				ms := []*metric.Metric{}
				if a.Status.Recommendation == nil || a.Status.Recommendation.ContainerRecommendations == nil {
					return &metric.Family{
						Metrics: ms,
					}
				}
				for _, c := range a.Status.Recommendation.ContainerRecommendations {
					ms = append(ms, vpaResourcesToMetrics(c.ContainerName, c.UncappedTarget)...)
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
	}
)

func vpaResourcesToMetrics(containerName string, resources v1.ResourceList) []*metric.Metric {
	ms := []*metric.Metric{}
	for resourceName, val := range resources {
		switch resourceName {
		case v1.ResourceCPU:
			ms = append(ms, &metric.Metric{
				LabelValues: []string{containerName, sanitizeLabelName(string(resourceName)), string(constant.UnitCore)},
				Value:       float64(val.MilliValue()) / 1000,
			})
		case v1.ResourceStorage:
			fallthrough
		case v1.ResourceEphemeralStorage:
			fallthrough
		case v1.ResourceMemory:
			ms = append(ms, &metric.Metric{
				LabelValues: []string{containerName, sanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
				Value:       float64(val.Value()),
			})
		}
	}
	for _, metric := range ms {
		metric.LabelKeys = []string{"container", "resource", "unit"}
	}
	return ms
}

func wrapVPAFunc(f func(*autoscaling.VerticalPodAutoscaler) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		vpa := obj.(*autoscaling.VerticalPodAutoscaler)

		metricFamily := f(vpa)
		targetRef := vpa.Spec.TargetRef

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descVerticalPodAutoscalerLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{vpa.Namespace, vpa.Name, targetRef.APIVersion, targetRef.Kind, targetRef.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createVPAListWatchFunc(vpaClient vpaclientset.Interface) func(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return func(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
		return &cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				return vpaClient.AutoscalingV1beta2().VerticalPodAutoscalers(ns).List(opts)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				return vpaClient.AutoscalingV1beta2().VerticalPodAutoscalers(ns).Watch(opts)
			},
		}
	}
}
