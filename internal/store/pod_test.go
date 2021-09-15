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

package store

import (
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func TestPodStore(t *testing.T) {
	var test = true
	runtimeclass := "foo"
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)

	cases := []generateMetricsTestCase{
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:        "container1",
							Image:       "k8s.gcr.io/hyperkube1",
							ImageID:     "docker://sha256:aaa",
							ContainerID: "docker://ab123",
						},
					},
				},
			},
			Want: `
			# HELP kube_pod_container_info Information about a container in a pod.
			# TYPE kube_pod_container_info gauge
			kube_pod_container_info{container="container1",container_id="docker://ab123",image="k8s.gcr.io/hyperkube1",image_id="docker://sha256:aaa",namespace="ns1",pod="pod1",uid="uid1"} 1`,
			MetricNames: []string{"kube_pod_container_info"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "ns2",
					UID:       "uid2",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:        "container2",
							Image:       "k8s.gcr.io/hyperkube2",
							ImageID:     "docker://sha256:bbb",
							ContainerID: "docker://cd456",
						},
						{
							Name:        "container3",
							Image:       "k8s.gcr.io/hyperkube3",
							ImageID:     "docker://sha256:ccc",
							ContainerID: "docker://ef789",
						},
					},
					InitContainerStatuses: []v1.ContainerStatus{
						{
							Name:        "initContainer",
							Image:       "k8s.gcr.io/initfoo",
							ImageID:     "docker://sha256:wxyz",
							ContainerID: "docker://ef123",
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_container_info Information about a container in a pod.
				# HELP kube_pod_init_container_info Information about an init container in a pod.
				# TYPE kube_pod_container_info gauge
				# TYPE kube_pod_init_container_info gauge
				kube_pod_container_info{container="container2",container_id="docker://cd456",image="k8s.gcr.io/hyperkube2",image_id="docker://sha256:bbb",namespace="ns2",pod="pod2",uid="uid2"} 1
				kube_pod_container_info{container="container3",container_id="docker://ef789",image="k8s.gcr.io/hyperkube3",image_id="docker://sha256:ccc",namespace="ns2",pod="pod2",uid="uid2"} 1
				kube_pod_init_container_info{container="initContainer",container_id="docker://ef123",image="k8s.gcr.io/initfoo",image_id="docker://sha256:wxyz",namespace="ns2",pod="pod2",uid="uid2"} 1`,
			MetricNames: []string{"kube_pod_container_info", "kube_pod_init_container_info"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:  "container1",
							Ready: true,
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_container_status_ready Describes whether the containers readiness check succeeded.
				# TYPE kube_pod_container_status_ready gauge
				kube_pod_container_status_ready{container="container1",namespace="ns1",pod="pod1",uid="uid1"} 1`,
			MetricNames: []string{"kube_pod_container_status_ready"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "ns2",
					UID:       "uid2",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:  "container2",
							Ready: true,
						},
						{
							Name:  "container3",
							Ready: false,
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_container_status_ready Describes whether the containers readiness check succeeded.
				# TYPE kube_pod_container_status_ready gauge
				kube_pod_container_status_ready{container="container2",namespace="ns2",pod="pod2",uid="uid2"} 1
				kube_pod_container_status_ready{container="container3",namespace="ns2",pod="pod2",uid="uid2"} 0
				`,
			MetricNames: []string{"kube_pod_container_status_ready"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod3",
					Namespace: "ns3",
					UID:       "uid3",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:  "container2",
							Ready: true,
						},
						{
							Name:  "container3",
							Ready: false,
						},
					},
					InitContainerStatuses: []v1.ContainerStatus{
						{
							Name:  "initcontainer1",
							Ready: true,
						},
						{
							Name:  "initcontainer2",
							Ready: false,
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_init_container_status_ready Describes whether the init containers readiness check succeeded.
				# TYPE kube_pod_init_container_status_ready gauge
				kube_pod_init_container_status_ready{container="initcontainer1",namespace="ns3",pod="pod3",uid="uid3"} 1
				kube_pod_init_container_status_ready{container="initcontainer2",namespace="ns3",pod="pod3",uid="uid3"} 0
				`,
			MetricNames: []string{"kube_pod_init_container_status_ready"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:         "container1",
							RestartCount: 0,
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_container_status_restarts_total The number of container restarts per container.
				# TYPE kube_pod_container_status_restarts_total counter
				kube_pod_container_status_restarts_total{container="container1",namespace="ns1",pod="pod1",uid="uid1"} 0
				`,
			MetricNames: []string{"kube_pod_container_status_restarts_total"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "ns2",
					UID:       "uid2",
				},
				Status: v1.PodStatus{
					InitContainerStatuses: []v1.ContainerStatus{
						{
							Name:         "initcontainer1",
							RestartCount: 1,
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_init_container_status_restarts_total The number of restarts for the init container.
				# TYPE kube_pod_init_container_status_restarts_total counter
				kube_pod_init_container_status_restarts_total{container="initcontainer1",namespace="ns2",pod="pod2",uid="uid2"} 1
				`,
			MetricNames: []string{"kube_pod_init_container_status_restarts_total"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "ns2",
					UID:       "uid2",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:         "container2",
							RestartCount: 0,
						},
						{
							Name:         "container3",
							RestartCount: 1,
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_container_status_restarts_total The number of container restarts per container.
				# TYPE kube_pod_container_status_restarts_total counter
				kube_pod_container_status_restarts_total{container="container2",namespace="ns2",pod="pod2",uid="uid2"} 0
				kube_pod_container_status_restarts_total{container="container3",namespace="ns2",pod="pod2",uid="uid2"} 1
				`,
			MetricNames: []string{"kube_pod_container_status_restarts_total"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "ns2",
					UID:       "uid2",
				},
				Status: v1.PodStatus{
					InitContainerStatuses: []v1.ContainerStatus{
						{
							Name:         "initcontainer2",
							RestartCount: 0,
						},
						{
							Name:         "initcontainer3",
							RestartCount: 1,
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_init_container_status_restarts_total The number of restarts for the init container.
				# TYPE kube_pod_init_container_status_restarts_total counter
				kube_pod_init_container_status_restarts_total{container="initcontainer2",namespace="ns2",pod="pod2",uid="uid2"} 0
				kube_pod_init_container_status_restarts_total{container="initcontainer3",namespace="ns2",pod="pod2",uid="uid2"} 1
				`,
			MetricNames: []string{"kube_pod_init_container_status_restarts_total"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "container1",
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{
									StartedAt: metav1.Time{
										Time: time.Unix(1501777018, 0),
									},
								},
							},
						},
					},
					InitContainerStatuses: []v1.ContainerStatus{
						{
							Name: "initcontainer1",
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{},
							},
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_container_status_running Describes whether the container is currently in running state.
				# HELP kube_pod_container_state_started Start time in unix timestamp for a pod container.
				# HELP kube_pod_container_status_terminated Describes whether the container is currently in terminated state.
				# HELP kube_pod_container_status_terminated_reason Describes the reason the container is currently in terminated state.
				# HELP kube_pod_container_status_waiting Describes whether the container is currently in waiting state.
				# HELP kube_pod_container_status_waiting_reason Describes the reason the container is currently in waiting state.
				# HELP kube_pod_init_container_status_running Describes whether the init container is currently in running state.
				# HELP kube_pod_init_container_status_terminated Describes whether the init container is currently in terminated state.
				# HELP kube_pod_init_container_status_terminated_reason Describes the reason the init container is currently in terminated state.
				# HELP kube_pod_init_container_status_waiting Describes whether the init container is currently in waiting state.
				# HELP kube_pod_init_container_status_waiting_reason Describes the reason the init container is currently in waiting state.
				# TYPE kube_pod_container_status_running gauge
				# TYPE kube_pod_container_state_started gauge
				# TYPE kube_pod_container_status_terminated gauge
				# TYPE kube_pod_container_status_terminated_reason gauge
				# TYPE kube_pod_container_status_waiting gauge
				# TYPE kube_pod_container_status_waiting_reason gauge
				# TYPE kube_pod_init_container_status_running gauge
				# TYPE kube_pod_init_container_status_terminated gauge
				# TYPE kube_pod_init_container_status_terminated_reason gauge
				# TYPE kube_pod_init_container_status_waiting gauge
				# TYPE kube_pod_init_container_status_waiting_reason gauge
				kube_pod_container_state_started{container="container1",namespace="ns1",pod="pod1",uid="uid1"} 1.501777018e+09
				kube_pod_container_status_running{container="container1",namespace="ns1",pod="pod1",uid="uid1"} 1
				kube_pod_container_status_terminated{container="container1",namespace="ns1",pod="pod1",uid="uid1"} 0
				kube_pod_container_status_waiting{container="container1",namespace="ns1",pod="pod1",uid="uid1"} 0
				kube_pod_init_container_status_running{container="initcontainer1",namespace="ns1",pod="pod1",uid="uid1"} 1
				kube_pod_init_container_status_terminated{container="initcontainer1",namespace="ns1",pod="pod1",uid="uid1"} 0
				kube_pod_init_container_status_waiting{container="initcontainer1",namespace="ns1",pod="pod1",uid="uid1"} 0
			`,
			MetricNames: []string{
				"kube_pod_container_status_running",
				"kube_pod_container_state_started",
				"kube_pod_container_status_waiting",
				"kube_pod_container_status_terminated",
				"kube_pod_container_status_terminated_reason",
				"kube_pod_init_container_status_running",
				"kube_pod_init_container_status_waiting",
				"kube_pod_init_container_status_waiting_reason",
				"kube_pod_init_container_status_terminated",
				"kube_pod_init_container_status_terminated_reason",
			},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "container1",
							State: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									StartedAt: metav1.Time{
										Time: time.Unix(1501777018, 0),
									},
									Reason: "Completed",
								},
							},
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_container_status_running Describes whether the container is currently in running state.
				# HELP kube_pod_container_state_started Start time in unix timestamp for a pod container.
				# HELP kube_pod_container_status_terminated Describes whether the container is currently in terminated state.
				# HELP kube_pod_container_status_terminated_reason Describes the reason the container is currently in terminated state.
				# HELP kube_pod_container_status_waiting Describes whether the container is currently in waiting state.
				# HELP kube_pod_container_status_waiting_reason Describes the reason the container is currently in waiting state.
				# TYPE kube_pod_container_status_running gauge
				# TYPE kube_pod_container_state_started gauge
				# TYPE kube_pod_container_status_terminated gauge
				# TYPE kube_pod_container_status_terminated_reason gauge
				# TYPE kube_pod_container_status_waiting gauge
				# TYPE kube_pod_container_status_waiting_reason gauge
				kube_pod_container_state_started{container="container1",namespace="ns1",pod="pod1",uid="uid1"} 1.501777018e+09
				kube_pod_container_status_running{container="container1",namespace="ns1",pod="pod1",uid="uid1"} 0
				kube_pod_container_status_terminated_reason{container="container1",namespace="ns1",pod="pod1",reason="Completed",uid="uid1"} 1
				kube_pod_container_status_terminated{container="container1",namespace="ns1",pod="pod1",uid="uid1"} 1
				kube_pod_container_status_waiting{container="container1",namespace="ns1",pod="pod1",uid="uid1"} 0
			`,
			MetricNames: []string{
				"kube_pod_container_status_running",
				"kube_pod_container_state_started",
				"kube_pod_container_status_waiting",
				"kube_pod_container_status_terminated",
				"kube_pod_container_status_terminated_reason",
			},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "ns2",
					UID:       "uid2",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "container2",
							State: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									StartedAt: metav1.Time{
										Time: time.Unix(1501777018, 0),
									},
									Reason: "OOMKilled",
								},
							},
						},
						{
							Name: "container3",
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{
									Reason: "ContainerCreating",
								},
							},
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_container_status_running Describes whether the container is currently in running state.
				# HELP kube_pod_container_state_started Start time in unix timestamp for a pod container.
				# HELP kube_pod_container_status_terminated Describes whether the container is currently in terminated state.
				# HELP kube_pod_container_status_terminated_reason Describes the reason the container is currently in terminated state.
				# HELP kube_pod_container_status_waiting Describes whether the container is currently in waiting state.
				# HELP kube_pod_container_status_waiting_reason Describes the reason the container is currently in waiting state.
				# TYPE kube_pod_container_status_running gauge
				# TYPE kube_pod_container_state_started gauge
				# TYPE kube_pod_container_status_terminated gauge
				# TYPE kube_pod_container_status_terminated_reason gauge
				# TYPE kube_pod_container_status_waiting gauge
				# TYPE kube_pod_container_status_waiting_reason gauge
				kube_pod_container_status_running{container="container2",namespace="ns2",pod="pod2",uid="uid2"} 0
				kube_pod_container_status_running{container="container3",namespace="ns2",pod="pod2",uid="uid2"} 0
				kube_pod_container_state_started{container="container2",namespace="ns2",pod="pod2",uid="uid2"} 1.501777018e+09
				kube_pod_container_status_terminated_reason{container="container2",namespace="ns2",pod="pod2",reason="OOMKilled",uid="uid2"} 1
				kube_pod_container_status_terminated{container="container2",namespace="ns2",pod="pod2",uid="uid2"} 1
				kube_pod_container_status_terminated{container="container3",namespace="ns2",pod="pod2",uid="uid2"} 0
				kube_pod_container_status_waiting_reason{container="container3",namespace="ns2",pod="pod2",reason="ContainerCreating",uid="uid2"} 1
				kube_pod_container_status_waiting{container="container2",namespace="ns2",pod="pod2",uid="uid2"} 0
				kube_pod_container_status_waiting{container="container3",namespace="ns2",pod="pod2",uid="uid2"} 1
`,
			MetricNames: []string{
				"kube_pod_container_status_running",
				"kube_pod_container_state_started",
				"kube_pod_container_status_waiting",
				"kube_pod_container_status_terminated",
				"kube_pod_container_status_terminated_reason",
			},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod3",
					Namespace: "ns3",
					UID:       "uid3",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "container4",
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{
									Reason: "CrashLoopBackOff",
								},
							},
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_container_status_last_terminated_reason Describes the last reason the container was in terminated state.
				# HELP kube_pod_container_status_running Describes whether the container is currently in running state.
				# HELP kube_pod_container_status_terminated Describes whether the container is currently in terminated state.
				# HELP kube_pod_container_status_terminated_reason Describes the reason the container is currently in terminated state.
				# HELP kube_pod_container_status_waiting Describes whether the container is currently in waiting state.
				# HELP kube_pod_container_status_waiting_reason Describes the reason the container is currently in waiting state.
				# TYPE kube_pod_container_status_last_terminated_reason gauge
				# TYPE kube_pod_container_status_running gauge
				# TYPE kube_pod_container_status_terminated gauge
				# TYPE kube_pod_container_status_terminated_reason gauge
				# TYPE kube_pod_container_status_waiting gauge
				# TYPE kube_pod_container_status_waiting_reason gauge
				kube_pod_container_status_running{container="container4",namespace="ns3",pod="pod3",uid="uid3"} 0
				kube_pod_container_status_terminated{container="container4",namespace="ns3",pod="pod3",uid="uid3"} 0
				kube_pod_container_status_waiting_reason{container="container4",namespace="ns3",pod="pod3",reason="CrashLoopBackOff",uid="uid3"} 1
				kube_pod_container_status_waiting{container="container4",namespace="ns3",pod="pod3",uid="uid3"} 1
`,
			MetricNames: []string{
				"kube_pod_container_status_running",
				"kube_pod_container_status_terminated",
				"kube_pod_container_status_terminated_reason",
				"kube_pod_container_status_terminated_reason",
				"kube_pod_container_status_terminated_reason",
				"kube_pod_container_status_terminated_reason",
				"kube_pod_container_status_waiting",
				"kube_pod_container_status_waiting_reason",
				"kube_pod_container_status_last_terminated_reason",
				"kube_pod_container_status_last_terminated_reason",
				"kube_pod_container_status_last_terminated_reason",
				"kube_pod_container_status_last_terminated_reason",
			},
		},
		{

			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod6",
					Namespace: "ns6",
					UID:       "uid6",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "container7",
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{
									StartedAt: metav1.Time{
										Time: time.Unix(1501777018, 0),
									},
								},
							},
							LastTerminationState: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									Reason: "OOMKilled",
								},
							},
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_container_status_last_terminated_reason Describes the last reason the container was in terminated state.
				# HELP kube_pod_container_status_running Describes whether the container is currently in running state.
				# HELP kube_pod_container_status_terminated Describes whether the container is currently in terminated state.
				# HELP kube_pod_container_status_terminated_reason Describes the reason the container is currently in terminated state.
				# HELP kube_pod_container_status_waiting Describes whether the container is currently in waiting state.
				# HELP kube_pod_container_status_waiting_reason Describes the reason the container is currently in waiting state.
				# HELP kube_pod_container_state_started Start time in unix timestamp for a pod container.
				# TYPE kube_pod_container_status_last_terminated_reason gauge
				# TYPE kube_pod_container_status_running gauge
				# TYPE kube_pod_container_status_terminated gauge
				# TYPE kube_pod_container_status_terminated_reason gauge
				# TYPE kube_pod_container_status_waiting gauge
				# TYPE kube_pod_container_status_waiting_reason gauge
				# TYPE kube_pod_container_state_started gauge
				kube_pod_container_status_running{container="container7",namespace="ns6",pod="pod6",uid="uid6"} 1
				kube_pod_container_state_started{container="container7",namespace="ns6",pod="pod6",uid="uid6"} 1.501777018e+09
				kube_pod_container_status_terminated{container="container7",namespace="ns6",pod="pod6",uid="uid6"} 0
				kube_pod_container_status_waiting{container="container7",namespace="ns6",pod="pod6",uid="uid6"} 0
				kube_pod_container_status_last_terminated_reason{container="container7",namespace="ns6",pod="pod6",reason="OOMKilled",uid="uid6"} 1
			`,
			MetricNames: []string{
				"kube_pod_container_status_last_terminated_reason",
				"kube_pod_container_status_running",
				"kube_pod_container_state_started",
				"kube_pod_container_status_terminated",
				"kube_pod_container_status_terminated_reason",
				"kube_pod_container_status_waiting",
			},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod7",
					Namespace: "ns7",
					UID:       "uid7",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "container7",
							State: v1.ContainerState{
								Running: &v1.ContainerStateRunning{
									StartedAt: metav1.Time{
										Time: time.Unix(1501777018, 0),
									},
								},
							},
							LastTerminationState: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									Reason: "DeadlineExceeded",
								},
							},
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_container_status_last_terminated_reason Describes the last reason the container was in terminated state.
				# HELP kube_pod_container_status_running Describes whether the container is currently in running state.
				# HELP kube_pod_container_state_started Start time in unix timestamp for a pod container.
				# HELP kube_pod_container_status_terminated Describes whether the container is currently in terminated state.
				# HELP kube_pod_container_status_terminated_reason Describes the reason the container is currently in terminated state.
				# HELP kube_pod_container_status_waiting Describes whether the container is currently in waiting state.
				# HELP kube_pod_container_status_waiting_reason Describes the reason the container is currently in waiting state.
				# TYPE kube_pod_container_status_last_terminated_reason gauge
				# TYPE kube_pod_container_status_running gauge
				# TYPE kube_pod_container_state_started gauge
				# TYPE kube_pod_container_status_terminated gauge
				# TYPE kube_pod_container_status_terminated_reason gauge
				# TYPE kube_pod_container_status_waiting gauge
				# TYPE kube_pod_container_status_waiting_reason gauge
				kube_pod_container_state_started{container="container7",namespace="ns7",pod="pod7",uid="uid7"} 1.501777018e+09
				kube_pod_container_status_last_terminated_reason{container="container7",namespace="ns7",pod="pod7",reason="DeadlineExceeded",uid="uid7"} 1
				kube_pod_container_status_running{container="container7",namespace="ns7",pod="pod7",uid="uid7"} 1
				kube_pod_container_status_terminated{container="container7",namespace="ns7",pod="pod7",uid="uid7"} 0
				kube_pod_container_status_waiting{container="container7",namespace="ns7",pod="pod7",uid="uid7"} 0
			`,
			MetricNames: []string{
				"kube_pod_container_status_running",
				"kube_pod_container_state_started",
				"kube_pod_container_status_terminated",
				"kube_pod_container_status_terminated_reason",
				"kube_pod_container_status_waiting",
				"kube_pod_container_status_last_terminated_reason",
			},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod4",
					Namespace: "ns4",
					UID:       "uid4",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "container5",
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{
									Reason: "ImagePullBackOff",
								},
							},
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_container_status_running Describes whether the container is currently in running state.
				# HELP kube_pod_container_status_terminated Describes whether the container is currently in terminated state.
				# HELP kube_pod_container_status_terminated_reason Describes the reason the container is currently in terminated state.
				# HELP kube_pod_container_status_waiting Describes whether the container is currently in waiting state.
				# HELP kube_pod_container_status_waiting_reason Describes the reason the container is currently in waiting state.
				# TYPE kube_pod_container_status_running gauge
				# TYPE kube_pod_container_status_terminated gauge
				# TYPE kube_pod_container_status_terminated_reason gauge
				# TYPE kube_pod_container_status_waiting gauge
				# TYPE kube_pod_container_status_waiting_reason gauge
				kube_pod_container_status_running{container="container5",namespace="ns4",pod="pod4",uid="uid4"} 0
				kube_pod_container_status_terminated{container="container5",namespace="ns4",pod="pod4",uid="uid4"} 0
				kube_pod_container_status_waiting_reason{container="container5",namespace="ns4",pod="pod4",reason="ImagePullBackOff",uid="uid4"} 1
				kube_pod_container_status_waiting{container="container5",namespace="ns4",pod="pod4",uid="uid4"} 1
`,
			MetricNames: []string{
				"kube_pod_container_status_running",
				"kube_pod_container_status_waiting",
				"kube_pod_container_status_waiting_reason",
				"kube_pod_container_status_terminated",
				"kube_pod_container_status_terminated_reason",
			},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod5",
					Namespace: "ns5",
					UID:       "uid5",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "container6",
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{
									Reason: "ErrImagePull",
								},
							},
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_container_status_running Describes whether the container is currently in running state.
				# HELP kube_pod_container_status_terminated Describes whether the container is currently in terminated state.
				# HELP kube_pod_container_status_terminated_reason Describes the reason the container is currently in terminated state.
				# HELP kube_pod_container_status_waiting Describes whether the container is currently in waiting state.
				# HELP kube_pod_container_status_waiting_reason Describes the reason the container is currently in waiting state.
				# TYPE kube_pod_container_status_running gauge
				# TYPE kube_pod_container_status_terminated gauge
				# TYPE kube_pod_container_status_terminated_reason gauge
				# TYPE kube_pod_container_status_waiting gauge
				# TYPE kube_pod_container_status_waiting_reason gauge
				kube_pod_container_status_running{container="container6",namespace="ns5",pod="pod5",uid="uid5"} 0
				kube_pod_container_status_terminated{container="container6",namespace="ns5",pod="pod5",uid="uid5"} 0
				kube_pod_container_status_waiting_reason{container="container6",namespace="ns5",pod="pod5",reason="ErrImagePull",uid="uid5"} 1
				kube_pod_container_status_waiting{container="container6",namespace="ns5",pod="pod5",uid="uid5"} 1
			`,
			MetricNames: []string{
				"kube_pod_container_status_running",
				"kube_pod_container_status_waiting",
				"kube_pod_container_status_waiting_reason",
				"kube_pod_container_status_terminated",
				"kube_pod_container_status_terminated_reason",
			},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod7",
					Namespace: "ns7",
					UID:       "uid7",
				},
				Status: v1.PodStatus{
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "container8",
							State: v1.ContainerState{
								Waiting: &v1.ContainerStateWaiting{
									Reason: "CreateContainerConfigError",
								},
							},
						},
					},
				},
			},
			Want: `
					# HELP kube_pod_container_status_running Describes whether the container is currently in running state.
					# HELP kube_pod_container_status_terminated Describes whether the container is currently in terminated state.
					# HELP kube_pod_container_status_terminated_reason Describes the reason the container is currently in terminated state.
					# HELP kube_pod_container_status_waiting Describes whether the container is currently in waiting state.
					# HELP kube_pod_container_status_waiting_reason Describes the reason the container is currently in waiting state.
					# TYPE kube_pod_container_status_running gauge
					# TYPE kube_pod_container_status_terminated gauge
					# TYPE kube_pod_container_status_terminated_reason gauge
					# TYPE kube_pod_container_status_waiting gauge
					# TYPE kube_pod_container_status_waiting_reason gauge
					kube_pod_container_status_running{container="container8",namespace="ns7",pod="pod7",uid="uid7"} 0
					kube_pod_container_status_terminated{container="container8",namespace="ns7",pod="pod7",uid="uid7"} 0
					kube_pod_container_status_waiting_reason{container="container8",namespace="ns7",pod="pod7",reason="CreateContainerConfigError",uid="uid7"} 1
					kube_pod_container_status_waiting{container="container8",namespace="ns7",pod="pod7",uid="uid7"} 1
			`,
			MetricNames: []string{
				"kube_pod_container_status_running",
				"kube_pod_container_status_terminated",
				"kube_pod_container_status_terminated_reason",
				"kube_pod_container_status_waiting",
				"kube_pod_container_status_waiting_reason",
			},
		},
		{

			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "ns1",
					UID:               "abc-123-xxx",
				},
				Spec: v1.PodSpec{
					NodeName:          "node1",
					PriorityClassName: "system-node-critical",
					HostNetwork:       true,
				},
				Status: v1.PodStatus{
					HostIP:    "1.1.1.1",
					PodIP:     "1.2.3.4",
					StartTime: &metav1StartTime,
				},
			},
			// TODO: Should it be '1501569018' instead?
			Want: `
				# HELP kube_pod_completion_time Completion time in unix timestamp for a pod.
				# HELP kube_pod_created Unix creation timestamp
				# HELP kube_pod_info Information about pod.
				# HELP kube_pod_owner Information about the Pod's owner.
				# HELP kube_pod_start_time Start time in unix timestamp for a pod.
				# TYPE kube_pod_completion_time gauge
				# TYPE kube_pod_created gauge
				# TYPE kube_pod_info gauge
				# TYPE kube_pod_owner gauge
				# TYPE kube_pod_start_time gauge
				kube_pod_created{namespace="ns1",pod="pod1",uid="abc-123-xxx"} 1.5e+09
				kube_pod_info{created_by_kind="<none>",created_by_name="<none>",host_ip="1.1.1.1",namespace="ns1",node="node1",pod="pod1",pod_ip="1.2.3.4",uid="abc-123-xxx",priority_class="system-node-critical",host_network="true"} 1
				kube_pod_start_time{namespace="ns1",pod="pod1",uid="abc-123-xxx"} 1.501569018e+09
				kube_pod_owner{namespace="ns1",owner_is_controller="<none>",owner_kind="<none>",owner_name="<none>",pod="pod1",uid="abc-123-xxx"} 1
`,
			MetricNames: []string{"kube_pod_created", "kube_pod_info", "kube_pod_start_time", "kube_pod_completion_time", "kube_pod_owner"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "ns1",
					UID:               "abc-123-xxx",
					DeletionTimestamp: &metav1.Time{Time: time.Unix(1800000000, 0)},
				},
				Spec: v1.PodSpec{
					NodeName:          "node1",
					PriorityClassName: "system-node-critical",
				},
				Status: v1.PodStatus{
					HostIP:    "1.1.1.1",
					PodIP:     "1.2.3.4",
					StartTime: &metav1StartTime,
				},
			},
			Want: `
				# HELP kube_pod_deletion_timestamp Unix deletion timestamp
				# TYPE kube_pod_deletion_timestamp gauge
				kube_pod_deletion_timestamp{namespace="ns1",pod="pod1",uid="abc-123-xxx"} 1.8e+09
`,
			MetricNames: []string{"kube_pod_deletion_timestamp"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "ns2",
					UID:       "uid2",
				},
				Spec: v1.PodSpec{
					RestartPolicy: v1.RestartPolicyAlways,
				},
			},
			Want: `
				# HELP kube_pod_restart_policy Describes the restart policy in use by this pod.
				# TYPE kube_pod_restart_policy gauge
				kube_pod_restart_policy{namespace="ns2",pod="pod2",type="Always",uid="uid2"} 1
				`,
			MetricNames: []string{"kube_pod_restart_policy"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "ns2",
					UID:       "uid2",
				},
				Spec: v1.PodSpec{
					RestartPolicy: v1.RestartPolicyOnFailure,
				},
			},
			Want: `
				# HELP kube_pod_restart_policy Describes the restart policy in use by this pod.
				# TYPE kube_pod_restart_policy gauge
				kube_pod_restart_policy{namespace="ns2",pod="pod2",type="OnFailure",uid="uid2"} 1
				`,
			MetricNames: []string{"kube_pod_restart_policy"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "ns2",
					UID:       "abc-456-xxx",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "ReplicaSet",
							Name:       "rs-name",
							Controller: &test,
						},
					},
				},
				Spec: v1.PodSpec{
					NodeName: "node2",
				},
				Status: v1.PodStatus{
					HostIP: "1.1.1.1",
					PodIP:  "2.3.4.5",
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name:        "container2_1",
							Image:       "k8s.gcr.io/hyperkube2",
							ImageID:     "docker://sha256:bbb",
							ContainerID: "docker://cd456",
							State: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									FinishedAt: metav1.Time{
										Time: time.Unix(1501777018, 0),
									},
								},
							},
						},
						{
							Name:        "container2_2",
							Image:       "k8s.gcr.io/hyperkube2",
							ImageID:     "docker://sha256:bbb",
							ContainerID: "docker://cd456",
							State: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									FinishedAt: metav1.Time{
										Time: time.Unix(1501888018, 0),
									},
								},
							},
						},
						{
							Name:        "container2_3",
							Image:       "k8s.gcr.io/hyperkube2",
							ImageID:     "docker://sha256:bbb",
							ContainerID: "docker://cd456",
							State: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									FinishedAt: metav1.Time{
										Time: time.Unix(1501666018, 0),
									},
								},
							},
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_completion_time Completion time in unix timestamp for a pod.
				# HELP kube_pod_created Unix creation timestamp
				# HELP kube_pod_info Information about pod.
				# HELP kube_pod_owner Information about the Pod's owner.
				# HELP kube_pod_start_time Start time in unix timestamp for a pod.
				# TYPE kube_pod_completion_time gauge
				# TYPE kube_pod_created gauge
				# TYPE kube_pod_info gauge
				# TYPE kube_pod_owner gauge
				# TYPE kube_pod_start_time gauge
				kube_pod_info{created_by_kind="ReplicaSet",created_by_name="rs-name",host_ip="1.1.1.1",namespace="ns2",node="node2",pod="pod2",pod_ip="2.3.4.5",uid="abc-456-xxx",priority_class="",host_network="false"} 1
				kube_pod_completion_time{namespace="ns2",pod="pod2",uid="abc-456-xxx"} 1.501888018e+09
				kube_pod_owner{namespace="ns2",owner_is_controller="true",owner_kind="ReplicaSet",owner_name="rs-name",pod="pod2",uid="abc-456-xxx"} 1
				`,
			MetricNames: []string{"kube_pod_created", "kube_pod_info", "kube_pod_start_time", "kube_pod_completion_time", "kube_pod_owner"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
				},
			},
			Want: `
				# HELP kube_pod_status_phase The pods current phase.
				# TYPE kube_pod_status_phase gauge
				kube_pod_status_phase{namespace="ns1",phase="Failed",pod="pod1",uid="uid1"} 0
				kube_pod_status_phase{namespace="ns1",phase="Pending",pod="pod1",uid="uid1"} 0
				kube_pod_status_phase{namespace="ns1",phase="Running",pod="pod1",uid="uid1"} 1
				kube_pod_status_phase{namespace="ns1",phase="Succeeded",pod="pod1",uid="uid1"} 0
				kube_pod_status_phase{namespace="ns1",phase="Unknown",pod="pod1",uid="uid1"} 0
`,
			MetricNames: []string{"kube_pod_status_phase"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "ns2",
					UID:       "uid2",
				},
				Status: v1.PodStatus{
					Phase: v1.PodPending,
				},
			},
			Want: `
				# HELP kube_pod_status_phase The pods current phase.
				# TYPE kube_pod_status_phase gauge
				kube_pod_status_phase{namespace="ns2",phase="Failed",pod="pod2",uid="uid2"} 0
				kube_pod_status_phase{namespace="ns2",phase="Pending",pod="pod2",uid="uid2"} 1
				kube_pod_status_phase{namespace="ns2",phase="Running",pod="pod2",uid="uid2"} 0
				kube_pod_status_phase{namespace="ns2",phase="Succeeded",pod="pod2",uid="uid2"} 0
				kube_pod_status_phase{namespace="ns2",phase="Unknown",pod="pod2",uid="uid2"} 0
`,
			MetricNames: []string{"kube_pod_status_phase"},
		},
		{

			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod3",
					Namespace: "ns3",
					UID:       "uid3",
				},
				Status: v1.PodStatus{
					Phase: v1.PodUnknown,
				},
			},
			Want: `
				# HELP kube_pod_status_phase The pods current phase.
				# TYPE kube_pod_status_phase gauge
				kube_pod_status_phase{namespace="ns3",phase="Failed",pod="pod3",uid="uid3"} 0
				kube_pod_status_phase{namespace="ns3",phase="Pending",pod="pod3",uid="uid3"} 0
				kube_pod_status_phase{namespace="ns3",phase="Running",pod="pod3",uid="uid3"} 0
				kube_pod_status_phase{namespace="ns3",phase="Succeeded",pod="pod3",uid="uid3"} 0
				kube_pod_status_phase{namespace="ns3",phase="Unknown",pod="pod3",uid="uid3"} 1
`,
			MetricNames: []string{"kube_pod_status_phase"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod4",
					Namespace:         "ns4",
					UID:               "uid4",
					DeletionTimestamp: &metav1.Time{},
				},
				Status: v1.PodStatus{
					Phase:  v1.PodRunning,
					Reason: "NodeLost",
				},
			},
			Want: `
				# HELP kube_pod_status_phase The pods current phase.
				# HELP kube_pod_status_reason The pod status reasons
				# TYPE kube_pod_status_phase gauge
				# TYPE kube_pod_status_reason gauge
				kube_pod_status_phase{namespace="ns4",phase="Failed",pod="pod4",uid="uid4"} 0
				kube_pod_status_phase{namespace="ns4",phase="Pending",pod="pod4",uid="uid4"} 0
				kube_pod_status_phase{namespace="ns4",phase="Running",pod="pod4",uid="uid4"} 1
				kube_pod_status_phase{namespace="ns4",phase="Succeeded",pod="pod4",uid="uid4"} 0
				kube_pod_status_phase{namespace="ns4",phase="Unknown",pod="pod4",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="Evicted",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="NodeAffinity",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="NodeLost",uid="uid4"} 1
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="Shutdown",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="UnexpectedAdmissionError",uid="uid4"} 0
`,
			MetricNames: []string{"kube_pod_status_phase", "kube_pod_status_reason"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod4",
					Namespace:         "ns4",
					UID:               "uid4",
					DeletionTimestamp: &metav1.Time{},
				},
				Status: v1.PodStatus{
					Phase:  v1.PodRunning,
					Reason: "Evicted",
				},
			},
			Want: `
				# HELP kube_pod_status_reason The pod status reasons
				# TYPE kube_pod_status_reason gauge
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="Evicted",uid="uid4"} 1
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="NodeAffinity",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="NodeLost",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="Shutdown",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="UnexpectedAdmissionError",uid="uid4"} 0
`,
			MetricNames: []string{"kube_pod_status_reason"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod4",
					Namespace:         "ns4",
					UID:               "uid4",
					DeletionTimestamp: &metav1.Time{},
				},
				Status: v1.PodStatus{
					Phase:  v1.PodRunning,
					Reason: "UnexpectedAdmissionError",
				},
			},
			Want: `
				# HELP kube_pod_status_reason The pod status reasons
				# TYPE kube_pod_status_reason gauge
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="Evicted",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="NodeAffinity",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="NodeLost",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="Shutdown",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="UnexpectedAdmissionError",uid="uid4"} 1
`,
			MetricNames: []string{"kube_pod_status_reason"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod4",
					Namespace:         "ns4",
					UID:               "uid4",
					DeletionTimestamp: &metav1.Time{},
				},
				Status: v1.PodStatus{
					Phase:  v1.PodRunning,
					Reason: "NodeAffinity",
				},
			},
			Want: `
				# HELP kube_pod_status_reason The pod status reasons
				# TYPE kube_pod_status_reason gauge
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="Evicted",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="NodeAffinity",uid="uid4"} 1
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="NodeLost",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="Shutdown",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="UnexpectedAdmissionError",uid="uid4"} 0
`,
			MetricNames: []string{"kube_pod_status_reason"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod4",
					Namespace:         "ns4",
					UID:               "uid4",
					DeletionTimestamp: &metav1.Time{},
				},
				Status: v1.PodStatus{
					Phase:  v1.PodRunning,
					Reason: "Shutdown",
				},
			},
			Want: `
				# HELP kube_pod_status_reason The pod status reasons
				# TYPE kube_pod_status_reason gauge
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="Evicted",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="NodeAffinity",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="NodeLost",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="Shutdown",uid="uid4"} 1
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="UnexpectedAdmissionError",uid="uid4"} 0
`,
			MetricNames: []string{"kube_pod_status_reason"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pod4",
					Namespace:         "ns4",
					UID:               "uid4",
					DeletionTimestamp: &metav1.Time{},
				},
				Status: v1.PodStatus{
					Phase:  v1.PodRunning,
					Reason: "other reason",
				},
			},
			Want: `
				# HELP kube_pod_status_reason The pod status reasons
				# TYPE kube_pod_status_reason gauge
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="Evicted",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="NodeAffinity",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="NodeLost",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="Shutdown",uid="uid4"} 0
				kube_pod_status_reason{namespace="ns4",pod="pod4",reason="UnexpectedAdmissionError",uid="uid4"} 0
`,
			MetricNames: []string{"kube_pod_status_reason"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{
							Type:   v1.PodReady,
							Status: v1.ConditionTrue,
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_status_ready Describes whether the pod is ready to serve requests.
				# TYPE kube_pod_status_ready gauge
				kube_pod_status_ready{condition="false",namespace="ns1",pod="pod1",uid="uid1"} 0
				kube_pod_status_ready{condition="true",namespace="ns1",pod="pod1",uid="uid1"} 1
				kube_pod_status_ready{condition="unknown",namespace="ns1",pod="pod1",uid="uid1"} 0
			`,
			MetricNames: []string{"kube_pod_status_ready"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "ns2",
					UID:       "uid2",
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{
							Type:   v1.PodReady,
							Status: v1.ConditionFalse,
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_status_ready Describes whether the pod is ready to serve requests.
				# TYPE kube_pod_status_ready gauge
				kube_pod_status_ready{condition="false",namespace="ns2",pod="pod2",uid="uid2"} 1
				kube_pod_status_ready{condition="true",namespace="ns2",pod="pod2",uid="uid2"} 0
				kube_pod_status_ready{condition="unknown",namespace="ns2",pod="pod2",uid="uid2"} 0
			`,
			MetricNames: []string{"kube_pod_status_ready"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{
							Type:   v1.PodScheduled,
							Status: v1.ConditionTrue,
							LastTransitionTime: metav1.Time{
								Time: time.Unix(1501666018, 0),
							},
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_status_scheduled Describes the status of the scheduling process for the pod.
				# HELP kube_pod_status_scheduled_time Unix timestamp when pod moved into scheduled status
				# TYPE kube_pod_status_scheduled gauge
				# TYPE kube_pod_status_scheduled_time gauge
				kube_pod_status_scheduled_time{namespace="ns1",pod="pod1",uid="uid1"} 1.501666018e+09
				kube_pod_status_scheduled{condition="false",namespace="ns1",pod="pod1",uid="uid1"} 0
				kube_pod_status_scheduled{condition="true",namespace="ns1",pod="pod1",uid="uid1"} 1
				kube_pod_status_scheduled{condition="unknown",namespace="ns1",pod="pod1",uid="uid1"} 0
			`,
			MetricNames: []string{"kube_pod_status_scheduled", "kube_pod_status_scheduled_time"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "ns2",
					UID:       "uid2",
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{
							Type:   v1.PodScheduled,
							Status: v1.ConditionFalse,
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_status_scheduled Describes the status of the scheduling process for the pod.
				# HELP kube_pod_status_scheduled_time Unix timestamp when pod moved into scheduled status
				# TYPE kube_pod_status_scheduled gauge
				# TYPE kube_pod_status_scheduled_time gauge
				kube_pod_status_scheduled{condition="false",namespace="ns2",pod="pod2",uid="uid2"} 1
				kube_pod_status_scheduled{condition="true",namespace="ns2",pod="pod2",uid="uid2"} 0
				kube_pod_status_scheduled{condition="unknown",namespace="ns2",pod="pod2",uid="uid2"} 0
			`,
			MetricNames: []string{"kube_pod_status_scheduled", "kube_pod_status_scheduled_time"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "ns2",
					UID:       "uid2",
				},
				Status: v1.PodStatus{
					Conditions: []v1.PodCondition{
						{
							Type:    v1.PodScheduled,
							Status:  v1.ConditionFalse,
							Reason:  "Unschedulable",
							Message: "0/3 nodes are available: 3 Insufficient cpu.",
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_status_unschedulable Describes the unschedulable status for the pod.
				# TYPE kube_pod_status_unschedulable gauge
				kube_pod_status_unschedulable{namespace="ns2",pod="pod2",uid="uid2"} 1
			`,
			MetricNames: []string{"kube_pod_status_unschedulable"},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "pod1_con1",
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:                    resource.MustParse("200m"),
									v1.ResourceMemory:                 resource.MustParse("100M"),
									v1.ResourceEphemeralStorage:       resource.MustParse("300M"),
									v1.ResourceStorage:                resource.MustParse("400M"),
									v1.ResourceName("nvidia.com/gpu"): resource.MustParse("1"),
								},
								Limits: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:                    resource.MustParse("200m"),
									v1.ResourceMemory:                 resource.MustParse("100M"),
									v1.ResourceEphemeralStorage:       resource.MustParse("300M"),
									v1.ResourceStorage:                resource.MustParse("400M"),
									v1.ResourceName("nvidia.com/gpu"): resource.MustParse("1"),
								},
							},
						},
						{
							Name: "pod1_con2",
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:    resource.MustParse("300m"),
									v1.ResourceMemory: resource.MustParse("200M"),
								},
								Limits: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:    resource.MustParse("300m"),
									v1.ResourceMemory: resource.MustParse("200M"),
								},
							},
						},
					},
					InitContainers: []v1.Container{
						{
							Name: "pod1_initcon1",
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:                    resource.MustParse("200m"),
									v1.ResourceMemory:                 resource.MustParse("100M"),
									v1.ResourceEphemeralStorage:       resource.MustParse("300M"),
									v1.ResourceStorage:                resource.MustParse("400M"),
									v1.ResourceName("nvidia.com/gpu"): resource.MustParse("1"),
								},
								Limits: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:                    resource.MustParse("200m"),
									v1.ResourceMemory:                 resource.MustParse("100M"),
									v1.ResourceEphemeralStorage:       resource.MustParse("300M"),
									v1.ResourceStorage:                resource.MustParse("400M"),
									v1.ResourceName("nvidia.com/gpu"): resource.MustParse("1"),
								},
							},
						},
					},
				},
			},
			Want: `
		# HELP kube_pod_container_resource_limits The number of requested limit resource by a container.
        # HELP kube_pod_container_resource_requests The number of requested request resource by a container.
        # HELP kube_pod_init_container_resource_limits The number of requested limit resource by an init container.
        # HELP kube_pod_init_container_resource_limits_cpu_cores The number of CPU cores requested limit by an init container.
        # HELP kube_pod_init_container_resource_limits_ephemeral_storage_bytes Bytes of ephemeral-storage requested limit by an init container.
        # HELP kube_pod_init_container_resource_limits_memory_bytes Bytes of memory requested limit by an init container.
        # HELP kube_pod_init_container_resource_limits_storage_bytes Bytes of storage requested limit by an init container.
        # HELP kube_pod_init_container_resource_requests The number of requested request resource by an init container.
        # HELP kube_pod_init_container_resource_requests_cpu_cores The number of CPU cores requested by an init container.
        # HELP kube_pod_init_container_resource_requests_ephemeral_storage_bytes Bytes of ephemeral-storage requested by an init container.
        # HELP kube_pod_init_container_resource_requests_memory_bytes Bytes of memory requested by an init container.
        # HELP kube_pod_init_container_resource_requests_storage_bytes Bytes of storage requested by an init container.
        # HELP kube_pod_init_container_status_last_terminated_reason Describes the last reason the init container was in terminated state.
        # TYPE kube_pod_container_resource_limits gauge
        # TYPE kube_pod_container_resource_requests gauge
        # TYPE kube_pod_init_container_resource_limits gauge
        # TYPE kube_pod_init_container_resource_limits_cpu_cores gauge
        # TYPE kube_pod_init_container_resource_limits_ephemeral_storage_bytes gauge
        # TYPE kube_pod_init_container_resource_limits_memory_bytes gauge
        # TYPE kube_pod_init_container_resource_limits_storage_bytes gauge
        # TYPE kube_pod_init_container_resource_requests gauge
        # TYPE kube_pod_init_container_resource_requests_cpu_cores gauge
        # TYPE kube_pod_init_container_resource_requests_ephemeral_storage_bytes gauge
        # TYPE kube_pod_init_container_resource_requests_memory_bytes gauge
        # TYPE kube_pod_init_container_resource_requests_storage_bytes gauge
        # TYPE kube_pod_init_container_status_last_terminated_reason gauge
        kube_pod_container_resource_limits{container="pod1_con1",namespace="ns1",node="",pod="pod1",resource="cpu",unit="core",uid="uid1"} 0.2
        kube_pod_container_resource_limits{container="pod1_con1",namespace="ns1",node="",pod="pod1",resource="ephemeral_storage",unit="byte",uid="uid1"} 3e+08
        kube_pod_container_resource_limits{container="pod1_con1",namespace="ns1",node="",pod="pod1",resource="memory",unit="byte",uid="uid1"} 1e+08
        kube_pod_container_resource_limits{container="pod1_con1",namespace="ns1",node="",pod="pod1",resource="nvidia_com_gpu",unit="integer",uid="uid1"} 1
        kube_pod_container_resource_limits{container="pod1_con1",namespace="ns1",node="",pod="pod1",resource="storage",unit="byte",uid="uid1"} 4e+08
        kube_pod_container_resource_limits{container="pod1_con2",namespace="ns1",node="",pod="pod1",resource="cpu",unit="core",uid="uid1"} 0.3
        kube_pod_container_resource_limits{container="pod1_con2",namespace="ns1",node="",pod="pod1",resource="memory",unit="byte",uid="uid1"} 2e+08
        kube_pod_container_resource_requests{container="pod1_con1",namespace="ns1",node="",pod="pod1",resource="cpu",unit="core",uid="uid1"} 0.2
        kube_pod_container_resource_requests{container="pod1_con1",namespace="ns1",node="",pod="pod1",resource="ephemeral_storage",unit="byte",uid="uid1"} 3e+08
        kube_pod_container_resource_requests{container="pod1_con1",namespace="ns1",node="",pod="pod1",resource="memory",unit="byte",uid="uid1"} 1e+08
        kube_pod_container_resource_requests{container="pod1_con1",namespace="ns1",node="",pod="pod1",resource="nvidia_com_gpu",unit="integer",uid="uid1"} 1
        kube_pod_container_resource_requests{container="pod1_con1",namespace="ns1",node="",pod="pod1",resource="storage",unit="byte",uid="uid1"} 4e+08
        kube_pod_container_resource_requests{container="pod1_con2",namespace="ns1",node="",pod="pod1",resource="cpu",unit="core",uid="uid1"} 0.3
        kube_pod_container_resource_requests{container="pod1_con2",namespace="ns1",node="",pod="pod1",resource="memory",unit="byte",uid="uid1"} 2e+08
        kube_pod_init_container_resource_limits_cpu_cores{container="pod1_initcon1",namespace="ns1",pod="pod1",uid="uid1"} 0.2
        kube_pod_init_container_resource_limits_ephemeral_storage_bytes{container="pod1_initcon1",namespace="ns1",pod="pod1",uid="uid1"} 3e+08
        kube_pod_init_container_resource_limits_memory_bytes{container="pod1_initcon1",namespace="ns1",pod="pod1",uid="uid1"} 1e+08
        kube_pod_init_container_resource_limits_storage_bytes{container="pod1_initcon1",namespace="ns1",pod="pod1",uid="uid1"} 4e+08
        kube_pod_init_container_resource_limits{container="pod1_initcon1",namespace="ns1",pod="pod1",resource="nvidia_com_gpu",unit="integer",uid="uid1"} 1
        kube_pod_init_container_resource_requests_cpu_cores{container="pod1_initcon1",namespace="ns1",pod="pod1",uid="uid1"} 0.2
        kube_pod_init_container_resource_requests_ephemeral_storage_bytes{container="pod1_initcon1",namespace="ns1",pod="pod1",uid="uid1"} 3e+08
        kube_pod_init_container_resource_requests_memory_bytes{container="pod1_initcon1",namespace="ns1",pod="pod1",uid="uid1"} 1e+08
        kube_pod_init_container_resource_requests_storage_bytes{container="pod1_initcon1",namespace="ns1",pod="pod1",uid="uid1"} 4e+08
        kube_pod_init_container_resource_requests{container="pod1_initcon1",namespace="ns1",pod="pod1",resource="nvidia_com_gpu",unit="integer",uid="uid1"} 1
		`,
			MetricNames: []string{
				"kube_pod_container_resource_requests",
				"kube_pod_container_resource_limits",
				"kube_pod_init_container_resource_limits",
				"kube_pod_init_container_resource_requests",
				"kube_pod_init_container_status_last_terminated_reason",
			},
		},
		{

			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod2",
					Namespace: "ns2",
					UID:       "uid2",
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "pod2_con1",
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:    resource.MustParse("400m"),
									v1.ResourceMemory: resource.MustParse("300M"),
								},
								Limits: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:    resource.MustParse("400m"),
									v1.ResourceMemory: resource.MustParse("300M"),
								},
							},
						},
						{
							Name: "pod2_con2",
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:    resource.MustParse("500m"),
									v1.ResourceMemory: resource.MustParse("400M"),
								},
								Limits: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:    resource.MustParse("500m"),
									v1.ResourceMemory: resource.MustParse("400M"),
								},
							},
						},
						// A container without a resource specification. No metrics will be emitted for that.
						{
							Name: "pod2_con3",
						},
					},
					InitContainers: []v1.Container{
						{
							Name: "pod2_initcon1",
							Resources: v1.ResourceRequirements{
								Requests: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:    resource.MustParse("400m"),
									v1.ResourceMemory: resource.MustParse("300M"),
								},
								Limits: map[v1.ResourceName]resource.Quantity{
									v1.ResourceCPU:    resource.MustParse("400m"),
									v1.ResourceMemory: resource.MustParse("300M"),
								},
							},
						},
					},
				},
			},
			Want: `
		# HELP kube_pod_init_container_resource_limits_cpu_cores The number of CPU cores requested limit by an init container.
        # HELP kube_pod_init_container_resource_limits_memory_bytes Bytes of memory requested limit by an init container.
        # TYPE kube_pod_init_container_resource_limits_cpu_cores gauge
        # TYPE kube_pod_init_container_resource_limits_memory_bytes gauge
        kube_pod_init_container_resource_limits_cpu_cores{container="pod2_initcon1",namespace="ns2",pod="pod2",uid="uid2"} 0.4
        kube_pod_init_container_resource_limits_memory_bytes{container="pod2_initcon1",namespace="ns2",pod="pod2",uid="uid2"} 3e+08
		`,
			MetricNames: []string{
				"kube_pod_init_container_resource_limits_cpu_cores",
				"kube_pod_init_container_resource_limits_memory_bytes",
			},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
					Labels: map[string]string{
						"app": "example",
					},
				},
				Spec: v1.PodSpec{},
			},
			Want: `
				# HELP kube_pod_labels Kubernetes labels converted to Prometheus labels.
				# TYPE kube_pod_labels gauge
				kube_pod_labels{namespace="ns1",pod="pod1",uid="uid1"} 1
		`,
			MetricNames: []string{
				"kube_pod_labels",
			},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
					Labels: map[string]string{
						"app": "example",
					},
				},
				Spec: v1.PodSpec{
					Volumes: []v1.Volume{
						{
							Name: "myvol",
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									ClaimName: "claim1",
									ReadOnly:  false,
								},
							},
						},
						{
							Name: "my-readonly-vol",
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									ClaimName: "claim2",
									ReadOnly:  true,
								},
							},
						},
						{
							Name: "not-pvc-vol",
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{
									Medium: "memory",
								},
							},
						},
					},
				},
			},
			Want: `
				# HELP kube_pod_spec_volumes_persistentvolumeclaims_info Information about persistentvolumeclaim volumes in a pod.
				# HELP kube_pod_spec_volumes_persistentvolumeclaims_readonly Describes whether a persistentvolumeclaim is mounted read only.
				# TYPE kube_pod_spec_volumes_persistentvolumeclaims_info gauge
				# TYPE kube_pod_spec_volumes_persistentvolumeclaims_readonly gauge
				kube_pod_spec_volumes_persistentvolumeclaims_info{namespace="ns1",persistentvolumeclaim="claim1",pod="pod1",volume="myvol",uid="uid1"} 1
				kube_pod_spec_volumes_persistentvolumeclaims_info{namespace="ns1",persistentvolumeclaim="claim2",pod="pod1",volume="my-readonly-vol",uid="uid1"} 1
				kube_pod_spec_volumes_persistentvolumeclaims_readonly{namespace="ns1",persistentvolumeclaim="claim1",pod="pod1",volume="myvol",uid="uid1"} 0
				kube_pod_spec_volumes_persistentvolumeclaims_readonly{namespace="ns1",persistentvolumeclaim="claim2",pod="pod1",volume="my-readonly-vol",uid="uid1"} 1

		`,
			MetricNames: []string{
				"kube_pod_spec_volumes_persistentvolumeclaims_info",
				"kube_pod_spec_volumes_persistentvolumeclaims_readonly",
			},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
					Labels: map[string]string{
						"app": "example",
					},
				},
				Spec: v1.PodSpec{
					RuntimeClassName: &runtimeclass,
				},
			},
			Want: `
				# HELP kube_pod_runtimeclass_name_info The runtimeclass associated with the pod.
				# TYPE kube_pod_runtimeclass_name_info gauge
				kube_pod_runtimeclass_name_info{namespace="ns1",pod="pod1",runtimeclass_name="foo",uid="uid1"} 1
			`,
			MetricNames: []string{
				"kube_pod_runtimeclass_name_info",
			},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
					Labels: map[string]string{
						"app": "example",
					},
				},
				Spec: v1.PodSpec{},
			},
			AllowLabelsList: []string{"wildcard-not-first", options.LabelWildcard},
			Want: `
				# HELP kube_pod_labels Kubernetes labels converted to Prometheus labels.
				# TYPE kube_pod_labels gauge
				kube_pod_labels{namespace="ns1",pod="pod1",uid="uid1"} 1
		`,
			MetricNames: []string{
				"kube_pod_labels",
			},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
					Labels: map[string]string{
						"app": "example",
					},
				},
				Spec: v1.PodSpec{},
			},
			AllowLabelsList: []string{options.LabelWildcard},
			Want: `
				# HELP kube_pod_labels Kubernetes labels converted to Prometheus labels.
				# TYPE kube_pod_labels gauge
				kube_pod_labels{label_app="example",namespace="ns1",pod="pod1",uid="uid1"} 1
		`,
			MetricNames: []string{
				"kube_pod_labels",
			},
		},
		{
			Obj: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "ns1",
					UID:       "uid1",
					Annotations: map[string]string{
						"app": "example",
					},
				},
				Spec: v1.PodSpec{},
			},
			AllowAnnotationsList: []string{options.LabelWildcard},
			Want: `
				# HELP kube_pod_annotations Kubernetes annotations converted to Prometheus labels.
				# TYPE kube_pod_annotations gauge
				kube_pod_annotations{annotation_app="example",namespace="ns1",pod="pod1",uid="uid1"} 1
		`,
			MetricNames: []string{
				"kube_pod_annotations",
			},
		},
	}

	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(podMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(podMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}

func BenchmarkPodStore(b *testing.B) {
	b.ReportAllocs()

	f := generator.ComposeMetricGenFuncs(podMetricFamilies(nil, nil))

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "ns1",
			UID:       "uid1",
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:         "container1",
					Image:        "k8s.gcr.io/hyperkube1",
					ImageID:      "docker://sha256:aaa",
					ContainerID:  "docker://ab123",
					Ready:        true,
					RestartCount: 0,
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
					LastTerminationState: v1.ContainerState{
						Terminated: &v1.ContainerStateTerminated{
							Reason: "OOMKilled",
						},
					},
				},
				{
					Name:         "container1",
					Image:        "k8s.gcr.io/hyperkube1",
					ImageID:      "docker://sha256:aaa",
					ContainerID:  "docker://ab123",
					Ready:        true,
					RestartCount: 0,
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
					LastTerminationState: v1.ContainerState{
						Terminated: &v1.ContainerStateTerminated{
							Reason: "OOMKilled",
						},
					},
				},
				{
					Name:         "container1",
					Image:        "k8s.gcr.io/hyperkube1",
					ImageID:      "docker://sha256:aaa",
					ContainerID:  "docker://ab123",
					Ready:        true,
					RestartCount: 0,
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
					LastTerminationState: v1.ContainerState{
						Terminated: &v1.ContainerStateTerminated{
							Reason: "OOMKilled",
						},
					},
				},
			},
		},
	}

	expectedFamilies := 51
	for n := 0; n < b.N; n++ {
		families := f(pod)
		if len(families) != expectedFamilies {
			b.Fatalf("expected %d but got %v", expectedFamilies, len(families))
		}
	}
}
