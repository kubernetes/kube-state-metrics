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

// In this package, the mentioned metrics need both the node list as well as the
// pod list.

package main

import (
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/api/resource"
	"k8s.io/client-go/1.4/pkg/api/v1"
)

var (
	descNodeCurrentRequestedCpuResources = prometheus.NewDesc(
		"kube_node_current_requested_cpu_cores",
		"The total number of cpu resources requested by all the running pods on one node",
		[]string{"node"}, nil,
	)
	descNodeCurrentRequestedMemoryResources = prometheus.NewDesc(
		"kube_node_current_requested_memory_bytes",
		"The total number of memory resources requested by all the running pods on one node",
		[]string{"node"}, nil,
	)
)

// nodePodCollector collects metrics that depend on both the nodes and pods in the cluster
type nodePodCollector struct {
	ns nodeStore
	ps podStore
}

// Describe implements the prometheus.Collector interface.
func (npc *nodePodCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descNodeCurrentRequestedCpuResources
	ch <- descNodeCurrentRequestedMemoryResources
}

// Collect implements the prometheus.Collector interface.
func (npc *nodePodCollector) Collect(ch chan<- prometheus.Metric) {
	nodes, err := npc.ns.List()
	if err != nil {
		glog.Errorf("listing nodes failed: %s", err)
		return
	}
	pods, err := npc.ps.List()
	if err != nil {
		glog.Errorf("listing pods failed: %s", err)
		return
	}

	for _, n := range nodes.Items {
		npc.collectNode(ch, n, pods)
	}
}

// isRelevantPod looks into the pod phase and podname.
// isRelevantPod returns true if the pod's resources should be accumulated to
// calculate the total used resource for the node with the given name.
func isRelevantPod(p v1.Pod, nodeName string) bool {
	isRunningOnNode := p.Spec.NodeName == nodeName
	isNotSucceeded := p.Status.Phase != v1.PodSucceeded
	isNotFailed := p.Status.Phase != v1.PodFailed

	return isRunningOnNode && isNotSucceeded && isNotFailed
}

func (npc *nodePodCollector) collectNode(ch chan<- prometheus.Metric,
	n v1.Node, pods []v1.Pod) {

	reqCpu := resource.Quantity{}
	reqMem := resource.Quantity{}
	nn := n.Name
	for _, p := range pods {
		// TODO: There should be a faster way to get the reduced set of
		// relevant pods with List(ap.ListOptions{FieldSelector})
		// Here we are iterating over all nodes every time and skipping
		// uninteresting ones.
		if isRelevantPod(p, nn) {
			var apiP api.Pod
			v1.Convert_v1_Pod_To_api_Pod(&p, &apiP, nil)
			req, _, err := api.PodRequestsAndLimits(&apiP)
			if err != nil {
				glog.Errorf("Getting resources for pod: %v failed: %s", p, err)
				return
			}
			cpu := req[api.ResourceCPU]
			mem := req[api.ResourceMemory]
			reqCpu.Add(cpu)
			reqMem.Add(mem)
		}
	}

	addRequestedResource := func(desc *prometheus.Desc, v float64) {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, n.Name)
	}
	addRequestedResource(descNodeCurrentRequestedCpuResources, float64(reqCpu.Value()))
	addRequestedResource(descNodeCurrentRequestedMemoryResources, float64(reqMem.Value()))
}
