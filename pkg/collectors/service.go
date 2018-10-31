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

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descServiceLabelsName          = "kube_service_labels"
	descServiceLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descServiceLabelsDefaultLabels = []string{"namespace", "service"}

	descServiceInfo = metrics.NewMetricFamilyDef(
		"kube_service_info",
		"Information about service.",
		append(descServiceLabelsDefaultLabels, "cluster_ip"),
		nil,
	)

	descServiceCreated = metrics.NewMetricFamilyDef(
		"kube_service_created",
		"Unix creation timestamp",
		descServiceLabelsDefaultLabels,
		nil,
	)

	descServiceSpecType = metrics.NewMetricFamilyDef(
		"kube_service_spec_type",
		"Type about service.",
		append(descServiceLabelsDefaultLabels, "type"),
		nil,
	)

	descServiceLabels = metrics.NewMetricFamilyDef(
		descServiceLabelsName,
		descServiceLabelsHelp,
		descServiceLabelsDefaultLabels,
		nil,
	)
)

func createServiceListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Services(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Services(ns).Watch(opts)
		},
	}
}

func serviceLabelsDesc(labelKeys []string) *metrics.MetricFamilyDef {
	return metrics.NewMetricFamilyDef(
		descServiceLabelsName,
		descServiceLabelsHelp,
		append(descServiceLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func generateServiceMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	sPointer := obj.(*v1.Service)
	s := *sPointer

	addConstMetric := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{s.Namespace, s.Name}, lv...)

		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}
	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		addConstMetric(desc, v, lv...)
	}
	addGauge(descServiceSpecType, 1, string(s.Spec.Type))

	addGauge(descServiceInfo, 1, s.Spec.ClusterIP)
	if !s.CreationTimestamp.IsZero() {
		addGauge(descServiceCreated, float64(s.CreationTimestamp.Unix()))
	}
	labelKeys, labelValues := kubeLabelsToPrometheusLabels(s.Labels)
	addGauge(serviceLabelsDesc(labelKeys), 1, labelValues...)

	return ms
}
