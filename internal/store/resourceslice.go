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
	"strconv"

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
	descResourceSliceLabelsDefaultLabels = []string{"resourceslice"}

	resourceSliceMetricFamilies = []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_resourceslice_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceSliceFunc(func(rs *resourcev1beta1.ResourceSlice) *metric.Family {
				ms := []*metric.Metric{}

				if !rs.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(rs.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_resourceslice_info",
			"Information about resource slice.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceSliceFunc(func(rs *resourcev1beta1.ResourceSlice) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"driver", "pool_name", "node_name", "all_nodes"},
							LabelValues: []string{rs.Spec.Driver, rs.Spec.Pool.Name, rs.Spec.NodeName, strconv.FormatBool(rs.Spec.AllNodes)},
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_resourceslice_devices_total",
			"The total count of devices published by this resource slice.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceSliceFunc(func(rs *resourcev1beta1.ResourceSlice) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"driver", "pool_name", "node_name"},
							LabelValues: []string{rs.Spec.Driver, rs.Spec.Pool.Name, rs.Spec.NodeName},
							Value:       float64(len(rs.Spec.Devices)),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_resourceslice_device_info",
			"Details of individual devices inside the resource slice.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceSliceFunc(func(rs *resourcev1beta1.ResourceSlice) *metric.Family {
				ms := make([]*metric.Metric, len(rs.Spec.Devices))
				for i, dev := range rs.Spec.Devices {
					ms[i] = &metric.Metric{
						LabelKeys:   []string{"driver", "pool_name", "node_name", "device_name"},
						LabelValues: []string{rs.Spec.Driver, rs.Spec.Pool.Name, rs.Spec.NodeName, dev.Name},
						Value:       1,
					}
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
)

func wrapResourceSliceFunc(f func(*resourcev1beta1.ResourceSlice) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		rs := obj.(*resourcev1beta1.ResourceSlice)

		metricFamily := f(rs)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descResourceSliceLabelsDefaultLabels, []string{rs.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createResourceSliceListWatch(kubeClient clientset.Interface, _ string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.ResourceV1beta1().ResourceSlices().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.ResourceV1beta1().ResourceSlices().Watch(context.TODO(), opts)
		},
	}
}
