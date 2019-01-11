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

package collectors

import (
	"k8s.io/kube-state-metrics/pkg/metrics"

	"k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descStatefulSetLabelsName          = "kube_statefulset_labels"
	descStatefulSetLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descStatefulSetLabelsDefaultLabels = []string{"namespace", "statefulset"}

	statefulSetMetricFamilies = []metrics.FamilyGenerator{
		metrics.FamilyGenerator{
			Name: "kube_statefulset_created",
			Type: metrics.MetricTypeGauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1beta1.StatefulSet) metrics.Family {
				f := metrics.Family{}

				if !s.CreationTimestamp.IsZero() {
					f = append(f, &metrics.Metric{
						Name:  "kube_statefulset_created",
						Value: float64(s.CreationTimestamp.Unix()),
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_statefulset_status_replicas",
			Type: metrics.MetricTypeGauge,
			Help: "The number of replicas per StatefulSet.",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1beta1.StatefulSet) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_statefulset_status_replicas",
					Value: float64(s.Status.Replicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_statefulset_status_replicas_current",
			Type: metrics.MetricTypeGauge,
			Help: "The number of current replicas per StatefulSet.",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1beta1.StatefulSet) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_statefulset_status_replicas_current",
					Value: float64(s.Status.CurrentReplicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_statefulset_status_replicas_ready",
			Type: metrics.MetricTypeGauge,
			Help: "The number of ready replicas per StatefulSet.",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1beta1.StatefulSet) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_statefulset_status_replicas_ready",
					Value: float64(s.Status.ReadyReplicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_statefulset_status_replicas_updated",
			Type: metrics.MetricTypeGauge,
			Help: "The number of updated replicas per StatefulSet.",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1beta1.StatefulSet) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_statefulset_status_replicas_updated",
					Value: float64(s.Status.UpdatedReplicas),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_statefulset_status_observed_generation",
			Type: metrics.MetricTypeGauge,
			Help: "The generation observed by the StatefulSet controller.",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1beta1.StatefulSet) metrics.Family {
				f := metrics.Family{}

				if s.Status.ObservedGeneration != nil {
					f = append(f, &metrics.Metric{
						Name:  "kube_statefulset_status_observed_generation",
						Value: float64(*s.Status.ObservedGeneration),
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_statefulset_replicas",
			Type: metrics.MetricTypeGauge,
			Help: "Number of desired pods for a StatefulSet.",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1beta1.StatefulSet) metrics.Family {
				f := metrics.Family{}

				if s.Spec.Replicas != nil {
					f = append(f, &metrics.Metric{
						Name:  "kube_statefulset_replicas",
						Value: float64(*s.Spec.Replicas),
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_statefulset_metadata_generation",
			Type: metrics.MetricTypeGauge,
			Help: "Sequence number representing a specific generation of the desired state for the StatefulSet.",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1beta1.StatefulSet) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_statefulset_metadata_generation",
					Value: float64(s.ObjectMeta.Generation),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: descStatefulSetLabelsName,
			Type: metrics.MetricTypeGauge,
			Help: descStatefulSetLabelsHelp,
			GenerateFunc: wrapStatefulSetFunc(func(s *v1beta1.StatefulSet) metrics.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(s.Labels)
				return metrics.Family{&metrics.Metric{
					Name:        descStatefulSetLabelsName,
					LabelKeys:   labelKeys,
					LabelValues: labelValues,
					Value:       1,
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_statefulset_status_current_revision",
			Type: metrics.MetricTypeGauge,
			Help: "Indicates the version of the StatefulSet used to generate Pods in the sequence [0,currentReplicas).",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1beta1.StatefulSet) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:        "kube_statefulset_status_current_revision",
					LabelKeys:   []string{"revision"},
					LabelValues: []string{s.Status.CurrentRevision},
					Value:       1,
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_statefulset_status_update_revision",
			Type: metrics.MetricTypeGauge,
			Help: "Indicates the version of the StatefulSet used to generate Pods in the sequence [replicas-updatedReplicas,replicas)",
			GenerateFunc: wrapStatefulSetFunc(func(s *v1beta1.StatefulSet) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:        "kube_statefulset_status_update_revision",
					LabelKeys:   []string{"revision"},
					LabelValues: []string{s.Status.UpdateRevision},
					Value:       1,
				}}
			}),
		},
	}
)

func wrapStatefulSetFunc(f func(*v1beta1.StatefulSet) metrics.Family) func(interface{}) metrics.Family {
	return func(obj interface{}) metrics.Family {
		statefulSet := obj.(*v1beta1.StatefulSet)

		metricFamily := f(statefulSet)

		for _, m := range metricFamily {
			m.LabelKeys = append(descStatefulSetLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{statefulSet.Namespace, statefulSet.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createStatefulSetListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AppsV1beta1().StatefulSets(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AppsV1beta1().StatefulSets(ns).Watch(opts)
		},
	}
}
