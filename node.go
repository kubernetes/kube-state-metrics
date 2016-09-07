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
	"k8s.io/kubernetes/pkg/api"
)

var (
	descNodeInfo = prometheus.NewDesc(
		"node_info",
		"Information about a cluster node.",
		[]string{
			"node",
			"kernel_version",
			"os_image",
			"container_runtime_version",
			"kubelet_version",
			"kubeproxy_version",
		}, nil,
	)

	descNodeStatusReady = prometheus.NewDesc(
		"node_status_ready",
		"The ready status of a cluster node.",
		[]string{"node", "condition"}, nil,
	)
)

type nodeStore interface {
	List() (api.NodeList, error)
}

// nodeCollector collects metrics about all nodes in the cluster.
type nodeCollector struct {
	store nodeStore
}

// Describe implements the prometheus.Collector interface.
func (nc *nodeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descNodeInfo
	ch <- descNodeStatusReady
}

// Collect implements the prometheus.Collector interface.
func (nc *nodeCollector) Collect(ch chan<- prometheus.Metric) {
	nodes, err := nc.store.List()
	if err != nil {
		glog.Errorf("listing nodes failed: %s", err)
		return
	}
	for _, n := range nodes.Items {
		nc.collectNode(ch, n)
	}
}

func (nc *nodeCollector) collectNode(ch chan<- prometheus.Metric, n api.Node) {
	// Collect node conditions and while default to false.
	// TODO(fabxc): add remaining conditions: NodeOutOfDisk, NodeMemoryPressure,  NodeDiskPressure, NodeNetworkUnavailable
	for _, c := range n.Status.Conditions {
		switch c.Type {
		case api.NodeReady:
			nodeStatusMetrics(ch, descNodeStatusReady, n.Name, c.Status)
		}
	}

	// NOTE: the instrumentation API requires providing label values in order of declaration
	// in the metric descriptor. Be careful when making modifications.
	ch <- prometheus.MustNewConstMetric(
		descNodeInfo, prometheus.GaugeValue, 1,
		n.Name,
		n.Status.NodeInfo.KernelVersion,
		n.Status.NodeInfo.OSImage,
		n.Status.NodeInfo.ContainerRuntimeVersion,
		n.Status.NodeInfo.KubeletVersion,
		n.Status.NodeInfo.KubeProxyVersion,
	)
}

// nodeStatusMetrics generates one metric for each possible node condition status.
func nodeStatusMetrics(ch chan<- prometheus.Metric, desc *prometheus.Desc, name string, cs api.ConditionStatus) {
	ch <- prometheus.MustNewConstMetric(
		desc, prometheus.GaugeValue, boolFloat64(cs == api.ConditionTrue),
		name, "true",
	)
	ch <- prometheus.MustNewConstMetric(
		desc, prometheus.GaugeValue, boolFloat64(cs == api.ConditionFalse),
		name, "false",
	)
	ch <- prometheus.MustNewConstMetric(
		desc, prometheus.GaugeValue, boolFloat64(cs == api.ConditionUnknown),
		name, "unknown",
	)
}

func boolFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
