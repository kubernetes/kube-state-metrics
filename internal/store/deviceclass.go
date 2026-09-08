/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

	resourcev1beta1 "k8s.io/api/resource/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descDeviceClassLabelsDefaultLabels = []string{"deviceclass"}

	deviceClassMetricFamilies = []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_deviceclass_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapDeviceClassFunc(func(dc *resourcev1beta1.DeviceClass) *metric.Family {
				ms := []*metric.Metric{}

				if !dc.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(dc.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_deviceclass_info",
			"Information about device class.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapDeviceClassFunc(func(dc *resourcev1beta1.DeviceClass) *metric.Family {
				extendedResourceName := ""
				if dc.Spec.ExtendedResourceName != nil {
					extendedResourceName = *dc.Spec.ExtendedResourceName
				}
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"extended_resource_name"},
							LabelValues: []string{extendedResourceName},
							Value:       1,
						},
					},
				}
			}),
		),
	}
)

func wrapDeviceClassFunc(f func(*resourcev1beta1.DeviceClass) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		dc := obj.(*resourcev1beta1.DeviceClass)

		metricFamily := f(dc)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descDeviceClassLabelsDefaultLabels, []string{dc.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createDeviceClassListWatch(kubeClient clientset.Interface, _ string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.ResourceV1beta1().DeviceClasses().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.ResourceV1beta1().DeviceClasses().Watch(context.TODO(), opts)
		},
	}
}
