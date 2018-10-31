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
	descConfigMapLabelsDefaultLabels = []string{"namespace", "configmap"}

	descConfigMapInfo = metrics.NewMetricFamilyDef(
		"kube_configmap_info",
		"Information about configmap.",
		descConfigMapLabelsDefaultLabels,
		nil,
	)

	descConfigMapCreated = metrics.NewMetricFamilyDef(
		"kube_configmap_created",
		"Unix creation timestamp",
		descConfigMapLabelsDefaultLabels,
		nil,
	)

	descConfigMapMetadataResourceVersion = metrics.NewMetricFamilyDef(
		"kube_configmap_metadata_resource_version",
		"Resource version representing a specific version of the configmap.",
		append(descConfigMapLabelsDefaultLabels, "resource_version"),
		nil,
	)
)

func createConfigMapListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().ConfigMaps(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().ConfigMaps(ns).Watch(opts)
		},
	}
}

func generateConfigMapMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	mPointer := obj.(*v1.ConfigMap)
	m := *mPointer

	addConstMetric := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{m.Namespace, m.Name}, lv...)

		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}
	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		addConstMetric(desc, v, lv...)
	}
	addGauge(descConfigMapInfo, 1)

	if !m.CreationTimestamp.IsZero() {
		addGauge(descConfigMapCreated, float64(m.CreationTimestamp.Unix()))
	}

	addGauge(descConfigMapMetadataResourceVersion, 1, string(m.ObjectMeta.ResourceVersion))

	return ms
}
