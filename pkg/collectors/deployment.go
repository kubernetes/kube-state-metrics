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

	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descDeploymentLabelsName          = "kube_deployment_labels"
	descDeploymentLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descDeploymentLabelsDefaultLabels = []string{"namespace", "deployment"}

	descDeploymentCreated = metrics.NewMetricFamilyDef(
		"kube_deployment_created",
		"Unix creation timestamp",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentStatusReplicas = metrics.NewMetricFamilyDef(
		"kube_deployment_status_replicas",
		"The number of replicas per deployment.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)
	descDeploymentStatusReplicasAvailable = metrics.NewMetricFamilyDef(
		"kube_deployment_status_replicas_available",
		"The number of available replicas per deployment.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)
	descDeploymentStatusReplicasUnavailable = metrics.NewMetricFamilyDef(
		"kube_deployment_status_replicas_unavailable",
		"The number of unavailable replicas per deployment.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)
	descDeploymentStatusReplicasUpdated = metrics.NewMetricFamilyDef(
		"kube_deployment_status_replicas_updated",
		"The number of updated replicas per deployment.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentStatusObservedGeneration = metrics.NewMetricFamilyDef(
		"kube_deployment_status_observed_generation",
		"The generation observed by the deployment controller.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentSpecReplicas = metrics.NewMetricFamilyDef(
		"kube_deployment_spec_replicas",
		"Number of desired pods for a deployment.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentSpecPaused = metrics.NewMetricFamilyDef(
		"kube_deployment_spec_paused",
		"Whether the deployment is paused and will not be processed by the deployment controller.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentStrategyRollingUpdateMaxUnavailable = metrics.NewMetricFamilyDef(
		"kube_deployment_spec_strategy_rollingupdate_max_unavailable",
		"Maximum number of unavailable replicas during a rolling update of a deployment.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentStrategyRollingUpdateMaxSurge = metrics.NewMetricFamilyDef(
		"kube_deployment_spec_strategy_rollingupdate_max_surge",
		"Maximum number of replicas that can be scheduled above the desired number of replicas during a rolling update of a deployment.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentMetadataGeneration = metrics.NewMetricFamilyDef(
		"kube_deployment_metadata_generation",
		"Sequence number representing a specific generation of the desired state.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentLabels = metrics.NewMetricFamilyDef(
		descDeploymentLabelsName,
		descDeploymentLabelsHelp,
		descDeploymentLabelsDefaultLabels, nil,
	)
)

func createDeploymentListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.ExtensionsV1beta1().Deployments(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.ExtensionsV1beta1().Deployments(ns).Watch(opts)
		},
	}
}

func deploymentLabelsDesc(labelKeys []string) *metrics.MetricFamilyDef {
	return metrics.NewMetricFamilyDef(
		descDeploymentLabelsName,
		descDeploymentLabelsHelp,
		append(descDeploymentLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func generateDeploymentMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	dPointer := obj.(*v1beta1.Deployment)
	d := *dPointer

	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{d.Namespace, d.Name}, lv...)

		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			// TODO: Handle!
			panic(err)
		}

		ms = append(ms, m)
	}
	labelKeys, labelValues := kubeLabelsToPrometheusLabels(d.Labels)
	addGauge(deploymentLabelsDesc(labelKeys), 1, labelValues...)
	if !d.CreationTimestamp.IsZero() {
		addGauge(descDeploymentCreated, float64(d.CreationTimestamp.Unix()))
	}
	addGauge(descDeploymentStatusReplicas, float64(d.Status.Replicas))
	addGauge(descDeploymentStatusReplicasAvailable, float64(d.Status.AvailableReplicas))
	addGauge(descDeploymentStatusReplicasUnavailable, float64(d.Status.UnavailableReplicas))
	addGauge(descDeploymentStatusReplicasUpdated, float64(d.Status.UpdatedReplicas))
	addGauge(descDeploymentStatusObservedGeneration, float64(d.Status.ObservedGeneration))
	addGauge(descDeploymentSpecPaused, boolFloat64(d.Spec.Paused))
	addGauge(descDeploymentSpecReplicas, float64(*d.Spec.Replicas))
	addGauge(descDeploymentMetadataGeneration, float64(d.ObjectMeta.Generation))

	if d.Spec.Strategy.RollingUpdate == nil {
		return nil
	}

	maxUnavailable, err := intstr.GetValueFromIntOrPercent(d.Spec.Strategy.RollingUpdate.MaxUnavailable, int(*d.Spec.Replicas), true)
	if err != nil {
		panic(err)
	} else {
		addGauge(descDeploymentStrategyRollingUpdateMaxUnavailable, float64(maxUnavailable))
	}

	maxSurge, err := intstr.GetValueFromIntOrPercent(d.Spec.Strategy.RollingUpdate.MaxSurge, int(*d.Spec.Replicas), true)
	if err != nil {
		panic(err)
	} else {
		addGauge(descDeploymentStrategyRollingUpdateMaxSurge, float64(maxSurge))
	}

	return ms
}
