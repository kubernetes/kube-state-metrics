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

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descPersistentVolumeClaimLabelsName          = "kube_persistentvolumeclaim_labels"
	descPersistentVolumeClaimLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descPersistentVolumeClaimLabelsDefaultLabels = []string{"namespace", "persistentvolumeclaim"}

	descPersistentVolumeClaimLabels = metrics.NewMetricFamilyDef(
		descPersistentVolumeClaimLabelsName,
		descPersistentVolumeClaimLabelsHelp,
		descPersistentVolumeClaimLabelsDefaultLabels,
		nil,
	)
	descPersistentVolumeClaimInfo = metrics.NewMetricFamilyDef(
		"kube_persistentvolumeclaim_info",
		"Information about persistent volume claim.",
		append(descPersistentVolumeClaimLabelsDefaultLabels, "storageclass", "volumename"),
		nil,
	)
	descPersistentVolumeClaimStatusPhase = metrics.NewMetricFamilyDef(
		"kube_persistentvolumeclaim_status_phase",
		"The phase the persistent volume claim is currently in.",
		append(descPersistentVolumeClaimLabelsDefaultLabels, "phase"),
		nil,
	)
	descPersistentVolumeClaimResourceRequestsStorage = metrics.NewMetricFamilyDef(
		"kube_persistentvolumeclaim_resource_requests_storage_bytes",
		"The capacity of storage requested by the persistent volume claim.",
		descPersistentVolumeClaimLabelsDefaultLabels,
		nil,
	)
)

func createPersistentVolumeClaimListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().PersistentVolumeClaims(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().PersistentVolumeClaims(ns).Watch(opts)
		},
	}
}
func persistentVolumeClaimLabelsDesc(labelKeys []string) *metrics.MetricFamilyDef {
	return metrics.NewMetricFamilyDef(
		descPersistentVolumeClaimLabelsName,
		descPersistentVolumeClaimLabelsHelp,
		append(descPersistentVolumeClaimLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

// getPersistentVolumeClaimClass returns StorageClassName. If no storage class was
// requested, it returns "".
func getPersistentVolumeClaimClass(claim *v1.PersistentVolumeClaim) string {
	// Use beta annotation first
	if class, found := claim.Annotations[v1.BetaStorageClassAnnotation]; found {
		return class
	}

	if claim.Spec.StorageClassName != nil {
		return *claim.Spec.StorageClassName
	}

	// Special non-empty string to indicate absence of storage class.
	return "<none>"
}

func generatePersistentVolumeClaimMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	pPointer := obj.(*v1.PersistentVolumeClaim)
	p := *pPointer

	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{p.Namespace, p.Name}, lv...)

		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}

	labelKeys, labelValues := kubeLabelsToPrometheusLabels(p.Labels)
	addGauge(persistentVolumeClaimLabelsDesc(labelKeys), 1, labelValues...)

	storageClassName := getPersistentVolumeClaimClass(&p)
	volumeName := p.Spec.VolumeName
	addGauge(descPersistentVolumeClaimInfo, 1, storageClassName, volumeName)

	// Set current phase to 1, others to 0 if it is set.
	if p := p.Status.Phase; p != "" {
		addGauge(descPersistentVolumeClaimStatusPhase, boolFloat64(p == v1.ClaimLost), string(v1.ClaimLost))
		addGauge(descPersistentVolumeClaimStatusPhase, boolFloat64(p == v1.ClaimBound), string(v1.ClaimBound))
		addGauge(descPersistentVolumeClaimStatusPhase, boolFloat64(p == v1.ClaimPending), string(v1.ClaimPending))
	}

	if storage, ok := p.Spec.Resources.Requests[v1.ResourceStorage]; ok {
		addGauge(descPersistentVolumeClaimResourceRequestsStorage, float64(storage.Value()))
	}

	return ms
}
