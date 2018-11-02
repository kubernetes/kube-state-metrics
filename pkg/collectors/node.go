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
	"k8s.io/kube-state-metrics/pkg/constant"
	"k8s.io/kube-state-metrics/pkg/metrics"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/apis/core/v1/helper"
)

var (
	descNodeLabelsName          = "kube_node_labels"
	descNodeLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descNodeLabelsDefaultLabels = []string{"node"}

	descNodeInfo = metrics.NewMetricFamilyDef(
		"kube_node_info",
		"Information about a cluster node.",
		append(descNodeLabelsDefaultLabels,
			"kernel_version",
			"os_image",
			"container_runtime_version",
			"kubelet_version",
			"kubeproxy_version",
			"provider_id"),
		nil,
	)
	descNodeCreated = metrics.NewMetricFamilyDef(
		"kube_node_created",
		"Unix creation timestamp",
		descNodeLabelsDefaultLabels,
		nil,
	)
	descNodeLabels = metrics.NewMetricFamilyDef(
		descNodeLabelsName,
		descNodeLabelsHelp,
		descNodeLabelsDefaultLabels,
		nil,
	)
	descNodeSpecUnschedulable = metrics.NewMetricFamilyDef(
		"kube_node_spec_unschedulable",
		"Whether a node can schedule new pods.",
		descNodeLabelsDefaultLabels,
		nil,
	)
	descNodeSpecTaint = metrics.NewMetricFamilyDef(
		"kube_node_spec_taint",
		"The taint of a cluster node.",
		append(descNodeLabelsDefaultLabels, "key", "value", "effect"),
		nil,
	)
	descNodeStatusCondition = metrics.NewMetricFamilyDef(
		"kube_node_status_condition",
		"The condition of a cluster node.",
		append(descNodeLabelsDefaultLabels, "condition", "status"),
		nil,
	)
	descNodeStatusPhase = metrics.NewMetricFamilyDef(
		"kube_node_status_phase",
		"The phase the node is currently in.",
		append(descNodeLabelsDefaultLabels, "phase"),
		nil,
	)
	descNodeStatusCapacity = metrics.NewMetricFamilyDef(
		"kube_node_status_capacity",
		"The capacity for different resources of a node.",
		append(descNodeLabelsDefaultLabels, "resource", "unit"),
		nil,
	)
	descNodeStatusCapacityPods = metrics.NewMetricFamilyDef(
		"kube_node_status_capacity_pods",
		"The total pod resources of the node.",
		descNodeLabelsDefaultLabels,
		nil,
	)
	descNodeStatusCapacityCPU = metrics.NewMetricFamilyDef(
		"kube_node_status_capacity_cpu_cores",
		"The total CPU resources of the node.",
		descNodeLabelsDefaultLabels,
		nil,
	)
	descNodeStatusCapacityMemory = metrics.NewMetricFamilyDef(
		"kube_node_status_capacity_memory_bytes",
		"The total memory resources of the node.",
		descNodeLabelsDefaultLabels,
		nil,
	)
	descNodeStatusAllocatable = metrics.NewMetricFamilyDef(
		"kube_node_status_allocatable",
		"The allocatable for different resources of a node that are available for scheduling.",
		append(descNodeLabelsDefaultLabels, "resource", "unit"),
		nil,
	)
	descNodeStatusAllocatablePods = metrics.NewMetricFamilyDef(
		"kube_node_status_allocatable_pods",
		"The pod resources of a node that are available for scheduling.",
		descNodeLabelsDefaultLabels,
		nil,
	)
	descNodeStatusAllocatableCPU = metrics.NewMetricFamilyDef(
		"kube_node_status_allocatable_cpu_cores",
		"The CPU resources of a node that are available for scheduling.",
		descNodeLabelsDefaultLabels,
		nil,
	)
	descNodeStatusAllocatableMemory = metrics.NewMetricFamilyDef(
		"kube_node_status_allocatable_memory_bytes",
		"The memory resources of a node that are available for scheduling.",
		descNodeLabelsDefaultLabels,
		nil,
	)
)

func createNodeListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Nodes().List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Nodes().Watch(opts)
		},
	}
}

func nodeLabelsDesc(labelKeys []string) *metrics.MetricFamilyDef {
	return metrics.NewMetricFamilyDef(
		descNodeLabelsName,
		descNodeLabelsHelp,
		append(descNodeLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func generateNodeMetrics(disableNodeNonGenericResourceMetrics bool, obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	nPointer := obj.(*v1.Node)
	n := *nPointer

	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{n.Name}, lv...)

		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}
	// NOTE: the instrumentation API requires providing label values in order of declaration
	// in the metric descriptor. Be careful when making modifications.
	addGauge(descNodeInfo, 1,
		n.Status.NodeInfo.KernelVersion,
		n.Status.NodeInfo.OSImage,
		n.Status.NodeInfo.ContainerRuntimeVersion,
		n.Status.NodeInfo.KubeletVersion,
		n.Status.NodeInfo.KubeProxyVersion,
		n.Spec.ProviderID,
	)
	if !n.CreationTimestamp.IsZero() {
		addGauge(descNodeCreated, float64(n.CreationTimestamp.Unix()))
	}
	labelKeys, labelValues := kubeLabelsToPrometheusLabels(n.Labels)
	addGauge(nodeLabelsDesc(labelKeys), 1, labelValues...)

	addGauge(descNodeSpecUnschedulable, boolFloat64(n.Spec.Unschedulable))

	// Collect node taints
	for _, taint := range n.Spec.Taints {
		// Taints are applied to repel pods from nodes that do not have a corresponding
		// toleration.  Many node conditions are optionally reflected as taints
		// by the node controller in order to simplify scheduling constraints.
		addGauge(descNodeSpecTaint, 1, taint.Key, taint.Value, string(taint.Effect))
	}

	// Collect node conditions and while default to false.
	for _, c := range n.Status.Conditions {
		// This all-in-one metric family contains all conditions for extensibility.
		// Third party plugin may report customized condition for cluster node
		// (e.g. node-problem-detector), and Kubernetes may add new core
		// conditions in future.
		ms = append(ms, addConditionMetrics(descNodeStatusCondition, c.Status, n.Name, string(c.Type))...)
	}

	// Set current phase to 1, others to 0 if it is set.
	if p := n.Status.Phase; p != "" {
		addGauge(descNodeStatusPhase, boolFloat64(p == v1.NodePending), string(v1.NodePending))
		addGauge(descNodeStatusPhase, boolFloat64(p == v1.NodeRunning), string(v1.NodeRunning))
		addGauge(descNodeStatusPhase, boolFloat64(p == v1.NodeTerminated), string(v1.NodeTerminated))
	}

	if !disableNodeNonGenericResourceMetrics {
		// Add capacity and allocatable resources if they are set.
		addResource := func(d *metrics.MetricFamilyDef, res v1.ResourceList, n v1.ResourceName) {
			if v, ok := res[n]; ok {
				addGauge(d, float64(v.MilliValue())/1000)
			}
		}

		addResource(descNodeStatusCapacityCPU, n.Status.Capacity, v1.ResourceCPU)
		addResource(descNodeStatusCapacityMemory, n.Status.Capacity, v1.ResourceMemory)
		addResource(descNodeStatusCapacityPods, n.Status.Capacity, v1.ResourcePods)

		addResource(descNodeStatusAllocatableCPU, n.Status.Allocatable, v1.ResourceCPU)
		addResource(descNodeStatusAllocatableMemory, n.Status.Allocatable, v1.ResourceMemory)
		addResource(descNodeStatusAllocatablePods, n.Status.Allocatable, v1.ResourcePods)
	}

	capacity := n.Status.Capacity
	allocatable := n.Status.Allocatable

	for resourceName, val := range capacity {
		switch resourceName {
		case v1.ResourceCPU:
			addGauge(descNodeStatusCapacity, float64(val.MilliValue())/1000, sanitizeLabelName(string(resourceName)), string(constant.UnitCore))
		case v1.ResourceStorage:
			fallthrough
		case v1.ResourceEphemeralStorage:
			fallthrough
		case v1.ResourceMemory:
			addGauge(descNodeStatusCapacity, float64(val.MilliValue())/1000, sanitizeLabelName(string(resourceName)), string(constant.UnitByte))
		case v1.ResourcePods:
			addGauge(descNodeStatusCapacity, float64(val.MilliValue())/1000, sanitizeLabelName(string(resourceName)), string(constant.UnitInteger))
		default:
			if helper.IsHugePageResourceName(resourceName) {
				addGauge(descNodeStatusCapacity, float64(val.MilliValue())/1000, sanitizeLabelName(string(resourceName)), string(constant.UnitByte))
			}
			if helper.IsAttachableVolumeResourceName(resourceName) {
				addGauge(descNodeStatusCapacity, float64(val.MilliValue())/1000, sanitizeLabelName(string(resourceName)), string(constant.UnitByte))
			}
			if helper.IsExtendedResourceName(resourceName) {
				addGauge(descNodeStatusCapacity, float64(val.MilliValue())/1000, sanitizeLabelName(string(resourceName)), string(constant.UnitInteger))
			}
		}
	}

	for resourceName, val := range allocatable {
		switch resourceName {
		case v1.ResourceCPU:
			addGauge(descNodeStatusAllocatable, float64(val.MilliValue())/1000, sanitizeLabelName(string(resourceName)), string(constant.UnitCore))
		case v1.ResourceStorage:
			fallthrough
		case v1.ResourceEphemeralStorage:
			fallthrough
		case v1.ResourceMemory:
			addGauge(descNodeStatusAllocatable, float64(val.MilliValue())/1000, sanitizeLabelName(string(resourceName)), string(constant.UnitByte))
		case v1.ResourcePods:
			addGauge(descNodeStatusAllocatable, float64(val.MilliValue())/1000, sanitizeLabelName(string(resourceName)), string(constant.UnitInteger))
		default:
			if helper.IsHugePageResourceName(resourceName) {
				addGauge(descNodeStatusAllocatable, float64(val.MilliValue())/1000, sanitizeLabelName(string(resourceName)), string(constant.UnitByte))
			}
			if helper.IsAttachableVolumeResourceName(resourceName) {
				addGauge(descNodeStatusAllocatable, float64(val.MilliValue())/1000, sanitizeLabelName(string(resourceName)), string(constant.UnitByte))
			}
			if helper.IsExtendedResourceName(resourceName) {
				addGauge(descNodeStatusAllocatable, float64(val.MilliValue())/1000, sanitizeLabelName(string(resourceName)), string(constant.UnitInteger))
			}
		}
	}

	return ms
}
