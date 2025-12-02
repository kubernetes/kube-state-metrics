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

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descStatefulSetAnnotationsName     = "kube_statefulset_annotations"
	descStatefulSetAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descStatefulSetLabelsName          = "kube_statefulset_labels"
	descStatefulSetLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descStatefulSetLabelsDefaultLabels = []string{"namespace", "statefulset"}
)

func statefulSetMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_statefulset_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
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
		*generator.NewFamilyGeneratorWithStability(
			"kube_statefulset_status_replicas",
			"The number of replicas per StatefulSet.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(s.Status.Replicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_statefulset_status_replicas_available",
			"The number of available replicas per StatefulSet.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(s.Status.AvailableReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_statefulset_status_replicas_current",
			"The number of current replicas per StatefulSet.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(s.Status.CurrentReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_statefulset_status_replicas_ready",
			"The number of ready replicas per StatefulSet.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(s.Status.ReadyReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_statefulset_status_replicas_updated",
			"The number of updated replicas per StatefulSet.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(s.Status.UpdatedReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_statefulset_status_observed_generation",
			"The generation observed by the StatefulSet controller.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(s.Status.ObservedGeneration),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_statefulset_replicas",
			"Number of desired pods for a StatefulSet.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				ms := []*metric.Metric{}

				if s.Spec.Replicas != nil {
					ms = append(ms, &metric.Metric{
						Value: float64(*s.Spec.Replicas),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_statefulset_ordinals_start",
			"Start ordinal of the StatefulSet.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				ms := []*metric.Metric{}

				if s.Spec.Ordinals != nil {
					ms = append(ms, &metric.Metric{
						Value: float64(s.Spec.Ordinals.Start),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_statefulset_metadata_generation",
			"Sequence number representing a specific generation of the desired state for the StatefulSet.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(s.Generation),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_statefulset_persistentvolumeclaim_retention_policy",
			"Count of retention policy for StatefulSet template PVCs",
			metric.Gauge,
			basemetrics.STABLE,

			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				deletedPolicyLabel := ""
				scaledPolicyLabel := ""
				if policy := s.Spec.PersistentVolumeClaimRetentionPolicy; policy != nil {
					deletedPolicyLabel = string(policy.WhenDeleted)
					scaledPolicyLabel = string(policy.WhenScaled)
				}
				ms := []*metric.Metric{}
				if deletedPolicyLabel != "" || scaledPolicyLabel != "" {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"when_deleted", "when_scaled"},
						LabelValues: []string{deletedPolicyLabel, scaledPolicyLabel},
						Value:       1,
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descStatefulSetAnnotationsName,
			descStatefulSetAnnotationsHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
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
			descStatefulSetLabelsName,
			descStatefulSetLabelsHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
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
		*generator.NewFamilyGeneratorWithStability(
			"kube_statefulset_status_current_revision",
			"Indicates the version of the StatefulSet used to generate Pods in the sequence [0,currentReplicas).",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"revision"},
							LabelValues: []string{s.Status.CurrentRevision},
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_statefulset_status_update_revision",
			"Indicates the version of the StatefulSet used to generate Pods in the sequence [replicas-updatedReplicas,replicas)",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"revision"},
							LabelValues: []string{s.Status.UpdateRevision},
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_statefulset_deletion_timestamp",
			"Unix deletion timestamp",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapStatefulSetFunc(func(s *v1.StatefulSet) *metric.Family {
				ms := []*metric.Metric{}

				if !s.DeletionTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(s.DeletionTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
}

func wrapStatefulSetFunc(f func(*v1.StatefulSet) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		statefulSet := obj.(*v1.StatefulSet)

		metricFamily := f(statefulSet)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descStatefulSetLabelsDefaultLabels, []string{statefulSet.Namespace, statefulSet.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createStatefulSetListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.AppsV1().StatefulSets(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.AppsV1().StatefulSets(ns).Watch(context.TODO(), opts)
		},
	}
}
