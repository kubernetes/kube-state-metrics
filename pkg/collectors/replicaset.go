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

package collectors

import (
	"strconv"

	"k8s.io/kube-state-metrics/pkg/metrics"

	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descReplicaSetLabelsDefaultLabels = []string{"namespace", "replicaset"}

	replicaSetMetricFamilies = []metrics.FamilyGenerator{
		metrics.FamilyGenerator{
			Name: "kube_replicaset_created",
			Type: metrics.MetricTypeGauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) metrics.Family {
				f := metrics.Family{}

				if !r.CreationTimestamp.IsZero() {
					f = append(f, &metrics.Metric{
						Name:  "kube_replicaset_created",
						Value: float64(r.CreationTimestamp.Unix()),
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_replicaset_status_replicas",
			Type: metrics.MetricTypeGauge,
			Help: "The number of replicas per ReplicaSet.",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_replicaset_status_replicas",
					Value: float64(r.Status.Replicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_replicaset_status_fully_labeled_replicas",
			Type: metrics.MetricTypeGauge,
			Help: "The number of fully labeled replicas per ReplicaSet.",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_replicaset_status_fully_labeled_replicas",
					Value: float64(r.Status.FullyLabeledReplicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_replicaset_status_ready_replicas",
			Type: metrics.MetricTypeGauge,
			Help: "The number of ready replicas per ReplicaSet.",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_replicaset_status_ready_replicas",
					Value: float64(r.Status.ReadyReplicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_replicaset_status_observed_generation",
			Type: metrics.MetricTypeGauge,
			Help: "The generation observed by the ReplicaSet controller.",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_replicaset_status_observed_generation",
					Value: float64(r.Status.ObservedGeneration),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_replicaset_spec_replicas",
			Type: metrics.MetricTypeGauge,
			Help: "Number of desired pods for a ReplicaSet.",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) metrics.Family {
				f := metrics.Family{}

				if r.Spec.Replicas != nil {
					f = append(f, &metrics.Metric{
						Name:  "kube_replicaset_spec_replicas",
						Value: float64(*r.Spec.Replicas),
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_replicaset_metadata_generation",
			Type: metrics.MetricTypeGauge,
			Help: "Sequence number representing a specific generation of the desired state.",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_replicaset_metadata_generation",
					Value: float64(r.ObjectMeta.Generation),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_replicaset_owner",
			Type: metrics.MetricTypeGauge,
			Help: "Information about the ReplicaSet's owner.",
			GenerateFunc: wrapReplicaSetFunc(func(r *v1beta1.ReplicaSet) metrics.Family {
				f := metrics.Family{}

				owners := r.GetOwnerReferences()
				if len(owners) == 0 {
					f = append(f, &metrics.Metric{
						LabelValues: []string{"<none>", "<none>", "<none>"},
					})
				} else {
					for _, owner := range owners {
						if owner.Controller != nil {
							f = append(f, &metrics.Metric{
								LabelValues: []string{owner.Kind, owner.Name, strconv.FormatBool(*owner.Controller)},
							})
						} else {
							f = append(f, &metrics.Metric{
								LabelValues: []string{owner.Kind, owner.Name, "false"},
							})
						}
					}
				}

				for _, m := range f {
					m.Name = "kube_replicaset_owner"
					m.LabelKeys = []string{"owner_kind", "owner_name", "owner_is_controller"}
					m.Value = 1
				}

				return f
			}),
		},
	}
)

func wrapReplicaSetFunc(f func(*v1beta1.ReplicaSet) metrics.Family) func(interface{}) metrics.Family {
	return func(obj interface{}) metrics.Family {
		replicaSet := obj.(*v1beta1.ReplicaSet)

		metricFamily := f(replicaSet)

		for _, m := range metricFamily {
			m.LabelKeys = append(descReplicaSetLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{replicaSet.Namespace, replicaSet.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createReplicaSetListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.ExtensionsV1beta1().ReplicaSets(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.ExtensionsV1beta1().ReplicaSets(ns).Watch(opts)
		},
	}
}
