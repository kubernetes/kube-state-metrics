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
	descResourceQuota = prometheus.NewDesc(
		"kube_resource_quota",
		"Information about resource quota.",
		[]string{
			"name",
			"namespace",
			"resource",
			"type",
		}, nil,
	)

	resources = []v1.ResourceName{
		v1.ResourceCPU,
		v1.ResourceMemory,
		v1.ResourceStorage,
		v1.ResourcePods,
		v1.ResourceServices,
		v1.ResourceReplicationControllers,
		v1.ResourceQuotas,
		v1.ResourceSecrets,
		v1.ResourceConfigMaps,
		v1.ResourcePersistentVolumeClaims,
		v1.ResourceServicesNodePorts,
		v1.ResourceServicesLoadBalancers,
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
	ch <- descResourceQuota
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
	for _, res := range resources {
		addResource(descResourceQuota, rq.Status.Hard, res, string(res), "hard")
		addResource(descResourceQuota, rq.Status.Used, res, string(res), "used")
	}

}
