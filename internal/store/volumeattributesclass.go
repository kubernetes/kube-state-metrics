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

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descVolumeAttributesClassAnnotationsName     = "kube_volumeattributesclass_annotations"
	descVolumeAttributesClassAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descVolumeAttributesClassLabelsName          = "kube_volumeattributesclass_labels"
	descVolumeAttributesClassLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descVolumeAttributesClassParametersName      = "kube_volumeattributesclass_parameters"
	descVolumeAttributesClassParametersHelp      = "VolumeAttributesClass parameters converted to Prometheus labels."
	descVolumeAttributesClassLabelsDefaultLabels = []string{"volumeattributesclass"}
)

func volumeAttributesClassMetricFamilies(allowAnnotationsList, allowLabelsList, allowParametersList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_volumeattributesclass_info",
			"Information about volumeattributesclass.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapVolumeAttributesClassFunc(func(v *storagev1.VolumeAttributesClass) *metric.Family {
				m := metric.Metric{
					LabelKeys:   []string{"driver_name"},
					LabelValues: []string{v.DriverName},
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_volumeattributesclass_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapVolumeAttributesClassFunc(func(v *storagev1.VolumeAttributesClass) *metric.Family {
				ms := []*metric.Metric{}
				if !v.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(v.CreationTimestamp.Unix()),
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descVolumeAttributesClassAnnotationsName,
			descVolumeAttributesClassAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapVolumeAttributesClassFunc(func(v *storagev1.VolumeAttributesClass) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", v.Annotations, allowAnnotationsList)
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
			descVolumeAttributesClassLabelsName,
			descVolumeAttributesClassLabelsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapVolumeAttributesClassFunc(func(v *storagev1.VolumeAttributesClass) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", v.Labels, allowLabelsList)
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
		*generator.NewFamilyGeneratorWithStability(
			descVolumeAttributesClassParametersName,
			descVolumeAttributesClassParametersHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapVolumeAttributesClassFunc(func(v *storagev1.VolumeAttributesClass) *metric.Family {
				if len(allowParametersList) == 0 {
					return &metric.Family{}
				}
				parameterKeys, parameterValues := createPrometheusLabelKeysValues("parameter", v.Parameters, allowParametersList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   parameterKeys,
							LabelValues: parameterValues,
							Value:       1,
						},
					},
				}
			}),
		),
	}
}

func wrapVolumeAttributesClassFunc(f func(*storagev1.VolumeAttributesClass) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		volumeAttributesClass := obj.(*storagev1.VolumeAttributesClass)

		metricFamily := f(volumeAttributesClass)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descVolumeAttributesClassLabelsDefaultLabels, []string{volumeAttributesClass.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createVolumeAttributesClassListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.StorageV1().VolumeAttributesClasses().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.StorageV1().VolumeAttributesClasses().Watch(context.TODO(), opts)
		},
	}
}
