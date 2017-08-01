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
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	descNodeLabelsName          = "kube_node_labels"
	descNodeLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descNodeLabelsDefaultLabels = []string{"node"}

	descNodeInfo = prometheus.NewDesc(
		"kube_node_info",
		"Information about a cluster node.",
		[]string{
			"node",
			"kernel_version",
			"os_image",
			"container_runtime_version",
			"kubelet_version",
			"kubeproxy_version",
			"provider_id",
		}, nil,
	)

	descNodeLabels = prometheus.NewDesc(
		descNodeLabelsName,
		descNodeLabelsHelp,
		descNodeLabelsDefaultLabels, nil,
	)

	descNodeSpecUnschedulable = prometheus.NewDesc(
		"kube_node_spec_unschedulable",
		"Whether a node can schedule new pods.",
		[]string{"node"}, nil,
	)

	descNodeStatusCondition = prometheus.NewDesc(
		"kube_node_status_condition",
		"The condition of a cluster node.",
		[]string{"node", "condition", "status"}, nil,
	)

	descNodeStatusPhase = prometheus.NewDesc(
		"kube_node_status_phase",
		"The phase the node is currently in.",
		[]string{"node", "phase"}, nil,
	)

	descNodeStatusCapacityPods = prometheus.NewDesc(
		"kube_node_status_capacity_pods",
		"The total pod resources of the node.",
		[]string{"node"}, nil,
	)
	descNodeStatusCapacityCPU = prometheus.NewDesc(
		"kube_node_status_capacity_cpu_cores",
		"The total CPU resources of the node.",
		[]string{"node"}, nil,
	)
	descNodeStatusCapacityMemory = prometheus.NewDesc(
		"kube_node_status_capacity_memory_bytes",
		"The total memory resources of the node.",
		[]string{"node"}, nil,
	)

	descNodeStatusAllocatablePods = prometheus.NewDesc(
		"kube_node_status_allocatable_pods",
		"The pod resources of a node that are available for scheduling.",
		[]string{"node"}, nil,
	)
	descNodeStatusAllocatableCPU = prometheus.NewDesc(
		"kube_node_status_allocatable_cpu_cores",
		"The CPU resources of a node that are available for scheduling.",
		[]string{"node"}, nil,
	)
	descNodeStatusAllocatableMemory = prometheus.NewDesc(
		"kube_node_status_allocatable_memory_bytes",
		"The memory resources of a node that are available for scheduling.",
		[]string{"node"}, nil,
	)
)

type NodeLister func() (v1.NodeList, error)

func (l NodeLister) List() (v1.NodeList, error) {
	return l()
}

func RegisterNodeCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface) {
	client := kubeClient.CoreV1().RESTClient()
	nlw := cache.NewListWatchFromClient(client, "nodes", api.NamespaceAll, nil)
	ninf := cache.NewSharedInformer(nlw, &v1.Node{}, resyncPeriod)

	nodeLister := NodeLister(func() (machines v1.NodeList, err error) {
		for _, m := range ninf.GetStore().List() {
			machines.Items = append(machines.Items, *(m.(*v1.Node)))
		}
		return machines, nil
	})

	registry.MustRegister(&nodeCollector{store: nodeLister})
	go ninf.Run(context.Background().Done())
}

type nodeStore interface {
	List() (v1.NodeList, error)
}

// nodeCollector collects metrics about all nodes in the cluster.
type nodeCollector struct {
	store nodeStore
}

// Describe implements the prometheus.Collector interface.
func (nc *nodeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descNodeInfo
	ch <- descNodeLabels
	ch <- descNodeSpecUnschedulable
	ch <- descNodeStatusCondition
	ch <- descNodeStatusPhase
	ch <- descNodeStatusCapacityCPU
	ch <- descNodeStatusCapacityMemory
	ch <- descNodeStatusCapacityPods
	ch <- descNodeStatusAllocatableCPU
	ch <- descNodeStatusAllocatableMemory
	ch <- descNodeStatusAllocatablePods
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

func nodeLabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descNodeLabelsName,
		descNodeLabelsHelp,
		append(descNodeLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func (nc *nodeCollector) collectNode(ch chan<- prometheus.Metric, n v1.Node) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{n.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	// NOTE: the instrumentation API requires providing label values in order of declaration
	// in the metric descriptor. Be careful when making modifications.
	addGauge(descNodeInfo, 1,
		n.Status.NodeInfo.KernelVersion,
		n.Status.NodeInfo.OSImage,
		n.Status.NodeInfo.ContainerRuntimeVersion,
		n.Status.NodeInfo.KubeletVersion,
		n.Status.NodeInfo.KubeProxyVersion,
		n.Spec.ProviderID,
	)
	labelKeys, labelValues := kubeLabelsToPrometheusLabels(n.Labels)
	addGauge(nodeLabelsDesc(labelKeys), 1, labelValues...)

	addGauge(descNodeSpecUnschedulable, boolFloat64(n.Spec.Unschedulable))

	// Collect node conditions and while default to false.
	for _, c := range n.Status.Conditions {
		// This all-in-one metric family contains all conditions for extensibility.
		// Third party plugin may report customized condition for cluster node
		// (e.g. node-problem-detector), and Kubernetes may add new core
		// conditions in future.
		addConditionMetrics(ch, descNodeStatusCondition, c.Status, n.Name, string(c.Type))
	}

	// Set current phase to 1, others to 0 if it is set.
	if p := n.Status.Phase; p != "" {
		addGauge(descNodeStatusPhase, boolFloat64(p == v1.NodePending), string(v1.NodePending))
		addGauge(descNodeStatusPhase, boolFloat64(p == v1.NodeRunning), string(v1.NodeRunning))
		addGauge(descNodeStatusPhase, boolFloat64(p == v1.NodeTerminated), string(v1.NodeTerminated))
	}

	// Add capacity and allocatable resources if they are set.
	addResource := func(d *prometheus.Desc, res v1.ResourceList, n v1.ResourceName) {
		if v, ok := res[n]; ok {
			addGauge(d, float64(v.MilliValue())/1000)
		}
	}
	addResource(descNodeStatusCapacityCPU, n.Status.Capacity, v1.ResourceCPU)
	addResource(descNodeStatusCapacityMemory, n.Status.Capacity, v1.ResourceMemory)
	addResource(descNodeStatusCapacityPods, n.Status.Capacity, v1.ResourcePods)

	addResource(descNodeStatusAllocatableCPU, n.Status.Allocatable, v1.ResourceCPU)
	addResource(descNodeStatusAllocatableMemory, n.Status.Allocatable, v1.ResourceMemory)
	addResource(descNodeStatusAllocatablePods, n.Status.Allocatable, v1.ResourcePods)
}

// addConditionMetrics generates one metric for each possible node condition
// status. For this function to work properly, the last label in the metric
// description must be the condition.
func addConditionMetrics(ch chan<- prometheus.Metric, desc *prometheus.Desc, cs v1.ConditionStatus, lv ...string) {
	ch <- prometheus.MustNewConstMetric(
		desc, prometheus.GaugeValue, boolFloat64(cs == v1.ConditionTrue),
		append(lv, "true")...,
	)
	ch <- prometheus.MustNewConstMetric(
		desc, prometheus.GaugeValue, boolFloat64(cs == v1.ConditionFalse),
		append(lv, "false")...,
	)
	ch <- prometheus.MustNewConstMetric(
		desc, prometheus.GaugeValue, boolFloat64(cs == v1.ConditionUnknown),
		append(lv, "unknown")...,
	)
}

func boolFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
