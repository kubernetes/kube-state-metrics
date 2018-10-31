/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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
	descSecretLabelsName          = "kube_secret_labels"
	descSecretLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descSecretLabelsDefaultLabels = []string{"namespace", "secret"}

	descSecretInfo = metrics.NewMetricFamilyDef(
		"kube_secret_info",
		"Information about secret.",
		descSecretLabelsDefaultLabels,
		nil,
	)

	descSecretType = metrics.NewMetricFamilyDef(
		"kube_secret_type",
		"Type about secret.",
		append(descSecretLabelsDefaultLabels, "type"),
		nil,
	)

	descSecretLabels = metrics.NewMetricFamilyDef(
		descSecretLabelsName,
		descSecretLabelsHelp,
		descSecretLabelsDefaultLabels,
		nil,
	)

	descSecretCreated = metrics.NewMetricFamilyDef(
		"kube_secret_created",
		"Unix creation timestamp",
		descSecretLabelsDefaultLabels,
		nil,
	)

	descSecretMetadataResourceVersion = metrics.NewMetricFamilyDef(
		"kube_secret_metadata_resource_version",
		"Resource version representing a specific version of secret.",
		append(descSecretLabelsDefaultLabels, "resource_version"),
		nil,
	)
)

func createSecretListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Secrets(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Secrets(ns).Watch(opts)
		},
	}
}
func secretLabelsDesc(labelKeys []string) *metrics.MetricFamilyDef {
	return metrics.NewMetricFamilyDef(
		descSecretLabelsName,
		descSecretLabelsHelp,
		append(descSecretLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func generateSecretMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	sPointer := obj.(*v1.Secret)
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
	addGauge(descSecretInfo, 1)

	addGauge(descSecretType, 1, string(s.Type))
	if !s.CreationTimestamp.IsZero() {
		addGauge(descSecretCreated, float64(s.CreationTimestamp.Unix()))
	}
	labelKeys, labelValues := kubeLabelsToPrometheusLabels(s.Labels)
	addGauge(secretLabelsDesc(labelKeys), 1, labelValues...)

	addGauge(descSecretMetadataResourceVersion, 1, string(s.ObjectMeta.ResourceVersion))

	return ms
}
