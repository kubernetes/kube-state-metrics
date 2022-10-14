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

	policyv1 "k8s.io/api/policy/v1"
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
	descPodDisruptionBudgetLabelsDefaultLabels = []string{"namespace", "poddisruptionbudget"}
	descPodDisruptionBudgetAnnotationsName     = "kube_poddisruptionbudget_annotations"
	descPodDisruptionBudgetAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descPodDisruptionBudgetLabelsName          = "kube_poddisruptionbudget_labels"
	descPodDisruptionBudgetLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
)

func podDisruptionBudgetMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			descPodDisruptionBudgetAnnotationsName,
			descPodDisruptionBudgetAnnotationsHelp,
			metric.Gauge,
			"",
			wrapPodDisruptionBudgetFunc(func(p *policyv1.PodDisruptionBudget) *metric.Family {
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", p.Annotations, allowAnnotationsList)
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
		*generator.NewFamilyGenerator(
			descPodDisruptionBudgetLabelsName,
			descPodDisruptionBudgetLabelsHelp,
			metric.Gauge,
			"",
			wrapPodDisruptionBudgetFunc(func(p *policyv1.PodDisruptionBudget) *metric.Family {
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", p.Labels, allowLabelsList)
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
			"kube_poddisruptionbudget_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapPodDisruptionBudgetFunc(func(p *policyv1.PodDisruptionBudget) *metric.Family {
				ms := []*metric.Metric{}

				if !p.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(p.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_poddisruptionbudget_status_current_healthy",
			"Current number of healthy pods",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapPodDisruptionBudgetFunc(func(p *policyv1.PodDisruptionBudget) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(p.Status.CurrentHealthy),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_poddisruptionbudget_status_desired_healthy",
			"Minimum desired number of healthy pods",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapPodDisruptionBudgetFunc(func(p *policyv1.PodDisruptionBudget) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(p.Status.DesiredHealthy),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_poddisruptionbudget_status_pod_disruptions_allowed",
			"Number of pod disruptions that are currently allowed",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapPodDisruptionBudgetFunc(func(p *policyv1.PodDisruptionBudget) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(p.Status.DisruptionsAllowed),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_poddisruptionbudget_status_expected_pods",
			"Total number of pods counted by this disruption budget",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapPodDisruptionBudgetFunc(func(p *policyv1.PodDisruptionBudget) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(p.Status.ExpectedPods),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_poddisruptionbudget_status_observed_generation",
			"Most recent generation observed when updating this PDB status",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapPodDisruptionBudgetFunc(func(p *policyv1.PodDisruptionBudget) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(p.Status.ObservedGeneration),
						},
					},
				}
			}),
		),
	}
}

func wrapPodDisruptionBudgetFunc(f func(*policyv1.PodDisruptionBudget) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		podDisruptionBudget := obj.(*policyv1.PodDisruptionBudget)

		metricFamily := f(podDisruptionBudget)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descPodDisruptionBudgetLabelsDefaultLabels, []string{podDisruptionBudget.Namespace, podDisruptionBudget.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createPodDisruptionBudgetListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.PolicyV1().PodDisruptionBudgets(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.PolicyV1().PodDisruptionBudgets(ns).Watch(context.TODO(), opts)
		},
	}
}
