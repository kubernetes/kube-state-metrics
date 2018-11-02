/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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

	"k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descStatefulSetLabelsName          = "kube_statefulset_labels"
	descStatefulSetLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descStatefulSetLabelsDefaultLabels = []string{"namespace", "statefulset"}

	descStatefulSetCreated = metrics.NewMetricFamilyDef(
		"kube_statefulset_created",
		"Unix creation timestamp",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetStatusReplicas = metrics.NewMetricFamilyDef(
		"kube_statefulset_status_replicas",
		"The number of replicas per StatefulSet.",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetStatusReplicasCurrent = metrics.NewMetricFamilyDef(
		"kube_statefulset_status_replicas_current",
		"The number of current replicas per StatefulSet.",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetStatusReplicasReady = metrics.NewMetricFamilyDef(
		"kube_statefulset_status_replicas_ready",
		"The number of ready replicas per StatefulSet.",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetStatusReplicasUpdated = metrics.NewMetricFamilyDef(
		"kube_statefulset_status_replicas_updated",
		"The number of updated replicas per StatefulSet.",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetStatusObservedGeneration = metrics.NewMetricFamilyDef(
		"kube_statefulset_status_observed_generation",
		"The generation observed by the StatefulSet controller.",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetSpecReplicas = metrics.NewMetricFamilyDef(
		"kube_statefulset_replicas",
		"Number of desired pods for a StatefulSet.",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetMetadataGeneration = metrics.NewMetricFamilyDef(
		"kube_statefulset_metadata_generation",
		"Sequence number representing a specific generation of the desired state for the StatefulSet.",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetLabels = metrics.NewMetricFamilyDef(
		descStatefulSetLabelsName,
		descStatefulSetLabelsHelp,
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetCurrentRevision = metrics.NewMetricFamilyDef(
		"kube_statefulset_status_current_revision",
		"Indicates the version of the StatefulSet used to generate Pods in the sequence [0,currentReplicas).",
		append(descStatefulSetLabelsDefaultLabels, "revision"),
		nil,
	)
	descStatefulSetUpdateRevision = metrics.NewMetricFamilyDef(
		"kube_statefulset_status_update_revision",
		"Indicates the version of the StatefulSet used to generate Pods in the sequence [replicas-updatedReplicas,replicas)",
		append(descStatefulSetLabelsDefaultLabels, "revision"),
		nil,
	)
)

func createStatefulSetListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AppsV1beta1().StatefulSets(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AppsV1beta1().StatefulSets(ns).Watch(opts)
		},
	}
}

func statefulSetLabelsDesc(labelKeys []string) *metrics.MetricFamilyDef {
	return metrics.NewMetricFamilyDef(
		descStatefulSetLabelsName,
		descStatefulSetLabelsHelp,
		append(descStatefulSetLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func generateStatefulSetMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	sPointer := obj.(*v1beta1.StatefulSet)
	s := *sPointer

	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{s.Namespace, s.Name}, lv...)
		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}
	if !s.CreationTimestamp.IsZero() {
		addGauge(descStatefulSetCreated, float64(s.CreationTimestamp.Unix()))
	}
	addGauge(descStatefulSetStatusReplicas, float64(s.Status.Replicas))
	addGauge(descStatefulSetStatusReplicasCurrent, float64(s.Status.CurrentReplicas))
	addGauge(descStatefulSetStatusReplicasReady, float64(s.Status.ReadyReplicas))
	addGauge(descStatefulSetStatusReplicasUpdated, float64(s.Status.UpdatedReplicas))
	if s.Status.ObservedGeneration != nil {
		addGauge(descStatefulSetStatusObservedGeneration, float64(*s.Status.ObservedGeneration))
	}

	if s.Spec.Replicas != nil {
		addGauge(descStatefulSetSpecReplicas, float64(*s.Spec.Replicas))
	}
	addGauge(descStatefulSetMetadataGeneration, float64(s.ObjectMeta.Generation))

	labelKeys, labelValues := kubeLabelsToPrometheusLabels(s.Labels)
	addGauge(statefulSetLabelsDesc(labelKeys), 1, labelValues...)

	addGauge(descStatefulSetCurrentRevision, 1, s.Status.CurrentRevision)
	addGauge(descStatefulSetUpdateRevision, 1, s.Status.UpdateRevision)
	return ms
}
