/*
Copyright 2020 The Kubernetes Authors All rights reserved.

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

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descLeaseLabelsDefaultLabels = []string{"lease"}

	leaseMetricFamilies = []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			"kube_lease_owner",
			"Information about the Lease's owner.",
			metric.Gauge,
			"",
			wrapLeaseFunc(func(l *coordinationv1.Lease) *metric.Family {
				labelKeys := []string{"owner_kind", "owner_name", "namespace", "lease_holder"}

				var holder string
				if l.Spec.HolderIdentity != nil {
					holder = *l.Spec.HolderIdentity
				}

				owners := l.GetOwnerReferences()
				if len(owners) == 0 {
					return &metric.Family{
						Metrics: []*metric.Metric{
							{
								LabelKeys:   labelKeys,
								LabelValues: []string{"", "", l.Namespace, holder},
								Value:       1,
							},
						},
					}
				}
				ms := make([]*metric.Metric, len(owners))

				for i, owner := range owners {
					ms[i] = &metric.Metric{
						LabelKeys:   labelKeys,
						LabelValues: []string{owner.Kind, owner.Name, l.Namespace, holder},
						Value:       1,
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_lease_renew_time",
			"Kube lease renew time.",
			metric.Gauge,
			"",
			wrapLeaseFunc(func(l *coordinationv1.Lease) *metric.Family {
				ms := []*metric.Metric{}

				if !l.Spec.RenewTime.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(l.Spec.RenewTime.Unix()),
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
)

func wrapLeaseFunc(f func(*coordinationv1.Lease) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		lease := obj.(*coordinationv1.Lease)

		metricFamily := f(lease)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descLeaseLabelsDefaultLabels, []string{lease.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createLeaseListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoordinationV1().Leases("").List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoordinationV1().Leases("").Watch(context.TODO(), opts)
		},
	}
}
