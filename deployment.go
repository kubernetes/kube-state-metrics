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
	"k8s.io/kubernetes/pkg/apis/extensions"
)

var (
	descDeploymentReplicas = prometheus.NewDesc(
		"deployment_replicas",
		"The number of replicas per deployment.",
		[]string{"namespace", "deployment"}, nil,
	)

	descDeploymentReplicasAvailable = prometheus.NewDesc(
		"deployment_replicas_available",
		"The number of available replicas per deployment.",
		[]string{"namespace", "deployment"}, nil,
	)
)

type deploymentStore interface {
	List() (deployments []extensions.Deployment, err error)
}

// deploymentCollector collects metrics about all deployments in the cluster.
type deploymentCollector struct {
	store deploymentStore
}

// Describe implements the prometheus.Collector interface.
func (dc *deploymentCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descDeploymentReplicas
	ch <- descDeploymentReplicasAvailable
}

// Collect implements the prometheus.Collector interface.
func (dc *deploymentCollector) Collect(ch chan<- prometheus.Metric) {
	dpls, err := dc.store.List()
	if err != nil {
		glog.Errorf("listing deployments failed: %s", err)
		return
	}
	for _, d := range dpls {
		for _, m := range dc.collectDeployment(d) {
			ch <- m
		}
	}
}

func (dc *deploymentCollector) collectDeployment(d extensions.Deployment) []prometheus.Metric {
	return []prometheus.Metric{
		prometheus.MustNewConstMetric(
			descDeploymentReplicas, prometheus.GaugeValue, float64(d.Status.Replicas),
			d.Namespace, d.Name,
		),
		prometheus.MustNewConstMetric(
			descDeploymentReplicasAvailable, prometheus.GaugeValue, float64(d.Status.AvailableReplicas),
			d.Namespace, d.Name,
		),
	}
}
