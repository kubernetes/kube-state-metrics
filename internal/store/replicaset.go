/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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
	"strconv"

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
	descReplicaSetLabelsDefaultLabels = []string{"namespace", "replicaset"}
	descReplicaSetAnnotationsName     = "kube_replicaset_annotations"
	descReplicaSetAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descReplicaSetLabelsName          = "kube_replicaset_labels"
	descReplicaSetLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
)

func replicaSetMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			"kube_replicaset_created",
			"Unix creation timestamp",
			metric.Gauge,
			"",
			wrapReplicaSetFunc(func(r *v1.ReplicaSet) *metric.Family {
				ms := []*metric.Metric{}

				if !r.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{

						Value: float64(r.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_replicaset_status_replicas",
			"The number of replicas per ReplicaSet.",
			metric.Gauge,
			"",
			wrapReplicaSetFunc(func(r *v1.ReplicaSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Status.Replicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_replicaset_status_fully_labeled_replicas",
			"The number of fully labeled replicas per ReplicaSet.",
			metric.Gauge,
			"",
			wrapReplicaSetFunc(func(r *v1.ReplicaSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Status.FullyLabeledReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_replicaset_status_ready_replicas",
			"The number of ready replicas per ReplicaSet.",
			metric.Gauge,
			"",
			wrapReplicaSetFunc(func(r *v1.ReplicaSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Status.ReadyReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_replicaset_status_observed_generation",
			"The generation observed by the ReplicaSet controller.",
			metric.Gauge,
			"",
			wrapReplicaSetFunc(func(r *v1.ReplicaSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Status.ObservedGeneration),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_replicaset_spec_replicas",
			"Number of desired pods for a ReplicaSet.",
			metric.Gauge,
			"",
			wrapReplicaSetFunc(func(r *v1.ReplicaSet) *metric.Family {
				ms := []*metric.Metric{}

				if r.Spec.Replicas != nil {
					ms = append(ms, &metric.Metric{
						Value: float64(*r.Spec.Replicas),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_replicaset_metadata_generation",
			"Sequence number representing a specific generation of the desired state.",
			metric.Gauge,
			"",
			wrapReplicaSetFunc(func(r *v1.ReplicaSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.ObjectMeta.Generation),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_replicaset_owner",
			"Information about the ReplicaSet's owner.",
			metric.Gauge,
			"",
			wrapReplicaSetFunc(func(r *v1.ReplicaSet) *metric.Family {
				owners := r.GetOwnerReferences()

				if len(owners) == 0 {
					return &metric.Family{
						Metrics: []*metric.Metric{
							{
								LabelKeys:   []string{"owner_kind", "owner_name", "owner_is_controller"},
								LabelValues: []string{"<none>", "<none>", "<none>"},
								Value:       1,
							},
						},
					}
				}

				ms := make([]*metric.Metric, len(owners))

				for i, owner := range owners {
					if owner.Controller != nil {
						ms[i] = &metric.Metric{
							LabelValues: []string{owner.Kind, owner.Name, strconv.FormatBool(*owner.Controller)},
						}
					} else {
						ms[i] = &metric.Metric{
							LabelValues: []string{owner.Kind, owner.Name, "false"},
						}
					}
				}

				for _, m := range ms {
					m.LabelKeys = []string{"owner_kind", "owner_name", "owner_is_controller"}
					m.Value = 1
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			descReplicaSetAnnotationsName,
			descReplicaSetAnnotationsHelp,
			metric.Gauge,
			"",
			wrapReplicaSetFunc(func(r *v1.ReplicaSet) *metric.Family {
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", r.Annotations, allowAnnotationsList)
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
		*generator.NewFamilyGenerator(
			descReplicaSetLabelsName,
			descReplicaSetLabelsHelp,
			metric.Gauge,
			"",
			wrapReplicaSetFunc(func(r *v1.ReplicaSet) *metric.Family {
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", r.Labels, allowLabelsList)
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

func wrapReplicaSetFunc(f func(*v1.ReplicaSet) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		replicaSet := obj.(*v1.ReplicaSet)

		metricFamily := f(replicaSet)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descReplicaSetLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{replicaSet.Namespace, replicaSet.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createReplicaSetListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AppsV1().ReplicaSets(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AppsV1().ReplicaSets(ns).Watch(context.TODO(), opts)
		},
	}
}
