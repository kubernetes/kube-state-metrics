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

package store

import (
	"context"

	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descPersistentVolumeClaimAnnotationsName     = "kube_persistentvolumeclaim_annotations"
	descPersistentVolumeClaimAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descPersistentVolumeClaimLabelsName          = "kube_persistentvolumeclaim_labels"
	descPersistentVolumeClaimLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descPersistentVolumeClaimLabelsDefaultLabels = []string{"namespace", "persistentvolumeclaim"}
)

func persistentVolumeClaimMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			descPersistentVolumeClaimLabelsName,
			descPersistentVolumeClaimLabelsHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapPersistentVolumeClaimFunc(func(p *v1.PersistentVolumeClaim) *metric.Family {
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", p.Labels, allowLabelsList)
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
		*generator.NewFamilyGenerator(
			descPersistentVolumeClaimAnnotationsName,
			descPersistentVolumeClaimAnnotationsHelp,
			metric.Gauge,
			"",
			wrapPersistentVolumeClaimFunc(func(p *v1.PersistentVolumeClaim) *metric.Family {
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", p.Annotations, allowAnnotationsList)
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
			"kube_persistentvolumeclaim_info",
			"Information about persistent volume claim.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapPersistentVolumeClaimFunc(func(p *v1.PersistentVolumeClaim) *metric.Family {
				storageClassName := getPersistentVolumeClaimClass(p)
				volumeName := p.Spec.VolumeName
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"storageclass", "volumename"},
							LabelValues: []string{storageClassName, volumeName},
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_persistentvolumeclaim_status_phase",
			"The phase the persistent volume claim is currently in.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapPersistentVolumeClaimFunc(func(p *v1.PersistentVolumeClaim) *metric.Family {
				phase := p.Status.Phase

				if phase == "" {
					return &metric.Family{
						Metrics: []*metric.Metric{},
					}
				}

				// Set current phase to 1, others to 0 if it is set.
				ms := []*metric.Metric{
					{
						LabelValues: []string{string(v1.ClaimLost)},
						Value:       boolFloat64(phase == v1.ClaimLost),
					},
					{
						LabelValues: []string{string(v1.ClaimBound)},
						Value:       boolFloat64(phase == v1.ClaimBound),
					},
					{
						LabelValues: []string{string(v1.ClaimPending)},
						Value:       boolFloat64(phase == v1.ClaimPending),
					},
				}

				for _, m := range ms {
					m.LabelKeys = []string{"phase"}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_persistentvolumeclaim_resource_requests_storage_bytes",
			"The capacity of storage requested by the persistent volume claim.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapPersistentVolumeClaimFunc(func(p *v1.PersistentVolumeClaim) *metric.Family {
				ms := []*metric.Metric{}

				if storage, ok := p.Spec.Resources.Requests[v1.ResourceStorage]; ok {
					ms = append(ms, &metric.Metric{
						Value: float64(storage.Value()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_persistentvolumeclaim_access_mode",
			"The access mode(s) specified by the persistent volume claim.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapPersistentVolumeClaimFunc(func(p *v1.PersistentVolumeClaim) *metric.Family {
				ms := make([]*metric.Metric, len(p.Spec.AccessModes))

				for i, mode := range p.Spec.AccessModes {
					ms[i] = &metric.Metric{
						LabelKeys:   []string{"access_mode"},
						LabelValues: []string{string(mode)},
						Value:       1,
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_persistentvolumeclaim_status_condition",
			"Information about status of different conditions of persistent volume claim.",
			metric.Gauge,
			"",
			wrapPersistentVolumeClaimFunc(func(p *v1.PersistentVolumeClaim) *metric.Family {
				ms := make([]*metric.Metric, len(p.Status.Conditions)*len(conditionStatuses))

				for i, c := range p.Status.Conditions {
					conditionMetrics := addConditionMetrics(c.Status)

					for j, m := range conditionMetrics {
						metric := m

						metric.LabelKeys = []string{"condition", "status"}
						metric.LabelValues = append([]string{string(c.Type)}, metric.LabelValues...)

						ms[i*len(conditionStatuses)+j] = metric
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_persistentvolumeclaim_created",
			"Unix creation timestamp",
			metric.Gauge,
			"",
			wrapPersistentVolumeClaimFunc(func(p *v1.PersistentVolumeClaim) *metric.Family {
				ms := []*metric.Metric{}

				if !p.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(p.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
}

func wrapPersistentVolumeClaimFunc(f func(*v1.PersistentVolumeClaim) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		persistentVolumeClaim := obj.(*v1.PersistentVolumeClaim)

		metricFamily := f(persistentVolumeClaim)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descPersistentVolumeClaimLabelsDefaultLabels, []string{persistentVolumeClaim.Namespace, persistentVolumeClaim.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createPersistentVolumeClaimListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().PersistentVolumeClaims(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().PersistentVolumeClaims(ns).Watch(context.TODO(), opts)
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
