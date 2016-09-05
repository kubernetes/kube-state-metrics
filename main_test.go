/*
Copyright 2015 The Kubernetes Authors All rights reserved.

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
	"reflect"
	"testing"

	"k8s.io/kubernetes/pkg/api"
)

type metricsRegistryMock struct {
	readyNodes        float64
	unreadyNodes      float64
	containerRestarts map[string]float64
}

func (mr *metricsRegistryMock) setReadyNodes(count float64) {
	mr.readyNodes = count
}

func (mr *metricsRegistryMock) setUnreadyNodes(count float64) {
	mr.unreadyNodes = count
}

func (mr *metricsRegistryMock) setContainerRestarts(name, namespace, podName string, count float64) {
	if mr.containerRestarts == nil {
		mr.containerRestarts = map[string]float64{}
	}
	mr.containerRestarts[name+"-"+podName+"-"+namespace] = count
}

func getNode(condition api.ConditionStatus) api.Node {
	return api.Node{
		Status: api.NodeStatus{
			Conditions: []api.NodeCondition{
				{
					Type:   api.NodeReady,
					Status: condition,
				},
			},
		},
	}
}

// This pod will have two containers - you can confiure the restart count on both.
func getPod(name, namespace string, containerStatuses []api.ContainerStatus) *api.Pod {
	return &api.Pod{
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: api.PodStatus{
			ContainerStatuses: containerStatuses,
		},
	}
}

func getContainerStatus(name string, restartCount int) api.ContainerStatus {
	return api.ContainerStatus{
		Name:         name,
		RestartCount: int32(restartCount),
	}
}

func TestRegisterNodeMetrics(t *testing.T) {
	cases := []struct {
		desc     string
		nodes    []api.Node
		registry *metricsRegistryMock
	}{
		{
			desc: "three ready nodes, one unready node, one unknown node",
			nodes: []api.Node{
				getNode(api.ConditionTrue),
				getNode(api.ConditionTrue),
				getNode(api.ConditionTrue),
				getNode(api.ConditionFalse),
				getNode(api.ConditionUnknown),
			},
			registry: &metricsRegistryMock{
				readyNodes:   3,
				unreadyNodes: 2,
			},
		},
	}

	for _, c := range cases {
		r := &metricsRegistryMock{}
		registerNodeMetrics(r, c.nodes)
		if !reflect.DeepEqual(r, c.registry) {
			t.Errorf("error in case \"%s\": actual %v does not equal expected %v", c.desc, r, c.registry)
		}
	}
}
