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

	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descDeploymentAnnotationsName     = "kube_deployment_annotations"
	descDeploymentAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descDeploymentLabelsName          = "kube_deployment_labels"
	descDeploymentLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descDeploymentLabelsDefaultLabels = []string{"namespace", "deployment"}
)

func deploymentMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_deployment_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				ms := []*metric.Metric{}

				if !d.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(d.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_deployment_status_replicas",
			"The number of replicas per deployment.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.Status.Replicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_deployment_status_replicas_ready",
			"The number of ready replicas per deployment.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.Status.ReadyReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_deployment_status_replicas_available",
			"The number of available replicas per deployment.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.Status.AvailableReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_deployment_status_replicas_unavailable",
			"The number of unavailable replicas per deployment.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.Status.UnavailableReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_deployment_status_replicas_updated",
			"The number of updated replicas per deployment.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.Status.UpdatedReplicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_deployment_status_observed_generation",
			"The generation observed by the deployment controller.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.Status.ObservedGeneration),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_deployment_status_condition",
			"The current status conditions of a deployment.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				ms := make([]*metric.Metric, len(d.Status.Conditions)*len(conditionStatuses))

				for i, c := range d.Status.Conditions {
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
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_deployment_spec_replicas",
			"Number of desired pods for a deployment.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(*d.Spec.Replicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_deployment_spec_paused",
			"Whether the deployment is paused and will not be processed by the deployment controller.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: boolFloat64(d.Spec.Paused),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_deployment_spec_strategy_rollingupdate_max_unavailable",
			"Maximum number of unavailable replicas during a rolling update of a deployment.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				if d.Spec.Strategy.RollingUpdate == nil {
					return &metric.Family{}
				}

				maxUnavailable, err := intstr.GetScaledValueFromIntOrPercent(d.Spec.Strategy.RollingUpdate.MaxUnavailable, int(*d.Spec.Replicas), false)
				if err != nil {
					panic(err)
				}

				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(maxUnavailable),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_deployment_spec_strategy_rollingupdate_max_surge",
			"Maximum number of replicas that can be scheduled above the desired number of replicas during a rolling update of a deployment.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				if d.Spec.Strategy.RollingUpdate == nil {
					return &metric.Family{}
				}

				maxSurge, err := intstr.GetScaledValueFromIntOrPercent(d.Spec.Strategy.RollingUpdate.MaxSurge, int(*d.Spec.Replicas), true)
				if err != nil {
					panic(err)
				}

				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(maxSurge),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_deployment_metadata_generation",
			"Sequence number representing a specific generation of the desired state.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(d.Generation),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descDeploymentAnnotationsName,
			descDeploymentAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", d.Annotations, allowAnnotationsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   annotationKeys,
							LabelValues: annotationValues,
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descDeploymentLabelsName,
			descDeploymentLabelsHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapDeploymentFunc(func(d *v1.Deployment) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", d.Labels, allowLabelsList)
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
		),
	}
}

func wrapDeploymentFunc(f func(*v1.Deployment) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		deployment := obj.(*v1.Deployment)

		metricFamily := f(deployment)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descDeploymentLabelsDefaultLabels, []string{deployment.Namespace, deployment.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createDeploymentListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.AppsV1().Deployments(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.AppsV1().Deployments(ns).Watch(context.TODO(), opts)
		},
	}
}
