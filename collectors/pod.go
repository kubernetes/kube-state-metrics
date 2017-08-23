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
	"encoding/json"
	"regexp"
	"strconv"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	invalidLabelCharRE         = regexp.MustCompile(`[^a-zA-Z0-9_]`)
	descPodLabelsName          = "kube_pod_labels"
	descPodLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descPodLabelsDefaultLabels = []string{"namespace", "pod"}

	descPodInfo = prometheus.NewDesc(
		"kube_pod_info",
		"Information about pod.",
		[]string{"namespace", "pod", "host_ip", "pod_ip", "node", "created_by_kind", "created_by_name"}, nil,
	)

	descPodStartTime = prometheus.NewDesc(
		"kube_pod_start_time",
		"Start time in unix timestamp for a pod.",
		[]string{"namespace", "pod"}, nil,
	)

	descPodOwner = prometheus.NewDesc(
		"kube_pod_owner",
		"Information about the Pod's owner.",
		[]string{"namespace", "pod", "owner_kind", "owner_name", "owner_is_controller"}, nil,
	)

	descPodLabels = prometheus.NewDesc(
		descPodLabelsName,
		descPodLabelsHelp,
		descPodLabelsDefaultLabels, nil,
	)

	descPodStatusPhase = prometheus.NewDesc(
		"kube_pod_status_phase",
		"The pods current phase.",
		[]string{"namespace", "pod", "phase"}, nil,
	)

	descPodStatusReady = prometheus.NewDesc(
		"kube_pod_status_ready",
		"Describes whether the pod is ready to serve requests.",
		[]string{"namespace", "pod", "condition"}, nil,
	)

	descPodStatusScheduled = prometheus.NewDesc(
		"kube_pod_status_scheduled",
		"Describes the status of the scheduling process for the pod.",
		[]string{"namespace", "pod", "condition"}, nil,
	)

	descPodContainerInfo = prometheus.NewDesc(
		"kube_pod_container_info",
		"Information about a container in a pod.",
		[]string{"namespace", "pod", "container", "image", "image_id", "container_id"}, nil,
	)

	descPodContainerStatusWaiting = prometheus.NewDesc(
		"kube_pod_container_status_waiting",
		"Describes whether the container is currently in waiting state.",
		[]string{"namespace", "pod", "container"}, nil,
	)

	descPodContainerStatusRunning = prometheus.NewDesc(
		"kube_pod_container_status_running",
		"Describes whether the container is currently in running state.",
		[]string{"namespace", "pod", "container"}, nil,
	)

	descPodContainerStatusTerminated = prometheus.NewDesc(
		"kube_pod_container_status_terminated",
		"Describes whether the container is currently in terminated state.",
		[]string{"namespace", "pod", "container"}, nil,
	)

	descPodContainerStatusReady = prometheus.NewDesc(
		"kube_pod_container_status_ready",
		"Describes whether the containers readiness check succeeded.",
		[]string{"namespace", "pod", "container"}, nil,
	)

	descPodContainerStatusRestarts = prometheus.NewDesc(
		"kube_pod_container_status_restarts",
		"The number of container restarts per container.",
		[]string{"namespace", "pod", "container"}, nil,
	)

	descPodContainerResourceRequestsCpuCores = prometheus.NewDesc(
		"kube_pod_container_resource_requests_cpu_cores",
		"The number of requested cpu cores by a container.",
		[]string{"namespace", "pod", "container", "node"}, nil,
	)

	descPodContainerResourceRequestsMemoryBytes = prometheus.NewDesc(
		"kube_pod_container_resource_requests_memory_bytes",
		"The number of requested memory bytes  by a container.",
		[]string{"namespace", "pod", "container", "node"}, nil,
	)

	descPodContainerResourceLimitsCpuCores = prometheus.NewDesc(
		"kube_pod_container_resource_limits_cpu_cores",
		"The limit on cpu cores to be used by a container.",
		[]string{"namespace", "pod", "container", "node"}, nil,
	)

	descPodContainerResourceLimitsMemoryBytes = prometheus.NewDesc(
		"kube_pod_container_resource_limits_memory_bytes",
		"The limit on memory to be used by a container in bytes.",
		[]string{"namespace", "pod", "container", "node"}, nil,
	)
)

type PodLister func() ([]v1.Pod, error)

func (l PodLister) List() ([]v1.Pod, error) {
	return l()
}

func RegisterPodCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface) {
	client := kubeClient.CoreV1().RESTClient()
	plw := cache.NewListWatchFromClient(client, "pods", api.NamespaceAll, nil)
	pinf := cache.NewSharedInformer(plw, &v1.Pod{}, resyncPeriod)

	podLister := PodLister(func() (pods []v1.Pod, err error) {
		for _, m := range pinf.GetStore().List() {
			pods = append(pods, *m.(*v1.Pod))
		}
		return pods, nil
	})

	registry.MustRegister(&podCollector{store: podLister})
	go pinf.Run(context.Background().Done())
}

type podStore interface {
	List() (pods []v1.Pod, err error)
}

// podCollector collects metrics about all pods in the cluster.
type podCollector struct {
	store podStore
}

// Describe implements the prometheus.Collector interface.
func (pc *podCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descPodInfo
	ch <- descPodStartTime
	ch <- descPodOwner
	ch <- descPodLabels
	ch <- descPodStatusPhase
	ch <- descPodStatusReady
	ch <- descPodStatusScheduled
	ch <- descPodContainerInfo
	ch <- descPodContainerStatusWaiting
	ch <- descPodContainerStatusRunning
	ch <- descPodContainerStatusTerminated
	ch <- descPodContainerStatusReady
	ch <- descPodContainerStatusRestarts
	ch <- descPodContainerResourceRequestsCpuCores
	ch <- descPodContainerResourceRequestsMemoryBytes
	ch <- descPodContainerResourceLimitsCpuCores
	ch <- descPodContainerResourceLimitsMemoryBytes
}

func extractCreatedBy(annotation map[string]string) *api.ObjectReference {
	value, ok := annotation[api.CreatedByAnnotation]
	if ok {
		var r api.SerializedReference
		err := json.Unmarshal([]byte(value), &r)
		if err == nil {
			return &r.Reference
		}
	}
	return nil
}

// Collect implements the prometheus.Collector interface.
func (pc *podCollector) Collect(ch chan<- prometheus.Metric) {
	pods, err := pc.store.List()
	if err != nil {
		glog.Errorf("listing pods failed: %s", err)
		return
	}
	for _, p := range pods {
		pc.collectPod(ch, p)
	}
}

func kubeLabelsToPrometheusLabels(labels map[string]string) ([]string, []string) {
	labelKeys := make([]string, len(labels))
	labelValues := make([]string, len(labels))
	i := 0
	for k, v := range labels {
		labelKeys[i] = "label_" + sanitizeLabelName(k)
		labelValues[i] = v
		i++
	}
	return labelKeys, labelValues
}

func sanitizeLabelName(s string) string {
	return invalidLabelCharRE.ReplaceAllString(s, "_")
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

	createdBy := extractCreatedBy(p.Annotations)
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

	addGauge(descPodInfo, 1, p.Status.HostIP, p.Status.PodIP, nodeName, createdByKind, createdByName)

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

	if p := p.Status.Phase; p != "" {
		addGauge(descPodStatusPhase, boolFloat64(p == v1.PodPending), string(v1.PodPending))
		addGauge(descPodStatusPhase, boolFloat64(p == v1.PodRunning), string(v1.PodRunning))
		addGauge(descPodStatusPhase, boolFloat64(p == v1.PodSucceeded), string(v1.PodSucceeded))
		addGauge(descPodStatusPhase, boolFloat64(p == v1.PodFailed), string(v1.PodFailed))
		addGauge(descPodStatusPhase, boolFloat64(p == v1.PodUnknown), string(v1.PodUnknown))
	}

	for _, c := range p.Status.Conditions {
		switch c.Type {
		case v1.PodReady:
			addConditionMetrics(ch, descPodStatusReady, c.Status, p.Namespace, p.Name)
		case v1.PodScheduled:
			addConditionMetrics(ch, descPodStatusScheduled, c.Status, p.Namespace, p.Name)
		}
	}

	for _, cs := range p.Status.ContainerStatuses {
		addGauge(descPodContainerInfo, 1,
			cs.Name, cs.Image, cs.ImageID, cs.ContainerID,
		)
		addGauge(descPodContainerStatusWaiting, boolFloat64(cs.State.Waiting != nil), cs.Name)
		addGauge(descPodContainerStatusRunning, boolFloat64(cs.State.Running != nil), cs.Name)
		addGauge(descPodContainerStatusTerminated, boolFloat64(cs.State.Terminated != nil), cs.Name)
		addGauge(descPodContainerStatusReady, boolFloat64(cs.Ready), cs.Name)
		addCounter(descPodContainerStatusRestarts, float64(cs.RestartCount), cs.Name)
	}

	for _, c := range p.Spec.Containers {
		req := c.Resources.Requests
		lim := c.Resources.Limits

		if cpu, ok := req[v1.ResourceCPU]; ok {
			addGauge(descPodContainerResourceRequestsCpuCores, float64(cpu.MilliValue())/1000,
				c.Name, nodeName)
		}
		if mem, ok := req[v1.ResourceMemory]; ok {
			addGauge(descPodContainerResourceRequestsMemoryBytes, float64(mem.Value()),
				c.Name, nodeName)
		}

		if cpu, ok := lim[v1.ResourceCPU]; ok {
			addGauge(descPodContainerResourceLimitsCpuCores, float64(cpu.MilliValue())/1000,
				c.Name, nodeName)
		}
		if mem, ok := lim[v1.ResourceMemory]; ok {
			addGauge(descPodContainerResourceLimitsMemoryBytes, float64(mem.Value()),
				c.Name, nodeName)
		}
	}
}
