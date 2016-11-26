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
	"k8s.io/client-go/1.5/pkg/api/v1"
)

var (
	descResourceQuotaCPU = prometheus.NewDesc(
		"kube_resource_quota_cpu",
		"Information about CPU resource quota.",
		[]string{
			"name",
			"namespace",
			"resource",
			"type",
		}, nil,
	)
	descResourceQuotaMemory = prometheus.NewDesc(
		"kube_resource_quota_memory",
		"Information about memory resource quota.",
		[]string{
			"name",
			"namespace",
			"resource",
			"type",
		}, nil,
	)
	descResourceQuotaPods = prometheus.NewDesc(
		"kube_resource_quota_pods",
		"Information about Pods resource quota.",
		[]string{
			"name",
			"namespace",
			"resource",
			"type",
		}, nil,
	)
	descResourceQuotaServices = prometheus.NewDesc(
		"kube_resource_quota_services",
		"Information about services resource quota.",
		[]string{
			"name",
			"namespace",
			"resource",
			"type",
		}, nil,
	)
	descResourceQuotaReplicationControllers = prometheus.NewDesc(
		"kube_resource_quota_replication_controllers",
		"Information about replication controllers resource quota.",
		[]string{
			"name",
			"namespace",
			"resource",
			"type",
		}, nil,
	)
	descResourceQuotaResourceQuotas = prometheus.NewDesc(
		"kube_resource_quota_resource_quota",
		"Information about resource quotas resource quota.",
		[]string{
			"name",
			"namespace",
			"resource",
			"type",
		}, nil,
	)
	descResourceQuotaSecrets = prometheus.NewDesc(
		"kube_resource_quota_secrets",
		"Information about secrets resource quota.",
		[]string{
			"name",
			"namespace",
			"resource",
			"type",
		}, nil,
	)
	descResourceQuotaConfigMaps = prometheus.NewDesc(
		"kube_resource_quota_config_maps",
		"Information about configmaps hard resource quota.",
		[]string{
			"name",
			"namespace",
			"resource",
			"type",
		}, nil,
	)
	descResourceQuotaPersistentVolumeClaims = prometheus.NewDesc(
		"kube_resource_quota_persistent_volume_claims",
		"Information about persistent volume claims resource quota.",
		[]string{
			"name",
			"namespace",
			"resource",
			"type",
		}, nil,
	)
	descResourceQuotaNodePorts = prometheus.NewDesc(
		"kube_resource_quota_node_ports",
		"Information about node ports resource quota.",
		[]string{
			"name",
			"namespace",
			"resource",
			"type",
		}, nil,
	)
	descResourceQuotaLoadBalancers = prometheus.NewDesc(
		"kube_resource_quota_load_balancers",
		"Information about load balancers resource quota.",
		[]string{
			"name",
			"namespace",
			"resource",
			"type",
		}, nil,
	)
	descResourceQuotaStorage = prometheus.NewDesc(
		"kube_resource_quota_storage",
		"Information about storage resource quota.",
		[]string{
			"name",
			"namespace",
			"resource",
			"type",
		}, nil,
	)

	resources = map[v1.ResourceName]*prometheus.Desc{
		v1.ResourceCPU:                    descResourceQuotaCPU,
		v1.ResourceMemory:                 descResourceQuotaMemory,
		v1.ResourceStorage:                descResourceQuotaStorage,
		v1.ResourcePods:                   descResourceQuotaPods,
		v1.ResourceServices:               descResourceQuotaServices,
		v1.ResourceReplicationControllers: descResourceQuotaReplicationControllers,
		v1.ResourceQuotas:                 descResourceQuotaResourceQuotas,
		v1.ResourceSecrets:                descResourceQuotaSecrets,
		v1.ResourceConfigMaps:             descResourceQuotaConfigMaps,
		v1.ResourcePersistentVolumeClaims: descResourceQuotaPersistentVolumeClaims,
		v1.ResourceServicesNodePorts:      descResourceQuotaNodePorts,
		v1.ResourceServicesLoadBalancers:  descResourceQuotaLoadBalancers,
	}
)

type resourceQuotaStore interface {
	List() (v1.ResourceQuotaList, error)
}

// resourceQuotaCollector collects metrics about all resource quotas in the cluster.
type resourceQuotaCollector struct {
	store resourceQuotaStore
}

// Describe implements the prometheus.Collector interface.
func (rqc *resourceQuotaCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descResourceQuotaCPU
	ch <- descResourceQuotaMemory
	ch <- descResourceQuotaPods
	ch <- descResourceQuotaServices
	ch <- descResourceQuotaReplicationControllers
	ch <- descResourceQuotaResourceQuotas
	ch <- descResourceQuotaSecrets
	ch <- descResourceQuotaConfigMaps
	ch <- descResourceQuotaPersistentVolumeClaims
	ch <- descResourceQuotaNodePorts
	ch <- descResourceQuotaLoadBalancers
	ch <- descResourceQuotaStorage
}

// Collect implements the prometheus.Collector interface.
func (rqc *resourceQuotaCollector) Collect(ch chan<- prometheus.Metric) {
	resourceQuota, err := rqc.store.List()
	if err != nil {
		glog.Errorf("listing resource quotas failed: %s", err)
		return
	}

	for _, rq := range resourceQuota.Items {
		rqc.collectResourceQuota(ch, rq)
	}
}

func (rqc *resourceQuotaCollector) collectResourceQuota(ch chan<- prometheus.Metric, rq v1.ResourceQuota) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{rq.Name, rq.Namespace}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	// Add capacity and allocatable resources if they are set.
	addResource := func(d *prometheus.Desc, res v1.ResourceList, n v1.ResourceName, labels ...string) {
		if v, ok := res[n]; ok {
			addGauge(d, float64(v.MilliValue())/1000, labels...)
		}
	}
	for res, desc := range resources {
		addResource(desc, rq.Status.Hard, res, string(res), "hard")
		addResource(desc, rq.Status.Used, res, string(res), "used")
	}

}
