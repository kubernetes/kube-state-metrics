/*
Copyright 2025 The Kubernetes Authors All rights reserved.

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
	basemetrics "k8s.io/component-base/metrics"

	metaapi "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kube-state-metrics/v2/pkg/metric"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descAnnotationsHelp = "Kubernetes annotations converted to Prometheus labels."
	descLabelsHelp      = "Kubernetes labels converted to Prometheus labels."
	descCreationHelp    = "Unix creation timestamp"
	descDeletionHelp    = "Unix deletion timestamp"
)

// createMetadataMetricFamiliesGenerator provides metadata metrics for all resources
func createMetadataMetricFamiliesGenerator(allowAnnotationsList, allowLabelsList, descLabelsDefaultLabels []string, descPrefixName string, w func(func(*metav1.ObjectMeta) *metric.Family, []string) func(any) *metric.Family) []generator.FamilyGenerator {
	wrapFunc := w

	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			descPrefixName+"_created",
			descCreationHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapFunc(func(t *metav1.ObjectMeta) *metric.Family {
				ms := []*metric.Metric{}

				if !t.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(t.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}, descLabelsDefaultLabels),
		),
		*generator.NewFamilyGeneratorWithStability(
			descPrefixName+"_deletion_timestamp",
			descDeletionHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapFunc(func(t *metav1.ObjectMeta) *metric.Family {
				ms := []*metric.Metric{}

				if t.DeletionTimestamp != nil && !t.DeletionTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(t.DeletionTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}, descLabelsDefaultLabels),
		),

		*generator.NewFamilyGeneratorWithStability(
			descPrefixName+"_annotations",
			descAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapFunc(func(t *metav1.ObjectMeta) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", t.Annotations, allowAnnotationsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   annotationKeys,
							LabelValues: annotationValues,
							Value:       1,
						},
					}}
			}, descLabelsDefaultLabels),
		),
		*generator.NewFamilyGeneratorWithStability(
			descPrefixName+"_labels",
			descLabelsHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapFunc(func(t *metav1.ObjectMeta) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", t.Labels, allowLabelsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
							Value:       1,
						},
					}}

			}, descLabelsDefaultLabels),
		),
	}
}

func wrapMetadataFunc(f func(*metav1.ObjectMeta) *metric.Family, defaultLabels []string) func(any) *metric.Family {
	return func(obj any) *metric.Family {
		o := obj.(metav1.Object)
		objectMeta := metaapi.AsPartialObjectMetadata(o).ObjectMeta
		metricFamily := f(&objectMeta)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(defaultLabels, []string{o.GetNamespace(), o.GetName()}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func wrapMetadataWithUIDFunc(f func(*metav1.ObjectMeta) *metric.Family, defaultLabels []string) func(any) *metric.Family {
	return func(obj any) *metric.Family {
		o := obj.(metav1.Object)
		objectMeta := metaapi.AsPartialObjectMetadata(o).ObjectMeta
		metricFamily := f(&objectMeta)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(defaultLabels, []string{o.GetNamespace(), o.GetName(), string(o.GetUID())}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}
