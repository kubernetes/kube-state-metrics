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

package collector

import (
	"k8s.io/kube-state-metrics/pkg/metric"

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

	persistentVolumeClaimMetricFamilies = []metric.FamilyGenerator{
		{
			Name: descPersistentVolumeClaimLabelsName,
			Type: metric.MetricTypeGauge,
			Help: descPersistentVolumeClaimLabelsHelp,
			GenerateFunc: wrapPersistentVolumeClaimFunc(func(p *v1.PersistentVolumeClaim) metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(p.Labels)
				return metric.Family{
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
			Name: "kube_persistentvolumeclaim_info",
			Type: metric.MetricTypeGauge,
			Help: "Information about persistent volume claim.",
			GenerateFunc: wrapPersistentVolumeClaimFunc(func(p *v1.PersistentVolumeClaim) metric.Family {
				storageClassName := getPersistentVolumeClaimClass(p)
				volumeName := p.Spec.VolumeName
				return metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"storageclass", "volumename"},
							LabelValues: []string{storageClassName, volumeName},
							Value:       1,
						},
					},
				}
			}),
		},
		{
			Name: "kube_persistentvolumeclaim_status_phase",
			Type: metric.MetricTypeGauge,
			Help: "The phase the persistent volume claim is currently in.",
			GenerateFunc: wrapPersistentVolumeClaimFunc(func(p *v1.PersistentVolumeClaim) metric.Family {
				ms := []*metric.Metric{}

				// Set current phase to 1, others to 0 if it is set.
				if p := p.Status.Phase; p != "" {
					ms = append(ms,
						&metric.Metric{
							LabelValues: []string{string(v1.ClaimLost)},
							Value:       boolFloat64(p == v1.ClaimLost),
						},
						&metric.Metric{
							LabelValues: []string{string(v1.ClaimBound)},
							Value:       boolFloat64(p == v1.ClaimBound),
						},
						&metric.Metric{
							LabelValues: []string{string(v1.ClaimPending)},
							Value:       boolFloat64(p == v1.ClaimPending),
						},
					)
				}

				for _, m := range ms {
					m.LabelKeys = []string{"phase"}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_persistentvolumeclaim_resource_requests_storage_bytes",
			Type: metric.MetricTypeGauge,
			Help: "The capacity of storage requested by the persistent volume claim.",
			GenerateFunc: wrapPersistentVolumeClaimFunc(func(p *v1.PersistentVolumeClaim) metric.Family {
				ms := []*metric.Metric{}

				if storage, ok := p.Spec.Resources.Requests[v1.ResourceStorage]; ok {
					ms = append(ms, &metric.Metric{
						Value: float64(storage.Value()),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
	}
)

func wrapPersistentVolumeClaimFunc(f func(*v1.PersistentVolumeClaim) metric.Family) func(interface{}) metric.Family {
	return func(obj interface{}) metric.Family {
		persistentVolumeClaim := obj.(*v1.PersistentVolumeClaim)

		metricFamily := f(persistentVolumeClaim)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descPersistentVolumeClaimLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{persistentVolumeClaim.Namespace, persistentVolumeClaim.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

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
