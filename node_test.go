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
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
)

type mockNodeStore struct {
	list func() (api.NodeList, error)
}

func (ns mockNodeStore) List() (api.NodeList, error) {
	return ns.list()
}

func TestNodeCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_node_info Information about a cluster node.
		# TYPE kube_node_info gauge
		# HELP kube_node_status_ready The ready status of a cluster node.
		# TYPE kube_node_status_ready gauge
		# TYPE kube_node_status_phase gauge
		# HELP kube_node_status_phase The phase the node is currently in.
		# TYPE kube_node_status_capacity_pods gauge
		# HELP kube_node_status_capacity_pods The total pod resources of the node.
		# TYPE kube_node_status_capacity_cpu_cores gauge
		# HELP kube_node_status_capacity_cpu_cores The total CPU resources of the node.
		# TYPE kube_node_status_capacity_memory_bytes gauge
		# HELP kube_node_status_capacity_memory_bytes The total memory resources of the node.
		# TYPE kube_node_status_allocateable_pods gauge
		# HELP kube_node_status_allocateable_pods The pod resources of a node that are available for scheduling.
		# TYPE kube_node_status_allocateable_cpu_cores gauge
		# HELP kube_node_status_allocateable_cpu_cores The CPU resources of a node that are available for scheduling.
		# TYPE kube_node_status_allocateable_memory_bytes gauge
		# HELP kube_node_status_allocateable_memory_bytes The memory resources of a node that are available for scheduling.
	`
	cases := []struct {
		nodes   []api.Node
		metrics []string // which metrics should be checked
		want    string
	}{
		// Verify populating base metrics and that metrics for unset fields are skipped.
		{
			nodes: []api.Node{
				{
					ObjectMeta: api.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: api.NodeStatus{
						NodeInfo: api.NodeSystemInfo{
							KernelVersion:           "kernel",
							KubeletVersion:          "kubelet",
							KubeProxyVersion:        "kubeproxy",
							OSImage:                 "osimage",
							ContainerRuntimeVersion: "rkt",
						},
					},
				},
			},
			want: metadata + `
				kube_node_info{container_runtime_version="rkt",kernel_version="kernel",kubelet_version="kubelet",kubeproxy_version="kubeproxy",node="127.0.0.1",os_image="osimage"} 1
			`,
		},
		// Verify resource metrics.
		{
			nodes: []api.Node{
				{
					ObjectMeta: api.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: api.NodeStatus{
						NodeInfo: api.NodeSystemInfo{
							KernelVersion:           "kernel",
							KubeletVersion:          "kubelet",
							KubeProxyVersion:        "kubeproxy",
							OSImage:                 "osimage",
							ContainerRuntimeVersion: "rkt",
						},
						Capacity: api.ResourceList{
							api.ResourceCPU:    resource.MustParse("4"),
							api.ResourceMemory: resource.MustParse("2G"),
							api.ResourcePods:   resource.MustParse("1000"),
						},
						Allocatable: api.ResourceList{
							api.ResourceCPU:    resource.MustParse("3"),
							api.ResourceMemory: resource.MustParse("1G"),
							api.ResourcePods:   resource.MustParse("555"),
						},
					},
				},
			},
			want: metadata + `
				kube_node_info{container_runtime_version="rkt",kernel_version="kernel",kubelet_version="kubelet",kubeproxy_version="kubeproxy",node="127.0.0.1",os_image="osimage"} 1
				kube_node_status_capacity_cpu_cores{node="127.0.0.1"} 4
				kube_node_status_capacity_memory_bytes{node="127.0.0.1"} 2e9
				kube_node_status_capacity_pods{node="127.0.0.1"} 1000
				kube_node_status_allocateable_cpu_cores{node="127.0.0.1"} 3
				kube_node_status_allocateable_memory_bytes{node="127.0.0.1"} 1e9
				kube_node_status_allocateable_pods{node="127.0.0.1"} 555
			`,
		},
		// Verify condition enumerations.
		{
			nodes: []api.Node{
				{
					ObjectMeta: api.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: api.NodeStatus{
						Conditions: []api.NodeCondition{
							{Type: api.NodeReady, Status: api.ConditionTrue},
						},
					},
				},
				{
					ObjectMeta: api.ObjectMeta{
						Name: "127.0.0.2",
					},
					Status: api.NodeStatus{
						Conditions: []api.NodeCondition{
							{Type: api.NodeReady, Status: api.ConditionUnknown},
						},
					},
				},
				{
					ObjectMeta: api.ObjectMeta{
						Name: "127.0.0.3",
					},
					Status: api.NodeStatus{
						Conditions: []api.NodeCondition{
							{Type: api.NodeReady, Status: api.ConditionFalse},
						},
					},
				},
			},
			want: metadata + `
				kube_node_status_ready{node="127.0.0.1",condition="true"} 1
				kube_node_status_ready{node="127.0.0.1",condition="false"} 0
				kube_node_status_ready{node="127.0.0.1",condition="unknown"} 0
				kube_node_status_ready{node="127.0.0.2",condition="true"} 0
				kube_node_status_ready{node="127.0.0.2",condition="false"} 0
				kube_node_status_ready{node="127.0.0.2",condition="unknown"} 1
				kube_node_status_ready{node="127.0.0.3",condition="true"} 0
				kube_node_status_ready{node="127.0.0.3",condition="false"} 1
				kube_node_status_ready{node="127.0.0.3",condition="unknown"} 0
			`,
			metrics: []string{"kube_node_status_ready"},
		},
		// Verify phase enumerations.
		{
			nodes: []api.Node{
				{
					ObjectMeta: api.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: api.NodeStatus{
						Phase: api.NodeRunning,
					},
				},
				{
					ObjectMeta: api.ObjectMeta{
						Name: "127.0.0.2",
					},
					Status: api.NodeStatus{
						Phase: api.NodePending,
					},
				},
				{
					ObjectMeta: api.ObjectMeta{
						Name: "127.0.0.3",
					},
					Status: api.NodeStatus{
						Phase: api.NodeTerminated,
					},
				},
			},
			want: metadata + `
				kube_node_status_phase{node="127.0.0.1",phase="Terminated"} 0
				kube_node_status_phase{node="127.0.0.1",phase="Running"} 1
				kube_node_status_phase{node="127.0.0.1",phase="Pending"} 0
				kube_node_status_phase{node="127.0.0.2",phase="Terminated"} 0
				kube_node_status_phase{node="127.0.0.2",phase="Running"} 0
				kube_node_status_phase{node="127.0.0.2",phase="Pending"} 1
				kube_node_status_phase{node="127.0.0.3",phase="Terminated"} 1
				kube_node_status_phase{node="127.0.0.3",phase="Running"} 0
				kube_node_status_phase{node="127.0.0.3",phase="Pending"} 0
			`,
			metrics: []string{"kube_node_status_phase"},
		},
	}
	for _, c := range cases {
		dc := &nodeCollector{
			store: &mockNodeStore{
				list: func() (api.NodeList, error) {
					return api.NodeList{Items: c.nodes}, nil
				},
			},
		}
		if err := gatherAndCompare(dc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
