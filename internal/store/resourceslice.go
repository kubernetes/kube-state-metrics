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

	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	basemetrics "k8s.io/component-base/metrics"
	"k8s.io/utils/ptr"

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
			wrapResourceSliceFunc(func(rs *resourcev1.ResourceSlice) *metric.Family {
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
			wrapResourceSliceFunc(func(rs *resourcev1.ResourceSlice) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"driver", "pool_name", "node_name", "node_selection"},
							LabelValues: []string{rs.Spec.Driver, rs.Spec.Pool.Name, ptr.Deref(rs.Spec.NodeName, ""), resourceSliceNodeSelection(rs)},
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_resourceslice_devices",
			"The total count of devices published by this resource slice.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceSliceFunc(func(rs *resourcev1.ResourceSlice) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"driver", "pool_name", "node_name"},
							LabelValues: []string{rs.Spec.Driver, rs.Spec.Pool.Name, ptr.Deref(rs.Spec.NodeName, "")},
							Value:       float64(len(rs.Spec.Devices)),
						},
					},
				}
			}),
		),
	}
)

func wrapResourceSliceFunc(f func(*resourcev1.ResourceSlice) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		rs := obj.(*resourcev1.ResourceSlice)

		metricFamily := f(rs)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descResourceSliceLabelsDefaultLabels, []string{rs.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

// resourceSliceNodeSelection reports which of the mutually exclusive node
// selection fields is set on the slice.
func resourceSliceNodeSelection(rs *resourcev1.ResourceSlice) string {
	switch {
	case ptr.Deref(rs.Spec.NodeName, "") != "":
		return "node_name"
	case rs.Spec.NodeSelector != nil:
		return "node_selector"
	case ptr.Deref(rs.Spec.AllNodes, false):
		return "all_nodes"
	case ptr.Deref(rs.Spec.PerDeviceNodeSelection, false):
		return "per_device"
	default:
		return ""
	}
}

func createResourceSliceListWatch(kubeClient clientset.Interface, _ string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.ResourceV1().ResourceSlices().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.ResourceV1().ResourceSlices().Watch(context.TODO(), opts)
		},
	}
}
