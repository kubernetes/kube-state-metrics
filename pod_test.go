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
	"k8s.io/kubernetes/pkg/labels"
)

type mockPodStore struct {
	f func() ([]*api.Pod, error)
}

func (ds mockPodStore) List(selector labels.Selector) (pods []*api.Pod, err error) {
	return ds.f()
}

func TestPodCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP pod_container_restarts The number of container restarts per container.
		# TYPE pod_container_restarts counter
	`
	cases := []struct {
		pods []*api.Pod
		want string
	}{
		{
			pods: []*api.Pod{
				{
					ObjectMeta: api.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
					Status: api.PodStatus{
						ContainerStatuses: []api.ContainerStatus{
							api.ContainerStatus{
								Name:         "container1",
								RestartCount: 0,
							},
						},
					},
				}, {
					ObjectMeta: api.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
					Status: api.PodStatus{
						ContainerStatuses: []api.ContainerStatus{
							api.ContainerStatus{
								Name:         "container2",
								RestartCount: 0,
							},
							api.ContainerStatus{
								Name:         "container3",
								RestartCount: 1,
							},
						},
					},
				},
			},
			want: metadata + `
				pod_container_restarts{container="container1",namespace="ns1",pod="pod1"} 0
				pod_container_restarts{container="container2",namespace="ns2",pod="pod2"} 0
				pod_container_restarts{container="container3",namespace="ns2",pod="pod2"} 1
			`,
		},
	}
	for _, c := range cases {
		pc := &podCollector{
			store: mockPodStore{
				f: func() ([]*api.Pod, error) { return c.pods, nil },
			},
		}
		if err := gatherAndCompare(pc, c.want); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
