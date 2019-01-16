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

package collector

import (
	"strconv"

	"k8s.io/kube-state-metrics/pkg/constant"
	"k8s.io/kube-state-metrics/pkg/metric"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/apis/core/v1/helper"
	"k8s.io/kubernetes/pkg/util/node"
)

var (
	descPodLabelsDefaultLabels = []string{"namespace", "pod"}
	containerWaitingReasons    = []string{"ContainerCreating", "CrashLoopBackOff", "CreateContainerConfigError", "ErrImagePull", "ImagePullBackOff"}
	containerTerminatedReasons = []string{"OOMKilled", "Completed", "Error", "ContainerCannotRun"}

	podMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_pod_info",
			Type: metric.MetricTypeGauge,
			Help: "Information about pod.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				createdBy := metav1.GetControllerOf(p)
				createdByKind := "<none>"
				createdByName := "<none>"
				if createdBy != nil {
					if createdBy.Kind != "" {
						createdByKind = createdBy.Kind
					}
					if createdBy.Name != "" {
						createdByName = createdBy.Name
					}
				}

				m := metric.Metric{

					LabelKeys:   []string{"host_ip", "pod_ip", "uid", "node", "created_by_kind", "created_by_name"},
					LabelValues: []string{p.Status.HostIP, p.Status.PodIP, string(p.UID), p.Spec.NodeName, createdByKind, createdByName},
					Value:       1,
				}

				return metric.Family{
					Metrics: []*metric.Metric{&m},
				}
			}),
		},
		{
			Name: "kube_pod_start_time",
			Type: metric.MetricTypeGauge,
			Help: "Start time in unix timestamp for a pod.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				if p.Status.StartTime != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64((*(p.Status.StartTime)).Unix()),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_completion_time",
			Type: metric.MetricTypeGauge,
			Help: "Completion time in unix timestamp for a pod.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				var lastFinishTime float64
				for _, cs := range p.Status.ContainerStatuses {
					if cs.State.Terminated != nil {
						if lastFinishTime == 0 || lastFinishTime < float64(cs.State.Terminated.FinishedAt.Unix()) {
							lastFinishTime = float64(cs.State.Terminated.FinishedAt.Unix())
						}
					}
				}

				if lastFinishTime > 0 {
					ms = append(ms, &metric.Metric{

						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       lastFinishTime,
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_owner",
			Type: metric.MetricTypeGauge,
			Help: "Information about the Pod's owner.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				labelKeys := []string{"owner_kind", "owner_name", "owner_is_controller"}
				ms := []*metric.Metric{}

				owners := p.GetOwnerReferences()
				if len(owners) == 0 {
					ms = append(ms, &metric.Metric{
						LabelKeys:   labelKeys,
						LabelValues: []string{"<none>", "<none>", "<none>"},
						Value:       1,
					})
				} else {
					for _, owner := range owners {
						if owner.Controller != nil {
							ms = append(ms, &metric.Metric{
								LabelKeys:   labelKeys,
								LabelValues: []string{owner.Kind, owner.Name, strconv.FormatBool(*owner.Controller)},
								Value:       1,
							})
						} else {
							ms = append(ms, &metric.Metric{
								LabelKeys:   labelKeys,
								LabelValues: []string{owner.Kind, owner.Name, "false"},
								Value:       1,
							})
						}
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_labels",
			Type: metric.MetricTypeGauge,
			Help: "Kubernetes labels converted to Prometheus labels.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(p.Labels)
				m := metric.Metric{
					LabelKeys:   labelKeys,
					LabelValues: labelValues,
					Value:       1,
				}
				return metric.Family{
					Metrics: []*metric.Metric{&m},
				}
			}),
		},
		{
			Name: "kube_pod_created",
			Type: metric.MetricTypeGauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				if !p.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(p.CreationTimestamp.Unix()),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_status_scheduled_time",
			Type: metric.MetricTypeGauge,
			Help: "Unix timestamp when pod moved into scheduled status",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, c := range p.Status.Conditions {
					switch c.Type {
					case v1.PodScheduled:
						if c.Status == v1.ConditionTrue {
							ms = append(ms, &metric.Metric{
								LabelKeys:   []string{},
								LabelValues: []string{},
								Value:       float64(c.LastTransitionTime.Unix()),
							})
						}
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_status_phase",
			Type: metric.MetricTypeGauge,
			Help: "The pods current phase.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				phase := p.Status.Phase
				if phase == "" {
					return metric.Family{
						Metrics: []*metric.Metric{},
					}
				}

				phases := []struct {
					v bool
					n string
				}{
					{phase == v1.PodPending, string(v1.PodPending)},
					{phase == v1.PodSucceeded, string(v1.PodSucceeded)},
					{phase == v1.PodFailed, string(v1.PodFailed)},
					// This logic is directly copied from: https://github.com/kubernetes/kubernetes/blob/d39bfa0d138368bbe72b0eaf434501dcb4ec9908/pkg/printers/internalversion/printers.go#L597-L601
					// For more info, please go to: https://github.com/kubernetes/kube-state-metrics/issues/410
					{phase == v1.PodRunning && !(p.DeletionTimestamp != nil && p.Status.Reason == node.NodeUnreachablePodReason), string(v1.PodRunning)},
					{phase == v1.PodUnknown || (p.DeletionTimestamp != nil && p.Status.Reason == node.NodeUnreachablePodReason), string(v1.PodUnknown)},
				}

				for _, p := range phases {
					ms = append(ms, &metric.Metric{

						LabelKeys:   []string{"phase"},
						LabelValues: []string{p.n},
						Value:       boolFloat64(p.v),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_status_ready",
			Type: metric.MetricTypeGauge,
			Help: "Describes whether the pod is ready to serve requests.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, c := range p.Status.Conditions {
					switch c.Type {
					case v1.PodReady:
						conditionMetrics := addConditionMetrics(c.Status)

						for _, m := range conditionMetrics {
							metric := m
							metric.LabelKeys = []string{"condition"}
							ms = append(ms, metric)
						}
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_status_scheduled",
			Type: metric.MetricTypeGauge,
			Help: "Describes the status of the scheduling process for the pod.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, c := range p.Status.Conditions {
					switch c.Type {
					case v1.PodScheduled:
						conditionMetrics := addConditionMetrics(c.Status)

						for _, m := range conditionMetrics {
							metric := m
							metric.LabelKeys = []string{"condition"}
							ms = append(ms, metric)
						}
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_info",
			Type: metric.MetricTypeGauge,
			Help: "Information about a container in a pod.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}
				labelKeys := []string{"container", "image", "image_id", "container_id"}

				for _, cs := range p.Status.ContainerStatuses {
					ms = append(ms, &metric.Metric{
						LabelKeys:   labelKeys,
						LabelValues: []string{cs.Name, cs.Image, cs.ImageID, cs.ContainerID},
						Value:       1,
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_status_waiting",
			Type: metric.MetricTypeGauge,
			Help: "Describes whether the container is currently in waiting state.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, cs := range p.Status.ContainerStatuses {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container"},
						LabelValues: []string{cs.Name},
						Value:       boolFloat64(cs.State.Waiting != nil),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_status_waiting_reason",
			Type: metric.MetricTypeGauge,
			Help: "Describes the reason the container is currently in waiting state.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, cs := range p.Status.ContainerStatuses {
					for _, reason := range containerWaitingReasons {
						ms = append(ms, &metric.Metric{
							LabelKeys:   []string{"container", "reason"},
							LabelValues: []string{cs.Name, reason},
							Value:       boolFloat64(waitingReason(cs, reason)),
						})
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_status_running",
			Type: metric.MetricTypeGauge,
			Help: "Describes whether the container is currently in running state.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, cs := range p.Status.ContainerStatuses {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container"},
						LabelValues: []string{cs.Name},
						Value:       boolFloat64(cs.State.Running != nil),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_status_terminated",
			Type: metric.MetricTypeGauge,
			Help: "Describes whether the container is currently in terminated state.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, cs := range p.Status.ContainerStatuses {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container"},
						LabelValues: []string{cs.Name},
						Value:       boolFloat64(cs.State.Terminated != nil),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_status_terminated_reason",
			Type: metric.MetricTypeGauge,
			Help: "Describes the reason the container is currently in terminated state.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, cs := range p.Status.ContainerStatuses {
					for _, reason := range containerTerminatedReasons {
						ms = append(ms, &metric.Metric{
							LabelKeys:   []string{"container", "reason"},
							LabelValues: []string{cs.Name, reason},
							Value:       boolFloat64(terminationReason(cs, reason)),
						})
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_status_last_terminated_reason",
			Type: metric.MetricTypeGauge,
			Help: "Describes the last reason the container was in terminated state.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, cs := range p.Status.ContainerStatuses {
					for _, reason := range containerTerminatedReasons {
						ms = append(ms, &metric.Metric{
							LabelKeys:   []string{"container", "reason"},
							LabelValues: []string{cs.Name, reason},
							Value:       boolFloat64(lastTerminationReason(cs, reason)),
						})
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_status_ready",
			Type: metric.MetricTypeGauge,
			Help: "Describes whether the containers readiness check succeeded.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, cs := range p.Status.ContainerStatuses {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container"},
						LabelValues: []string{cs.Name},
						Value:       boolFloat64(cs.Ready),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_status_restarts_total",
			Type: metric.MetricTypeCounter,
			Help: "The number of container restarts per container.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, cs := range p.Status.ContainerStatuses {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container"},
						LabelValues: []string{cs.Name},
						Value:       float64(cs.RestartCount),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_resource_requests",
			Type: metric.MetricTypeGauge,
			Help: "The number of requested request resource by a container.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, c := range p.Spec.Containers {
					req := c.Resources.Requests

					for resourceName, val := range req {
						switch resourceName {
						case v1.ResourceCPU:
							ms = append(ms, &metric.Metric{
								LabelValues: []string{c.Name, p.Spec.NodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitCore)},
								Value:       float64(val.MilliValue()) / 1000,
							})
						case v1.ResourceStorage:
							fallthrough
						case v1.ResourceEphemeralStorage:
							fallthrough
						case v1.ResourceMemory:
							ms = append(ms, &metric.Metric{
								LabelValues: []string{c.Name, p.Spec.NodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
								Value:       float64(val.Value()),
							})
						default:
							if helper.IsHugePageResourceName(resourceName) {
								ms = append(ms, &metric.Metric{
									LabelValues: []string{c.Name, p.Spec.NodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
									Value:       float64(val.Value()),
								})
							}
							if helper.IsAttachableVolumeResourceName(resourceName) {
								ms = append(ms, &metric.Metric{
									LabelValues: []string{c.Name, p.Spec.NodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
									Value:       float64(val.Value()),
								})
							}
							if helper.IsExtendedResourceName(resourceName) {
								ms = append(ms, &metric.Metric{
									LabelValues: []string{c.Name, p.Spec.NodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitInteger)},
									Value:       float64(val.Value()),
								})
							}
						}
					}
				}

				for _, metric := range ms {
					metric.LabelKeys = []string{"container", "node", "resource", "unit"}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_resource_limits",
			Type: metric.MetricTypeGauge,
			Help: "The number of requested limit resource by a container.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, c := range p.Spec.Containers {
					lim := c.Resources.Limits

					for resourceName, val := range lim {
						switch resourceName {
						case v1.ResourceCPU:
							ms = append(ms, &metric.Metric{
								Value:       float64(val.MilliValue()) / 1000,
								LabelValues: []string{c.Name, p.Spec.NodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitCore)},
							})
						case v1.ResourceStorage:
							fallthrough
						case v1.ResourceEphemeralStorage:
							fallthrough
						case v1.ResourceMemory:
							ms = append(ms, &metric.Metric{
								LabelValues: []string{c.Name, p.Spec.NodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
								Value:       float64(val.Value()),
							})
						default:
							if helper.IsHugePageResourceName(resourceName) {
								ms = append(ms, &metric.Metric{
									LabelValues: []string{c.Name, p.Spec.NodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
									Value:       float64(val.Value()),
								})
							}
							if helper.IsAttachableVolumeResourceName(resourceName) {
								ms = append(ms, &metric.Metric{
									Value:       float64(val.Value()),
									LabelValues: []string{c.Name, p.Spec.NodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
								})
							}
							if helper.IsExtendedResourceName(resourceName) {
								ms = append(ms, &metric.Metric{
									Value:       float64(val.Value()),
									LabelValues: []string{c.Name, p.Spec.NodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitInteger)},
								})
							}
						}
					}
				}

				for _, metric := range ms {
					metric.LabelKeys = []string{"container", "node", "resource", "unit"}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_resource_requests_cpu_cores",
			Type: metric.MetricTypeGauge,
			Help: "The number of requested cpu cores by a container.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, c := range p.Spec.Containers {
					req := c.Resources.Requests
					if cpu, ok := req[v1.ResourceCPU]; ok {
						ms = append(ms, &metric.Metric{
							LabelKeys:   []string{"container", "node"},
							LabelValues: []string{c.Name, p.Spec.NodeName},
							Value:       float64(cpu.MilliValue()) / 1000,
						})
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_resource_requests_memory_bytes",
			Type: metric.MetricTypeGauge,
			Help: "The number of requested memory bytes by a container.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, c := range p.Spec.Containers {
					req := c.Resources.Requests
					if mem, ok := req[v1.ResourceMemory]; ok {
						ms = append(ms, &metric.Metric{
							LabelKeys:   []string{"container", "node"},
							LabelValues: []string{c.Name, p.Spec.NodeName},
							Value:       float64(mem.Value()),
						})
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_resource_limits_cpu_cores",
			Type: metric.MetricTypeGauge,
			Help: "The limit on cpu cores to be used by a container.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, c := range p.Spec.Containers {
					lim := c.Resources.Limits
					if cpu, ok := lim[v1.ResourceCPU]; ok {
						ms = append(ms, &metric.Metric{
							LabelKeys:   []string{"container", "node"},
							LabelValues: []string{c.Name, p.Spec.NodeName},
							Value:       float64(cpu.MilliValue()) / 1000,
						})
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_container_resource_limits_memory_bytes",
			Type: metric.MetricTypeGauge,
			Help: "The limit on memory to be used by a container in bytes.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, c := range p.Spec.Containers {
					lim := c.Resources.Limits

					if mem, ok := lim[v1.ResourceMemory]; ok {
						ms = append(ms, &metric.Metric{
							LabelKeys:   []string{"container", "node"},
							LabelValues: []string{c.Name, p.Spec.NodeName},
							Value:       float64(mem.Value()),
						})
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_spec_volumes_persistentvolumeclaims_info",
			Type: metric.MetricTypeGauge,
			Help: "Information about persistentvolumeclaim volumes in a pod.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, v := range p.Spec.Volumes {
					if v.PersistentVolumeClaim != nil {
						ms = append(ms, &metric.Metric{
							LabelKeys:   []string{"volume", "persistentvolumeclaim"},
							LabelValues: []string{v.Name, v.PersistentVolumeClaim.ClaimName},
							Value:       1,
						})
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_pod_spec_volumes_persistentvolumeclaims_readonly",
			Type: metric.MetricTypeGauge,
			Help: "Describes whether a persistentvolumeclaim is mounted read only.",
			GenerateFunc: wrapPodFunc(func(p *v1.Pod) metric.Family {
				ms := []*metric.Metric{}

				for _, v := range p.Spec.Volumes {
					if v.PersistentVolumeClaim != nil {
						ms = append(ms, &metric.Metric{
							LabelKeys:   []string{"volume", "persistentvolumeclaim"},
							LabelValues: []string{v.Name, v.PersistentVolumeClaim.ClaimName},
							Value:       boolFloat64(v.PersistentVolumeClaim.ReadOnly),
						})
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
	}
)

func wrapPodFunc(f func(*v1.Pod) metric.Family) func(interface{}) metric.Family {
	return func(obj interface{}) metric.Family {
		pod := obj.(*v1.Pod)

		metricFamily := f(pod)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descPodLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{pod.Namespace, pod.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createPodListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Pods(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Pods(ns).Watch(opts)
		},
	}
}

func waitingReason(cs v1.ContainerStatus, reason string) bool {
	if cs.State.Waiting == nil {
		return false
	}
	return cs.State.Waiting.Reason == reason
}

func terminationReason(cs v1.ContainerStatus, reason string) bool {
	if cs.State.Terminated == nil {
		return false
	}
	return cs.State.Terminated.Reason == reason
}

func lastTerminationReason(cs v1.ContainerStatus, reason string) bool {
	if cs.LastTerminationState.Terminated == nil {
		return false
	}
	return cs.LastTerminationState.Terminated.Reason == reason
}
