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

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descVolumeAttachmentLabelsName          = "kube_volumeattachment_labels"
	descVolumeAttachmentLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descVolumeAttachmentLabelsDefaultLabels = []string{"volumeattachment"}

	volumeAttachmentMetricFamilies = []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			descVolumeAttachmentLabelsName,
			descVolumeAttachmentLabelsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapVolumeAttachmentFunc(func(va *storagev1.VolumeAttachment) *metric.Family {
				labelKeys, labelValues := kubeMapToPrometheusLabels("label", va.Labels)
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
			"kube_volumeattachment_info",
			"Information about volumeattachment.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapVolumeAttachmentFunc(func(va *storagev1.VolumeAttachment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"attacher", "node"},
							LabelValues: []string{va.Spec.Attacher, va.Spec.NodeName},
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_volumeattachment_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapVolumeAttachmentFunc(func(va *storagev1.VolumeAttachment) *metric.Family {
				if !va.CreationTimestamp.IsZero() {
					m := metric.Metric{
						LabelKeys:   nil,
						LabelValues: nil,
						Value:       float64(va.CreationTimestamp.Unix()),
					}
					return &metric.Family{Metrics: []*metric.Metric{&m}}
				}
				return &metric.Family{Metrics: []*metric.Metric{}}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_volumeattachment_spec_source_persistentvolume",
			"PersistentVolume source reference.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapVolumeAttachmentFunc(func(va *storagev1.VolumeAttachment) *metric.Family {
				if va.Spec.Source.PersistentVolumeName != nil {
					return &metric.Family{
						Metrics: []*metric.Metric{
							{
								LabelKeys:   []string{"volumename"},
								LabelValues: []string{*va.Spec.Source.PersistentVolumeName},
								Value:       1,
							},
						},
					}
				}
				return &metric.Family{}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_volumeattachment_status_attached",
			"Information about volumeattachment.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapVolumeAttachmentFunc(func(va *storagev1.VolumeAttachment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   nil,
							LabelValues: nil,
							Value:       boolFloat64(va.Status.Attached),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_volumeattachment_status_attachment_metadata",
			"volumeattachment metadata.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapVolumeAttachmentFunc(func(va *storagev1.VolumeAttachment) *metric.Family {
				labelKeys, labelValues := mapToPrometheusLabels(va.Status.AttachmentMetadata, "metadata")
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
)

func wrapVolumeAttachmentFunc(f func(*storagev1.VolumeAttachment) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		va := obj.(*storagev1.VolumeAttachment)

		metricFamily := f(va)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descVolumeAttachmentLabelsDefaultLabels, []string{va.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createVolumeAttachmentListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcherWithContext {
	return &cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.StorageV1().VolumeAttachments().List(ctx, opts)
		},
		WatchFuncWithContext: func(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.StorageV1().VolumeAttachments().Watch(ctx, opts)
		},
	}
}
