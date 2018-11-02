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
	descPersistentVolumeLabelsName          = "kube_persistentvolume_labels"
	descPersistentVolumeLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descPersistentVolumeLabelsDefaultLabels = []string{"persistentvolume"}

	descPersistentVolumeLabels = metrics.NewMetricFamilyDef(
		descPersistentVolumeLabelsName,
		descPersistentVolumeLabelsHelp,
		descPersistentVolumeLabelsDefaultLabels,
		nil,
	)
	descPersistentVolumeStatusPhase = metrics.NewMetricFamilyDef(
		"kube_persistentvolume_status_phase",
		"The phase indicates if a volume is available, bound to a claim, or released by a claim.",
		append(descPersistentVolumeLabelsDefaultLabels, "phase"),
		nil,
	)
	descPersistentVolumeInfo = metrics.NewMetricFamilyDef(
		"kube_persistentvolume_info",
		"Information about persistentvolume.",
		append(descPersistentVolumeLabelsDefaultLabels, "storageclass"),
		nil,
	)
)

func createPersistentVolumeListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().PersistentVolumes().List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().PersistentVolumes().Watch(opts)
		},
	}
}

func persistentVolumeLabelsDesc(labelKeys []string) *metrics.MetricFamilyDef {
	return metrics.NewMetricFamilyDef(
		descPersistentVolumeLabelsName,
		descPersistentVolumeLabelsHelp,
		append(descPersistentVolumeLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func generatePersistentVolumeMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	pPointer := obj.(*v1.PersistentVolume)
	p := *pPointer

	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{p.Name}, lv...)

		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}

	labelKeys, labelValues := kubeLabelsToPrometheusLabels(p.Labels)
	addGauge(persistentVolumeLabelsDesc(labelKeys), 1, labelValues...)

	addGauge(descPersistentVolumeInfo, 1, p.Spec.StorageClassName)
	// Set current phase to 1, others to 0 if it is set.
	if p := p.Status.Phase; p != "" {
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == v1.VolumePending), string(v1.VolumePending))
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == v1.VolumeAvailable), string(v1.VolumeAvailable))
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == v1.VolumeBound), string(v1.VolumeBound))
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == v1.VolumeReleased), string(v1.VolumeReleased))
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == v1.VolumeFailed), string(v1.VolumeFailed))
	}

	return ms
}
