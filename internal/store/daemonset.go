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

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descDaemonSetAnnotationsName     = "kube_daemonset_annotations"
	descDaemonSetAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descDaemonSetLabelsName          = "kube_daemonset_labels"
	descDaemonSetLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descDaemonSetLabelsDefaultLabels = []string{"namespace", "daemonset"}
)

func daemonSetMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			"kube_daemonset_created",
			"Unix creation timestamp",
			metric.Gauge,
			"",
			wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				ms := []*metric.Metric{}

				if !d.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(d.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_daemonset_status_current_number_scheduled",
			"The number of nodes running at least one daemon pod and are supposed to.",
			metric.Gauge,
			"",
			wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.Status.CurrentNumberScheduled),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_daemonset_status_desired_number_scheduled",
			"The number of nodes that should be running the daemon pod.",
			metric.Gauge,
			"",
			wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.Status.DesiredNumberScheduled),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_daemonset_status_number_available",
			"The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and available",
			metric.Gauge,
			"",
			wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.Status.NumberAvailable),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_daemonset_status_number_misscheduled",
			"The number of nodes running a daemon pod but are not supposed to.",
			metric.Gauge,
			"",
			wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.Status.NumberMisscheduled),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_daemonset_status_number_ready",
			"The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and ready.",
			metric.Gauge,
			"",
			wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.Status.NumberReady),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_daemonset_status_number_unavailable",
			"The number of nodes that should be running the daemon pod and have none of the daemon pod running and available",
			metric.Gauge,
			"",
			wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.Status.NumberUnavailable),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_daemonset_status_observed_generation",
			"The most recent generation observed by the daemon set controller.",
			metric.Gauge,
			"",
			wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.Status.ObservedGeneration),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_daemonset_status_updated_number_scheduled",
			"The total number of nodes that are running updated daemon pod",
			metric.Gauge,
			"",
			wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.Status.UpdatedNumberScheduled),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_daemonset_metadata_generation",
			"Sequence number representing a specific generation of the desired state.",
			metric.Gauge,
			"",
			wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(d.ObjectMeta.Generation),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			descDaemonSetAnnotationsName,
			descDaemonSetAnnotationsHelp,
			metric.Gauge,
			"",
			wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", d.Annotations, allowAnnotationsList)
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
			descDaemonSetLabelsName,
			descDaemonSetLabelsHelp,
			metric.Gauge,
			"",
			wrapDaemonSetFunc(func(d *v1.DaemonSet) *metric.Family {
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", d.Labels, allowLabelsList)
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

func wrapDaemonSetFunc(f func(*v1.DaemonSet) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		daemonSet := obj.(*v1.DaemonSet)

		metricFamily := f(daemonSet)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descDaemonSetLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{daemonSet.Namespace, daemonSet.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createDaemonSetListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AppsV1().DaemonSets(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AppsV1().DaemonSets(ns).Watch(context.TODO(), opts)
		},
	}
}
