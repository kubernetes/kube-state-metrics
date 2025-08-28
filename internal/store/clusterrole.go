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

package store

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
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
	descClusterRoleLabelsDefaultLabels = []string{"clusterrole"}
)

func clusterRoleMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	metadataFamilies := createMetadataMetricFamiliesGenerator(allowAnnotationsList, allowLabelsList, descClusterRoleLabelsDefaultLabels, "kube_clusterrole", wrapMetadataFunc)
	return append(metadataFamilies,
		*generator.NewFamilyGeneratorWithStability(
			"kube_clusterrole_info",
			"Information about cluster role.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapClusterRoleFunc(func(_ *rbacv1.ClusterRole) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       1,
					}},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_clusterrole_metadata_resource_version",
			"Resource version representing a specific version of the cluster role.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapClusterRoleFunc(func(r *rbacv1.ClusterRole) *metric.Family {
				return &metric.Family{
					Metrics: resourceVersionMetric(r.ResourceVersion),
				}
			}),
		),
	)
}

func createClusterRoleListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.RbacV1().ClusterRoles().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.RbacV1().ClusterRoles().Watch(context.TODO(), opts)
		},
	}
}

func wrapClusterRoleFunc(f func(*rbacv1.ClusterRole) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		clusterrole := obj.(*rbacv1.ClusterRole)

		metricFamily := f(clusterrole)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descClusterRoleLabelsDefaultLabels, []string{clusterrole.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}
