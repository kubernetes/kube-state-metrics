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

	nodeMetricFamilies = []metrics.FamilyGenerator{
		metrics.FamilyGenerator{
			Name: "kube_node_info",
			Type: metrics.MetricTypeGauge,
			Help: "Information about a cluster node.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name: "kube_node_info",
					LabelKeys: []string{
						"kernel_version",
						"os_image",
						"container_runtime_version",
						"kubelet_version",
						"kubeproxy_version",
						"provider_id",
					},
					LabelValues: []string{
						n.Status.NodeInfo.KernelVersion,
						n.Status.NodeInfo.OSImage,
						n.Status.NodeInfo.ContainerRuntimeVersion,
						n.Status.NodeInfo.KubeletVersion,
						n.Status.NodeInfo.KubeProxyVersion,
						n.Spec.ProviderID,
					},
					Value: 1,
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_node_created",
			Type: metrics.MetricTypeGauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				f := metrics.Family{}

				if !n.CreationTimestamp.IsZero() {
					f = append(f, &metrics.Metric{
						Name:  "kube_node_created",
						Value: float64(n.CreationTimestamp.Unix()),
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: descNodeLabelsName,
			Type: metrics.MetricTypeGauge,
			Help: descNodeLabelsHelp,
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(n.Labels)
				return metrics.Family{&metrics.Metric{
					Name:        descNodeLabelsName,
					LabelKeys:   labelKeys,
					LabelValues: labelValues,
					Value:       1,
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_node_spec_unschedulable",
			Type: metrics.MetricTypeGauge,
			Help: "Whether a node can schedule new pods.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				return metrics.Family{&metrics.Metric{
					Name:  "kube_node_spec_unschedulable",
					Value: boolFloat64(n.Spec.Unschedulable),
				}}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_node_spec_taint",
			Type: metrics.MetricTypeGauge,
			Help: "The taint of a cluster node.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				f := metrics.Family{}

				for _, taint := range n.Spec.Taints {
					// Taints are applied to repel pods from nodes that do not have a corresponding
					// toleration.  Many node conditions are optionally reflected as taints
					// by the node controller in order to simplify scheduling constraints.
					f = append(f, &metrics.Metric{
						Name:        "kube_node_spec_taint",
						LabelKeys:   []string{"key", "value", "effect"},
						LabelValues: []string{taint.Key, taint.Value, string(taint.Effect)},
						Value:       1,
					})
				}

				return f
			}),
		},
		// This all-in-one metric family contains all conditions for extensibility.
		// Third party plugin may report customized condition for cluster node
		// (e.g. node-problem-detector), and Kubernetes may add new core
		// conditions in future.
		metrics.FamilyGenerator{
			Name: "kube_node_status_condition",
			Type: metrics.MetricTypeGauge,
			Help: "The condition of a cluster node.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				f := metrics.Family{}

				// Collect node conditions and while default to false.
				for _, c := range n.Status.Conditions {
					ms := addConditionMetrics(c.Status)
					for _, metric := range ms {
						metric.Name = "kube_node_status_condition"
						metric.LabelKeys = []string{"condition", "status"}
						metric.LabelValues = append([]string{string(c.Type)}, metric.LabelValues...)
					}
					f = append(f, ms...)
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_node_status_phase",
			Type: metrics.MetricTypeGauge,
			Help: "The phase the node is currently in.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				f := metrics.Family{}

				// Set current phase to 1, others to 0 if it is set.
				if p := n.Status.Phase; p != "" {
					f = append(f,
						&metrics.Metric{
							LabelValues: []string{string(v1.NodePending)},
							Value:       boolFloat64(p == v1.NodePending),
						},
						&metrics.Metric{
							LabelValues: []string{string(v1.NodeRunning)},
							Value:       boolFloat64(p == v1.NodeRunning),
						},
						&metrics.Metric{
							LabelValues: []string{string(v1.NodeTerminated)},
							Value:       boolFloat64(p == v1.NodeTerminated),
						},
					)
				}

				for _, metric := range f {
					metric.Name = "kube_node_status_phase"
					metric.LabelKeys = []string{"phase"}
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_node_status_capacity",
			Type: metrics.MetricTypeGauge,
			Help: "The capacity for different resources of a node.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				f := metrics.Family{}

				capacity := n.Status.Capacity
				for resourceName, val := range capacity {
					switch resourceName {
					case v1.ResourceCPU:
						f = append(f, &metrics.Metric{
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
						f = append(f, &metrics.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitByte),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					case v1.ResourcePods:
						f = append(f, &metrics.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitInteger),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					default:
						if helper.IsHugePageResourceName(resourceName) {
							f = append(f, &metrics.Metric{
								LabelValues: []string{
									sanitizeLabelName(string(resourceName)),
									string(constant.UnitByte),
								},
								Value: float64(val.MilliValue()) / 1000,
							})
						}
						if helper.IsAttachableVolumeResourceName(resourceName) {
							f = append(f, &metrics.Metric{
								LabelValues: []string{
									sanitizeLabelName(string(resourceName)),
									string(constant.UnitByte),
								},
								Value: float64(val.MilliValue()) / 1000,
							})
						}
						if helper.IsExtendedResourceName(resourceName) {
							f = append(f, &metrics.Metric{
								LabelValues: []string{
									sanitizeLabelName(string(resourceName)),
									string(constant.UnitInteger),
								},
								Value: float64(val.MilliValue()) / 1000,
							})
						}
					}
				}

				for _, metric := range f {
					metric.Name = "kube_node_status_capacity"
					metric.LabelKeys = []string{"resource", "unit"}
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_node_status_capacity_pods",
			Type: metrics.MetricTypeGauge,
			Help: "The total pod resources of the node.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				f := metrics.Family{}

				// Add capacity and allocatable resources if they are set.
				if v, ok := n.Status.Capacity[v1.ResourcePods]; ok {
					f = append(f, &metrics.Metric{
						Name:  "kube_node_status_capacity_pods",
						Value: float64(v.MilliValue()) / 1000,
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_node_status_capacity_cpu_cores",
			Type: metrics.MetricTypeGauge,
			Help: "The total CPU resources of the node.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				f := metrics.Family{}

				// Add capacity and allocatable resources if they are set.
				if v, ok := n.Status.Capacity[v1.ResourceCPU]; ok {
					f = append(f, &metrics.Metric{
						Name:  "kube_node_status_capacity_cpu_cores",
						Value: float64(v.MilliValue()) / 1000,
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_node_status_capacity_memory_bytes",
			Type: metrics.MetricTypeGauge,
			Help: "The total memory resources of the node.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				f := metrics.Family{}

				// Add capacity and allocatable resources if they are set.
				if v, ok := n.Status.Capacity[v1.ResourceMemory]; ok {
					f = append(f, &metrics.Metric{
						Name:  "kube_node_status_capacity_memory_bytes",
						Value: float64(v.MilliValue()) / 1000,
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_node_status_allocatable",
			Type: metrics.MetricTypeGauge,
			Help: "The allocatable for different resources of a node that are available for scheduling.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				f := metrics.Family{}

				allocatable := n.Status.Allocatable

				for resourceName, val := range allocatable {
					switch resourceName {
					case v1.ResourceCPU:
						f = append(f, &metrics.Metric{
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
						f = append(f, &metrics.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitByte),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					case v1.ResourcePods:
						f = append(f, &metrics.Metric{
							LabelValues: []string{
								sanitizeLabelName(string(resourceName)),
								string(constant.UnitInteger),
							},
							Value: float64(val.MilliValue()) / 1000,
						})
					default:
						if helper.IsHugePageResourceName(resourceName) {
							f = append(f, &metrics.Metric{
								LabelValues: []string{
									sanitizeLabelName(string(resourceName)),
									string(constant.UnitByte),
								},
								Value: float64(val.MilliValue()) / 1000,
							})
						}
						if helper.IsAttachableVolumeResourceName(resourceName) {
							f = append(f, &metrics.Metric{
								LabelValues: []string{
									sanitizeLabelName(string(resourceName)),
									string(constant.UnitByte),
								},
								Value: float64(val.MilliValue()) / 1000,
							})
						}
						if helper.IsExtendedResourceName(resourceName) {
							f = append(f, &metrics.Metric{
								LabelValues: []string{
									sanitizeLabelName(string(resourceName)),
									string(constant.UnitInteger),
								},
								Value: float64(val.MilliValue()) / 1000,
							})
						}
					}
				}

				for _, m := range f {
					m.Name = "kube_node_status_allocatable"
					m.LabelKeys = []string{"resource", "unit"}
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_node_status_allocatable_pods",
			Type: metrics.MetricTypeGauge,
			Help: "The pod resources of a node that are available for scheduling.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				f := metrics.Family{}

				// Add capacity and allocatable resources if they are set.
				if v, ok := n.Status.Allocatable[v1.ResourcePods]; ok {
					f = append(f, &metrics.Metric{
						Name:  "kube_node_status_allocatable_pods",
						Value: float64(v.MilliValue()) / 1000,
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_node_status_allocatable_cpu_cores",
			Type: metrics.MetricTypeGauge,
			Help: "The CPU resources of a node that are available for scheduling.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				f := metrics.Family{}

				// Add capacity and allocatable resources if they are set.
				if v, ok := n.Status.Allocatable[v1.ResourceCPU]; ok {
					f = append(f, &metrics.Metric{
						Name:  "kube_node_status_allocatable_cpu_cores",
						Value: float64(v.MilliValue()) / 1000,
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_node_status_allocatable_memory_bytes",
			Type: metrics.MetricTypeGauge,
			Help: "The memory resources of a node that are available for scheduling.",
			GenerateFunc: wrapNodeFunc(func(n *v1.Node) metrics.Family {
				f := metrics.Family{}

				// Add capacity and allocatable resources if they are set.
				if v, ok := n.Status.Allocatable[v1.ResourceMemory]; ok {
					f = append(f, &metrics.Metric{
						Name:  "kube_node_status_allocatable_memory_bytes",
						Value: float64(v.MilliValue()) / 1000,
					})
				}

				return f
			}),
		},
	}
)

func wrapNodeFunc(f func(*v1.Node) metrics.Family) func(interface{}) metrics.Family {
	return func(obj interface{}) metrics.Family {
		node := obj.(*v1.Node)

		metricFamily := f(node)

		for _, m := range metricFamily {
			m.LabelKeys = append(descNodeLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{node.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

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
