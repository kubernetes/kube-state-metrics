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
	descEndpointLabelsName          = "kube_endpoint_labels"
	descEndpointLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descEndpointLabelsDefaultLabels = []string{"namespace", "endpoint"}

	descEndpointInfo = metrics.NewMetricFamilyDef(
		"kube_endpoint_info",
		"Information about endpoint.",
		descEndpointLabelsDefaultLabels,
		nil,
	)

	descEndpointCreated = metrics.NewMetricFamilyDef(
		"kube_endpoint_created",
		"Unix creation timestamp",
		descEndpointLabelsDefaultLabels,
		nil,
	)

	descEndpointLabels = metrics.NewMetricFamilyDef(
		descEndpointLabelsName,
		descEndpointLabelsHelp,
		descEndpointLabelsDefaultLabels,
		nil,
	)

	descEndpointAddressAvailable = metrics.NewMetricFamilyDef(
		"kube_endpoint_address_available",
		"Number of addresses available in endpoint.",
		descEndpointLabelsDefaultLabels,
		nil,
	)

	descEndpointAddressNotReady = metrics.NewMetricFamilyDef(
		"kube_endpoint_address_not_ready",
		"Number of addresses not ready in endpoint",
		descEndpointLabelsDefaultLabels,
		nil,
	)
)

func createEndpointsListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Endpoints(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Endpoints(ns).Watch(opts)
		},
	}
}

func generateEndpointsMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	ePointer := obj.(*v1.Endpoints)
	e := *ePointer

	addConstMetric := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{e.Namespace, e.Name}, lv...)

		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}
	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		addConstMetric(desc, v, lv...)
	}

	addGauge(descEndpointInfo, 1)
	if !e.CreationTimestamp.IsZero() {
		addGauge(descEndpointCreated, float64(e.CreationTimestamp.Unix()))
	}
	labelKeys, labelValues := kubeLabelsToPrometheusLabels(e.Labels)
	addGauge(endpointLabelsDesc(labelKeys), 1, labelValues...)

	var available int
	for _, s := range e.Subsets {
		available += len(s.Addresses) * len(s.Ports)
	}
	addGauge(descEndpointAddressAvailable, float64(available))

	var notReady int
	for _, s := range e.Subsets {
		notReady += len(s.NotReadyAddresses) * len(s.Ports)
	}
	addGauge(descEndpointAddressNotReady, float64(notReady))

	return ms
}

func endpointLabelsDesc(labelKeys []string) *metrics.MetricFamilyDef {
	return metrics.NewMetricFamilyDef(
		descEndpointLabelsName,
		descEndpointLabelsHelp,
		append(descEndpointLabelsDefaultLabels, labelKeys...),
		nil,
	)
}
