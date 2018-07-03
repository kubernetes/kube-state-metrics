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
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/pkg/collectors/testutils"
	"k8s.io/kube-state-metrics/pkg/options"
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
		# HELP kube_node_created Unix creation timestamp
		# TYPE kube_node_created gauge
		# HELP kube_node_info Information about a cluster node.
		# TYPE kube_node_info gauge
		# HELP kube_node_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_node_labels gauge
		# HELP kube_node_spec_unschedulable Whether a node can schedule new pods.
		# TYPE kube_node_spec_unschedulable gauge
		# HELP kube_node_spec_taint The taint of a cluster node.
		# TYPE kube_node_spec_taint gauge
		# TYPE kube_node_status_phase gauge
		# HELP kube_node_status_phase The phase the node is currently in.
		# TYPE kube_node_status_capacity gauge
		# HELP kube_node_status_capacity The capacity for different resources of a node.
		# TYPE kube_node_status_capacity_pods gauge
		# HELP kube_node_status_capacity_pods The total pod resources of the node.
		# TYPE kube_node_status_capacity_cpu_cores gauge
		# HELP kube_node_status_capacity_cpu_cores The total CPU resources of the node.
		# TYPE kube_node_status_capacity_memory_bytes gauge
		# HELP kube_node_status_capacity_memory_bytes The total memory resources of the node.
		# TYPE kube_node_status_allocatable gauge
		# HELP kube_node_status_allocatable The allocatable for different resources of a node that are available for scheduling.
		# TYPE kube_node_status_allocatable_pods gauge
		# HELP kube_node_status_allocatable_pods The pod resources of a node that are available for scheduling.
		# TYPE kube_node_status_allocatable_cpu_cores gauge
		# HELP kube_node_status_allocatable_cpu_cores The CPU resources of a node that are available for scheduling.
		# TYPE kube_node_status_allocatable_memory_bytes gauge
		# HELP kube_node_status_allocatable_memory_bytes The memory resources of a node that are available for scheduling.
		# HELP kube_node_status_condition The condition of a cluster node.
		# TYPE kube_node_status_condition gauge
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
					ObjectMeta: metav1.ObjectMeta{
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
					Spec: v1.NodeSpec{
						ProviderID: "provider://i-uniqueid",
					},
				},
			},
			want: metadata + `
				kube_node_info{container_runtime_version="rkt",kernel_version="kernel",kubelet_version="kubelet",kubeproxy_version="kubeproxy",node="127.0.0.1",os_image="osimage",provider_id="provider://i-uniqueid"} 1
				kube_node_labels{node="127.0.0.1"} 1
				kube_node_spec_unschedulable{node="127.0.0.1"} 0
			`,
		},
		// Verify resource metrics.
		{
			nodes: []v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "127.0.0.1",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
						Labels: map[string]string{
							"type": "master",
						},
					},
					Spec: v1.NodeSpec{
						Unschedulable: true,
						ProviderID:    "provider://i-randomidentifier",
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
							v1.ResourceCPU:                    resource.MustParse("4.3"),
							v1.ResourceMemory:                 resource.MustParse("2G"),
							v1.ResourcePods:                   resource.MustParse("1000"),
							v1.ResourceStorage:                resource.MustParse("3G"),
							v1.ResourceEphemeralStorage:       resource.MustParse("4G"),
							v1.ResourceName("nvidia.com/gpu"): resource.MustParse("4"),
						},
						Allocatable: v1.ResourceList{
							v1.ResourceCPU:                    resource.MustParse("3"),
							v1.ResourceMemory:                 resource.MustParse("1G"),
							v1.ResourcePods:                   resource.MustParse("555"),
							v1.ResourceStorage:                resource.MustParse("2G"),
							v1.ResourceEphemeralStorage:       resource.MustParse("3G"),
							v1.ResourceName("nvidia.com/gpu"): resource.MustParse("1"),
						},
					},
				},
			},
			want: metadata + `
				kube_node_created{node="127.0.0.1"} 1.5e+09
				kube_node_info{container_runtime_version="rkt",kernel_version="kernel",kubelet_version="kubelet",kubeproxy_version="kubeproxy",node="127.0.0.1",os_image="osimage",provider_id="provider://i-randomidentifier"} 1
				kube_node_labels{label_type="master",node="127.0.0.1"} 1
				kube_node_spec_unschedulable{node="127.0.0.1"} 1
				kube_node_status_capacity{node="127.0.0.1",resource="cpu",unit="core"} 4.3
				kube_node_status_capacity{node="127.0.0.1",resource="memory",unit="byte"}2e9
				kube_node_status_capacity{node="127.0.0.1",resource="pods",unit="integer"} 1000
				kube_node_status_capacity{node="127.0.0.1",resource="nvidia_com_gpu",unit="integer"} 4
				kube_node_status_capacity{node="127.0.0.1",resource="storage",unit="byte"} 3e9
				kube_node_status_capacity{node="127.0.0.1",resource="ephemeral_storage",unit="byte"} 4e9
				kube_node_status_capacity_cpu_cores{node="127.0.0.1"} 4.3
				kube_node_status_capacity_memory_bytes{node="127.0.0.1"} 2e9
				kube_node_status_capacity_pods{node="127.0.0.1"} 1000
				kube_node_status_allocatable{node="127.0.0.1",resource="cpu",unit="core"} 3
				kube_node_status_allocatable{node="127.0.0.1",resource="memory",unit="byte"} 1e9
				kube_node_status_allocatable{node="127.0.0.1",resource="pods",unit="integer"} 555
				kube_node_status_allocatable{node="127.0.0.1",resource="storage",unit="byte"} 2e9
				kube_node_status_allocatable{node="127.0.0.1",resource="nvidia_com_gpu",unit="integer"} 1
				kube_node_status_allocatable{node="127.0.0.1",resource="ephemeral_storage",unit="byte"} 3e9
				kube_node_status_allocatable_cpu_cores{node="127.0.0.1"} 3
				kube_node_status_allocatable_memory_bytes{node="127.0.0.1"} 1e9
				kube_node_status_allocatable_pods{node="127.0.0.1"} 555
			`,
		},
		// Verify phase enumerations.
		{
			nodes: []v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: v1.NodeStatus{
						Phase: v1.NodeRunning,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "127.0.0.2",
					},
					Status: v1.NodeStatus{
						Phase: v1.NodePending,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
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
		// Verify StatusCondition
		{
			nodes: []v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "127.0.0.1",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeNetworkUnavailable, Status: v1.ConditionTrue},
							{Type: v1.NodeReady, Status: v1.ConditionTrue},
							{Type: v1.NodeConditionType("CustomizedType"), Status: v1.ConditionTrue},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "127.0.0.2",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeNetworkUnavailable, Status: v1.ConditionUnknown},
							{Type: v1.NodeReady, Status: v1.ConditionUnknown},
							{Type: v1.NodeConditionType("CustomizedType"), Status: v1.ConditionUnknown},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "127.0.0.3",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeNetworkUnavailable, Status: v1.ConditionFalse},
							{Type: v1.NodeReady, Status: v1.ConditionFalse},
							{Type: v1.NodeConditionType("CustomizedType"), Status: v1.ConditionFalse},
						},
					},
				},
			},
			want: metadata + `
				kube_node_status_condition{node="127.0.0.1",condition="NetworkUnavailable",status="true"} 1
				kube_node_status_condition{node="127.0.0.1",condition="NetworkUnavailable",status="false"} 0
				kube_node_status_condition{node="127.0.0.1",condition="NetworkUnavailable",status="unknown"} 0
				kube_node_status_condition{node="127.0.0.2",condition="NetworkUnavailable",status="true"} 0
				kube_node_status_condition{node="127.0.0.2",condition="NetworkUnavailable",status="false"} 0
				kube_node_status_condition{node="127.0.0.2",condition="NetworkUnavailable",status="unknown"} 1
				kube_node_status_condition{node="127.0.0.3",condition="NetworkUnavailable",status="true"} 0
				kube_node_status_condition{node="127.0.0.3",condition="NetworkUnavailable",status="false"} 1
				kube_node_status_condition{node="127.0.0.3",condition="NetworkUnavailable",status="unknown"} 0
				kube_node_status_condition{node="127.0.0.1",condition="Ready",status="true"} 1
				kube_node_status_condition{node="127.0.0.1",condition="Ready",status="false"} 0
				kube_node_status_condition{node="127.0.0.1",condition="Ready",status="unknown"} 0
				kube_node_status_condition{node="127.0.0.2",condition="Ready",status="true"} 0
				kube_node_status_condition{node="127.0.0.2",condition="Ready",status="false"} 0
				kube_node_status_condition{node="127.0.0.2",condition="Ready",status="unknown"} 1
				kube_node_status_condition{node="127.0.0.3",condition="Ready",status="true"} 0
				kube_node_status_condition{node="127.0.0.3",condition="Ready",status="false"} 1
				kube_node_status_condition{node="127.0.0.3",condition="Ready",status="unknown"} 0
				kube_node_status_condition{node="127.0.0.1",condition="CustomizedType",status="true"} 1
				kube_node_status_condition{node="127.0.0.1",condition="CustomizedType",status="false"} 0
				kube_node_status_condition{node="127.0.0.1",condition="CustomizedType",status="unknown"} 0
				kube_node_status_condition{node="127.0.0.2",condition="CustomizedType",status="true"} 0
				kube_node_status_condition{node="127.0.0.2",condition="CustomizedType",status="false"} 0
				kube_node_status_condition{node="127.0.0.2",condition="CustomizedType",status="unknown"} 1
				kube_node_status_condition{node="127.0.0.3",condition="CustomizedType",status="true"} 0
				kube_node_status_condition{node="127.0.0.3",condition="CustomizedType",status="false"} 1
				kube_node_status_condition{node="127.0.0.3",condition="CustomizedType",status="unknown"} 0
			`,
			metrics: []string{"kube_node_status_condition"},
		},
		// Verify SpecTaints
		{
			nodes: []v1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "127.0.0.1",
					},
					Spec: v1.NodeSpec{
						Taints: []v1.Taint{
							{Key: "node.kubernetes.io/memory-pressure", Value: "true", Effect: v1.TaintEffectPreferNoSchedule},
							{Key: "Accelerated", Value: "gpu", Effect: v1.TaintEffectPreferNoSchedule},
							{Key: "Dedicated", Effect: v1.TaintEffectPreferNoSchedule},
						},
					},
				},
			},
			want: metadata + `
				kube_node_spec_taint{effect="PreferNoSchedule",key="Dedicated",node="127.0.0.1",value=""} 1
				kube_node_spec_taint{effect="PreferNoSchedule",key="Accelerated",node="127.0.0.1",value="gpu"} 1
				kube_node_spec_taint{effect="PreferNoSchedule",key="node.kubernetes.io/memory-pressure",node="127.0.0.1",value="true"} 1
			`,
			metrics: []string{"kube_node_spec_taint"},
		},
	}
	for _, c := range cases {
		dc := &nodeCollector{
			store: &mockNodeStore{
				list: func() (v1.NodeList, error) {
					return v1.NodeList{Items: c.nodes}, nil
				},
			},
			opts: &options.Options{},
		}
		if err := testutils.GatherAndCompare(dc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
