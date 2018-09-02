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
	"strconv"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/constant"
	"k8s.io/kube-state-metrics/pkg/options"
	"k8s.io/kubernetes/pkg/apis/core/v1/helper"
	"k8s.io/kubernetes/pkg/util/node"
)

var (
	descPodLabelsName          = "kube_pod_labels"
	descPodLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descPodLabelsDefaultLabels = []string{"namespace", "pod"}
	containerWaitingReasons    = []string{"ContainerCreating", "CrashLoopBackOff", "ErrImagePull", "ImagePullBackOff"}
	containerTerminatedReasons = []string{"OOMKilled", "Completed", "Error", "ContainerCannotRun"}

	descPodInfo = prometheus.NewDesc(
		"kube_pod_info",
		"Information about pod.",
		append(descPodLabelsDefaultLabels, "host_ip", "pod_ip", "uid", "node", "created_by_kind", "created_by_name"),
		nil,
	)
	descPodStartTime = prometheus.NewDesc(
		"kube_pod_start_time",
		"Start time in unix timestamp for a pod.",
		descPodLabelsDefaultLabels,
		nil,
	)
	descPodCompletionTime = prometheus.NewDesc(
		"kube_pod_completion_time",
		"Completion time in unix timestamp for a pod.",
		descPodLabelsDefaultLabels,
		nil,
	)
	descPodOwner = prometheus.NewDesc(
		"kube_pod_owner",
		"Information about the Pod's owner.",
		append(descPodLabelsDefaultLabels, "owner_kind", "owner_name", "owner_is_controller"),
		nil,
	)
	descPodLabels = prometheus.NewDesc(
		descPodLabelsName,
		descPodLabelsHelp,
		descPodLabelsDefaultLabels,
		nil,
	)
	descPodCreated = prometheus.NewDesc(
		"kube_pod_created",
		"Unix creation timestamp",
		descPodLabelsDefaultLabels,
		nil,
	)
	descPodStatusScheduledTime = prometheus.NewDesc(
		"kube_pod_status_scheduled_time",
		"Unix timestamp when pod moved into scheduled status",
		descPodLabelsDefaultLabels,
		nil,
	)
	descPodStatusPhase = prometheus.NewDesc(
		"kube_pod_status_phase",
		"The pods current phase.",
		append(descPodLabelsDefaultLabels, "phase"),
		nil,
	)
	descPodStatusReady = prometheus.NewDesc(
		"kube_pod_status_ready",
		"Describes whether the pod is ready to serve requests.",
		append(descPodLabelsDefaultLabels, "condition"),
		nil,
	)
	descPodStatusScheduled = prometheus.NewDesc(
		"kube_pod_status_scheduled",
		"Describes the status of the scheduling process for the pod.",
		append(descPodLabelsDefaultLabels, "condition"),
		nil,
	)
	descPodContainerInfo = prometheus.NewDesc(
		"kube_pod_container_info",
		"Information about a container in a pod.",
		append(descPodLabelsDefaultLabels, "container", "image", "image_id", "container_id"),
		nil,
	)
	descPodContainerStatusWaiting = prometheus.NewDesc(
		"kube_pod_container_status_waiting",
		"Describes whether the container is currently in waiting state.",
		append(descPodLabelsDefaultLabels, "container"),
		nil,
	)
	descPodContainerStatusWaitingReason = prometheus.NewDesc(
		"kube_pod_container_status_waiting_reason",
		"Describes the reason the container is currently in waiting state.",
		append(descPodLabelsDefaultLabels, "container", "reason"),
		nil,
	)
	descPodContainerStatusRunning = prometheus.NewDesc(
		"kube_pod_container_status_running",
		"Describes whether the container is currently in running state.",
		append(descPodLabelsDefaultLabels, "container"),
		nil,
	)
	descPodContainerStatusTerminated = prometheus.NewDesc(
		"kube_pod_container_status_terminated",
		"Describes whether the container is currently in terminated state.",
		append(descPodLabelsDefaultLabels, "container"),
		nil,
	)
	descPodContainerStatusTerminatedReason = prometheus.NewDesc(
		"kube_pod_container_status_terminated_reason",
		"Describes the reason the container is currently in terminated state.",
		append(descPodLabelsDefaultLabels, "container", "reason"),
		nil,
	)
	descPodContainerStatusLastTerminatedReason = prometheus.NewDesc(
		"kube_pod_container_status_last_terminated_reason",
		"Describes the last reason the container was in terminated state.",
		append(descPodLabelsDefaultLabels, "container", "reason"),
		nil,
	)

	descPodContainerStatusReady = prometheus.NewDesc(
		"kube_pod_container_status_ready",
		"Describes whether the containers readiness check succeeded.",
		append(descPodLabelsDefaultLabels, "container"),
		nil,
	)
	descPodContainerStatusRestarts = prometheus.NewDesc(
		"kube_pod_container_status_restarts_total",
		"The number of container restarts per container.",
		append(descPodLabelsDefaultLabels, "container"),
		nil,
	)
	descPodContainerResourceRequests = prometheus.NewDesc(
		"kube_pod_container_resource_requests",
		"The number of requested request resource by a container.",
		append(descPodLabelsDefaultLabels, "container", "node", "resource", "unit"),
		nil,
	)
	descPodContainerResourceLimits = prometheus.NewDesc(
		"kube_pod_container_resource_limits",
		"The number of requested limit resource by a container.",
		append(descPodLabelsDefaultLabels, "container", "node", "resource", "unit"),
		nil,
	)
	descPodContainerResourceRequestsCPUCores = prometheus.NewDesc(
		"kube_pod_container_resource_requests_cpu_cores",
		"The number of requested cpu cores by a container.",
		append(descPodLabelsDefaultLabels, "container", "node"),
		nil,
	)
	descPodContainerResourceRequestsMemoryBytes = prometheus.NewDesc(
		"kube_pod_container_resource_requests_memory_bytes",
		"The number of requested memory bytes by a container.",
		append(descPodLabelsDefaultLabels, "container", "node"),
		nil,
	)
	descPodContainerResourceLimitsCPUCores = prometheus.NewDesc(
		"kube_pod_container_resource_limits_cpu_cores",
		"The limit on cpu cores to be used by a container.",
		append(descPodLabelsDefaultLabels, "container", "node"),
		nil,
	)
	descPodContainerResourceLimitsMemoryBytes = prometheus.NewDesc(
		"kube_pod_container_resource_limits_memory_bytes",
		"The limit on memory to be used by a container in bytes.",
		append(descPodLabelsDefaultLabels, "container", "node"),
		nil,
	)
	descPodSpecVolumesPersistentVolumeClaimsInfo = prometheus.NewDesc(
		"kube_pod_spec_volumes_persistentvolumeclaims_info",
		"Information about persistentvolumeclaim volumes in a pod.",
		append(descPodLabelsDefaultLabels, "volume", "persistentvolumeclaim"),
		nil,
	)
	descPodSpecVolumesPersistentVolumeClaimsReadOnly = prometheus.NewDesc(
		"kube_pod_spec_volumes_persistentvolumeclaims_readonly",
		"Describes whether a persistentvolumeclaim is mounted read only.",
		append(descPodLabelsDefaultLabels, "volume", "persistentvolumeclaim"),
		nil,
	)
)

type PodLister func() ([]v1.Pod, error)

func (l PodLister) List() ([]v1.Pod, error) {
	return l()
}

func RegisterPodCollector(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options) {

	infs := SharedInformerList{}
	for _, f := range informerFactories {
		infs = append(infs, f.Core().V1().Pods().Informer().(cache.SharedInformer))
	}

	podLister := PodLister(func() (pods []v1.Pod, err error) {
		for _, pinf := range infs {
			for _, m := range pinf.GetStore().List() {
				pods = append(pods, *m.(*v1.Pod))
			}
		}
		return pods, nil
	})

	registry.MustRegister(&podCollector{store: podLister, opts: opts})
	infs.Run(context.Background().Done())
}

type podStore interface {
	List() (pods []v1.Pod, err error)
}

// podCollector collects metrics about all pods in the cluster.
type podCollector struct {
	store podStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (pc *podCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descPodInfo
	ch <- descPodStartTime
	ch <- descPodCompletionTime
	ch <- descPodOwner
	ch <- descPodLabels
	ch <- descPodCreated
	ch <- descPodStatusScheduledTime
	ch <- descPodStatusPhase
	ch <- descPodStatusReady
	ch <- descPodStatusScheduled
	ch <- descPodContainerInfo
	ch <- descPodContainerStatusWaiting
	ch <- descPodContainerStatusWaitingReason
	ch <- descPodContainerStatusRunning
	ch <- descPodContainerStatusTerminated
	ch <- descPodContainerStatusTerminatedReason
	ch <- descPodContainerStatusLastTerminatedReason
	ch <- descPodContainerStatusReady
	ch <- descPodContainerStatusRestarts
	ch <- descPodSpecVolumesPersistentVolumeClaimsInfo
	ch <- descPodSpecVolumesPersistentVolumeClaimsReadOnly
	ch <- descPodContainerResourceRequests
	ch <- descPodContainerResourceLimits

	if !pc.opts.DisablePodNonGenericResourceMetrics {
		ch <- descPodContainerResourceRequestsCPUCores
		ch <- descPodContainerResourceRequestsMemoryBytes
		ch <- descPodContainerResourceLimitsCPUCores
		ch <- descPodContainerResourceLimitsMemoryBytes
	}
}

// Collect implements the prometheus.Collector interface.
func (pc *podCollector) Collect(ch chan<- prometheus.Metric) {
	pods, err := pc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "pod"}).Inc()
		glog.Errorf("listing pods failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "pod"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "pod"}).Observe(float64(len(pods)))
	for _, p := range pods {
		pc.collectPod(ch, p)
	}

	glog.V(4).Infof("collected %d pods", len(pods))
}

func podLabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descPodLabelsName,
		descPodLabelsHelp,
		append(descPodLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func (pc *podCollector) collectPod(ch chan<- prometheus.Metric, p v1.Pod) {
	nodeName := p.Spec.NodeName
	addConstMetric := func(desc *prometheus.Desc, t prometheus.ValueType, v float64, lv ...string) {
		lv = append([]string{p.Namespace, p.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, t, v, lv...)
	}
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		addConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	addCounter := func(desc *prometheus.Desc, v float64, lv ...string) {
		addConstMetric(desc, prometheus.CounterValue, v, lv...)
	}

	createdBy := metav1.GetControllerOf(&p)
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

	if p.Status.StartTime != nil {
		addGauge(descPodStartTime, float64((*(p.Status.StartTime)).Unix()))
	}

	addGauge(descPodInfo, 1, p.Status.HostIP, p.Status.PodIP, string(p.UID), nodeName, createdByKind, createdByName)

	owners := p.GetOwnerReferences()
	if len(owners) == 0 {
		addGauge(descPodOwner, 1, "<none>", "<none>", "<none>")
	} else {
		for _, owner := range owners {
			if owner.Controller != nil {
				addGauge(descPodOwner, 1, owner.Kind, owner.Name, strconv.FormatBool(*owner.Controller))
			} else {
				addGauge(descPodOwner, 1, owner.Kind, owner.Name, "false")
			}
		}
	}

	labelKeys, labelValues := kubeLabelsToPrometheusLabels(p.Labels)
	addGauge(podLabelsDesc(labelKeys), 1, labelValues...)

	if phase := p.Status.Phase; phase != "" {
		addGauge(descPodStatusPhase, boolFloat64(phase == v1.PodPending), string(v1.PodPending))
		addGauge(descPodStatusPhase, boolFloat64(phase == v1.PodSucceeded), string(v1.PodSucceeded))
		addGauge(descPodStatusPhase, boolFloat64(phase == v1.PodFailed), string(v1.PodFailed))
		// This logic is directly copied from: https://github.com/kubernetes/kubernetes/blob/d39bfa0d138368bbe72b0eaf434501dcb4ec9908/pkg/printers/internalversion/printers.go#L597-L601
		// For more info, please go to: https://github.com/kubernetes/kube-state-metrics/issues/410
		addGauge(descPodStatusPhase, boolFloat64(phase == v1.PodRunning && !(p.DeletionTimestamp != nil && p.Status.Reason == node.NodeUnreachablePodReason)), string(v1.PodRunning))
		addGauge(descPodStatusPhase, boolFloat64(phase == v1.PodUnknown || (p.DeletionTimestamp != nil && p.Status.Reason == node.NodeUnreachablePodReason)), string(v1.PodUnknown))
	}

	if !p.CreationTimestamp.IsZero() {
		addGauge(descPodCreated, float64(p.CreationTimestamp.Unix()))
	}

	for _, c := range p.Status.Conditions {
		switch c.Type {
		case v1.PodReady:
			addConditionMetrics(ch, descPodStatusReady, c.Status, p.Namespace, p.Name)
		case v1.PodScheduled:
			addConditionMetrics(ch, descPodStatusScheduled, c.Status, p.Namespace, p.Name)
			if c.Status == v1.ConditionTrue {
				addGauge(descPodStatusScheduledTime, float64(c.LastTransitionTime.Unix()))
			}
		}
	}

	waitingReason := func(cs v1.ContainerStatus, reason string) bool {
		if cs.State.Waiting == nil {
			return false
		}
		return cs.State.Waiting.Reason == reason
	}

	terminationReason := func(cs v1.ContainerStatus, reason string) bool {
		if cs.State.Terminated == nil {
			return false
		}
		return cs.State.Terminated.Reason == reason
	}

	lastTerminationReason := func(cs v1.ContainerStatus, reason string) bool {
		if cs.LastTerminationState.Terminated == nil {
			return false
		}
		return cs.LastTerminationState.Terminated.Reason == reason
	}

	var lastFinishTime float64

	for _, cs := range p.Status.ContainerStatuses {
		addGauge(descPodContainerInfo, 1,
			cs.Name, cs.Image, cs.ImageID, cs.ContainerID,
		)
		addGauge(descPodContainerStatusWaiting, boolFloat64(cs.State.Waiting != nil), cs.Name)
		for _, reason := range containerWaitingReasons {
			addGauge(descPodContainerStatusWaitingReason, boolFloat64(waitingReason(cs, reason)), cs.Name, reason)
		}
		addGauge(descPodContainerStatusRunning, boolFloat64(cs.State.Running != nil), cs.Name)
		addGauge(descPodContainerStatusTerminated, boolFloat64(cs.State.Terminated != nil), cs.Name)
		for _, reason := range containerTerminatedReasons {
			addGauge(descPodContainerStatusTerminatedReason, boolFloat64(terminationReason(cs, reason)), cs.Name, reason)
		}
		for _, reason := range containerTerminatedReasons {
			addGauge(descPodContainerStatusLastTerminatedReason, boolFloat64(lastTerminationReason(cs, reason)), cs.Name, reason)
		}
		addGauge(descPodContainerStatusReady, boolFloat64(cs.Ready), cs.Name)
		addCounter(descPodContainerStatusRestarts, float64(cs.RestartCount), cs.Name)

		if cs.State.Terminated != nil {
			if lastFinishTime == 0 || lastFinishTime < float64(cs.State.Terminated.FinishedAt.Unix()) {
				lastFinishTime = float64(cs.State.Terminated.FinishedAt.Unix())
			}
		}
	}

	if lastFinishTime > 0 {
		addGauge(descPodCompletionTime, lastFinishTime)
	}

	if !pc.opts.DisablePodNonGenericResourceMetrics {
		for _, c := range p.Spec.Containers {
			req := c.Resources.Requests
			lim := c.Resources.Limits

			if cpu, ok := req[v1.ResourceCPU]; ok {
				addGauge(descPodContainerResourceRequestsCPUCores, float64(cpu.MilliValue())/1000,
					c.Name, nodeName)
			}
			if mem, ok := req[v1.ResourceMemory]; ok {
				addGauge(descPodContainerResourceRequestsMemoryBytes, float64(mem.Value()),
					c.Name, nodeName)
			}

			if cpu, ok := lim[v1.ResourceCPU]; ok {
				addGauge(descPodContainerResourceLimitsCPUCores, float64(cpu.MilliValue())/1000,
					c.Name, nodeName)
			}

			if mem, ok := lim[v1.ResourceMemory]; ok {
				addGauge(descPodContainerResourceLimitsMemoryBytes, float64(mem.Value()),
					c.Name, nodeName)
			}
		}
	}

	for _, c := range p.Spec.Containers {
		req := c.Resources.Requests
		lim := c.Resources.Limits

		for resourceName, val := range req {
			switch resourceName {
			case v1.ResourceCPU:
				addGauge(descPodContainerResourceRequests, float64(val.MilliValue())/1000,
					c.Name, nodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitCore))
			case v1.ResourceStorage:
				fallthrough
			case v1.ResourceEphemeralStorage:
				fallthrough
			case v1.ResourceMemory:
				addGauge(descPodContainerResourceRequests, float64(val.Value()),
					c.Name, nodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitByte))
			default:
				if helper.IsHugePageResourceName(resourceName) {
					addGauge(descPodContainerResourceRequests, float64(val.Value()),
						c.Name, nodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitByte))
				}
				if helper.IsAttachableVolumeResourceName(resourceName) {
					addGauge(descPodContainerResourceRequests, float64(val.Value()),
						c.Name, nodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitByte))
				}
				if helper.IsExtendedResourceName(resourceName) {
					addGauge(descPodContainerResourceRequests, float64(val.Value()),
						c.Name, nodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitInteger))
				}
			}
		}

		for resourceName, val := range lim {
			switch resourceName {
			case v1.ResourceCPU:
				addGauge(descPodContainerResourceLimits, float64(val.MilliValue())/1000,
					c.Name, nodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitCore))
			case v1.ResourceStorage:
				fallthrough
			case v1.ResourceEphemeralStorage:
				fallthrough
			case v1.ResourceMemory:
				addGauge(descPodContainerResourceLimits, float64(val.Value()),
					c.Name, nodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitByte))
			default:
				if helper.IsHugePageResourceName(resourceName) {
					addGauge(descPodContainerResourceLimits, float64(val.Value()),
						c.Name, nodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitByte))
				}
				if helper.IsAttachableVolumeResourceName(resourceName) {
					addGauge(descPodContainerResourceLimits, float64(val.Value()),
						c.Name, nodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitByte))
				}
				if helper.IsExtendedResourceName(resourceName) {
					addGauge(descPodContainerResourceLimits, float64(val.Value()),
						c.Name, nodeName, sanitizeLabelName(string(resourceName)), string(constant.UnitInteger))
				}
			}
		}
	}

	for _, v := range p.Spec.Volumes {
		if v.PersistentVolumeClaim != nil {
			addGauge(descPodSpecVolumesPersistentVolumeClaimsInfo, 1, v.Name, v.PersistentVolumeClaim.ClaimName)
			readOnly := 0.0
			if v.PersistentVolumeClaim.ReadOnly {
				readOnly = 1.0
			}
			addGauge(descPodSpecVolumesPersistentVolumeClaimsReadOnly, readOnly, v.Name, v.PersistentVolumeClaim.ClaimName)
		}
	}
}
