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
	"context"
	"strconv"

	basemetrics "k8s.io/component-base/metrics"
	"k8s.io/utils/net"

	"k8s.io/kube-state-metrics/v2/pkg/constant"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descPodLabelsDefaultLabels = []string{"namespace", "pod", "uid"}
	podStatusReasons           = []string{"Evicted", "NodeAffinity", "NodeLost", "Shutdown", "UnexpectedAdmissionError"}
)

func podMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		createPodCompletionTimeFamilyGenerator(),
		createPodContainerInfoFamilyGenerator(),
		createPodContainerResourceLimitsFamilyGenerator(),
		createPodContainerResourceRequestsFamilyGenerator(),
		createPodContainerStateStartedFamilyGenerator(),
		createPodContainerStatusLastTerminatedReasonFamilyGenerator(),
		createPodContainerStatusLastTerminatedExitCodeFamilyGenerator(),
		createPodContainerStatusLastTerminatedTimestampFamilyGenerator(),
		createPodContainerStatusReadyFamilyGenerator(),
		createPodContainerStatusRestartsTotalFamilyGenerator(),
		createPodContainerStatusRunningFamilyGenerator(),
		createPodContainerStatusTerminatedFamilyGenerator(),
		createPodContainerStatusTerminatedReasonFamilyGenerator(),
		createPodContainerStatusWaitingFamilyGenerator(),
		createPodContainerStatusWaitingReasonFamilyGenerator(),
		createPodCreatedFamilyGenerator(),
		createPodDeletionTimestampFamilyGenerator(),
		createPodInfoFamilyGenerator(),
		createPodIPFamilyGenerator(),
		createPodInitContainerInfoFamilyGenerator(),
		createPodInitContainerResourceLimitsFamilyGenerator(),
		createPodInitContainerResourceRequestsFamilyGenerator(),
		createPodInitContainerStatusLastTerminatedReasonFamilyGenerator(),
		createPodInitContainerStatusReadyFamilyGenerator(),
		createPodInitContainerStatusRestartsTotalFamilyGenerator(),
		createPodInitContainerStatusRunningFamilyGenerator(),
		createPodInitContainerStatusTerminatedFamilyGenerator(),
		createPodInitContainerStatusTerminatedReasonFamilyGenerator(),
		createPodInitContainerStatusWaitingFamilyGenerator(),
		createPodInitContainerStatusWaitingReasonFamilyGenerator(),
		createPodAnnotationsGenerator(allowAnnotationsList),
		createPodLabelsGenerator(allowLabelsList),
		createPodOverheadCPUCoresFamilyGenerator(),
		createPodOverheadMemoryBytesFamilyGenerator(),
		createPodOwnerFamilyGenerator(),
		createPodRestartPolicyFamilyGenerator(),
		createPodRuntimeClassNameInfoFamilyGenerator(),
		createPodSpecVolumesPersistentVolumeClaimsInfoFamilyGenerator(),
		createPodSpecVolumesPersistentVolumeClaimsReadonlyFamilyGenerator(),
		createPodStartTimeFamilyGenerator(),
		createPodStatusPhaseFamilyGenerator(),
		createPodStatusQosClassFamilyGenerator(),
		createPodStatusReadyFamilyGenerator(),
		createPodStatusReadyTimeFamilyGenerator(),
		createPodStatusInitializedTimeFamilyGenerator(),
		createPodStatusContainerReadyTimeFamilyGenerator(),
		createPodStatusReasonFamilyGenerator(),
		createPodStatusScheduledFamilyGenerator(),
		createPodStatusScheduledTimeFamilyGenerator(),
		createPodStatusUnschedulableFamilyGenerator(),
		createPodStatusUnscheduledTimeFamilyGenerator(),
		createPodTolerationsFamilyGenerator(),
		createPodNodeSelectorsFamilyGenerator(),
		createPodServiceAccountFamilyGenerator(),
		createPodSchedulerNameFamilyGenerator(),
	}
}

func createPodCompletionTimeFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_completion_time",
		"Completion time in unix timestamp for a pod.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
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

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodContainerInfoFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_container_info",
		"Information about a container in a pod.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}
			labelKeys := []string{"container", "image_spec", "image", "image_id", "container_id"}

			for _, c := range p.Spec.Containers {
				for _, cs := range p.Status.ContainerStatuses {
					if cs.Name != c.Name {
						continue
					}
					ms = append(ms, &metric.Metric{
						LabelKeys:   labelKeys,
						LabelValues: []string{cs.Name, c.Image, cs.Image, cs.ImageID, cs.ContainerID},
						Value:       1,
					})
				}
			}
			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodContainerResourceLimitsFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_container_resource_limits",
		"The number of requested limit resource by a container. It is recommended to use the kube_pod_resource_limits metric exposed by kube-scheduler instead, as it is more precise.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, c := range p.Spec.Containers {
				lim := c.Resources.Limits

				for resourceName, val := range lim {
					switch resourceName {
					case v1.ResourceCPU:
						ms = append(ms, &metric.Metric{
							LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitCore)},
							Value:       convertValueToFloat64(&val),
						})
					case v1.ResourceStorage:
						fallthrough
					case v1.ResourceEphemeralStorage:
						fallthrough
					case v1.ResourceMemory:
						ms = append(ms, &metric.Metric{
							LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
							Value:       float64(val.Value()),
						})
					default:
						if isHugePageResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
								Value:       float64(val.Value()),
							})
						}
						if isAttachableVolumeResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								Value:       float64(val.Value()),
								LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
							})
						}
						if isExtendedResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								Value:       float64(val.Value()),
								LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitInteger)},
							})

						}
					}
				}
			}

			for _, metric := range ms {
				metric.LabelKeys = []string{"container", "node", "resource", "unit"}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodContainerResourceRequestsFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_container_resource_requests",
		"The number of requested request resource by a container. It is recommended to use the kube_pod_resource_requests metric exposed by kube-scheduler instead, as it is more precise.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, c := range p.Spec.Containers {
				req := c.Resources.Requests

				for resourceName, val := range req {
					switch resourceName {
					case v1.ResourceCPU:
						ms = append(ms, &metric.Metric{
							LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitCore)},
							Value:       convertValueToFloat64(&val),
						})
					case v1.ResourceStorage:
						fallthrough
					case v1.ResourceEphemeralStorage:
						fallthrough
					case v1.ResourceMemory:
						ms = append(ms, &metric.Metric{
							LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
							Value:       float64(val.Value()),
						})
					default:
						if isHugePageResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
								Value:       float64(val.Value()),
							})
						}
						if isAttachableVolumeResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
								Value:       float64(val.Value()),
							})
						}
						if isExtendedResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitInteger)},
								Value:       float64(val.Value()),
							})
						}
					}
				}
			}

			for _, metric := range ms {
				metric.LabelKeys = []string{"container", "node", "resource", "unit"}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodContainerStateStartedFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_container_state_started",
		"Start time in unix timestamp for a pod container.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, cs := range p.Status.ContainerStatuses {
				if cs.State.Running != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container"},
						LabelValues: []string{cs.Name},
						Value:       float64((cs.State.Running.StartedAt).Unix()),
					})
				} else if cs.State.Terminated != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container"},
						LabelValues: []string{cs.Name},
						Value:       float64((cs.State.Terminated.StartedAt).Unix()),
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodContainerStatusLastTerminatedReasonFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_container_status_last_terminated_reason",
		"Describes the last reason the container was in terminated state.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, 0, len(p.Status.ContainerStatuses))
			for _, cs := range p.Status.ContainerStatuses {
				if cs.LastTerminationState.Terminated != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container", "reason"},
						LabelValues: []string{cs.Name, cs.LastTerminationState.Terminated.Reason},
						Value:       1,
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodContainerStatusLastTerminatedExitCodeFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_container_status_last_terminated_exitcode",
		"Describes the exit code for the last container in terminated state.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, 0, len(p.Status.ContainerStatuses))
			for _, cs := range p.Status.ContainerStatuses {
				if cs.LastTerminationState.Terminated != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container"},
						LabelValues: []string{cs.Name},
						Value:       float64(cs.LastTerminationState.Terminated.ExitCode),
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodContainerStatusLastTerminatedTimestampFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_container_status_last_terminated_timestamp",
		"Last terminated time for a pod container in unix timestamp.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, 0, len(p.Status.ContainerStatuses))
			for _, cs := range p.Status.ContainerStatuses {
				if cs.LastTerminationState.Terminated != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container"},
						LabelValues: []string{cs.Name},
						Value:       float64(cs.LastTerminationState.Terminated.FinishedAt.Unix()),
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodContainerStatusReadyFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_container_status_ready",
		"Describes whether the containers readiness check succeeded.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, len(p.Status.ContainerStatuses))
			for i, cs := range p.Status.ContainerStatuses {
				ms[i] = &metric.Metric{
					LabelKeys:   []string{"container"},
					LabelValues: []string{cs.Name},
					Value:       boolFloat64(cs.Ready),
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodContainerStatusRestartsTotalFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_container_status_restarts_total",
		"The number of container restarts per container.",
		metric.Counter, basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, len(p.Status.ContainerStatuses))

			for i, cs := range p.Status.ContainerStatuses {
				ms[i] = &metric.Metric{
					LabelKeys:   []string{"container"},
					LabelValues: []string{cs.Name},
					Value:       float64(cs.RestartCount),
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodContainerStatusRunningFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_container_status_running",
		"Describes whether the container is currently in running state.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, len(p.Status.ContainerStatuses))

			for i, cs := range p.Status.ContainerStatuses {
				ms[i] = &metric.Metric{
					LabelKeys:   []string{"container"},
					LabelValues: []string{cs.Name},
					Value:       boolFloat64(cs.State.Running != nil),
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodContainerStatusTerminatedFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_container_status_terminated",
		"Describes whether the container is currently in terminated state.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, len(p.Status.ContainerStatuses))

			for i, cs := range p.Status.ContainerStatuses {
				ms[i] = &metric.Metric{
					LabelKeys:   []string{"container"},
					LabelValues: []string{cs.Name},
					Value:       boolFloat64(cs.State.Terminated != nil),
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodContainerStatusTerminatedReasonFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_container_status_terminated_reason",
		"Describes the reason the container is currently in terminated state.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, 0, len(p.Status.ContainerStatuses))
			for _, cs := range p.Status.ContainerStatuses {
				if cs.State.Terminated != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container", "reason"},
						LabelValues: []string{cs.Name, cs.State.Terminated.Reason},
						Value:       1,
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodContainerStatusWaitingFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_container_status_waiting",
		"Describes whether the container is currently in waiting state.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, len(p.Status.ContainerStatuses))

			for i, cs := range p.Status.ContainerStatuses {
				ms[i] = &metric.Metric{
					LabelKeys:   []string{"container"},
					LabelValues: []string{cs.Name},
					Value:       boolFloat64(cs.State.Waiting != nil),
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodContainerStatusWaitingReasonFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_container_status_waiting_reason",
		"Describes the reason the container is currently in waiting state.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, 0, len(p.Status.ContainerStatuses))
			for _, cs := range p.Status.ContainerStatuses {
				// Skip creating series for running containers.
				if cs.State.Waiting != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container", "reason"},
						LabelValues: []string{cs.Name, cs.State.Waiting.Reason},
						Value:       1,
					})
				}
			}
			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodCreatedFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_created",
		"Unix creation timestamp",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			if !p.CreationTimestamp.IsZero() {
				ms = append(ms, &metric.Metric{
					LabelKeys:   []string{},
					LabelValues: []string{},
					Value:       float64(p.CreationTimestamp.Unix()),
				})
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodDeletionTimestampFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_deletion_timestamp",
		"Unix deletion timestamp",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			if p.DeletionTimestamp != nil && !p.DeletionTimestamp.IsZero() {
				ms = append(ms, &metric.Metric{
					LabelKeys:   []string{},
					LabelValues: []string{},
					Value:       float64(p.DeletionTimestamp.Unix()),
				})
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodInfoFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_info",
		"Information about pod.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			createdBy := metav1.GetControllerOf(p)
			createdByKind := ""
			createdByName := ""
			if createdBy != nil {
				if createdBy.Kind != "" {
					createdByKind = createdBy.Kind
				}
				if createdBy.Name != "" {
					createdByName = createdBy.Name
				}
			}

			m := metric.Metric{
				LabelKeys:   []string{"host_ip", "pod_ip", "node", "created_by_kind", "created_by_name", "priority_class", "host_network"},
				LabelValues: []string{p.Status.HostIP, p.Status.PodIP, p.Spec.NodeName, createdByKind, createdByName, p.Spec.PriorityClassName, strconv.FormatBool(p.Spec.HostNetwork)},
				Value:       1,
			}

			return &metric.Family{
				Metrics: []*metric.Metric{&m},
			}
		}),
	)
}

func createPodIPFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_ips",
		"Pod IP addresses",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, len(p.Status.PodIPs))
			labelKeys := []string{"ip", "ip_family"}

			for i, ip := range p.Status.PodIPs {
				netIP := net.ParseIPSloppy(ip.IP)
				var ipFamily net.IPFamily
				switch {
				case net.IsIPv4(netIP):
					ipFamily = net.IPv4
				case net.IsIPv6(netIP):
					ipFamily = net.IPv6
				default:
					continue // nil from ParseIPSloppy indicates failure to parse, so we don't include that in our metrics series
				}
				ms[i] = &metric.Metric{
					LabelKeys:   labelKeys,
					LabelValues: []string{ip.IP, string(ipFamily)},
					Value:       1,
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}))
}

func createPodInitContainerInfoFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_init_container_info",
		"Information about an init container in a pod.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}
			labelKeys := []string{"container", "image_spec", "image", "image_id", "container_id", "restart_policy"}

			for _, c := range p.Spec.InitContainers {
				restartPolicy := ""
				if c.RestartPolicy != nil {
					restartPolicy = string(*c.RestartPolicy)
				}

				for _, cs := range p.Status.InitContainerStatuses {
					if cs.Name != c.Name {
						continue
					}
					ms = append(ms, &metric.Metric{
						LabelKeys:   labelKeys,
						LabelValues: []string{cs.Name, c.Image, cs.Image, cs.ImageID, cs.ContainerID, restartPolicy},
						Value:       1,
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodInitContainerResourceLimitsFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_init_container_resource_limits",
		"The number of requested limit resource by an init container.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, c := range p.Spec.InitContainers {
				lim := c.Resources.Limits

				for resourceName, val := range lim {
					switch resourceName {
					case v1.ResourceCPU:
						ms = append(ms, &metric.Metric{
							LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitCore)},
							Value:       convertValueToFloat64(&val),
						})
					case v1.ResourceStorage:
						fallthrough
					case v1.ResourceEphemeralStorage:
						fallthrough
					case v1.ResourceMemory:
						ms = append(ms, &metric.Metric{
							LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
							Value:       float64(val.Value()),
						})
					default:
						if isHugePageResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
								Value:       float64(val.Value()),
							})
						}
						if isAttachableVolumeResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								Value:       float64(val.Value()),
								LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
							})
						}
						if isExtendedResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								Value:       float64(val.Value()),
								LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitInteger)},
							})

						}
					}
				}
			}

			for _, metric := range ms {
				metric.LabelKeys = []string{"container", "node", "resource", "unit"}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodInitContainerResourceRequestsFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_init_container_resource_requests",
		"The number of requested request resource by an init container.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, c := range p.Spec.InitContainers {
				req := c.Resources.Requests

				for resourceName, val := range req {
					switch resourceName {
					case v1.ResourceCPU:
						ms = append(ms, &metric.Metric{
							LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitCore)},
							Value:       convertValueToFloat64(&val),
						})
					case v1.ResourceStorage:
						fallthrough
					case v1.ResourceEphemeralStorage:
						fallthrough
					case v1.ResourceMemory:
						ms = append(ms, &metric.Metric{
							LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
							Value:       float64(val.Value()),
						})
					default:
						if isHugePageResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
								Value:       float64(val.Value()),
							})
						}
						if isAttachableVolumeResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitByte)},
								Value:       float64(val.Value()),
							})
						}
						if isExtendedResourceName(resourceName) {
							ms = append(ms, &metric.Metric{
								LabelValues: []string{c.Name, p.Spec.NodeName, SanitizeLabelName(string(resourceName)), string(constant.UnitInteger)},
								Value:       float64(val.Value()),
							})
						}
					}
				}
			}

			for _, metric := range ms {
				metric.LabelKeys = []string{"container", "node", "resource", "unit"}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodInitContainerStatusLastTerminatedReasonFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_init_container_status_last_terminated_reason",
		"Describes the last reason the init container was in terminated state.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, 0, len(p.Status.InitContainerStatuses))
			for _, cs := range p.Status.InitContainerStatuses {
				if cs.LastTerminationState.Terminated != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container", "reason"},
						LabelValues: []string{cs.Name, cs.LastTerminationState.Terminated.Reason},
						Value:       1,
					})
				}
			}
			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodInitContainerStatusReadyFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_init_container_status_ready",
		"Describes whether the init containers readiness check succeeded.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, len(p.Status.InitContainerStatuses))

			for i, cs := range p.Status.InitContainerStatuses {
				ms[i] = &metric.Metric{
					LabelKeys:   []string{"container"},
					LabelValues: []string{cs.Name},
					Value:       boolFloat64(cs.Ready),
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodInitContainerStatusRestartsTotalFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_init_container_status_restarts_total",
		"The number of restarts for the init container.",
		metric.Counter, basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, len(p.Status.InitContainerStatuses))

			for i, cs := range p.Status.InitContainerStatuses {
				ms[i] = &metric.Metric{
					LabelKeys:   []string{"container"},
					LabelValues: []string{cs.Name},
					Value:       float64(cs.RestartCount),
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodInitContainerStatusRunningFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_init_container_status_running",
		"Describes whether the init container is currently in running state.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, len(p.Status.InitContainerStatuses))

			for i, cs := range p.Status.InitContainerStatuses {
				ms[i] = &metric.Metric{
					LabelKeys:   []string{"container"},
					LabelValues: []string{cs.Name},
					Value:       boolFloat64(cs.State.Running != nil),
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodInitContainerStatusTerminatedFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_init_container_status_terminated",
		"Describes whether the init container is currently in terminated state.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, len(p.Status.InitContainerStatuses))

			for i, cs := range p.Status.InitContainerStatuses {
				ms[i] = &metric.Metric{
					LabelKeys:   []string{"container"},
					LabelValues: []string{cs.Name},
					Value:       boolFloat64(cs.State.Terminated != nil),
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodInitContainerStatusTerminatedReasonFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_init_container_status_terminated_reason",
		"Describes the reason the init container is currently in terminated state.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, 0, len(p.Status.InitContainerStatuses))
			for _, cs := range p.Status.InitContainerStatuses {
				if cs.State.Terminated != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container", "reason"},
						LabelValues: []string{cs.Name, cs.State.Terminated.Reason},
						Value:       1,
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodInitContainerStatusWaitingFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_init_container_status_waiting",
		"Describes whether the init container is currently in waiting state.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, len(p.Status.InitContainerStatuses))

			for i, cs := range p.Status.InitContainerStatuses {
				ms[i] = &metric.Metric{
					LabelKeys:   []string{"container"},
					LabelValues: []string{cs.Name},
					Value:       boolFloat64(cs.State.Waiting != nil),
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodInitContainerStatusWaitingReasonFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_init_container_status_waiting_reason",
		"Describes the reason the init container is currently in waiting state.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := make([]*metric.Metric, 0, len(p.Status.InitContainerStatuses))
			for _, cs := range p.Status.InitContainerStatuses {
				// Skip creating series for running containers.
				if cs.State.Waiting != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"container", "reason"},
						LabelValues: []string{cs.Name, cs.State.Waiting.Reason},
						Value:       1,
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodAnnotationsGenerator(allowAnnotations []string) generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_annotations",
		"Kubernetes annotations converted to Prometheus labels.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			if len(allowAnnotations) == 0 {
				return &metric.Family{}
			}
			annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", p.Annotations, allowAnnotations)
			m := metric.Metric{
				LabelKeys:   annotationKeys,
				LabelValues: annotationValues,
				Value:       1,
			}
			return &metric.Family{
				Metrics: []*metric.Metric{&m},
			}
		}),
	)
}

func createPodLabelsGenerator(allowLabelsList []string) generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_labels",
		"Kubernetes labels converted to Prometheus labels.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			if len(allowLabelsList) == 0 {
				return &metric.Family{}
			}
			labelKeys, labelValues := createPrometheusLabelKeysValues("label", p.Labels, allowLabelsList)
			m := metric.Metric{
				LabelKeys:   labelKeys,
				LabelValues: labelValues,
				Value:       1,
			}
			return &metric.Family{
				Metrics: []*metric.Metric{&m},
			}
		}),
	)
}

func createPodOverheadCPUCoresFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_overhead_cpu_cores",
		"The pod overhead in regards to cpu cores associated with running a pod.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			if p.Spec.Overhead != nil {
				for resourceName, val := range p.Spec.Overhead {
					if resourceName == v1.ResourceCPU {
						ms = append(ms, &metric.Metric{
							Value: convertValueToFloat64(&val),
						})
					}
				}
			}
			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodOverheadMemoryBytesFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_overhead_memory_bytes",
		"The pod overhead in regards to memory associated with running a pod.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			if p.Spec.Overhead != nil {
				for resourceName, val := range p.Spec.Overhead {
					if resourceName == v1.ResourceMemory {
						ms = append(ms, &metric.Metric{
							Value: float64(val.Value()),
						})
					}
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodOwnerFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_owner",
		"Information about the Pod's owner.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			labelKeys := []string{"owner_kind", "owner_name", "owner_is_controller"}

			owners := p.GetOwnerReferences()
			if len(owners) == 0 {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   labelKeys,
							LabelValues: []string{"", "", ""},
							Value:       1,
						},
					},
				}
			}

			ms := make([]*metric.Metric, len(owners))

			for i, owner := range owners {
				if owner.Controller != nil {
					ms[i] = &metric.Metric{
						LabelKeys:   labelKeys,
						LabelValues: []string{owner.Kind, owner.Name, strconv.FormatBool(*owner.Controller)},
						Value:       1,
					}
				} else {
					ms[i] = &metric.Metric{
						LabelKeys:   labelKeys,
						LabelValues: []string{owner.Kind, owner.Name, "false"},
						Value:       1,
					}
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodRestartPolicyFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_restart_policy",
		"Describes the restart policy in use by this pod.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						LabelKeys:   []string{"type"},
						LabelValues: []string{string(p.Spec.RestartPolicy)},
						Value:       float64(1),
					},
				},
			}
		}),
	)
}

func createPodRuntimeClassNameInfoFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_runtimeclass_name_info",
		"The runtimeclass associated with the pod.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			if p.Spec.RuntimeClassName != nil {
				ms = append(ms, &metric.Metric{
					LabelKeys:   []string{"runtimeclass_name"},
					LabelValues: []string{*p.Spec.RuntimeClassName},
					Value:       1,
				})
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodSpecVolumesPersistentVolumeClaimsInfoFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_spec_volumes_persistentvolumeclaims_info",
		"Information about persistentvolumeclaim volumes in a pod.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
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

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodSpecVolumesPersistentVolumeClaimsReadonlyFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_spec_volumes_persistentvolumeclaims_readonly",
		"Describes whether a persistentvolumeclaim is mounted read only.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
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

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodStartTimeFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_start_time",
		"Start time in unix timestamp for a pod.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			if p.Status.StartTime != nil {
				ms = append(ms, &metric.Metric{
					LabelKeys:   []string{},
					LabelValues: []string{},
					Value:       float64((p.Status.StartTime).Unix()),
				})
			}
			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodStatusPhaseFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_status_phase",
		"The pods current phase.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			phase := p.Status.Phase
			if phase == "" {
				return &metric.Family{
					Metrics: []*metric.Metric{},
				}
			}

			phases := []struct {
				n string
				v bool
			}{
				{string(v1.PodPending), phase == v1.PodPending},
				{string(v1.PodSucceeded), phase == v1.PodSucceeded},
				{string(v1.PodFailed), phase == v1.PodFailed},
				{string(v1.PodUnknown), phase == v1.PodUnknown},
				{string(v1.PodRunning), phase == v1.PodRunning},
			}

			ms := make([]*metric.Metric, len(phases))

			for i, p := range phases {
				ms[i] = &metric.Metric{

					LabelKeys:   []string{"phase"},
					LabelValues: []string{p.n},
					Value:       boolFloat64(p.v),
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodStatusInitializedTimeFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_status_initialized_time",
		"Initialized time in unix timestamp for a pod.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, c := range p.Status.Conditions {
				if c.Type == v1.PodInitialized && c.Status == v1.ConditionTrue {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64((c.LastTransitionTime).Unix()),
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodStatusContainerReadyTimeFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_status_container_ready_time",
		"Readiness achieved time in unix timestamp for a pod containers.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, c := range p.Status.Conditions {
				if c.Type == v1.ContainersReady && c.Status == v1.ConditionTrue {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64((c.LastTransitionTime).Unix()),
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodStatusReadyTimeFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_status_ready_time",
		"Readiness achieved time in unix timestamp for a pod.",
		metric.Gauge,
		basemetrics.ALPHA,

		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, c := range p.Status.Conditions {
				if c.Type == v1.PodReady && c.Status == v1.ConditionTrue {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64((c.LastTransitionTime).Unix()),
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodStatusQosClassFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_status_qos_class",
		"The pods current qosClass.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			class := p.Status.QOSClass
			if class == "" {
				return &metric.Family{
					Metrics: []*metric.Metric{},
				}
			}

			qosClasses := []struct {
				n string
				v bool
			}{
				{string(v1.PodQOSBestEffort), class == v1.PodQOSBestEffort},
				{string(v1.PodQOSBurstable), class == v1.PodQOSBurstable},
				{string(v1.PodQOSGuaranteed), class == v1.PodQOSGuaranteed},
			}

			ms := make([]*metric.Metric, len(qosClasses))

			for i, p := range qosClasses {
				ms[i] = &metric.Metric{

					LabelKeys:   []string{"qos_class"},
					LabelValues: []string{p.n},
					Value:       boolFloat64(p.v),
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodStatusReadyFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_status_ready",
		"Describes whether the pod is ready to serve requests.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, c := range p.Status.Conditions {
				if c.Type == v1.PodReady {
					conditionMetrics := addConditionMetrics(c.Status)

					for _, m := range conditionMetrics {
						metric := m
						metric.LabelKeys = []string{"condition"}
						ms = append(ms, metric)
					}
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodStatusReasonFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_status_reason",
		"The pod status reasons",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, reason := range podStatusReasons {
				m := &metric.Metric{
					LabelKeys:   []string{"reason"},
					LabelValues: []string{reason},
					Value:       getPodStatusReasonValue(p, reason),
				}
				ms = append(ms, m)
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func getPodStatusReasonValue(p *v1.Pod, reason string) float64 {
	if p.Status.Reason == reason {
		return 1
	}
	for _, cond := range p.Status.Conditions {
		if cond.Reason == reason {
			return 1
		}
	}
	for _, cs := range p.Status.ContainerStatuses {
		if cs.State.Terminated != nil && cs.State.Terminated.Reason == reason {
			return 1
		}
	}
	return 0
}

func createPodStatusScheduledFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_status_scheduled",
		"Describes the status of the scheduling process for the pod.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, c := range p.Status.Conditions {
				if c.Type == v1.PodScheduled {
					conditionMetrics := addConditionMetrics(c.Status)

					for _, m := range conditionMetrics {
						metric := m
						metric.LabelKeys = []string{"condition"}
						ms = append(ms, metric)
					}
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodStatusScheduledTimeFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_status_scheduled_time",
		"Unix timestamp when pod moved into scheduled status",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, c := range p.Status.Conditions {
				if c.Type == v1.PodScheduled && c.Status == v1.ConditionTrue {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(c.LastTransitionTime.Unix()),
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodStatusUnschedulableFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_status_unschedulable",
		"Describes the unschedulable status for the pod.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, c := range p.Status.Conditions {
				if c.Type == v1.PodScheduled && c.Status == v1.ConditionFalse {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       1,
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodStatusUnscheduledTimeFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_status_unscheduled_time",
		"Unix timestamp when pod moved into unscheduled status",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			ms := []*metric.Metric{}

			for _, c := range p.Status.Conditions {
				if c.Type == v1.PodScheduled && c.Status == v1.ConditionFalse {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(c.LastTransitionTime.Unix()),
					})
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

// getUniqueTolerations takes an array
func getUniqueTolerations(tolerations []v1.Toleration) []v1.Toleration {
	uniqueTolerationsMap := make(map[v1.Toleration]struct{})
	var uniqueTolerations []v1.Toleration

	for _, toleration := range tolerations {
		_, exists := uniqueTolerationsMap[toleration]
		if !exists {
			uniqueTolerationsMap[toleration] = struct{}{}
			uniqueTolerations = append(uniqueTolerations, toleration)
		}
	}
	return uniqueTolerations
}

func createPodTolerationsFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_tolerations",
		"Information about the pod tolerations",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			var ms []*metric.Metric
			uniqueTolerations := getUniqueTolerations(p.Spec.Tolerations)

			for _, t := range uniqueTolerations {
				var key, operator, value, effect, tolerationSeconds string

				key = t.Key
				if t.Operator != "" {
					operator = string(t.Operator)
				}

				value = t.Value

				if t.Effect != "" {
					effect = string(t.Effect)
				}

				if t.TolerationSeconds != nil {
					tolerationSeconds = strconv.FormatInt(*t.TolerationSeconds, 10)
				}

				ms = append(ms, &metric.Metric{
					LabelKeys:   []string{"key", "operator", "value", "effect", "toleration_seconds"},
					LabelValues: []string{key, operator, value, effect, tolerationSeconds},
					Value:       1,
				})
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPodNodeSelectorsFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewOptInFamilyGenerator(
		"kube_pod_nodeselectors",
		"Describes the Pod nodeSelectors.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			labelKeys, labelValues := kubeMapToPrometheusLabels("nodeselector", p.Spec.NodeSelector)
			m := metric.Metric{
				LabelKeys:   labelKeys,
				LabelValues: labelValues,
				Value:       1,
			}
			return &metric.Family{
				Metrics: []*metric.Metric{&m},
			}
		}),
	)
}

func createPodServiceAccountFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_service_account",
		"The service account for a pod.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			m := metric.Metric{
				LabelKeys:   []string{"service_account"},
				LabelValues: []string{p.Spec.ServiceAccountName},
				Value:       1,
			}

			return &metric.Family{
				Metrics: []*metric.Metric{&m},
			}
		}),
	)
}

func createPodSchedulerNameFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_pod_scheduler",
		"The scheduler for a pod.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPodFunc(func(p *v1.Pod) *metric.Family {
			m := metric.Metric{
				LabelKeys:   []string{"name"},
				LabelValues: []string{p.Spec.SchedulerName},
				Value:       1,
			}

			return &metric.Family{
				Metrics: []*metric.Metric{&m},
			}
		}),
	)
}

func wrapPodFunc(f func(*v1.Pod) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		pod := obj.(*v1.Pod)

		metricFamily := f(pod)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descPodLabelsDefaultLabels, []string{pod.Namespace, pod.Name, string(pod.UID)}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createPodListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().Pods(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().Pods(ns).Watch(context.TODO(), opts)
		},
	}
}
