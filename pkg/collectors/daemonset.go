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
	"k8s.io/kube-state-metrics/pkg/metrics"

	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descDaemonSetLabelsName          = "kube_daemonset_labels"
	descDaemonSetLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descDaemonSetLabelsDefaultLabels = []string{"namespace", "daemonset"}

	descDaemonSetCreated = metrics.NewMetricFamilyDef(
		"kube_daemonset_created",
		"Unix creation timestamp",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetCurrentNumberScheduled = metrics.NewMetricFamilyDef(
		"kube_daemonset_status_current_number_scheduled",
		"The number of nodes running at least one daemon pod and are supposed to.",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetDesiredNumberScheduled = metrics.NewMetricFamilyDef(
		"kube_daemonset_status_desired_number_scheduled",
		"The number of nodes that should be running the daemon pod.",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetNumberAvailable = metrics.NewMetricFamilyDef(
		"kube_daemonset_status_number_available",
		"The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and available",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetNumberMisscheduled = metrics.NewMetricFamilyDef(
		"kube_daemonset_status_number_misscheduled",
		"The number of nodes running a daemon pod but are not supposed to.",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetNumberReady = metrics.NewMetricFamilyDef(
		"kube_daemonset_status_number_ready",
		"The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and ready.",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetNumberUnavailable = metrics.NewMetricFamilyDef(
		"kube_daemonset_status_number_unavailable",
		"The number of nodes that should be running the daemon pod and have none of the daemon pod running and available",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetUpdatedNumberScheduled = metrics.NewMetricFamilyDef(
		"kube_daemonset_updated_number_scheduled",
		"The total number of nodes that are running updated daemon pod",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetMetadataGeneration = metrics.NewMetricFamilyDef(
		"kube_daemonset_metadata_generation",
		"Sequence number representing a specific generation of the desired state.",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetLabels = metrics.NewMetricFamilyDef(
		descDaemonSetLabelsName,
		descDaemonSetLabelsHelp,
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
)

// TODO: Not necessary without HELP and TYPE line
// // Describe implements the prometheus.Collector interface.
// func (dc *daemonsetCollector) Describe(ch chan<- *metrics.MetricFamilyDef) {
// 	ch <- descDaemonSetCreated
// 	ch <- descDaemonSetCurrentNumberScheduled
// 	ch <- descDaemonSetNumberAvailable
// 	ch <- descDaemonSetNumberMisscheduled
// 	ch <- descDaemonSetNumberUnavailable
// 	ch <- descDaemonSetDesiredNumberScheduled
// 	ch <- descDaemonSetNumberReady
// 	ch <- descDaemonSetUpdatedNumberScheduled
// 	ch <- descDaemonSetMetadataGeneration
// 	ch <- descDaemonSetLabels
// }

func createDaemonSetListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.ExtensionsV1beta1().DaemonSets(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.ExtensionsV1beta1().DaemonSets(ns).Watch(opts)
		},
	}
}

func DaemonSetLabelsDesc(labelKeys []string) *metrics.MetricFamilyDef {
	return metrics.NewMetricFamilyDef(
		descDaemonSetLabelsName,
		descDaemonSetLabelsHelp,
		append(descDaemonSetLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func generateDaemonSetMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	dPointer := obj.(*v1beta1.DaemonSet)
	d := *dPointer

	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{d.Namespace, d.Name}, lv...)

		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}
	if !d.CreationTimestamp.IsZero() {
		addGauge(descDaemonSetCreated, float64(d.CreationTimestamp.Unix()))
	}
	addGauge(descDaemonSetCurrentNumberScheduled, float64(d.Status.CurrentNumberScheduled))
	addGauge(descDaemonSetNumberAvailable, float64(d.Status.NumberAvailable))
	addGauge(descDaemonSetNumberUnavailable, float64(d.Status.NumberUnavailable))
	addGauge(descDaemonSetNumberMisscheduled, float64(d.Status.NumberMisscheduled))
	addGauge(descDaemonSetDesiredNumberScheduled, float64(d.Status.DesiredNumberScheduled))
	addGauge(descDaemonSetNumberReady, float64(d.Status.NumberReady))
	addGauge(descDaemonSetUpdatedNumberScheduled, float64(d.Status.UpdatedNumberScheduled))
	addGauge(descDaemonSetMetadataGeneration, float64(d.ObjectMeta.Generation))

	labelKeys, labelValues := kubeLabelsToPrometheusLabels(d.ObjectMeta.Labels)
	addGauge(DaemonSetLabelsDesc(labelKeys), 1, labelValues...)

	return ms
}
