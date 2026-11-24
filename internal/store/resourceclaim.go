/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	basemetrics "k8s.io/component-base/metrics"
	"k8s.io/utils/ptr"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descResourceClaimLabelsDefaultLabels = []string{"resourceclaim", "namespace"}
	descResourceClaimAnnotationsName     = "kube_resourceclaim_annotations"
	descResourceClaimAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descResourceClaimLabelsName          = "kube_resourceclaim_labels" //nolint:gosec
	descResourceClaimLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
)

func resourceClaimMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_resourceclaim_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceClaimFunc(func(rc *resourcev1.ResourceClaim) *metric.Family {
				ms := []*metric.Metric{}

				if !rc.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(rc.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_resourceclaim_status_allocated",
			"Indicates whether the resource claim has been allocated.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceClaimFunc(func(rc *resourcev1.ResourceClaim) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: boolFloat64(rc.Status.Allocation != nil),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_resourceclaim_status_reserved_for",
			"Indicates which consumers have currently reserved the resource claim.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceClaimFunc(func(rc *resourcev1.ResourceClaim) *metric.Family {
				ms := make([]*metric.Metric, len(rc.Status.ReservedFor))
				for i, res := range rc.Status.ReservedFor {
					ms[i] = &metric.Metric{
						LabelKeys:   []string{"consumer_apigroup", "consumer_resource", "consumer_name", "consumer_uid"},
						LabelValues: []string{res.APIGroup, res.Resource, res.Name, string(res.UID)},
						Value:       1,
					}
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_resourceclaim_status_allocation_device_info",
			"Allocation information about the devices allocated to the resource claim.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceClaimFunc(func(rc *resourcev1.ResourceClaim) *metric.Family {
				if rc.Status.Allocation == nil {
					return &metric.Family{}
				}
				results := rc.Status.Allocation.Devices.Results
				ms := make([]*metric.Metric, len(results))
				for i, res := range results {
					ms[i] = &metric.Metric{
						LabelKeys:   []string{"request", "driver", "pool", "device", "share_id", "admin_access"},
						LabelValues: []string{res.Request, res.Driver, res.Pool, res.Device, string(ptr.Deref(res.ShareID, "")), strconv.FormatBool(ptr.Deref(res.AdminAccess, false))},
						Value:       1,
					}
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descResourceClaimAnnotationsName,
			descResourceClaimAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceClaimFunc(func(rc *resourcev1.ResourceClaim) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				keys, values := createPrometheusLabelKeysValues("annotation", rc.Annotations, allowAnnotationsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   keys,
							LabelValues: values,
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descResourceClaimLabelsName,
			descResourceClaimLabelsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceClaimFunc(func(rc *resourcev1.ResourceClaim) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				keys, values := createPrometheusLabelKeysValues("label", rc.Labels, allowLabelsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   keys,
							LabelValues: values,
							Value:       1,
						},
					},
				}
			}),
		),
	}
}

func wrapResourceClaimFunc(f func(*resourcev1.ResourceClaim) *metric.Family) func(any) *metric.Family {
	return func(obj any) *metric.Family {
		rc := obj.(*resourcev1.ResourceClaim)

		metricFamily := f(rc)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descResourceClaimLabelsDefaultLabels, []string{rc.Name, rc.Namespace}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createResourceClaimListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.ResourceV1().ResourceClaims(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.ResourceV1().ResourceClaims(ns).Watch(context.TODO(), opts)
		},
	}
}
