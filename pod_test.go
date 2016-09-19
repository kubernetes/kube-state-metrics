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

	"k8s.io/client-go/1.4/pkg/api/v1"
)

type mockPodStore struct {
	f func() ([]v1.Pod, error)
}

func (ds mockPodStore) List() (pods []v1.Pod, err error) {
	return ds.f()
}

func TestPodCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_pod_container_info Information about a container in a pod.
		# TYPE kube_pod_container_info gauge
		# HELP kube_pod_container_status_ready Describes whether the containers readiness check succeeded.
		# TYPE kube_pod_container_status_ready gauge
		# HELP kube_pod_container_status_restarts The number of container restarts per container.
		# TYPE kube_pod_container_status_restarts counter
		# HELP kube_pod_container_status_running Describes whether the container is currently in running state.
		# TYPE kube_pod_container_status_running gauge
		# HELP kube_pod_container_status_terminated Describes whether the container is currently in terminated state.
		# TYPE kube_pod_container_status_terminated gauge
		# HELP kube_pod_container_status_waiting Describes whether the container is currently in waiting state.
		# TYPE kube_pod_container_status_waiting gauge
		# HELP kube_pod_info Information about pod.
		# TYPE kube_pod_info gauge
		# HELP kube_pod_status_phase The pods current phase.
		# TYPE kube_pod_status_phase gauge
		# HELP kube_pod_status_ready Describes whether the pod is ready to serve requests.
		# TYPE kube_pod_status_ready gauge
		# HELP kube_pod_status_scheduled Describes the status of the scheduling process for the pod.
		# TYPE kube_pod_status_scheduled gauge
	`
	cases := []struct {
		pods    []v1.Pod
		metrics []string
		want    string
	}{
		{
			pods: []v1.Pod{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
					Status: v1.PodStatus{
						ContainerStatuses: []v1.ContainerStatus{
							v1.ContainerStatus{
								Name:        "container1",
								Image:       "gcr.io/google_containers/hyperkube1",
								ImageID:     "docker://sha256:aaa",
								ContainerID: "docker://ab123",
							},
						},
					},
				}, {
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
					Status: v1.PodStatus{
						ContainerStatuses: []v1.ContainerStatus{
							v1.ContainerStatus{
								Name:        "container2",
								Image:       "gcr.io/google_containers/hyperkube2",
								ImageID:     "docker://sha256:bbb",
								ContainerID: "docker://cd456",
							},
							v1.ContainerStatus{
								Name:        "container3",
								Image:       "gcr.io/google_containers/hyperkube3",
								ImageID:     "docker://sha256:ccc",
								ContainerID: "docker://ef789",
							},
						},
					},
				},
			},
			want: metadata + `
				kube_pod_container_info{container="container1",container_id="docker://ab123",image="gcr.io/google_containers/hyperkube1",image_id="docker://sha256:aaa",namespace="ns1",pod="pod1"} 1
				kube_pod_container_info{container="container2",container_id="docker://cd456",image="gcr.io/google_containers/hyperkube2",image_id="docker://sha256:bbb",namespace="ns2",pod="pod2"} 1
				kube_pod_container_info{container="container3",container_id="docker://ef789",image="gcr.io/google_containers/hyperkube3",image_id="docker://sha256:ccc",namespace="ns2",pod="pod2"} 1
				`,
			metrics: []string{"kube_pod_container_info"},
		}, {
			pods: []v1.Pod{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
					Status: v1.PodStatus{
						ContainerStatuses: []v1.ContainerStatus{
							v1.ContainerStatus{
								Name:  "container1",
								Ready: true,
							},
						},
					},
				}, {
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
					Status: v1.PodStatus{
						ContainerStatuses: []v1.ContainerStatus{
							v1.ContainerStatus{
								Name:  "container2",
								Ready: true,
							},
							v1.ContainerStatus{
								Name:  "container3",
								Ready: false,
							},
						},
					},
				},
			},
			want: metadata + `
				kube_pod_container_status_ready{container="container1",namespace="ns1",pod="pod1"} 1
				kube_pod_container_status_ready{container="container2",namespace="ns2",pod="pod2"} 1
				kube_pod_container_status_ready{container="container3",namespace="ns2",pod="pod2"} 0
				`,
			metrics: []string{"kube_pod_container_status_ready"},
		}, {
			pods: []v1.Pod{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
					Status: v1.PodStatus{
						ContainerStatuses: []v1.ContainerStatus{
							v1.ContainerStatus{
								Name:         "container1",
								RestartCount: 0,
							},
						},
					},
				}, {
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
					Status: v1.PodStatus{
						ContainerStatuses: []v1.ContainerStatus{
							v1.ContainerStatus{
								Name:         "container2",
								RestartCount: 0,
							},
							v1.ContainerStatus{
								Name:         "container3",
								RestartCount: 1,
							},
						},
					},
				},
			},
			want: metadata + `
				kube_pod_container_status_restarts{container="container1",namespace="ns1",pod="pod1"} 0
				kube_pod_container_status_restarts{container="container2",namespace="ns2",pod="pod2"} 0
				kube_pod_container_status_restarts{container="container3",namespace="ns2",pod="pod2"} 1
				`,
			metrics: []string{"kube_pod_container_status_restarts"},
		}, {
			pods: []v1.Pod{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
					Status: v1.PodStatus{
						ContainerStatuses: []v1.ContainerStatus{
							v1.ContainerStatus{
								Name: "container1",
								State: v1.ContainerState{
									Running: &v1.ContainerStateRunning{},
								},
							},
						},
					},
				}, {
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
					Status: v1.PodStatus{
						ContainerStatuses: []v1.ContainerStatus{
							v1.ContainerStatus{
								Name: "container2",
								State: v1.ContainerState{
									Terminated: &v1.ContainerStateTerminated{},
								},
							},
							v1.ContainerStatus{
								Name: "container3",
								State: v1.ContainerState{
									Waiting: &v1.ContainerStateWaiting{},
								},
							},
						},
					},
				},
			},
			want: metadata + `
				kube_pod_container_status_running{container="container1",namespace="ns1",pod="pod1"} 1
				kube_pod_container_status_running{container="container2",namespace="ns2",pod="pod2"} 0
				kube_pod_container_status_running{container="container3",namespace="ns2",pod="pod2"} 0
				kube_pod_container_status_terminated{container="container1",namespace="ns1",pod="pod1"} 0
				kube_pod_container_status_terminated{container="container2",namespace="ns2",pod="pod2"} 1
				kube_pod_container_status_terminated{container="container3",namespace="ns2",pod="pod2"} 0
				kube_pod_container_status_waiting{container="container1",namespace="ns1",pod="pod1"} 0
				kube_pod_container_status_waiting{container="container2",namespace="ns2",pod="pod2"} 0
				kube_pod_container_status_waiting{container="container3",namespace="ns2",pod="pod2"} 1
				`,
			metrics: []string{
				"kube_pod_container_status_running",
				"kube_pod_container_status_waiting",
				"kube_pod_container_status_terminated",
			},
		}, {
			pods: []v1.Pod{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
					Status: v1.PodStatus{
						HostIP: "1.1.1.1",
						PodIP:  "1.2.3.4",
					},
				}, {
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
					Status: v1.PodStatus{
						HostIP: "1.1.1.1",
						PodIP:  "2.3.4.5",
					},
				},
			},
			want: metadata + `
				kube_pod_info{host_ip="1.1.1.1",namespace="ns1",pod="pod1",pod_ip="1.2.3.4"} 1
				kube_pod_info{host_ip="1.1.1.1",namespace="ns2",pod="pod2",pod_ip="2.3.4.5"} 1
				`,
			metrics: []string{"kube_pod_info"},
		}, {
			pods: []v1.Pod{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
					Status: v1.PodStatus{
						Phase: "Running",
					},
				}, {
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
					Status: v1.PodStatus{
						Phase: "Pending",
					},
				},
			},
			want: metadata + `
				kube_pod_status_phase{namespace="ns1",phase="Running",pod="pod1"} 1
				kube_pod_status_phase{namespace="ns2",phase="Pending",pod="pod2"} 1
				`,
			metrics: []string{"kube_pod_status_phase"},
		}, {
			pods: []v1.Pod{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
					Status: v1.PodStatus{
						Conditions: []v1.PodCondition{
							v1.PodCondition{
								Type:   v1.PodReady,
								Status: v1.ConditionTrue,
							},
						},
					},
				}, {
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
					Status: v1.PodStatus{
						Conditions: []v1.PodCondition{
							v1.PodCondition{
								Type:   v1.PodReady,
								Status: v1.ConditionFalse,
							},
						},
					},
				},
			},
			want: metadata + `
				kube_pod_status_ready{condition="false",namespace="ns1",pod="pod1"} 0
				kube_pod_status_ready{condition="false",namespace="ns2",pod="pod2"} 1
				kube_pod_status_ready{condition="true",namespace="ns1",pod="pod1"} 1
				kube_pod_status_ready{condition="true",namespace="ns2",pod="pod2"} 0
				kube_pod_status_ready{condition="unknown",namespace="ns1",pod="pod1"} 0
				kube_pod_status_ready{condition="unknown",namespace="ns2",pod="pod2"} 0
			`,
			metrics: []string{"kube_pod_status_ready"},
		}, {
			pods: []v1.Pod{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
					Status: v1.PodStatus{
						Conditions: []v1.PodCondition{
							v1.PodCondition{
								Type:   v1.PodScheduled,
								Status: v1.ConditionTrue,
							},
						},
					},
				}, {
					ObjectMeta: v1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
					Status: v1.PodStatus{
						Conditions: []v1.PodCondition{
							v1.PodCondition{
								Type:   v1.PodScheduled,
								Status: v1.ConditionFalse,
							},
						},
					},
				},
			},
			want: metadata + `
				kube_pod_status_scheduled{condition="false",namespace="ns1",pod="pod1"} 0
				kube_pod_status_scheduled{condition="false",namespace="ns2",pod="pod2"} 1
				kube_pod_status_scheduled{condition="true",namespace="ns1",pod="pod1"} 1
				kube_pod_status_scheduled{condition="true",namespace="ns2",pod="pod2"} 0
				kube_pod_status_scheduled{condition="unknown",namespace="ns1",pod="pod1"} 0
				kube_pod_status_scheduled{condition="unknown",namespace="ns2",pod="pod2"} 0
			`,
			metrics: []string{"kube_pod_status_scheduled"},
		},
	}
	for _, c := range cases {
		pc := &podCollector{
			store: mockPodStore{
				f: func() ([]v1.Pod, error) { return c.pods, nil },
			},
		}
		if err := gatherAndCompare(pc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
