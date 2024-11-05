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

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

var (
	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
	matchAllCap        = regexp.MustCompile("([a-z0-9])([A-Z])")
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

func kubeMapToPrometheusLabels(prefix string, input map[string]string) ([]string, []string) {
	return mapToPrometheusLabels(input, prefix)
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
		labelKey := labelName(prefix, k)
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

func labelName(prefix, labelName string) string {
	return prefix + "_" + lintLabelName(SanitizeLabelName(labelName))
}

// SanitizeLabelName replaces all invalid characters with an underscore.
func SanitizeLabelName(s string) string {
	return invalidLabelCharRE.ReplaceAllString(s, "_")
}

func lintLabelName(s string) string {
	return toSnakeCase(s)
}

func toSnakeCase(s string) string {
	snake := matchAllCap.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(snake)
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

// createPrometheusLabelKeysValues takes in passed kubernetes annotations/labels
// and associated allowed list in kubernetes label format.
// It returns only those allowed annotations/labels that exist in the list and converts them to Prometheus labels.
func createPrometheusLabelKeysValues(prefix string, allKubeData map[string]string, allowList []string) ([]string, []string) {
	allowedKubeData := make(map[string]string)

	if len(allowList) > 0 {
		if allowList[0] == options.LabelWildcard {
			return kubeMapToPrometheusLabels(prefix, allKubeData)
		}

		for _, l := range allowList {
			v, found := allKubeData[l]
			if found {
				allowedKubeData[l] = v
			}
		}
	}
	return kubeMapToPrometheusLabels(prefix, allowedKubeData)
}

// mergeKeyValues merges label keys and values slice pairs into a single slice pair.
// Arguments are passed as equal-length pairs of slices, where the first slice contains keys and second contains values.
// Example: mergeKeyValues(keys1, values1, keys2, values2) => (keys1+keys2, values1+values2)
func mergeKeyValues(keyValues ...[]string) (keys, values []string) {
	capacity := 0
	for i := 0; i < len(keyValues); i += 2 {
		capacity += len(keyValues[i])
	}

	// Allocate one contiguous block, then split it up to keys and values zero'd slices.
	keysValues := make([]string, 0, capacity*2)
	keys = (keysValues[0:capacity:capacity])[:0]
	values = (keysValues[capacity : capacity*2])[:0]

	for i := 0; i < len(keyValues); i += 2 {
		keys = append(keys, keyValues[i]...)
		values = append(values, keyValues[i+1]...)
	}

	return keys, values
}

// convertValueToFloat64 converts a resource.Quantity to a float64 and checks for a possible overflow in the value.
func convertValueToFloat64(q *resource.Quantity) float64 {
	if q.Value() > resource.MaxMilliValue {
		return float64(q.Value())
	}
	return float64(q.MilliValue()) / 1000
}
