/*
Copyright 2022 The Kubernetes Authors All rights reserved.

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
	descClusterRoleBindingAnnotationsName     = "kube_clusterrolebinding_annotations"
	descClusterRoleBindingAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descClusterRoleBindingLabelsName          = "kube_clusterrolebinding_labels"
	descClusterRoleBindingLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descClusterRoleBindingLabelsDefaultLabels = []string{"clusterrolebinding"}
)

func clusterRoleBindingMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			descClusterRoleBindingAnnotationsName,
			descClusterRoleBindingAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapClusterRoleBindingFunc(func(r *rbacv1.ClusterRoleBinding) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", r.Annotations, allowAnnotationsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   annotationKeys,
							LabelValues: annotationValues,
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descClusterRoleBindingLabelsName,
			descClusterRoleBindingLabelsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapClusterRoleBindingFunc(func(r *rbacv1.ClusterRoleBinding) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", r.Labels, allowLabelsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_clusterrolebinding_info",
			"Information about clusterrolebinding.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapClusterRoleBindingFunc(func(r *rbacv1.ClusterRoleBinding) *metric.Family {
				labelKeys := []string{"roleref_kind", "roleref_name"}
				labelValues := []string{r.RoleRef.Kind, r.RoleRef.Name}
				return &metric.Family{
					Metrics: []*metric.Metric{{
						LabelKeys:   labelKeys,
						LabelValues: labelValues,
						Value:       1,
					}},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_clusterrolebinding_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapClusterRoleBindingFunc(func(r *rbacv1.ClusterRoleBinding) *metric.Family {
				ms := []*metric.Metric{}

				if !r.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(r.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_clusterrolebinding_metadata_resource_version",
			"Resource version representing a specific version of the clusterrolebinding.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapClusterRoleBindingFunc(func(r *rbacv1.ClusterRoleBinding) *metric.Family {
				return &metric.Family{
					Metrics: resourceVersionMetric(r.ResourceVersion),
				}
			}),
		),
	}
}

func createClusterRoleBindingListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.RbacV1().ClusterRoleBindings().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.RbacV1().ClusterRoleBindings().Watch(context.TODO(), opts)
		},
	}
}

func wrapClusterRoleBindingFunc(f func(*rbacv1.ClusterRoleBinding) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		clusterrolebinding := obj.(*rbacv1.ClusterRoleBinding)

		metricFamily := f(clusterrolebinding)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descClusterRoleBindingLabelsDefaultLabels, []string{clusterrolebinding.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}
