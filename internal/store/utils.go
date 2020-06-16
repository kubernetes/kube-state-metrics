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
	labelValues := make([]string, 0, len(labels))

	sortedKeys := make([]string, 0)
	for key := range labels {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	// conflictDesc holds some metadata for resolving potential label conflicts
	type conflictDesc struct {
		// the number of conflicting label keys we saw so far
		count int

		// the offset of the initial conflicting label key, so we could
		// later go back and rename "label_foo" to "label_foo_conflict1"
		initial int
	}

	conflicts := make(map[string]*conflictDesc)
	for _, k := range sortedKeys {
		labelKey := prefix + "_" + sanitizeLabelName(k)
		if conflict, ok := conflicts[labelKey]; ok {
			if conflict.count == 1 {
				// this is the first conflict for the label,
				// so we have to go back and rename the initial label that we've already added
				labelKeys[conflict.initial] = labelConflictSuffix(labelKeys[conflict.initial], conflict.count)
			}

			conflict.count++
			labelKey = labelConflictSuffix(labelKey, conflict.count)
		} else {
			// we'll need this info later in case there are conflicts
			conflicts[labelKey] = &conflictDesc{
				count:   1,
				initial: len(labelKeys),
			}
		}
		labelKeys = append(labelKeys, labelKey)
		labelValues = append(labelValues, labels[k])
	}
	return labelKeys, labelValues
}

func sanitizeLabelName(s string) string {
	return invalidLabelCharRE.ReplaceAllString(s, "_")
}

func labelConflictSuffix(label string, count int) string {
	return fmt.Sprintf("%s_conflict%d", label, count)
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
