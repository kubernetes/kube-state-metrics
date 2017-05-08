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
	"testing"

	"k8s.io/client-go/pkg/api/resource"
	"k8s.io/client-go/pkg/api/v1"
)

type mockNodeStore struct {
	list func() (v1.NodeList, error)
}

func (ns mockNodeStore) List() (v1.NodeList, error) {
	return ns.list()
}

func TestNodeCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_node_info Information about a cluster node.
		# TYPE kube_node_info gauge
		# HELP kube_node_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_node_labels gauge
		# HELP kube_node_spec_unschedulable Whether a node can schedule new pods.
		# TYPE kube_node_spec_unschedulable gauge
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
		# TYPE kube_node_status_allocatable_pods gauge
		# HELP kube_node_status_allocatable_pods The pod resources of a node that are available for scheduling.
		# TYPE kube_node_status_allocatable_cpu_cores gauge
		# HELP kube_node_status_allocatable_cpu_cores The CPU resources of a node that are available for scheduling.
		# TYPE kube_node_status_allocatable_memory_bytes gauge
		# HELP kube_node_status_allocatable_memory_bytes The memory resources of a node that are available for scheduling.
		# HELP kube_node_status_memory_pressure Whether the kubelet is under pressure due to insufficient available memory.
		# TYPE kube_node_status_memory_pressure gauge
		# HELP kube_node_status_disk_pressure Whether the kubelet is under pressure due to insufficient available disk.
		# TYPE kube_node_status_disk_pressure gauge
		# HELP kube_node_status_network_unavailable Whether the network is correctly configured for the node.
		# TYPE kube_node_status_network_unavailable gauge
	`
	cases := []struct {
		nodes   []v1.Node
		metrics []string // which metrics should be checked
		want    string
	}{
		// Verify populating base metrics and that metrics for unset fields are skipped.
		{
			nodes: []v1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: v1.NodeStatus{
						NodeInfo: v1.NodeSystemInfo{
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
				kube_node_labels{node="127.0.0.1"} 1
				kube_node_spec_unschedulable{node="127.0.0.1"} 0
			`,
		},
		// Verify resource metrics.
		{
			nodes: []v1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.1",
						Labels: map[string]string{
							"type": "master",
						},
					},
					Spec: v1.NodeSpec{
						Unschedulable: true,
					},
					Status: v1.NodeStatus{
						NodeInfo: v1.NodeSystemInfo{
							KernelVersion:           "kernel",
							KubeletVersion:          "kubelet",
							KubeProxyVersion:        "kubeproxy",
							OSImage:                 "osimage",
							ContainerRuntimeVersion: "rkt",
						},
						Capacity: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("4.3"),
							v1.ResourceMemory: resource.MustParse("2G"),
							v1.ResourcePods:   resource.MustParse("1000"),
						},
						Allocatable: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("3"),
							v1.ResourceMemory: resource.MustParse("1G"),
							v1.ResourcePods:   resource.MustParse("555"),
						},
					},
				},
			},
			want: metadata + `
				kube_node_info{container_runtime_version="rkt",kernel_version="kernel",kubelet_version="kubelet",kubeproxy_version="kubeproxy",node="127.0.0.1",os_image="osimage"} 1
				kube_node_labels{label_type="master",node="127.0.0.1"} 1
				kube_node_spec_unschedulable{node="127.0.0.1"} 1
				kube_node_status_capacity_cpu_cores{node="127.0.0.1"} 4.3
				kube_node_status_capacity_memory_bytes{node="127.0.0.1"} 2e9
				kube_node_status_capacity_pods{node="127.0.0.1"} 1000
				kube_node_status_allocatable_cpu_cores{node="127.0.0.1"} 3
				kube_node_status_allocatable_memory_bytes{node="127.0.0.1"} 1e9
				kube_node_status_allocatable_pods{node="127.0.0.1"} 555
			`,
		},
		// Verify condition enumerations.
		{
			nodes: []v1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeReady, Status: v1.ConditionTrue},
						},
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.2",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeReady, Status: v1.ConditionUnknown},
						},
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.3",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeReady, Status: v1.ConditionFalse},
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
			nodes: []v1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: v1.NodeStatus{
						Phase: v1.NodeRunning,
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.2",
					},
					Status: v1.NodeStatus{
						Phase: v1.NodePending,
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.3",
					},
					Status: v1.NodeStatus{
						Phase: v1.NodeTerminated,
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
		// Verify MemoryPressure
		{
			nodes: []v1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeMemoryPressure, Status: v1.ConditionTrue},
						},
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.2",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeMemoryPressure, Status: v1.ConditionUnknown},
						},
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.3",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeMemoryPressure, Status: v1.ConditionFalse},
						},
					},
				},
			},
			want: metadata + `
				kube_node_status_memory_pressure{node="127.0.0.1",condition="true"} 1
				kube_node_status_memory_pressure{node="127.0.0.1",condition="false"} 0
				kube_node_status_memory_pressure{node="127.0.0.1",condition="unknown"} 0
				kube_node_status_memory_pressure{node="127.0.0.2",condition="true"} 0
				kube_node_status_memory_pressure{node="127.0.0.2",condition="false"} 0
				kube_node_status_memory_pressure{node="127.0.0.2",condition="unknown"} 1
				kube_node_status_memory_pressure{node="127.0.0.3",condition="true"} 0
				kube_node_status_memory_pressure{node="127.0.0.3",condition="false"} 1
				kube_node_status_memory_pressure{node="127.0.0.3",condition="unknown"} 0
			`,
			metrics: []string{"kube_node_status_memory_pressure"},
		},
		// Verify DiskPressure
		{
			nodes: []v1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeDiskPressure, Status: v1.ConditionTrue},
						},
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.2",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeDiskPressure, Status: v1.ConditionUnknown},
						},
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.3",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeDiskPressure, Status: v1.ConditionFalse},
						},
					},
				},
			},
			want: metadata + `
				kube_node_status_disk_pressure{node="127.0.0.1",condition="true"} 1
				kube_node_status_disk_pressure{node="127.0.0.1",condition="false"} 0
				kube_node_status_disk_pressure{node="127.0.0.1",condition="unknown"} 0
				kube_node_status_disk_pressure{node="127.0.0.2",condition="true"} 0
				kube_node_status_disk_pressure{node="127.0.0.2",condition="false"} 0
				kube_node_status_disk_pressure{node="127.0.0.2",condition="unknown"} 1
				kube_node_status_disk_pressure{node="127.0.0.3",condition="true"} 0
				kube_node_status_disk_pressure{node="127.0.0.3",condition="false"} 1
				kube_node_status_disk_pressure{node="127.0.0.3",condition="unknown"} 0
			`,
			metrics: []string{"kube_node_status_disk_pressure"},
		},
		// Verify NetworkUnavailable
		{
			nodes: []v1.Node{
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeNetworkUnavailable, Status: v1.ConditionTrue},
						},
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.2",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeNetworkUnavailable, Status: v1.ConditionUnknown},
						},
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Name: "127.0.0.3",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeNetworkUnavailable, Status: v1.ConditionFalse},
						},
					},
				},
			},
			want: metadata + `
				kube_node_status_network_unavailable{node="127.0.0.1",condition="true"} 1
				kube_node_status_network_unavailable{node="127.0.0.1",condition="false"} 0
				kube_node_status_network_unavailable{node="127.0.0.1",condition="unknown"} 0
				kube_node_status_network_unavailable{node="127.0.0.2",condition="true"} 0
				kube_node_status_network_unavailable{node="127.0.0.2",condition="false"} 0
				kube_node_status_network_unavailable{node="127.0.0.2",condition="unknown"} 1
				kube_node_status_network_unavailable{node="127.0.0.3",condition="true"} 0
				kube_node_status_network_unavailable{node="127.0.0.3",condition="false"} 1
				kube_node_status_network_unavailable{node="127.0.0.3",condition="unknown"} 0
			`,
			metrics: []string{"kube_node_status_network_unavailable"},
		},
	}
	for _, c := range cases {
		dc := &nodeCollector{
			store: &mockNodeStore{
				list: func() (v1.NodeList, error) {
					return v1.NodeList{Items: c.nodes}, nil
				},
			},
		}
		if err := gatherAndCompare(dc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
