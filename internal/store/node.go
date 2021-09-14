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
	"strings"

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
	descNodeLabelsName          = "kube_node_labels"
	descNodeLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descNodeLabelsDefaultLabels = []string{"node"}
)

func nodeMetricFamilies(allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		createNodeCreatedFamilyGenerator(),
		createNodeInfoFamilyGenerator(),
		createNodeLabelsGenerator(allowLabelsList),
		createNodeRoleFamilyGenerator(),
		createNodeSpecTaintFamilyGenerator(),
		createNodeSpecUnschedulableFamilyGenerator(),
		createNodeStatusAllocatableFamilyGenerator(),
		createNodeStatusCapacityFamilyGenerator(),
		createNodeStatusConditionFamilyGenerator(),
	}
}

func createNodeCreatedFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGenerator(
		"kube_node_created",
		"Unix creation timestamp",
		metric.Gauge,
		"",
		wrapNodeFunc(func(n *v1.Node) *metric.Family {
			ms := []*metric.Metric{}

			if !n.CreationTimestamp.IsZero() {
				ms = append(ms, &metric.Metric{

					Value: float64(n.CreationTimestamp.Unix()),
				})
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createNodeInfoFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGenerator(
		"kube_node_info",
		"Information about a cluster node.",
		metric.Gauge,
		"",
		wrapNodeFunc(func(n *v1.Node) *metric.Family {
			labelKeys := []string{
				"kernel_version",
				"os_image",
				"container_runtime_version",
				"kubelet_version",
				"kubeproxy_version",
				"provider_id",
				"pod_cidr",
			}
			labelValues := []string{
				n.Status.NodeInfo.KernelVersion,
				n.Status.NodeInfo.OSImage,
				n.Status.NodeInfo.ContainerRuntimeVersion,
				n.Status.NodeInfo.KubeletVersion,
				n.Status.NodeInfo.KubeProxyVersion,
				n.Spec.ProviderID,
				n.Spec.PodCIDR,
			}

			internalIP := ""
			for _, address := range n.Status.Addresses {
				if address.Type == "InternalIP" {
					internalIP = address.Address
				}
			}
			labelKeys = append(labelKeys, "internal_ip")
			labelValues = append(labelValues, internalIP)

			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						LabelKeys:   labelKeys,
						LabelValues: labelValues,
						Value:       1,
					},
				},
			}
		}),
	)
}

func createNodeLabelsGenerator(allowLabelsList []string) generator.FamilyGenerator {
	return *generator.NewFamilyGenerator(
		descNodeLabelsName,
		descNodeLabelsHelp,
		metric.Gauge,
		"",
		wrapNodeFunc(func(n *v1.Node) *metric.Family {
			labelKeys, labelValues := createLabelKeysValues(n.Labels, allowLabelsList)
			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						LabelKeys:   labelKeys,
						LabelValues: labelValues,
						Value:       1,
					},
				},
			}
		}),
	)
}

func createNodeRoleFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGenerator(
		"kube_node_role",
		"The role of a cluster node.",
		metric.Gauge,
		"",
		wrapNodeFunc(func(n *v1.Node) *metric.Family {
			const prefix = "node-role.kubernetes.io/"
			ms := []*metric.Metric{}
			for lbl := range n.Labels {
				if strings.HasPrefix(lbl, prefix) {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"role"},
						LabelValues: []string{strings.TrimPrefix(lbl, prefix)},
						Value:       float64(1),
					})
				}
			}
			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createNodeSpecTaintFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGenerator(
		"kube_node_spec_taint",
		"The taint of a cluster node.",
		metric.Gauge,
		"",
		wrapNodeFunc(func(n *v1.Node) *metric.Family {
			ms := make([]*metric.Metric, len(n.Spec.Taints))

			for i, taint := range n.Spec.Taints {
				// Taints are applied to repel pods from nodes that do not have a corresponding
				// toleration.  Many node conditions are optionally reflected as taints
				// by the node controller in order to simplify scheduling constraints.
				ms[i] = &metric.Metric{
					LabelKeys:   []string{"key", "value", "effect"},
					LabelValues: []string{taint.Key, taint.Value, string(taint.Effect)},
					Value:       1,
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createNodeSpecUnschedulableFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGenerator(
		"kube_node_spec_unschedulable",
		"Whether a node can schedule new pods.",
		metric.Gauge,
		"",
		wrapNodeFunc(func(n *v1.Node) *metric.Family {
			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						Value: boolFloat64(n.Spec.Unschedulable),
					},
				},
			}
		}),
	)
}

func createNodeStatusAllocatableFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGenerator(
		"kube_node_status_allocatable",
		"The allocatable for different resources of a node that are available for scheduling.",
		metric.Gauge,
		"",
		wrapNodeFunc(func(n *v1.Node) *metric.Family {
			ms := []*metric.Metric{}

			allocatable := n.Status.Allocatable

			for resourceName, val := range allocatable {
				switch resourceName {
				case v1.ResourceCPU:
					ms = append(ms, &metric.Metric{
						LabelValues: []string{
							sanitizeLabelName(string(resourceName)),
							string(constant.UnitCore),
						},
						Value: float64(val.MilliValue()) / 1000,
					})
				case v1.ResourceStorage:
					fallthrough
				case v1.ResourceEphemeralStorage:
					fallthrough
				case v1.ResourceMemory:
					ms = append(ms, &metric.Metric{
						LabelValues: []string{
							sanitizeLabelName(string(resourceName)),
							string(constant.UnitByte),
						},
						Value: float64(val.MilliValue()) / 1000,
					})
				case v1.ResourcePods:
					ms = append(ms, &metric.Metric{
						LabelValues: []string{
							sanitizeLabelName(string(resourceName)),
							string(constant.UnitInteger),
						},
						Value: float64(val.MilliValue()) / 1000,
					})
				default:
					if isHugePageResourceName(resourceName) {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitByte),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					}
					if isAttachableVolumeResourceName(resourceName) {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitByte),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					}
					if isExtendedResourceName(resourceName) {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitInteger),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					}
				}
			}

			for _, m := range ms {
				m.LabelKeys = []string{"resource", "unit"}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createNodeStatusCapacityFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGenerator(
		"kube_node_status_capacity",
		"The capacity for different resources of a node.",
		metric.Gauge,
		"",
		wrapNodeFunc(func(n *v1.Node) *metric.Family {
			ms := []*metric.Metric{}

			capacity := n.Status.Capacity
			for resourceName, val := range capacity {
				switch resourceName {
				case v1.ResourceCPU:
					ms = append(ms, &metric.Metric{
						LabelValues: []string{
							sanitizeLabelName(string(resourceName)),
							string(constant.UnitCore),
						},
						Value: float64(val.MilliValue()) / 1000,
					})
				case v1.ResourceStorage:
					fallthrough
				case v1.ResourceEphemeralStorage:
					fallthrough
				case v1.ResourceMemory:
					ms = append(ms, &metric.Metric{
						LabelValues: []string{
							sanitizeLabelName(string(resourceName)),
							string(constant.UnitByte),
						},
						Value: float64(val.MilliValue()) / 1000,
					})
				case v1.ResourcePods:
					ms = append(ms, &metric.Metric{
						LabelValues: []string{
							sanitizeLabelName(string(resourceName)),
							string(constant.UnitInteger),
						},
						Value: float64(val.MilliValue()) / 1000,
					})
				default:
					if isHugePageResourceName(resourceName) {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitByte),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					}
					if isAttachableVolumeResourceName(resourceName) {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitByte),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					}
					if isExtendedResourceName(resourceName) {
						ms = append(ms, &metric.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitInteger),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					}
				}
			}

			for _, metric := range ms {
				metric.LabelKeys = []string{"resource", "unit"}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

// createNodeStatusConditionFamilyGenerator returns an all-in-one metric family
// containing all conditions for extensibility. Third party plugin may report
// customized condition for cluster node (e.g. node-problem-detector), and
// Kubernetes may add new core conditions in future.
func createNodeStatusConditionFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGenerator(
		"kube_node_status_condition",
		"The condition of a cluster node.",
		metric.Gauge,
		"",
		wrapNodeFunc(func(n *v1.Node) *metric.Family {
			ms := make([]*metric.Metric, len(n.Status.Conditions)*len(conditionStatuses))

			// Collect node conditions and while default to false.
			for i, c := range n.Status.Conditions {
				conditionMetrics := addConditionMetrics(c.Status)

				for j, m := range conditionMetrics {
					metric := m

					metric.LabelKeys = []string{"condition", "status"}
					metric.LabelValues = append([]string{string(c.Type)}, metric.LabelValues...)

					ms[i*len(conditionStatuses)+j] = metric
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func wrapNodeFunc(f func(*v1.Node) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		node := obj.(*v1.Node)

		metricFamily := f(node)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descNodeLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{node.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createNodeListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Nodes().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Nodes().Watch(context.TODO(), opts)
		},
	}
}
