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
	"slices"
	"strconv"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

var (
	invalidLabelCharRE    = regexp.MustCompile(`[^a-zA-Z0-9_]`)
	matchAllCap           = regexp.MustCompile("([a-z0-9])([A-Z])")
	conditionStatuses     = []v1.ConditionStatus{v1.ConditionTrue, v1.ConditionFalse, v1.ConditionUnknown}
	allowListPatternCache sync.Map
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

	sortedKeys := make([]string, 0, len(labels))
	for key := range labels {
		sortedKeys = append(sortedKeys, key)
	}
	slices.Sort(sortedKeys)

	// used maps an emitted label name to its offset in labelKeys, so that the
	// first time a name collides the entry already emitted under it can be
	// renamed. conflicts counts the suffixes handed out per sanitized name.
	// Prometheus rejects a sample carrying the same label name twice, so a
	// generated "_conflictN" name must be checked against used as well: it can
	// collide with a name some other key sanitized to.
	used := make(map[string]int, len(labels))
	conflicts := make(map[string]int)

	for _, k := range sortedKeys {
		base := labelName(prefix, k)
		name := base
		if count, seen := conflicts[base]; seen {
			// This name has collided before, so the unsuffixed form is no longer
			// in use and every further occurrence just takes the next suffix.
			name = nextFreeLabelName(base, &count, used)
			conflicts[base] = count
		} else if idx, taken := used[name]; taken {
			// First collision for this name: the entry already emitted is still
			// unsuffixed, so rename it to "_conflict1" and take "_conflict2".
			count := 0
			renamed := nextFreeLabelName(base, &count, used)
			delete(used, name)
			labelKeys[idx] = renamed
			used[renamed] = idx

			name = nextFreeLabelName(base, &count, used)
			conflicts[base] = count
		}
		used[name] = len(labelKeys)
		labelKeys = append(labelKeys, name)
		labelValues = append(labelValues, labels[k])
	}
	return labelKeys, labelValues
}

// nextFreeLabelName advances count until base with that conflict suffix is a
// name no label is using yet, and returns it.
func nextFreeLabelName(base string, count *int, used map[string]int) string {
	for {
		*count++
		candidate := labelConflictSuffix(base, *count)
		if _, taken := used[candidate]; !taken {
			return candidate
		}
	}
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
// Full wildcards (*) can only be set as first in the allow list, partial wildcards can appear anywhere in the list
func createPrometheusLabelKeysValues(prefix string, allKubeData map[string]string, allowList []string) ([]string, []string) {
	allowedKubeData := make(map[string]string)

	for i, l := range allowList {
		// only the first label can be the wildcard label
		if l == options.LabelWildcard {
			if i == 0 {
				return kubeMapToPrometheusLabels(prefix, allKubeData)
			}
			continue
		}

		if !strings.Contains(l, options.LabelWildcard) {
			// exact key — direct lookup, no regexp needed
			if v, ok := allKubeData[l]; ok {
				allowedKubeData[l] = v
			}
			continue
		}

		// look up or compile the pattern, caching the result so we only compile once per unique pattern
		re := cachedCompileAllowListPattern(l)
		if re == nil {
			continue
		}
		for k, v := range allKubeData {
			if re.MatchString(k) {
				allowedKubeData[k] = v
			}
		}
	}

	return kubeMapToPrometheusLabels(prefix, allowedKubeData)
}

// expandWildcard expands wildcards (*) to regular expressions, up to a limited number of wildcards
func expandWildcard(pattern string, limit uint) string {
	var result strings.Builder
	var replacements uint
	for i, literal := range strings.Split(pattern, options.LabelWildcard) {
		if i > 0 {
			result.WriteString(".*")
			replacements++
		}

		result.WriteString(regexp.QuoteMeta(literal))
		if replacements >= limit {
			break
		}
	}
	return "^" + result.String() + "$"
}

// cachedCompileAllowListPattern returns a compiled regexp for the given wildcard pattern,
// using allowListPatternCache to avoid recompiling on every object event.
// Patterns with more wildcards than MaxPartialWildcardsPerLabel are rejected outright so
// that an over-wildcarded entry like "*key*" fails closed rather than silently matching
// as a truncated pattern (e.g. "^.*key$"). A warning is logged on the first rejection or
// compile error so the operator can fix the configuration.
func cachedCompileAllowListPattern(pattern string) *regexp.Regexp {
	if strings.Count(pattern, options.LabelWildcard) > options.MaxPartialWildcardsPerLabel {
		klog.Warningf("kube-state-metrics: ignoring allowlist pattern %q: exceeds maximum of %d wildcard(s)", pattern, options.MaxPartialWildcardsPerLabel)
		return nil
	}
	expanded := expandWildcard(pattern, options.MaxPartialWildcardsPerLabel)
	if v, ok := allowListPatternCache.Load(expanded); ok {
		if v == nil {
			return nil
		}
		return v.(*regexp.Regexp)
	}
	re, err := regexp.Compile(expanded)
	if err != nil {
		klog.Warningf("kube-state-metrics: ignoring invalid allowlist pattern %q: %v", pattern, err)
		allowListPatternCache.Store(expanded, nil)
		return nil
	}
	allowListPatternCache.Store(expanded, re)
	return re
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

	for i := 0; i+1 < len(keyValues); i += 2 {
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
