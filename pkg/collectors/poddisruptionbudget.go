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
	"k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/metrics"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
)

var (
	descPodDisruptionBudgetLabelsDefaultLabels = []string{"poddisruptionbudget", "namespace"}

	descPodDisruptionBudgetCreated = metrics.NewMetricFamilyDef(
		"kube_poddisruptionbudget_created",
		"Unix creation timestamp",
		descPodDisruptionBudgetLabelsDefaultLabels,
		nil,
	)

	descPodDisruptionBudgetStatusCurrentHealthy = metrics.NewMetricFamilyDef(
		"kube_poddisruptionbudget_status_current_healthy",
		"Current number of healthy pods",
		descPodDisruptionBudgetLabelsDefaultLabels,
		nil,
	)
	descPodDisruptionBudgetStatusDesiredHealthy = metrics.NewMetricFamilyDef(
		"kube_poddisruptionbudget_status_desired_healthy",
		"Minimum desired number of healthy pods",
		descPodDisruptionBudgetLabelsDefaultLabels,
		nil,
	)
	descPodDisruptionBudgetStatusPodDisruptionsAllowed = metrics.NewMetricFamilyDef(
		"kube_poddisruptionbudget_status_pod_disruptions_allowed",
		"Number of pod disruptions that are currently allowed",
		descPodDisruptionBudgetLabelsDefaultLabels,
		nil,
	)
	descPodDisruptionBudgetStatusExpectedPods = metrics.NewMetricFamilyDef(
		"kube_poddisruptionbudget_status_expected_pods",
		"Total number of pods counted by this disruption budget",
		descPodDisruptionBudgetLabelsDefaultLabels,
		nil,
	)
	descPodDisruptionBudgetStatusObservedGeneration = metrics.NewMetricFamilyDef(
		"kube_poddisruptionbudget_status_observed_generation",
		"Most recent generation observed when updating this PDB status",
		descPodDisruptionBudgetLabelsDefaultLabels,
		nil,
	)
)

func createPodDisruptionBudgetListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.PolicyV1beta1().PodDisruptionBudgets(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.PolicyV1beta1().PodDisruptionBudgets(ns).Watch(opts)
		},
	}
}

func generatePodDisruptionBudgetMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	pPointer := obj.(*v1beta1.PodDisruptionBudget)
	p := *pPointer

	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{p.Name, p.Namespace}, lv...)
		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}

	if !p.CreationTimestamp.IsZero() {
		addGauge(descPodDisruptionBudgetCreated, float64(p.CreationTimestamp.Unix()))
	}
	addGauge(descPodDisruptionBudgetStatusCurrentHealthy, float64(p.Status.CurrentHealthy))
	addGauge(descPodDisruptionBudgetStatusDesiredHealthy, float64(p.Status.DesiredHealthy))
	addGauge(descPodDisruptionBudgetStatusPodDisruptionsAllowed, float64(p.Status.PodDisruptionsAllowed))
	addGauge(descPodDisruptionBudgetStatusExpectedPods, float64(p.Status.ExpectedPods))
	addGauge(descPodDisruptionBudgetStatusObservedGeneration, float64(p.Status.ObservedGeneration))

	return ms
}
