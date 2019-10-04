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
	"k8s.io/kube-state-metrics/pkg/metric"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descStorageClassLabelsName          = "kube_storageclass_labels"
	descStorageClassLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descStorageClassLabelsDefaultLabels = []string{"storageclass"}
	defaultReclaimPolicy                = v1.PersistentVolumeReclaimDelete
	defaultVolumeBindingMode            = storagev1.VolumeBindingImmediate

	storageClassMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_storageclass_info",
			Type: metric.Gauge,
			Help: "Information about storageclass.",
			GenerateFunc: wrapStorageClassFunc(func(s *storagev1.StorageClass) *metric.Family {

				// Add default values if missing.
				if s.ReclaimPolicy == nil {
					s.ReclaimPolicy = &defaultReclaimPolicy
				}

				if s.VolumeBindingMode == nil {
					s.VolumeBindingMode = &defaultVolumeBindingMode
				}

				m := metric.Metric{
					LabelKeys:   []string{"provisioner", "reclaimPolicy", "volumeBindingMode"},
					LabelValues: []string{s.Provisioner, string(*s.ReclaimPolicy), string(*s.VolumeBindingMode)},
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		},
		{
			Name: "kube_storageclass_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapStorageClassFunc(func(s *storagev1.StorageClass) *metric.Family {
				ms := []*metric.Metric{}
				if !s.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(s.CreationTimestamp.Unix()),
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: descStorageClassLabelsName,
			Type: metric.Gauge,
			Help: descStorageClassLabelsHelp,
			GenerateFunc: wrapStorageClassFunc(func(s *storagev1.StorageClass) *metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(s.Labels)
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
	}
)

func wrapStorageClassFunc(f func(*storagev1.StorageClass) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		storageClass := obj.(*storagev1.StorageClass)

		metricFamily := f(storageClass)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descStorageClassLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{storageClass.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createStorageClassListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.StorageV1().StorageClasses().List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.StorageV1().StorageClasses().Watch(opts)
		},
	}
}
