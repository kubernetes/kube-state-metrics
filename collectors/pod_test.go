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

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
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
	var test = true

	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)

	const metadata = `
		# HELP kube_pod_container_info Information about a container in a pod.
		# TYPE kube_pod_container_info gauge
		# HELP kube_pod_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_pod_labels gauge
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
		# HELP kube_pod_start_time Start time in unix timestamp for a pod.
		# TYPE kube_pod_start_time gauge
		# HELP kube_pod_owner Information about the Pod's owner.
		# TYPE kube_pod_owner gauge
		# HELP kube_pod_status_phase The pods current phase.
		# TYPE kube_pod_status_phase gauge
		# HELP kube_pod_status_ready Describes whether the pod is ready to serve requests.
		# TYPE kube_pod_status_ready gauge
		# HELP kube_pod_status_scheduled Describes the status of the scheduling process for the pod.
		# TYPE kube_pod_status_scheduled gauge
		# HELP kube_pod_container_resource_requests_cpu_cores The number of requested cpu cores by a container.
		# TYPE kube_pod_container_resource_requests_cpu_cores gauge
		# HELP kube_pod_container_resource_requests_memory_bytes The number of requested memory bytes  by a container.
		# TYPE kube_pod_container_resource_requests_memory_bytes gauge
		# HELP kube_pod_container_resource_limits_cpu_cores The limit on cpu cores to be used by a container.
		# TYPE kube_pod_container_resource_limits_cpu_cores gauge
		# HELP kube_pod_container_resource_limits_memory_bytes The limit on memory to be used by a container in bytes.
		# TYPE kube_pod_container_resource_limits_memory_bytes gauge
	`
	cases := []struct {
		pods    []v1.Pod
		metrics []string
		want    string
	}{
		{
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
					Spec: v1.PodSpec{
						NodeName: "node1",
					},
					Status: v1.PodStatus{
						HostIP:    "1.1.1.1",
						PodIP:     "1.2.3.4",
						StartTime: &metav1StartTime,
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
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
					},
				},
			},
			want: metadata + `
				kube_pod_info{created_by_kind="<none>",created_by_name="<none>",host_ip="1.1.1.1",namespace="ns1",pod="pod1",node="node1",pod_ip="1.2.3.4"} 1
				kube_pod_info{created_by_kind="<none>",created_by_name="<none>",host_ip="1.1.1.1",namespace="ns2",pod="pod2",node="node2",pod_ip="2.3.4.5"} 1
				kube_pod_start_time{namespace="ns1",pod="pod1"} 1501569018
				kube_pod_owner{namespace="ns1",pod="pod1",owner_kind="<none>",owner_name="<none>",owner_is_controller="<none>"} 1
				kube_pod_owner{namespace="ns2",pod="pod2",owner_kind="ReplicaSet",owner_name="rs-name",owner_is_controller="true"} 1
				`,
			metrics: []string{"kube_pod_info", "kube_pod_start_time", "kube_pod_owner"},
		}, {
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
					Status: v1.PodStatus{
						Phase: "Running",
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
					Status: v1.PodStatus{
						Phase: "Pending",
					},
				},
			},
			want: metadata + `
				kube_pod_status_phase{namespace="ns1",phase="Failed",pod="pod1"} 0
				kube_pod_status_phase{namespace="ns1",phase="Pending",pod="pod1"} 0
				kube_pod_status_phase{namespace="ns1",phase="Running",pod="pod1"} 1
				kube_pod_status_phase{namespace="ns1",phase="Succeeded",pod="pod1"} 0
				kube_pod_status_phase{namespace="ns1",phase="Unknown",pod="pod1"} 0
				kube_pod_status_phase{namespace="ns2",phase="Failed",pod="pod2"} 0
				kube_pod_status_phase{namespace="ns2",phase="Pending",pod="pod2"} 1
				kube_pod_status_phase{namespace="ns2",phase="Running",pod="pod2"} 0
				kube_pod_status_phase{namespace="ns2",phase="Succeeded",pod="pod2"} 0
				kube_pod_status_phase{namespace="ns2",phase="Unknown",pod="pod2"} 0
				`,
			metrics: []string{"kube_pod_status_phase"},
		}, {
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
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
					ObjectMeta: metav1.ObjectMeta{
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
		}, {
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
					Spec: v1.PodSpec{
						NodeName: "node1",
						Containers: []v1.Container{
							v1.Container{
								Name: "pod1_con1",
								Resources: v1.ResourceRequirements{
									Requests: map[v1.ResourceName]resource.Quantity{
										v1.ResourceCPU:    resource.MustParse("200m"),
										v1.ResourceMemory: resource.MustParse("100M"),
									},
									Limits: map[v1.ResourceName]resource.Quantity{
										v1.ResourceCPU:    resource.MustParse("200m"),
										v1.ResourceMemory: resource.MustParse("100M"),
									},
								},
							},
							v1.Container{
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
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
					Spec: v1.PodSpec{
						NodeName: "node2",
						Containers: []v1.Container{
							v1.Container{
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
							v1.Container{
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
							// A container without a resource specicication. No metrics will be emitted for that.
							v1.Container{
								Name: "pod2_con3",
							},
						},
					},
				},
			},
			want: metadata + `
				kube_pod_container_resource_requests_cpu_cores{container="pod1_con1",namespace="ns1",node="node1",pod="pod1"} 0.2
				kube_pod_container_resource_requests_cpu_cores{container="pod1_con2",namespace="ns1",node="node1",pod="pod1"} 0.3
				kube_pod_container_resource_requests_cpu_cores{container="pod2_con1",namespace="ns2",node="node2",pod="pod2"} 0.4
				kube_pod_container_resource_requests_cpu_cores{container="pod2_con2",namespace="ns2",node="node2",pod="pod2"} 0.5
				kube_pod_container_resource_requests_memory_bytes{container="pod1_con1",namespace="ns1",node="node1",pod="pod1"} 1e+08
				kube_pod_container_resource_requests_memory_bytes{container="pod1_con2",namespace="ns1",node="node1",pod="pod1"} 2e+08
				kube_pod_container_resource_requests_memory_bytes{container="pod2_con1",namespace="ns2",node="node2",pod="pod2"} 3e+08
				kube_pod_container_resource_requests_memory_bytes{container="pod2_con2",namespace="ns2",node="node2",pod="pod2"} 4e+08
				kube_pod_container_resource_limits_cpu_cores{container="pod1_con1",namespace="ns1",node="node1",pod="pod1"} 0.2
				kube_pod_container_resource_limits_cpu_cores{container="pod1_con2",namespace="ns1",node="node1",pod="pod1"} 0.3
				kube_pod_container_resource_limits_cpu_cores{container="pod2_con1",namespace="ns2",node="node2",pod="pod2"} 0.4
				kube_pod_container_resource_limits_cpu_cores{container="pod2_con2",namespace="ns2",node="node2",pod="pod2"} 0.5
				kube_pod_container_resource_limits_memory_bytes{container="pod1_con1",namespace="ns1",node="node1",pod="pod1"} 1e+08
				kube_pod_container_resource_limits_memory_bytes{container="pod1_con2",namespace="ns1",node="node1",pod="pod1"} 2e+08
				kube_pod_container_resource_limits_memory_bytes{container="pod2_con1",namespace="ns2",node="node2",pod="pod2"} 3e+08
				kube_pod_container_resource_limits_memory_bytes{container="pod2_con2",namespace="ns2",node="node2",pod="pod2"} 4e+08
		`,
			metrics: []string{
				"kube_pod_container_resource_requests_cpu_cores",
				"kube_pod_container_resource_requests_memory_bytes",
				"kube_pod_container_resource_limits_cpu_cores",
				"kube_pod_container_resource_limits_memory_bytes",
			},
		}, {
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
						Labels: map[string]string{
							"app": "example",
						},
					},
					Spec: v1.PodSpec{},
				},
			},
			want: metadata + `
				kube_pod_labels{label_app="example",namespace="ns1",pod="pod1"} 1
		`,
			metrics: []string{
				"kube_pod_labels",
			},
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
