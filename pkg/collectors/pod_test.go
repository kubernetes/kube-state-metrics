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
	"fmt"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/pkg/collectors/testutils"
	"k8s.io/kube-state-metrics/pkg/options"
	"k8s.io/kubernetes/pkg/util/node"
	"strings"
	"testing"
	"text/template"
	"time"
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
		# HELP kube_pod_created Unix creation timestamp
		# TYPE kube_pod_created gauge
		# HELP kube_pod_container_info Information about a container in a pod.
		# TYPE kube_pod_container_info gauge
		# HELP kube_pod_init_container_info Information about an init container in a pod.
		# TYPE kube_pod_init_container_info gauge
		# HELP kube_pod_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_pod_labels gauge
		# HELP kube_pod_container_status_ready Describes whether the containers readiness check succeeded.
		# TYPE kube_pod_container_status_ready gauge
		# HELP kube_pod_container_status_restarts_total The number of container restarts per container.
		# TYPE kube_pod_container_status_restarts_total counter
		# HELP kube_pod_init_container_status_restarts_total The number of init container restarts per container.
		# TYPE kube_pod_init_container_status_restarts_total counter
		# HELP kube_pod_container_status_running Describes whether the container is currently in running state.
		# TYPE kube_pod_container_status_running gauge
		# HELP kube_pod_init_container_status_running Describes whether the init container is currently in running state.
		# TYPE kube_pod_init_container_status_running gauge
		# HELP kube_pod_container_status_terminated Describes whether the container is currently in terminated state.
		# TYPE kube_pod_container_status_terminated gauge
		# HELP kube_pod_init_container_status_terminated Describes whether the init container is currently in terminated state.
		# TYPE kube_pod_init_container_status_terminated gauge
		# HELP kube_pod_container_status_terminated_reason Describes the reason the container is currently in terminated state.
		# TYPE kube_pod_container_status_terminated_reason gauge
		# HELP kube_pod_container_status_last_terminated_reason Describes the last reason the container was in terminated state.
		# TYPE kube_pod_container_status_last_terminated_reason gauge
		# HELP kube_pod_init_container_status_terminated_reason Describes the reason the init container is currently in terminated state.
		# TYPE kube_pod_init_container_status_terminated_reason gauge
		# HELP kube_pod_container_status_waiting Describes whether the container is currently in waiting state.
		# TYPE kube_pod_container_status_waiting gauge
		# HELP kube_pod_init_container_status_waiting Describes whether the init container is currently in waiting state.
		# TYPE kube_pod_init_container_status_waiting gauge
		# HELP kube_pod_container_status_waiting_reason Describes the reason the container is currently in waiting state.
		# TYPE kube_pod_container_status_waiting_reason gauge
		# HELP kube_pod_init_container_status_waiting_reason Describes the reason the init container is currently in waiting state.
		# TYPE kube_pod_init_container_status_waiting_reason gauge
		# HELP kube_pod_info Information about pod.
		# TYPE kube_pod_info gauge
		# HELP kube_pod_status_scheduled_time Unix timestamp when pod moved into scheduled status
		# TYPE kube_pod_status_scheduled_time gauge
		# HELP kube_pod_start_time Start time in unix timestamp for a pod.
		# TYPE kube_pod_start_time gauge
		# HELP kube_pod_completion_time Completion time in unix timestamp for a pod.
		# TYPE kube_pod_completion_time gauge
		# HELP kube_pod_owner Information about the Pod's owner.
		# TYPE kube_pod_owner gauge
		# HELP kube_pod_status_phase The pods current phase.
		# TYPE kube_pod_status_phase gauge
		# HELP kube_pod_status_ready Describes whether the pod is ready to serve requests.
		# TYPE kube_pod_status_ready gauge
		# HELP kube_pod_status_scheduled Describes the status of the scheduling process for the pod.
		# TYPE kube_pod_status_scheduled gauge
		# HELP kube_pod_container_resource_requests The number of requested request resource by a container.
		# TYPE kube_pod_container_resource_requests gauge
		# HELP kube_pod_container_resource_limits The number of requested limit resource by a container.
		# TYPE kube_pod_container_resource_limits gauge
		# HELP kube_pod_container_resource_requests_cpu_cores The number of requested cpu cores by a container.
		# TYPE kube_pod_container_resource_requests_cpu_cores gauge
		# HELP kube_pod_container_resource_requests_memory_bytes The number of requested memory bytes by a container.
		# TYPE kube_pod_container_resource_requests_memory_bytes gauge
		# HELP kube_pod_container_resource_limits_cpu_cores The limit on cpu cores to be used by a container.
		# TYPE kube_pod_container_resource_limits_cpu_cores gauge
		# HELP kube_pod_container_resource_limits_memory_bytes The limit on memory to be used by a container in bytes.
		# TYPE kube_pod_container_resource_limits_memory_bytes gauge
		# HELP kube_pod_spec_volumes_persistentvolumeclaims_info Information about persistentvolumeclaim volumes in a pod.
		# TYPE kube_pod_spec_volumes_persistentvolumeclaims_info gauge
		# HELP kube_pod_spec_volumes_persistentvolumeclaims_readonly Describes whether a persistentvolumeclaim is mounted read only.
		# TYPE kube_pod_spec_volumes_persistentvolumeclaims_readonly gauge
	`

	type csToProm func(cs v1.ContainerStatus, ns string, pod string) string

	podToProm := func(p v1.Pod, fn_cs csToProm, fn_ics csToProm) string {
		var sb strings.Builder
		if p.Status.ContainerStatuses != nil {
			for _, cs := range p.Status.ContainerStatuses {
				sb.WriteString(fn_cs(cs, p.Namespace, p.Name))
				sb.WriteString("\n")
			}
		}

		if p.Status.InitContainerStatuses != nil {
			for _, ics := range p.Status.InitContainerStatuses {
				sb.WriteString(fn_ics(ics, p.Namespace, p.Name))
				sb.WriteString("\n")
			}
		}
		return sb.String()
	}

	mkCs := func(name string, id int, ready bool, rc int32) v1.ContainerStatus {
		return v1.ContainerStatus{
			Name:         fmt.Sprintf("%s%d", name, id),
			Image:        fmt.Sprintf("k8s.gcr.io/hyperkube%d", id),
			ImageID:      "docker://sha256:aaa",
			ContainerID:  "docker://ab123",
			Ready:        ready,
			RestartCount: rc,
		}
	}

	mkPod := func(cses []v1.ContainerStatus, icses []v1.ContainerStatus, name string, ns string) v1.Pod {
		return v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Status: v1.PodStatus{
				ContainerStatuses:     cses,
				InitContainerStatuses: icses,
			},
		}
	}

	mkPod1 := func(cs v1.ContainerStatus, ics v1.ContainerStatus) v1.Pod {
		return mkPod([]v1.ContainerStatus{cs}, []v1.ContainerStatus{ics}, "pod1", "ns1")
	}

	mkPod2 := func(cses []v1.ContainerStatus, icses []v1.ContainerStatus) v1.Pod {
		return mkPod(cses, icses, "pod2", "ns2")
	}

	cs1 := mkCs("container", 1, true, 0)
	ics1 := mkCs("initcontainer", 1, true, 0)

	cs2 := mkCs("container", 2, true, 0)
	ics2 := mkCs("initcontainer", 2, true, 0)

	cs3 := mkCs("container", 3, false, 1)
	ics3 := mkCs("initcontainer", 3, false, 1)

	type Case struct {
		pods    []v1.Pod
		metrics []string
		want    string
	}

	mkCsInfoCase := func() Case {
		mkCsi := func(cs v1.ContainerStatus) v1.ContainerStatus {
			return v1.ContainerStatus{
				Name:        cs.Name,
				Image:       cs.Image,
				ImageID:     cs.ImageID,
				ContainerID: cs.ContainerID,
			}
		}

		p1 := mkPod1(mkCsi(cs1), mkCsi(ics1))
		p2 := mkPod2([]v1.ContainerStatus{mkCsi(cs2), mkCsi(cs3)}, []v1.ContainerStatus{mkCsi(ics2), mkCsi(ics3)})

		csToProm := func(cs v1.ContainerStatus, ns string, pod string) string {
			return fmt.Sprintf(`kube_pod_container_info{container="%s",container_id="%s",image="%s",image_id="%s",namespace="%s",pod="%s"} 1`, cs.Name, cs.ContainerID, cs.Image, cs.ImageID, ns, pod)
		}

		icsToProm := func(cs v1.ContainerStatus, ns string, pod string) string {
			return fmt.Sprintf(`kube_pod_init_container_info{container="%s",container_id="%s",image="%s",image_id="%s",namespace="%s",pod="%s"} 1`, cs.Name, cs.ContainerID, cs.Image, cs.ImageID, ns, pod)
		}

		return Case{
			pods: []v1.Pod{
				p1, p2,
			},
			want:    metadata + podToProm(p1, csToProm, icsToProm) + podToProm(p2, csToProm, icsToProm),
			metrics: []string{"kube_pod_container_info", "kube_pod_init_container_info"},
		}
	}

	// ready
	mkCsrCase := func() Case {
		mk_csr := func(cs v1.ContainerStatus) v1.ContainerStatus {
			return v1.ContainerStatus{
				Name:  cs.Name,
				Ready: cs.Ready,
			}
		}

		return Case{
			pods: []v1.Pod{
				mkPod1(mk_csr(cs1), mk_csr(ics1)),
				mkPod2([]v1.ContainerStatus{mk_csr(cs2), mk_csr(cs3)}, []v1.ContainerStatus{mk_csr(ics2), mk_csr(ics3)}),
			},
			want: metadata + `
                   kube_pod_container_status_ready{container="container1",namespace="ns1",pod="pod1"} 1
                   kube_pod_container_status_ready{container="container2",namespace="ns2",pod="pod2"} 1
                   kube_pod_container_status_ready{container="container3",namespace="ns2",pod="pod2"} 0
                   `,
			metrics: []string{"kube_pod_container_status_ready"},
		}
	}

	mkCsRcCase := func() Case {
		mk_cs_rc := func(cs v1.ContainerStatus) v1.ContainerStatus {
			return v1.ContainerStatus{
				Name:         cs.Name,
				RestartCount: cs.RestartCount,
			}
		}

		p1 := mkPod1(mk_cs_rc(cs1), mk_cs_rc(ics1))
		p2 := mkPod2([]v1.ContainerStatus{mk_cs_rc(cs2), mk_cs_rc(cs3)}, []v1.ContainerStatus{mk_cs_rc(ics2), mk_cs_rc(ics3)})

		csToProm := func(cs v1.ContainerStatus, ns string, pod string) string {
			return fmt.Sprintf(`kube_pod_container_status_restarts_total{container="%s",namespace="%s",pod="%s"} %d`, cs.Name, ns, pod, cs.RestartCount)
		}

		icsToProm := func(cs v1.ContainerStatus, ns string, pod string) string {
			return fmt.Sprintf(`kube_pod_init_container_status_restarts_total{container="%s",namespace="%s",pod="%s"} %d`, cs.Name, ns, pod, cs.RestartCount)
		}

		return Case{
			pods: []v1.Pod{
				p1, p2,
			},
			want:    metadata + podToProm(p1, csToProm, icsToProm) + podToProm(p2, csToProm, icsToProm),
			metrics: []string{"kube_pod_container_status_restarts_total", "kube_pod_init_container_status_restarts_total"},
		}
	}

	mkCsAllCase := func() Case {
		p1 := mkPod(
			[]v1.ContainerStatus{
				v1.ContainerStatus{
					Name: "container1",
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
				},
			},
			[]v1.ContainerStatus{
				v1.ContainerStatus{
					Name: "init_container1",
					State: v1.ContainerState{
						Terminated: &v1.ContainerStateTerminated{
							Reason: "Completed",
						},
					},
				},
			}, "pod1", "ns1")

		p2 := mkPod(
			[]v1.ContainerStatus{
				v1.ContainerStatus{
					Name: "container2",
					State: v1.ContainerState{
						Terminated: &v1.ContainerStateTerminated{
							Reason: "OOMKilled",
						},
					},
				},
				v1.ContainerStatus{
					Name: "container3",
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason: "ContainerCreating",
						},
					},
				},
			},
			[]v1.ContainerStatus{}, "pod2", "ns2")

		p3 := mkPod(
			[]v1.ContainerStatus{
				v1.ContainerStatus{
					Name: "container4",
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
			[]v1.ContainerStatus{}, "pod3", "ns3")

		p4 := mkPod(
			[]v1.ContainerStatus{
				v1.ContainerStatus{
					Name: "container5",
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason: "ImagePullBackOff",
						},
					},
				},
			},
			[]v1.ContainerStatus{}, "pod4", "ns4")

		p5 := mkPod(
			[]v1.ContainerStatus{
				v1.ContainerStatus{
					Name: "container6",
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason: "ErrImagePull",
						},
					},
				},
			},
			[]v1.ContainerStatus{}, "pod5", "ns5")

		p7 := mkPod(
			[]v1.ContainerStatus{
				v1.ContainerStatus{
					Name: "container7",
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
			[]v1.ContainerStatus{}, "pod7", "ns7")

		p6 := mkPod(
			[]v1.ContainerStatus{},
			[]v1.ContainerStatus{
				v1.ContainerStatus{
					Name: "init_container2",
					State: v1.ContainerState{
						Terminated: &v1.ContainerStateTerminated{
							Reason: "OOMKilled",
						},
					},
				},
				v1.ContainerStatus{
					Name: "init_container3",
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason: "ContainerCreating",
						},
					},
				},
				v1.ContainerStatus{
					Name: "init_container4",
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason: "ErrImagePull",
						},
					},
				},
				v1.ContainerStatus{
					Name: "init_container5",
					State: v1.ContainerState{
						Terminated: &v1.ContainerStateTerminated{
							Reason: "Error",
						},
					},
				},
			}, "pod6", "ns3")

		toProm := func(cs v1.ContainerStatus, ns string, pod string, typ string) string {
			data := map[string]interface{}{
				"Name":       cs.Name,
				"Pod":        pod,
				"Ns":         ns,
				"Type":       typ,
				"Running":    0,
				"Terminated": 0,
				"TR_C":       0,
				"TR_CCR":     0,
				"TR_E":       0,
				"TR_O":       0,
				"Waiting":    0,
				"WR_CC":      0,
				"WR_IP":      0,
				"WR_CL":      0,
				"WR_EI":      0,
				"LTR_C":      0,
				"LTR_CCR":    0,
				"LTR_E":      0,
				"LTR_O":      0,
			}

			if cs.State.Running != nil {
				data["Running"] = 1
			} else if cs.State.Terminated != nil {
				data["Terminated"] = 1
				switch cs.State.Terminated.Reason {
				case "Completed":
					data["TR_C"] = 1
				case "ContainerCannotRun":
					data["TR_CCR"] = 1
				case "Error":
					data["TR_E"] = 1
				case "OOMKilled":
					data["TR_O"] = 1
				}
			} else if cs.State.Waiting != nil {
				data["Waiting"] = 1
				switch cs.State.Waiting.Reason {
				case "ContainerCreating":
					data["WR_CC"] = 1
				case "ImagePullBackOff":
					data["WR_IP"] = 1
				case "CrashLoopBackOff":
					data["WR_CL"] = 1
				case "ErrImagePull":
					data["WR_EI"] = 1
				}
			}

			if cs.LastTerminationState.Terminated != nil {
				switch cs.LastTerminationState.Terminated.Reason {
				case "Completed":
					data["LTR_C"] = 1
				case "ContainerCannotRun":
					data["LTR_CCR"] = 1
				case "Error":
					data["LTR_E"] = 1
				case "OOMKilled":
					data["LTR_O"] = 1
				}
			}

			tmpl := `
				kube_pod_{{ .Type }}_status_running{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}"} {{ .Running }}
				kube_pod_{{ .Type }}_status_terminated{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}"} {{ .Terminated }}
				kube_pod_{{ .Type }}_status_terminated_reason{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}",reason="Completed"} {{ .TR_C }}
				kube_pod_{{ .Type }}_status_terminated_reason{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}",reason="ContainerCannotRun"} {{ .TR_CCR }}
				kube_pod_{{ .Type }}_status_terminated_reason{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}",reason="Error"} {{ .TR_E }}
				kube_pod_{{ .Type }}_status_terminated_reason{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}",reason="OOMKilled"} {{ .TR_O }}
				kube_pod_{{ .Type }}_status_waiting{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}"} {{ .Waiting }}
				kube_pod_{{ .Type }}_status_waiting_reason{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}",reason="ContainerCreating"} {{ .WR_CC }}
				kube_pod_{{ .Type }}_status_waiting_reason{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}",reason="ImagePullBackOff"} {{ .WR_IP }}
				kube_pod_{{ .Type }}_status_waiting_reason{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}",reason="CrashLoopBackOff"} {{ .WR_CL }}
				kube_pod_{{ .Type }}_status_waiting_reason{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}",reason="ErrImagePull"} {{ .WR_EI }}
            `
			if typ == "container" {
				tmpl = tmpl + `
                kube_pod_{{ .Type }}_status_last_terminated_reason{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}",reason="Completed"} {{ .LTR_C }}
				kube_pod_{{ .Type }}_status_last_terminated_reason{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}",reason="ContainerCannotRun"} {{ .LTR_CCR }}
				kube_pod_{{ .Type }}_status_last_terminated_reason{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}",reason="Error"} {{ .LTR_E }}
				kube_pod_{{ .Type }}_status_last_terminated_reason{container="{{ .Name }}",namespace="{{ .Ns }}",pod="{{ .Pod }}",reason="OOMKilled"} {{ .LTR_O }}
                `
			}

			t := template.Must(template.New("prom").Parse(tmpl))
			builder := &strings.Builder{}
			if err := t.Execute(builder, data); err != nil {
				panic(err)
			}
			return builder.String()
		}

		csToProm := func(cs v1.ContainerStatus, ns string, pod string) string {
			return toProm(cs, ns, pod, "container")
		}

		icsToProm := func(cs v1.ContainerStatus, ns string, pod string) string {
			return toProm(cs, ns, pod, "init_container")
		}

		return Case{
			pods: []v1.Pod{
				p1, p2, p3, p4, p5, p6, p7,
			},
			want: metadata +
				podToProm(p1, csToProm, icsToProm) +
				podToProm(p2, csToProm, icsToProm) +
				podToProm(p3, csToProm, icsToProm) +
				podToProm(p4, csToProm, icsToProm) +
				podToProm(p5, csToProm, icsToProm) +
				podToProm(p6, csToProm, icsToProm) +
				podToProm(p7, csToProm, icsToProm),
			metrics: []string{
				"kube_pod_container_status_running",
				"kube_pod_container_status_waiting",
				"kube_pod_container_status_waiting_reason",
				"kube_pod_container_status_terminated",
				"kube_pod_container_status_terminated_reason",
				"kube_pod_container_status_last_terminated_reason",
				"kube_pod_init_container_status_running",
				"kube_pod_init_container_status_waiting",
				"kube_pod_init_container_status_waiting_reason",
				"kube_pod_init_container_status_terminated",
				"kube_pod_init_container_status_terminated_reason",
			},
		}
	}

	cases := []Case{
		mkCsInfoCase(),
		mkCsrCase(),
		mkCsRcCase(),
		mkCsAllCase(),
		{
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "pod1",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
						Namespace:         "ns1",
						UID:               "abc-123-xxx",
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
							v1.ContainerStatus{
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
							v1.ContainerStatus{
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
							v1.ContainerStatus{
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
			},
			want: metadata + `
				kube_pod_created{namespace="ns1",pod="pod1"} 1.5e+09
				kube_pod_info{created_by_kind="<none>",created_by_name="<none>",host_ip="1.1.1.1",namespace="ns1",pod="pod1",node="node1",pod_ip="1.2.3.4",uid="abc-123-xxx"} 1
				kube_pod_info{created_by_kind="ReplicaSet",created_by_name="rs-name",host_ip="1.1.1.1",namespace="ns2",pod="pod2",node="node2",pod_ip="2.3.4.5",uid="abc-456-xxx"} 1
				kube_pod_start_time{namespace="ns1",pod="pod1"} 1501569018
				kube_pod_completion_time{namespace="ns2",pod="pod2"} 1501888018
				kube_pod_owner{namespace="ns1",pod="pod1",owner_kind="<none>",owner_name="<none>",owner_is_controller="<none>"} 1
				kube_pod_owner{namespace="ns2",pod="pod2",owner_kind="ReplicaSet",owner_name="rs-name",owner_is_controller="true"} 1
				`,
			metrics: []string{"kube_pod_created", "kube_pod_info", "kube_pod_start_time", "kube_pod_completion_time", "kube_pod_owner"},
		}, {
			pods: []v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
					Status: v1.PodStatus{
						Phase: v1.PodRunning,
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
					Status: v1.PodStatus{
						Phase: v1.PodPending,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod3",
						Namespace: "ns3",
					},
					Status: v1.PodStatus{
						Phase: v1.PodUnknown,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "pod4",
						Namespace:         "ns4",
						DeletionTimestamp: &metav1.Time{},
					},
					Status: v1.PodStatus{
						Phase:  v1.PodRunning,
						Reason: node.NodeUnreachablePodReason,
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
				kube_pod_status_phase{namespace="ns3",phase="Failed",pod="pod3"} 0
				kube_pod_status_phase{namespace="ns3",phase="Pending",pod="pod3"} 0
				kube_pod_status_phase{namespace="ns3",phase="Running",pod="pod3"} 0
				kube_pod_status_phase{namespace="ns3",phase="Succeeded",pod="pod3"} 0
				kube_pod_status_phase{namespace="ns3",phase="Unknown",pod="pod3"} 1
				kube_pod_status_phase{namespace="ns4",phase="Failed",pod="pod4"} 0
				kube_pod_status_phase{namespace="ns4",phase="Pending",pod="pod4"} 0
				kube_pod_status_phase{namespace="ns4",phase="Running",pod="pod4"} 0
				kube_pod_status_phase{namespace="ns4",phase="Succeeded",pod="pod4"} 0
				kube_pod_status_phase{namespace="ns4",phase="Unknown",pod="pod4"} 1
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
								LastTransitionTime: metav1.Time{
									Time: time.Unix(1501666018, 0),
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
				kube_pod_status_scheduled_time{namespace="ns1",pod="pod1"} 1.501666018e+09
				kube_pod_status_scheduled{condition="false",namespace="ns1",pod="pod1"} 0
				kube_pod_status_scheduled{condition="false",namespace="ns2",pod="pod2"} 1
				kube_pod_status_scheduled{condition="true",namespace="ns1",pod="pod1"} 1
				kube_pod_status_scheduled{condition="true",namespace="ns2",pod="pod2"} 0
				kube_pod_status_scheduled{condition="unknown",namespace="ns1",pod="pod1"} 0
				kube_pod_status_scheduled{condition="unknown",namespace="ns2",pod="pod2"} 0
			`,
			metrics: []string{"kube_pod_status_scheduled", "kube_pod_status_scheduled_time"},
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
				kube_pod_container_resource_requests{container="pod1_con1",namespace="ns1",node="node1",pod="pod1",resource="cpu",unit="core"} 0.2
				kube_pod_container_resource_requests{container="pod1_con2",namespace="ns1",node="node1",pod="pod1",resource="cpu",unit="core"} 0.3
				kube_pod_container_resource_requests{container="pod1_con1",namespace="ns1",node="node1",pod="pod1",resource="nvidia_com_gpu",unit="integer"} 1
				kube_pod_container_resource_requests{container="pod2_con1",namespace="ns2",node="node2",pod="pod2",resource="cpu",unit="core"} 0.4
				kube_pod_container_resource_requests{container="pod2_con2",namespace="ns2",node="node2",pod="pod2",resource="cpu",unit="core"} 0.5
				kube_pod_container_resource_requests{container="pod1_con1",namespace="ns1",node="node1",pod="pod1",resource="memory",unit="byte"} 1e+08
				kube_pod_container_resource_requests{container="pod1_con2",namespace="ns1",node="node1",pod="pod1",resource="memory",unit="byte"} 2e+08
				kube_pod_container_resource_requests{container="pod1_con1",namespace="ns1",node="node1",pod="pod1",resource="storage",unit="byte"} 4e+08
				kube_pod_container_resource_requests{container="pod1_con1",namespace="ns1",node="node1",pod="pod1",resource="ephemeral_storage",unit="byte"} 3e+08
				kube_pod_container_resource_requests{container="pod2_con1",namespace="ns2",node="node2",pod="pod2",resource="memory",unit="byte"} 3e+08
				kube_pod_container_resource_requests{container="pod2_con2",namespace="ns2",node="node2",pod="pod2",resource="memory",unit="byte"} 4e+08
				kube_pod_container_resource_limits{container="pod1_con1",namespace="ns1",node="node1",pod="pod1",resource="cpu",unit="core"} 0.2
				kube_pod_container_resource_limits{container="pod1_con1",namespace="ns1",node="node1",pod="pod1",resource="nvidia_com_gpu",unit="integer"} 1
				kube_pod_container_resource_limits{container="pod1_con2",namespace="ns1",node="node1",pod="pod1",resource="cpu",unit="core"} 0.3
				kube_pod_container_resource_limits{container="pod2_con1",namespace="ns2",node="node2",pod="pod2",resource="cpu",unit="core"} 0.4
				kube_pod_container_resource_limits{container="pod2_con2",namespace="ns2",node="node2",pod="pod2",resource="cpu",unit="core"} 0.5
				kube_pod_container_resource_limits{container="pod1_con1",namespace="ns1",node="node1",pod="pod1",resource="memory",unit="byte"} 1e+08
				kube_pod_container_resource_limits{container="pod1_con2",namespace="ns1",node="node1",pod="pod1",resource="memory",unit="byte"} 2e+08
				kube_pod_container_resource_limits{container="pod1_con1",namespace="ns1",node="node1",pod="pod1",resource="storage",unit="byte"} 4e+08
				kube_pod_container_resource_limits{container="pod1_con1",namespace="ns1",node="node1",pod="pod1",resource="ephemeral_storage",unit="byte"} 3e+08
				kube_pod_container_resource_limits{container="pod2_con1",namespace="ns2",node="node2",pod="pod2",resource="memory",unit="byte"} 3e+08
				kube_pod_container_resource_limits{container="pod2_con2",namespace="ns2",node="node2",pod="pod2",resource="memory",unit="byte"} 4e+08
		`,
			metrics: []string{
				"kube_pod_container_resource_requests_cpu_cores",
				"kube_pod_container_resource_requests_memory_bytes",
				"kube_pod_container_resource_limits_cpu_cores",
				"kube_pod_container_resource_limits_memory_bytes",
				"kube_pod_container_resource_requests",
				"kube_pod_container_resource_limits",
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
					Spec: v1.PodSpec{
						Volumes: []v1.Volume{
							v1.Volume{
								Name: "myvol",
								VolumeSource: v1.VolumeSource{
									PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
										ClaimName: "claim1",
										ReadOnly:  false,
									},
								},
							},
							v1.Volume{
								Name: "my-readonly-vol",
								VolumeSource: v1.VolumeSource{
									PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
										ClaimName: "claim2",
										ReadOnly:  true,
									},
								},
							},
							v1.Volume{
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
			},
			want: metadata + `
				kube_pod_spec_volumes_persistentvolumeclaims_info{namespace="ns1",persistentvolumeclaim="claim1",pod="pod1",volume="myvol"} 1
				kube_pod_spec_volumes_persistentvolumeclaims_info{namespace="ns1",persistentvolumeclaim="claim2",pod="pod1",volume="my-readonly-vol"} 1
				kube_pod_spec_volumes_persistentvolumeclaims_readonly{namespace="ns1",persistentvolumeclaim="claim1",pod="pod1",volume="myvol"} 0
				kube_pod_spec_volumes_persistentvolumeclaims_readonly{namespace="ns1",persistentvolumeclaim="claim2",pod="pod1",volume="my-readonly-vol"} 1

		`,
			metrics: []string{
				"kube_pod_spec_volumes_persistentvolumeclaims_info",
				"kube_pod_spec_volumes_persistentvolumeclaims_readonly",
			},
		}}
	for _, c := range cases {
		pc := &podCollector{
			store: mockPodStore{
				f: func() ([]v1.Pod, error) { return c.pods, nil },
			},
			opts: &options.Options{},
		}
		if err := testutils.GatherAndCompare(pc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
