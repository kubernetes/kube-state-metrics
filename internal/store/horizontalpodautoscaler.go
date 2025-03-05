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

	autoscaling "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	basemetrics "k8s.io/component-base/metrics"

	"slices"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

type metricTargetType int

const (
	value metricTargetType = iota
	utilization
	average
)

func (m metricTargetType) String() string {
	return [...]string{"value", "utilization", "average"}[m]
}

var (
	descHorizontalPodAutoscalerAnnotationsName     = "kube_horizontalpodautoscaler_annotations"
	descHorizontalPodAutoscalerAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descHorizontalPodAutoscalerLabelsName          = "kube_horizontalpodautoscaler_labels"
	descHorizontalPodAutoscalerLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descHorizontalPodAutoscalerLabelsDefaultLabels = []string{"namespace", "horizontalpodautoscaler"}

	targetMetricLabels    = []string{"metric_name", "metric_target_type"}
	containerMetricLabels = []string{"metric_name", "metric_target_type", "container_name"}
	objectMetricLabels    = []string{"metric_name", "metric_target_type", "target_name"}
)

func hpaMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		createHPAInfo(),
		createHPAMetaDataGeneration(),
		createHPASpecMaxReplicas(),
		createHPASpecMinReplicas(),
		createHPASpecTargetContainerMetric(),
		createHPAStatusTargetContainerMetric(),
		createHPAStatusTargetObjectMetric(),
		createHPASpecTargetMetric(),
		createHPASpecTargetObjectMetric(),
		createHPAStatusTargetMetric(),
		createHPAStatusCurrentReplicas(),
		createHPAStatusDesiredReplicas(),
		createHPAAnnotations(allowAnnotationsList),
		createHPALabels(allowLabelsList),
		createHPAStatusCondition(),
	}
}

func wrapHPAFunc(f func(*autoscaling.HorizontalPodAutoscaler) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		hpa := obj.(*autoscaling.HorizontalPodAutoscaler)

		metricFamily := f(hpa)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descHorizontalPodAutoscalerLabelsDefaultLabels, []string{hpa.Namespace, hpa.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createHPAListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.AutoscalingV2().HorizontalPodAutoscalers(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.AutoscalingV2().HorizontalPodAutoscalers(ns).Watch(context.TODO(), opts)
		},
	}
}

func createHPAInfo() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_horizontalpodautoscaler_info",
		"Information about this autoscaler.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
			labelKeys := []string{"scaletargetref_kind", "scaletargetref_name"}
			labelValues := []string{a.Spec.ScaleTargetRef.Kind, a.Spec.ScaleTargetRef.Name}
			if a.Spec.ScaleTargetRef.APIVersion != "" {
				labelKeys = append([]string{"scaletargetref_api_version"}, labelKeys...)
				labelValues = append([]string{a.Spec.ScaleTargetRef.APIVersion}, labelValues...)
			}
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

func createHPAMetaDataGeneration() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_horizontalpodautoscaler_metadata_generation",
		"The generation observed by the HorizontalPodAutoscaler controller.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						Value: float64(a.ObjectMeta.Generation),
					},
				},
			}
		}),
	)
}

func createHPASpecMaxReplicas() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_horizontalpodautoscaler_spec_max_replicas",
		"Upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						Value: float64(a.Spec.MaxReplicas),
					},
				},
			}
		}),
	)
}

func createHPASpecMinReplicas() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_horizontalpodautoscaler_spec_min_replicas",
		"Lower limit for the number of pods that can be set by the autoscaler, default 1.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						Value: float64(*a.Spec.MinReplicas),
					},
				},
			}
		}),
	)
}

func createHPASpecTargetContainerMetric() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_horizontalpodautoscaler_spec_target_container_metric",
		"The container metric specifications used by this autoscaler when calculating the desired replica count.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
			ms := make([]*metric.Metric, 0, len(a.Spec.Metrics))
			for _, m := range a.Spec.Metrics {
				var metricName string
				var metricTarget autoscaling.MetricTarget
				var containerName string
				// The variable maps the type of metric to the corresponding value
				metricMap := make(map[metricTargetType]float64)

				switch m.Type {
				case autoscaling.ContainerResourceMetricSourceType:
					metricName = string(m.ContainerResource.Name)
					metricTarget = m.ContainerResource.Target
					containerName = m.ContainerResource.Container
				default:
					// Skip unsupported metric type
					continue
				}

				if metricTarget.Value != nil {
					metricMap[value] = convertValueToFloat64(metricTarget.Value)
				}
				if metricTarget.AverageValue != nil {
					metricMap[average] = convertValueToFloat64(metricTarget.AverageValue)
				}
				if metricTarget.AverageUtilization != nil {
					metricMap[utilization] = float64(*metricTarget.AverageUtilization)
				}

				for metricTypeIndex, metricValue := range metricMap {
					ms = append(ms, &metric.Metric{
						LabelKeys:   containerMetricLabels,
						LabelValues: []string{metricName, metricTypeIndex.String(), containerName},
						Value:       metricValue,
					})
				}
			}

			return &metric.Family{Metrics: ms}
		}),
	)
}

func createHPASpecTarget(allowedTypes []autoscaling.MetricSourceType) generator.FamilyGenerator {
	metricName := "kube_horizontalpodautoscaler_spec_target_metric"
	metricDescription := "The metric specifications used by this autoscaler when calculating the desired replica count."
	if len(allowedTypes) == 1 && allowedTypes[0] == autoscaling.ObjectMetricSourceType {
		metricName = "kube_horizontalpodautoscaler_spec_target_object_metric"
		metricDescription = "The object metric specifications used by this autoscaler when calculating the desired replica count."
	}

	return *generator.NewFamilyGeneratorWithStability(
		metricName,
		metricDescription,
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
			ms := make([]*metric.Metric, 0, len(a.Spec.Metrics))
			for _, m := range a.Spec.Metrics {
				// Check whether the metric type is allowed.
				allowed := slices.Contains(allowedTypes, m.Type)
				if !allowed {
					continue
				}

				var metricName string
				var metricTarget autoscaling.MetricTarget
				var fullTargetName string // only used for ObjectMetricSourceType

				switch m.Type {
				case autoscaling.PodsMetricSourceType:
					metricName = m.Pods.Metric.Name
					metricTarget = m.Pods.Target
				case autoscaling.ResourceMetricSourceType:
					metricName = string(m.Resource.Name)
					metricTarget = m.Resource.Target
				case autoscaling.ExternalMetricSourceType:
					metricName = m.External.Metric.Name
					metricTarget = m.External.Target
				case autoscaling.ObjectMetricSourceType:
					metricName = m.Object.Metric.Name
					metricTarget = m.Object.Target
					fullTargetName = m.Object.DescribedObject.Name
				default:
					// Skip unsupported metric type.
					continue
				}
				// The variable maps the type of metric to the corresponding value
				metricMap := make(map[metricTargetType]float64)
				if metricTarget.Value != nil {
					metricMap[value] = convertValueToFloat64(metricTarget.Value)
				}
				if metricTarget.AverageValue != nil {
					metricMap[average] = convertValueToFloat64(metricTarget.AverageValue)
				}
				if metricTarget.AverageUtilization != nil {
					metricMap[utilization] = float64(*metricTarget.AverageUtilization)
				}

				for metricTypeIndex, metricValue := range metricMap {
					labelValues := []string{metricName, metricTypeIndex.String()}
					metricLabels := targetMetricLabels
					if m.Type == autoscaling.ObjectMetricSourceType {
						labelValues = append(labelValues, fullTargetName)
						metricLabels = objectMetricLabels
					}
					ms = append(ms, &metric.Metric{
						LabelKeys:   metricLabels,
						LabelValues: labelValues,
						Value:       metricValue,
					})
				}
			}
			return &metric.Family{Metrics: ms}
		}),
	)
}

func createHPASpecTargetObjectMetric() generator.FamilyGenerator {
	return createHPASpecTarget([]autoscaling.MetricSourceType{
		autoscaling.ObjectMetricSourceType,
	})
}

func createHPASpecTargetMetric() generator.FamilyGenerator {
	return createHPASpecTarget([]autoscaling.MetricSourceType{
		autoscaling.PodsMetricSourceType,
		autoscaling.ResourceMetricSourceType,
		autoscaling.ExternalMetricSourceType,
	})
}

func createHPAStatusTargetContainerMetric() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_horizontalpodautoscaler_status_target_container_metric",
		"The current container metric status used by this autoscaler when calculating the desired replica count.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
			ms := make([]*metric.Metric, 0, len(a.Status.CurrentMetrics))
			for _, m := range a.Status.CurrentMetrics {
				var metricName string
				var currentMetric autoscaling.MetricValueStatus
				var containerName string
				// The variable maps the type of metric to the corresponding value
				metricMap := make(map[metricTargetType]float64)

				switch m.Type {
				case autoscaling.ContainerResourceMetricSourceType:
					metricName = string(m.ContainerResource.Name)
					currentMetric = m.ContainerResource.Current
					containerName = m.ContainerResource.Container
				default:
					// Skip unsupported metric type
					continue
				}

				if currentMetric.Value != nil {
					metricMap[value] = convertValueToFloat64(currentMetric.Value)
				}
				if currentMetric.AverageValue != nil {
					metricMap[average] = convertValueToFloat64(currentMetric.AverageValue)
				}
				if currentMetric.AverageUtilization != nil {
					metricMap[utilization] = float64(*currentMetric.AverageUtilization)
				}

				for metricTypeIndex, metricValue := range metricMap {
					ms = append(ms, &metric.Metric{
						LabelKeys:   containerMetricLabels,
						LabelValues: []string{metricName, metricTypeIndex.String(), containerName},
						Value:       metricValue,
					})
				}
			}
			return &metric.Family{Metrics: ms}
		}),
	)
}

func createHPAStatusTargetObjectMetric() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_horizontalpodautoscaler_status_target_object_metric",
		"The current object metric status used by this autoscaler when calculating the desired replica count.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
			ms := make([]*metric.Metric, 0, len(a.Status.CurrentMetrics))
			for _, m := range a.Status.CurrentMetrics {
				var metricName string
				var currentMetric autoscaling.MetricValueStatus
				var fullTargetName string
				// The variable maps the type of metric to the corresponding value
				metricMap := make(map[metricTargetType]float64)

				switch m.Type {
				case autoscaling.ObjectMetricSourceType:
					metricName = m.Object.Metric.Name
					currentMetric = m.Object.Current
					fullTargetName = m.Object.DescribedObject.Name
				default:
					// Skip unsupported metric type
					continue
				}

				if currentMetric.Value != nil {
					metricMap[value] = convertValueToFloat64(currentMetric.Value)
				}
				if currentMetric.AverageValue != nil {
					metricMap[average] = convertValueToFloat64(currentMetric.AverageValue)
				}
				if currentMetric.AverageUtilization != nil {
					metricMap[utilization] = float64(*currentMetric.AverageUtilization)
				}

				for metricTypeIndex, metricValue := range metricMap {
					ms = append(ms, &metric.Metric{
						LabelKeys:   objectMetricLabels,
						LabelValues: []string{metricName, metricTypeIndex.String(), fullTargetName},
						Value:       metricValue,
					})
				}
			}
			return &metric.Family{Metrics: ms}
		}),
	)
}

func createHPAStatusTargetMetric() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_horizontalpodautoscaler_status_target_metric",
		"The current metric status used by this autoscaler when calculating the desired replica count.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
			ms := make([]*metric.Metric, 0, len(a.Status.CurrentMetrics))
			for _, m := range a.Status.CurrentMetrics {
				var metricName string
				var currentMetric autoscaling.MetricValueStatus
				// The variable maps the type of metric to the corresponding value
				metricMap := make(map[metricTargetType]float64)

				switch m.Type {
				case autoscaling.PodsMetricSourceType:
					metricName = m.Pods.Metric.Name
					currentMetric = m.Pods.Current
				case autoscaling.ResourceMetricSourceType:
					metricName = string(m.Resource.Name)
					currentMetric = m.Resource.Current
				case autoscaling.ExternalMetricSourceType:
					metricName = m.External.Metric.Name
					currentMetric = m.External.Current
				default:
					// Skip unsupported metric type
					continue
				}

				if currentMetric.Value != nil {
					metricMap[value] = convertValueToFloat64(currentMetric.Value)
				}
				if currentMetric.AverageValue != nil {
					metricMap[average] = convertValueToFloat64(currentMetric.AverageValue)
				}
				if currentMetric.AverageUtilization != nil {
					metricMap[utilization] = float64(*currentMetric.AverageUtilization)
				}

				for metricTypeIndex, metricValue := range metricMap {
					ms = append(ms, &metric.Metric{
						LabelKeys:   targetMetricLabels,
						LabelValues: []string{metricName, metricTypeIndex.String()},
						Value:       metricValue,
					})
				}
			}
			return &metric.Family{Metrics: ms}
		}),
	)
}

func createHPAStatusCurrentReplicas() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_horizontalpodautoscaler_status_current_replicas",
		"Current number of replicas of pods managed by this autoscaler.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						Value: float64(a.Status.CurrentReplicas),
					},
				},
			}
		}),
	)
}

func createHPAStatusDesiredReplicas() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_horizontalpodautoscaler_status_desired_replicas",
		"Desired number of replicas of pods managed by this autoscaler.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						Value: float64(a.Status.DesiredReplicas),
					},
				},
			}
		}),
	)
}

func createHPAAnnotations(allowAnnotationsList []string) generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		descHorizontalPodAutoscalerAnnotationsName,
		descHorizontalPodAutoscalerAnnotationsHelp,
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
			if len(allowAnnotationsList) == 0 {
				return &metric.Family{}
			}
			annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", a.Annotations, allowAnnotationsList)
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
	)
}

func createHPALabels(allowLabelsList []string) generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		descHorizontalPodAutoscalerLabelsName,
		descHorizontalPodAutoscalerLabelsHelp,
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
			if len(allowLabelsList) == 0 {
				return &metric.Family{}
			}
			labelKeys, labelValues := createPrometheusLabelKeysValues("label", a.Labels, allowLabelsList)
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

func createHPAStatusCondition() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_horizontalpodautoscaler_status_condition",
		"The condition of this autoscaler.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapHPAFunc(func(a *autoscaling.HorizontalPodAutoscaler) *metric.Family {
			ms := make([]*metric.Metric, 0, len(a.Status.Conditions)*len(conditionStatuses))

			for _, c := range a.Status.Conditions {
				metrics := addConditionMetrics(c.Status)

				for _, m := range metrics {
					metric := m
					metric.LabelKeys = []string{"condition", "status"}
					metric.LabelValues = append([]string{string(c.Type)}, metric.LabelValues...)
					ms = append(ms, metric)
				}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}
