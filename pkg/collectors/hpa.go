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

	descHorizontalPodAutoscalerMetadataGeneration = metrics.NewMetricFamilyDef(
		"kube_hpa_metadata_generation",
		"The generation observed by the HorizontalPodAutoscaler controller.",
		descHorizontalPodAutoscalerLabelsDefaultLabels,
		nil,
	)
	descHorizontalPodAutoscalerSpecMaxReplicas = metrics.NewMetricFamilyDef(
		"kube_hpa_spec_max_replicas",
		"Upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas.",
		descHorizontalPodAutoscalerLabelsDefaultLabels,
		nil,
	)
	descHorizontalPodAutoscalerSpecMinReplicas = metrics.NewMetricFamilyDef(
		"kube_hpa_spec_min_replicas",
		"Lower limit for the number of pods that can be set by the autoscaler, default 1.",
		descHorizontalPodAutoscalerLabelsDefaultLabels,
		nil,
	)
	descHorizontalPodAutoscalerStatusCurrentReplicas = metrics.NewMetricFamilyDef(
		"kube_hpa_status_current_replicas",
		"Current number of replicas of pods managed by this autoscaler.",
		descHorizontalPodAutoscalerLabelsDefaultLabels,
		nil,
	)
	descHorizontalPodAutoscalerStatusDesiredReplicas = metrics.NewMetricFamilyDef(
		"kube_hpa_status_desired_replicas",
		"Desired number of replicas of pods managed by this autoscaler.",
		descHorizontalPodAutoscalerLabelsDefaultLabels,
		nil,
	)
	descHorizontalPodAutoscalerLabels = metrics.NewMetricFamilyDef(
		descHorizontalPodAutoscalerLabelsName,
		descHorizontalPodAutoscalerLabelsHelp,
		descHorizontalPodAutoscalerLabelsDefaultLabels,
		nil,
	)
	descHorizontalPodAutoscalerCondition = metrics.NewMetricFamilyDef(
		"kube_hpa_status_condition",
		"The condition of this autoscaler.",
		append(descHorizontalPodAutoscalerLabelsDefaultLabels, "condition", "status"),
		nil,
	)
)

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

func hpaLabelsDesc(labelKeys []string) *metrics.MetricFamilyDef {
	return metrics.NewMetricFamilyDef(
		descHorizontalPodAutoscalerLabelsName,
		descHorizontalPodAutoscalerLabelsHelp,
		append(descHorizontalPodAutoscalerLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func generateHPAMetrics(obj interface{}) []*metrics.Metric {

	ms := []*metrics.Metric{}

	// TODO: Refactor
	hPointer := obj.(*autoscaling.HorizontalPodAutoscaler)
	h := *hPointer

	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{h.Namespace, h.Name}, lv...)

		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}
	labelKeys, labelValues := kubeLabelsToPrometheusLabels(h.Labels)
	addGauge(hpaLabelsDesc(labelKeys), 1, labelValues...)
	addGauge(descHorizontalPodAutoscalerMetadataGeneration, float64(h.ObjectMeta.Generation))
	addGauge(descHorizontalPodAutoscalerSpecMaxReplicas, float64(h.Spec.MaxReplicas))
	addGauge(descHorizontalPodAutoscalerSpecMinReplicas, float64(*h.Spec.MinReplicas))
	addGauge(descHorizontalPodAutoscalerStatusCurrentReplicas, float64(h.Status.CurrentReplicas))
	addGauge(descHorizontalPodAutoscalerStatusDesiredReplicas, float64(h.Status.DesiredReplicas))

	for _, c := range h.Status.Conditions {
		ms = append(ms, addConditionMetrics(descHorizontalPodAutoscalerCondition, c.Status, h.Namespace, h.Name, string(c.Type))...)
	}

	return ms
}
