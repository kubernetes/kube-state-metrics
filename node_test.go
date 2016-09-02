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
		# HELP node_info Information about a cluster node.
		# TYPE node_info gauge
		# HELP node_status_ready The ready status of a cluster node.
		# TYPE node_status_ready gauge
	`
	cases := []struct {
		nodes []api.Node
		want  string
	}{
		// Verify populating of node_info metric.
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
				node_info{container_runtime_version="rkt",kernel_version="kernel",kubelet_version="kubelet",kubeproxy_version="kubeproxy",node="127.0.0.1",os_image="osimage"} 1
			`,
		},
		// Verify condition mappings to 1, 0, and NaN.
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
				node_status_ready{node="127.0.0.1",condition="true"} 1
				node_status_ready{node="127.0.0.1",condition="false"} 0
				node_status_ready{node="127.0.0.1",condition="unknown"} 0
				node_status_ready{node="127.0.0.2",condition="true"} 0
				node_status_ready{node="127.0.0.2",condition="false"} 0
				node_status_ready{node="127.0.0.2",condition="unknown"} 1
				node_status_ready{node="127.0.0.3",condition="true"} 0
				node_status_ready{node="127.0.0.3",condition="false"} 1
				node_status_ready{node="127.0.0.3",condition="unknown"} 0
				node_info{container_runtime_version="",kernel_version="",kubelet_version="",kubeproxy_version="",node="127.0.0.1",os_image=""} 1
				node_info{container_runtime_version="",kernel_version="",kubelet_version="",kubeproxy_version="",node="127.0.0.2",os_image=""} 1
				node_info{container_runtime_version="",kernel_version="",kubelet_version="",kubeproxy_version="",node="127.0.0.3",os_image=""} 1
			`,
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
		if err := gatherAndCompare(dc, c.want); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
