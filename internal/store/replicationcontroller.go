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

	v1 "k8s.io/api/core/v1"
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
	descReplicationControllerLabelsDefaultLabels = []string{"namespace", "replicationcontroller"}

	replicationControllerMetricFamilies = []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_replicationcontroller_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
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
		*generator.NewFamilyGeneratorWithStability(
			"kube_replicationcontroller_status_replicas",
			"The number of replicas per ReplicationController.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Status.Replicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_replicationcontroller_status_fully_labeled_replicas",
			"The number of fully labeled replicas per ReplicationController.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Status.FullyLabeledReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_replicationcontroller_status_ready_replicas",
			"The number of ready replicas per ReplicationController.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Status.ReadyReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_replicationcontroller_status_available_replicas",
			"The number of available replicas per ReplicationController.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Status.AvailableReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_replicationcontroller_status_observed_generation",
			"The generation observed by the ReplicationController controller.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Status.ObservedGeneration),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_replicationcontroller_spec_replicas",
			"Number of desired pods for a ReplicationController.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
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
		*generator.NewFamilyGeneratorWithStability(
			"kube_replicationcontroller_metadata_generation",
			"Sequence number representing a specific generation of the desired state.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
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
			"kube_replicationcontroller_owner",
			"Information about the ReplicationController's owner.",
			metric.Gauge,
			"",
			wrapReplicationControllerFunc(func(r *v1.ReplicationController) *metric.Family {
				labelKeys := []string{"owner_kind", "owner_name", "owner_is_controller"}
				ms := []*metric.Metric{}

				owners := r.GetOwnerReferences()
				if len(owners) == 0 {
					ms = append(ms, &metric.Metric{
						LabelKeys:   labelKeys,
						LabelValues: []string{"", "", ""},
						Value:       1,
					})
				} else {
					for _, owner := range owners {
						ownerIsController := "false"
						if owner.Controller != nil {
							ownerIsController = strconv.FormatBool(*owner.Controller)
						}

						ms = append(ms, &metric.Metric{
							LabelKeys:   labelKeys,
							LabelValues: []string{owner.Kind, owner.Name, ownerIsController},
							Value:       1,
						})
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
)

func wrapReplicationControllerFunc(f func(*v1.ReplicationController) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		replicationController := obj.(*v1.ReplicationController)

		metricFamily := f(replicationController)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descReplicationControllerLabelsDefaultLabels, []string{replicationController.Namespace, replicationController.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createReplicationControllerListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().ReplicationControllers(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().ReplicationControllers(ns).Watch(context.TODO(), opts)
		},
	}
}
