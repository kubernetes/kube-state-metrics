/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"

	v1 "k8s.io/api/core/v1"

	"k8s.io/kube-state-metrics/pkg/metric"
)

var (
	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
	conditionStatuses  = []v1.ConditionStatus{v1.ConditionTrue, v1.ConditionFalse, v1.ConditionUnknown}
)

func resourceVersionMetric(rv string) []*metric.Metric {
	v, err := strconv.ParseFloat(rv, 64)
	if err != nil {
		return []*metric.Metric{}
	}

	return []*metric.Metric{
		{
			Value: v,
		},
	}

}

func boolFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// addConditionMetrics generates one metric for each possible condition
// status. For this function to work properly, the last label in the metric
// description must be the condition.
func addConditionMetrics(cs v1.ConditionStatus) []*metric.Metric {
	ms := make([]*metric.Metric, len(conditionStatuses))

	for i, status := range conditionStatuses {
		ms[i] = &metric.Metric{
			LabelValues: []string{strings.ToLower(string(status))},
			Value:       boolFloat64(cs == status),
		}
	}

	return ms
}

func kubeLabelsToPrometheusLabels(labels map[string]string) ([]string, []string) {
	return mapToPrometheusLabels(labels, "label")
}

func mapToPrometheusLabels(labels map[string]string, prefix string) ([]string, []string) {
	labelKeys := make([]string, 0, len(labels))
	for k := range labels {
		labelKeys = append(labelKeys, k)
	}
	sort.Strings(labelKeys)

	labelValues := make([]string, 0, len(labels))
	for i, k := range labelKeys {
		labelKeys[i] = prefix + "_" + sanitizeLabelName(k)
		labelValues = append(labelValues, labels[k])
	}
	return labelKeys, labelValues
}

func sanitizeLabelName(s string) string {
	return invalidLabelCharRE.ReplaceAllString(s, "_")
}

func isHugePageResourceName(name v1.ResourceName) bool {
	return strings.HasPrefix(string(name), v1.ResourceHugePagesPrefix)
}

func isAttachableVolumeResourceName(name v1.ResourceName) bool {
	return strings.HasPrefix(string(name), v1.ResourceAttachableVolumesPrefix)
}

func isExtendedResourceName(name v1.ResourceName) bool {
	if isNativeResource(name) || strings.HasPrefix(string(name), v1.DefaultResourceRequestsPrefix) {
		return false
	}
	// Ensure it satisfies the rules in IsQualifiedName() after converted into quota resource name
	nameForQuota := fmt.Sprintf("%s%s", v1.DefaultResourceRequestsPrefix, string(name))
	if errs := validation.IsQualifiedName(nameForQuota); len(errs) != 0 {
		return false
	}
	return true
}

func isNativeResource(name v1.ResourceName) bool {
	return !strings.Contains(string(name), "/") ||
		isPrefixedNativeResource(name)
}

func isPrefixedNativeResource(name v1.ResourceName) bool {
	return strings.Contains(string(name), v1.ResourceDefaultNamespacePrefix)
}
