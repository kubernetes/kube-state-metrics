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
	"context"

	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descStorageClassAnnotationsName     = "kube_storageclass_annotations"
	descStorageClassAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descStorageClassLabelsName          = "kube_storageclass_labels"
	descStorageClassLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descStorageClassLabelsDefaultLabels = []string{"storageclass"}
	defaultReclaimPolicy                = v1.PersistentVolumeReclaimDelete
	defaultVolumeBindingMode            = storagev1.VolumeBindingImmediate
)

func storageClassMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_storageclass_info",
			"Information about storageclass.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStorageClassFunc(func(s *storagev1.StorageClass) *metric.Family {

				// Add default values if missing.
				if s.ReclaimPolicy == nil {
					s.ReclaimPolicy = &defaultReclaimPolicy
				}

				if s.VolumeBindingMode == nil {
					s.VolumeBindingMode = &defaultVolumeBindingMode
				}

				m := metric.Metric{
					LabelKeys:   []string{"provisioner", "reclaim_policy", "volume_binding_mode"},
					LabelValues: []string{s.Provisioner, string(*s.ReclaimPolicy), string(*s.VolumeBindingMode)},
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_storageclass_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStorageClassFunc(func(s *storagev1.StorageClass) *metric.Family {
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
		),
		*generator.NewFamilyGenerator(
			descStorageClassAnnotationsName,
			descStorageClassAnnotationsHelp,
			metric.Gauge,
			"",
			wrapStorageClassFunc(func(s *storagev1.StorageClass) *metric.Family {
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", s.Annotations, allowAnnotationsList)
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
		*generator.NewFamilyGeneratorWithStability(
			descStorageClassLabelsName,
			descStorageClassLabelsHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStorageClassFunc(func(s *storagev1.StorageClass) *metric.Family {
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", s.Labels, allowLabelsList)
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
	}
}

func wrapStorageClassFunc(f func(*storagev1.StorageClass) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		storageClass := obj.(*storagev1.StorageClass)

		metricFamily := f(storageClass)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descStorageClassLabelsDefaultLabels, []string{storageClass.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createStorageClassListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.StorageV1().StorageClasses().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.StorageV1().StorageClasses().Watch(context.TODO(), opts)
		},
	}
}
