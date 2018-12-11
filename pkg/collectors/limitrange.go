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
	descLimitRangeLabelsDefaultLabels = []string{"namespace", "limitrange"}

	limitRangeMetricFamilies = []metrics.FamilyGenerator{
		metrics.FamilyGenerator{
			Name: "kube_limitrange",
			Type: metrics.MetricTypeGauge,
			Help: "Information about limit range.",
			GenerateFunc: wrapLimitRangeFunc(func(r *v1.LimitRange) metrics.Family {
				f := metrics.Family{}

				rawLimitRanges := r.Spec.Limits
				for _, rawLimitRange := range rawLimitRanges {
					for resource, min := range rawLimitRange.Min {
						f = append(f, &metrics.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "min"},
							Value:       float64(min.MilliValue()) / 1000,
						})
					}

					for resource, max := range rawLimitRange.Max {
						f = append(f, &metrics.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "max"},
							Value:       float64(max.MilliValue()) / 1000,
						})
					}

					for resource, df := range rawLimitRange.Default {
						f = append(f, &metrics.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "default"},
							Value:       float64(df.MilliValue()) / 1000,
						})
					}

					for resource, dfR := range rawLimitRange.DefaultRequest {
						f = append(f, &metrics.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "defaultRequest"},
							Value:       float64(dfR.MilliValue()) / 1000,
						})
					}

					for resource, mLR := range rawLimitRange.MaxLimitRequestRatio {
						f = append(f, &metrics.Metric{
							LabelValues: []string{string(resource), string(rawLimitRange.Type), "maxLimitRequestRatio"},
							Value:       float64(mLR.MilliValue()) / 1000,
						})
					}
				}

				for _, m := range f {
					m.Name = "kube_limitrange"
					m.LabelKeys = []string{"resource", "type", "constraint"}
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_limitrange_created",
			Type: metrics.MetricTypeGauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapLimitRangeFunc(func(r *v1.LimitRange) metrics.Family {
				f := metrics.Family{}

				if !r.CreationTimestamp.IsZero() {
					f = append(f, &metrics.Metric{
						Name:  "kube_limitrange_created",
						Value: float64(r.CreationTimestamp.Unix()),
					})
				}

				return f
			}),
		},
	}
)

func wrapLimitRangeFunc(f func(*v1.LimitRange) metrics.Family) func(interface{}) metrics.Family {
	return func(obj interface{}) metrics.Family {
		limitRange := obj.(*v1.LimitRange)

		metricFamily := f(limitRange)

		for _, m := range metricFamily {
			m.LabelKeys = append(descLimitRangeLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{limitRange.Namespace, limitRange.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

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
