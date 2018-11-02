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

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descResourceQuotaLabelsDefaultLabels = []string{"resourcequota", "namespace"}

	descResourceQuotaCreated = metrics.NewMetricFamilyDef(
		"kube_resourcequota_created",
		"Unix creation timestamp",
		descResourceQuotaLabelsDefaultLabels,
		nil,
	)
	descResourceQuota = metrics.NewMetricFamilyDef(
		"kube_resourcequota",
		"Information about resource quota.",
		append(descResourceQuotaLabelsDefaultLabels,
			"resource",
			"type",
		), nil,
	)
)

func createResourceQuotaListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().ResourceQuotas(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().ResourceQuotas(ns).Watch(opts)
		},
	}
}

func generateResourceQuotaMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	rPointer := obj.(*v1.ResourceQuota)
	r := *rPointer

	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{r.Name, r.Namespace}, lv...)
		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}

	if !r.CreationTimestamp.IsZero() {
		addGauge(descResourceQuotaCreated, float64(r.CreationTimestamp.Unix()))
	}
	for res, qty := range r.Status.Hard {
		addGauge(descResourceQuota, float64(qty.MilliValue())/1000, string(res), "hard")
	}
	for res, qty := range r.Status.Used {
		addGauge(descResourceQuota, float64(qty.MilliValue())/1000, string(res), "used")
	}

	return ms
}
