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

package store

import (
	"context"

	v1 "k8s.io/api/core/v1"
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
	descLimitRangeLabelsDefaultLabels = []string{"namespace", "limitrange"}

	limitRangeMetricFamilies = []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_limitrange",
			"Information about limit range.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapLimitRangeFunc(func(r *v1.LimitRange) *metric.Family {
				ms := []*metric.Metric{}

				rawLimitRanges := r.Spec.Limits
				for _, rawLimitRange := range rawLimitRanges {
					for resource, min := range rawLimitRange.Min {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "min"},
							Value:       convertValueToFloat64(&min),
						})
					}

					for resource, max := range rawLimitRange.Max {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "max"},
							Value:       convertValueToFloat64(&max),
						})
					}

					for resource, df := range rawLimitRange.Default {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "default"},
							Value:       convertValueToFloat64(&df),
						})
					}

					for resource, dfR := range rawLimitRange.DefaultRequest {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "defaultRequest"},
							Value:       convertValueToFloat64(&dfR),
						})
					}

					for resource, mLR := range rawLimitRange.MaxLimitRequestRatio {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "maxLimitRequestRatio"},
							Value:       convertValueToFloat64(&mLR),
						})
					}
				}

				for _, m := range ms {
					m.LabelKeys = []string{"resource", "type", "constraint"}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_limitrange_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapLimitRangeFunc(func(r *v1.LimitRange) *metric.Family {
				ms := []*metric.Metric{}

				if !r.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{

						Value: float64(r.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
)

func wrapLimitRangeFunc(f func(*v1.LimitRange) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		limitRange := obj.(*v1.LimitRange)

		metricFamily := f(limitRange)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descLimitRangeLabelsDefaultLabels, []string{limitRange.Namespace, limitRange.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createLimitRangeListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcherWithContext {
	return &cache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().LimitRanges(ns).List(context.TODO(), opts)
		},
		WatchFuncWithContext: func(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().LimitRanges(ns).Watch(context.TODO(), opts)
		},
	}
}
