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
	descLimitRangeLabelsDefaultLabels = []string{"limitrange", "namespace"}
	descLimitRange                    = metrics.NewMetricFamilyDef(
		"kube_limitrange",
		"Information about limit range.",
		append(descLimitRangeLabelsDefaultLabels, "resource", "type", "constraint"),
		nil,
	)

	descLimitRangeCreated = metrics.NewMetricFamilyDef(
		"kube_limitrange_created",
		"Unix creation timestamp",
		descLimitRangeLabelsDefaultLabels,
		nil,
	)
)

func createLimitRangeListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().LimitRanges(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().LimitRanges(ns).Watch(opts)
		},
	}
}
func generateLimitRangeMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	lPointer := obj.(*v1.LimitRange)
	l := *lPointer

	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{l.Name, l.Namespace}, lv...)

		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}
	if !l.CreationTimestamp.IsZero() {
		addGauge(descLimitRangeCreated, float64(l.CreationTimestamp.Unix()))
	}

	rawLimitRanges := l.Spec.Limits
	for _, rawLimitRange := range rawLimitRanges {
		for resource, min := range rawLimitRange.Min {
			addGauge(descLimitRange, float64(min.MilliValue())/1000, string(resource), string(rawLimitRange.Type), "min")
		}

		for resource, max := range rawLimitRange.Max {
			addGauge(descLimitRange, float64(max.MilliValue())/1000, string(resource), string(rawLimitRange.Type), "max")
		}

		for resource, df := range rawLimitRange.Default {
			addGauge(descLimitRange, float64(df.MilliValue())/1000, string(resource), string(rawLimitRange.Type), "default")
		}

		for resource, dfR := range rawLimitRange.DefaultRequest {
			addGauge(descLimitRange, float64(dfR.MilliValue())/1000, string(resource), string(rawLimitRange.Type), "defaultRequest")
		}

		for resource, mLR := range rawLimitRange.MaxLimitRequestRatio {
			addGauge(descLimitRange, float64(mLR.MilliValue())/1000, string(resource), string(rawLimitRange.Type), "maxLimitRequestRatio")
		}

	}

	return ms
}
