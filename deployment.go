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

package main

import (
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/1.4/pkg/apis/extensions/v1beta1"
)

var (
	descDeploymentStatusReplicas = prometheus.NewDesc(
		"kube_deployment_status_replicas",
		"The number of replicas per deployment.",
		[]string{"namespace", "deployment"}, nil,
	)
	descDeploymentStatusReplicasAvailable = prometheus.NewDesc(
		"kube_deployment_status_replicas_available",
		"The number of available replicas per deployment.",
		[]string{"namespace", "deployment"}, nil,
	)
	descDeploymentStatusReplicasUnavailable = prometheus.NewDesc(
		"kube_deployment_status_replicas_unavailable",
		"The number of unavailable replicas per deployment.",
		[]string{"namespace", "deployment"}, nil,
	)
	descDeploymentStatusReplicasUpdated = prometheus.NewDesc(
		"kube_deployment_status_replicas_updated",
		"The number of updated replicas per deployment.",
		[]string{"namespace", "deployment"}, nil,
	)

	descDeploymentStatusObservedGeneration = prometheus.NewDesc(
		"kube_deployment_status_observed_generation",
		"The generation observed by the deployment controller.",
		[]string{"namespace", "deployment"}, nil,
	)

	descDeploymentSpecReplicas = prometheus.NewDesc(
		"kube_deployment_spec_replicas",
		"Number of desired pods for a deployment.",
		[]string{"namespace", "deployment"}, nil,
	)

	descDeploymentSpecPaused = prometheus.NewDesc(
		"kube_deployment_spec_paused",
		"Whether the deployment is paused and will not be processed by the deployment controller.",
		[]string{"namespace", "deployment"}, nil,
	)

	descDeploymentMetadataGeneration = prometheus.NewDesc(
		"kube_deployment_metadata_generation",
		"Sequence number representing a specific generation of the desired state.",
		[]string{"namespace", "deployment"}, nil,
	)
)

type deploymentStore interface {
	List() (deployments []v1beta1.Deployment, err error)
}

// deploymentCollector collects metrics about all deployments in the cluster.
type deploymentCollector struct {
	store deploymentStore
}

// Describe implements the prometheus.Collector interface.
func (dc *deploymentCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descDeploymentStatusReplicas
	ch <- descDeploymentStatusReplicasAvailable
	ch <- descDeploymentStatusReplicasUnavailable
	ch <- descDeploymentStatusReplicasUpdated
	ch <- descDeploymentStatusObservedGeneration
	ch <- descDeploymentSpecPaused
	ch <- descDeploymentSpecReplicas
	ch <- descDeploymentMetadataGeneration
}

// Collect implements the prometheus.Collector interface.
func (dc *deploymentCollector) Collect(ch chan<- prometheus.Metric) {
	dpls, err := dc.store.List()
	if err != nil {
		glog.Errorf("listing deployments failed: %s", err)
		return
	}
	for _, d := range dpls {
		dc.collectDeployment(ch, d)
	}
}

func (dc *deploymentCollector) collectDeployment(ch chan<- prometheus.Metric, d v1beta1.Deployment) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{d.Namespace, d.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	addGauge(descDeploymentStatusReplicas, float64(d.Status.Replicas))
	addGauge(descDeploymentStatusReplicasAvailable, float64(d.Status.AvailableReplicas))
	addGauge(descDeploymentStatusReplicasUnavailable, float64(d.Status.UnavailableReplicas))
	addGauge(descDeploymentStatusReplicasUpdated, float64(d.Status.UpdatedReplicas))
	addGauge(descDeploymentStatusObservedGeneration, float64(d.Status.ObservedGeneration))
	addGauge(descDeploymentSpecPaused, boolFloat64(d.Spec.Paused))
	addGauge(descDeploymentSpecReplicas, float64(*d.Spec.Replicas))
	addGauge(descDeploymentMetadataGeneration, float64(d.ObjectMeta.Generation))
}
