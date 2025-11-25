/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descIngressAnnotationsName     = "kube_ingress_annotations" //nolint:gosec
	descIngressAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descIngressLabelsName          = "kube_ingress_labels"
	descIngressLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descIngressLabelsDefaultLabels = []string{"namespace", "ingress"}
)

func ingressMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_ingress_info",
			"Information about ingress.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapIngressFunc(func(i *networkingv1.Ingress) *metric.Family {
				ingressClassName := "_default"
				if i.Spec.IngressClassName != nil {
					ingressClassName = *i.Spec.IngressClassName
				}
				if className, ok := i.Annotations["kubernetes.io/ingress.class"]; ok {
					ingressClassName = className
				}

				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"ingressclass"},
							LabelValues: []string{ingressClassName},
							Value:       1,
						},
					}}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descIngressAnnotationsName,
			descIngressAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapIngressFunc(func(i *networkingv1.Ingress) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", i.Annotations, allowAnnotationsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   annotationKeys,
							LabelValues: annotationValues,
							Value:       1,
						},
					}}

			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descIngressLabelsName,
			descIngressLabelsHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapIngressFunc(func(i *networkingv1.Ingress) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", i.Labels, allowLabelsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
							Value:       1,
						},
					}}

			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_ingress_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapIngressFunc(func(i *networkingv1.Ingress) *metric.Family {
				ms := []*metric.Metric{}

				if !i.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(i.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_ingress_metadata_resource_version",
			"Resource version representing a specific version of ingress.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapIngressFunc(func(i *networkingv1.Ingress) *metric.Family {
				return &metric.Family{
					Metrics: resourceVersionMetric(i.ResourceVersion),
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_ingress_path",
			"Ingress host, paths and backend service information.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapIngressFunc(func(i *networkingv1.Ingress) *metric.Family {
				ms := []*metric.Metric{}
				for _, rule := range i.Spec.Rules {
					if rule.HTTP != nil {
						for _, path := range rule.HTTP.Paths {
							pathType := ""
							if path.PathType != nil {
								pathType = string(*path.PathType)
							}
							if path.Backend.Service != nil {
								ms = append(ms, &metric.Metric{
									LabelKeys:   []string{"host", "path", "path_type", "service_name", "service_port"},
									LabelValues: []string{rule.Host, path.Path, pathType, path.Backend.Service.Name, strconv.Itoa(int(path.Backend.Service.Port.Number))},
									Value:       1,
								})
							} else {
								apiGroup := ""
								if path.Backend.Resource.APIGroup != nil {
									apiGroup = *path.Backend.Resource.APIGroup
								}
								ms = append(ms, &metric.Metric{
									LabelKeys:   []string{"host", "path", "path_type", "resource_api_group", "resource_kind", "resource_name"},
									LabelValues: []string{rule.Host, path.Path, pathType, apiGroup, path.Backend.Resource.Kind, path.Backend.Resource.Name},
									Value:       1,
								})
							}
						}
					}
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_ingress_tls",
			"Ingress TLS host and secret information.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapIngressFunc(func(i *networkingv1.Ingress) *metric.Family {
				ms := []*metric.Metric{}
				for _, tls := range i.Spec.TLS {
					for _, host := range tls.Hosts {
						ms = append(ms, &metric.Metric{
							LabelKeys:   []string{"tls_host", "secret"},
							LabelValues: []string{host, tls.SecretName},
							Value:       1,
						})
					}
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
}

func wrapIngressFunc(f func(*networkingv1.Ingress) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		ingress := obj.(*networkingv1.Ingress)

		metricFamily := f(ingress)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descIngressLabelsDefaultLabels, []string{ingress.Namespace, ingress.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createIngressListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.NetworkingV1().Ingresses(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.NetworkingV1().Ingresses(ns).Watch(context.TODO(), opts)
		},
	}
}
