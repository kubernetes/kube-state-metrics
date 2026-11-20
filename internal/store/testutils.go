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

// TODO: Does this file need to be renamed to not be compiled in production?

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/google/go-cmp/cmp"
	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

type generateMetricsTestCase struct {
	Obj                  any
	Func                 func(any) []metric.FamilyInterface
	Want                 string
	MetricNames          []string
	AllowAnnotationsList []string
	AllowLabelsList      []string
	Headers              []string
	FamilyGens           []generator.FamilyGenerator
}

func (testCase *generateMetricsTestCase) run() error {
	metricFamilies := testCase.Func(testCase.Obj)

	if err := testCase.validateLabels(metricFamilies); err != nil {
		return err
	}

	metricFamilyStrings := []string{}
	for _, f := range metricFamilies {
		metricFamilyStrings = append(metricFamilyStrings, string(f.ByteSlice()))
	}
	metric := strings.Split(strings.Join(metricFamilyStrings, ""), "\n")
	filteredMetrics := filterMetricNames(metric, testCase.MetricNames)
	filteredHeaders := filterMetricNames(testCase.Headers, testCase.MetricNames)
	headers := strings.Join(filteredHeaders, "\n")
	metrics := strings.Join(filteredMetrics, "\n")
	out := headers + "\n" + metrics

	if err := compareOutput(testCase.Want, out); err != nil {
		return fmt.Errorf("expected wanted output to equal output: %w", err)
	}

	return nil
}

func compareOutput(expected, actual string) error {
	entities := []string{expected, actual}
	// Align wanted and actual
	for i := range entities {
		for _, f := range []func(string) string{removeUnusedWhitespace, sortLabels, sortByLine} {
			entities[i] = f(entities[i])
		}
	}

	if diff := cmp.Diff(entities[0], entities[1]); diff != "" {
		return fmt.Errorf("(-want, +got):\n%s", diff)
	}

	return nil
}

// sortLabels sorts the order of labels in each line of the given metric. The
// Prometheus exposition format does not force ordering of labels. Hence a test
// should not fail due to different metric orders.
func sortLabels(s string) string {
	sorted := []string{}

	for line := range strings.SplitSeq(s, "\n") {
		// skipping if its headers
		if strings.HasPrefix(line, "# ") {
			sorted = append(sorted, line)
			continue
		}
		split := strings.Split(line, "{")
		if len(split) != 2 {
			panic(fmt.Sprintf("failed to sort labels in \"%v\"", line))
		}
		name := split[0]

		split = strings.Split(split[1], "}")
		value := split[1]

		labels := strings.Split(split[0], ",")
		slices.Sort(labels)

		sorted = append(sorted, fmt.Sprintf("%v{%v}%v", name, strings.Join(labels, ","), value))
	}

	return strings.Join(sorted, "\n")
}

func sortByLine(s string) string {
	split := strings.Split(s, "\n")
	slices.Sort(split)
	return strings.Join(split, "\n")
}

// filterMetricNames removes those metrics and headers that
// are not part of the names.
func filterMetricNames(ms []string, names []string) []string {
	// In case the test case is based on all returned metric, MetricNames does
	// not need to me defined.
	if names == nil {
		return ms
	}
	filtered := []string{}

	regexps := []*regexp.Regexp{}
	for _, n := range names {
		regexps = append(regexps, regexp.MustCompile(fmt.Sprintf("(?m).*%v.*$", n)))
	}

	for _, m := range ms {
		drop := true
		for _, r := range regexps {
			if r.MatchString(m) {
				drop = false
				break
			}
		}
		if !drop {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func removeUnusedWhitespace(s string) string {
	var (
		trimmedLine  string
		trimmedLines []string
		lines        = strings.Split(s, "\n")
	)

	for _, l := range lines {
		trimmedLine = strings.TrimLeft(l, " \t")

		if len(trimmedLine) > 0 {
			trimmedLines = append(trimmedLines, trimmedLine)
		}
	}

	return strings.Join(trimmedLines, "\n")
}

// legacyVariableLabelSets lists pre-existing STABLE metrics whose series emit
// one of several fixed label-key sets depending on object state rather than a
// single uniform label set across all series.
var legacyVariableLabelSets = map[string][][]string{
	"kube_ingress_path": {
		{"namespace", "ingress", "host", "path", "path_type", "service_name", "service_port"},
		{"namespace", "ingress", "host", "path", "path_type", "resource_api_group", "resource_kind", "resource_name"},
	},
	"kube_job_status_failed": {
		{"namespace", "job_name"},
		{"namespace", "job_name", "reason"},
	},
}

func (testCase *generateMetricsTestCase) validateLabels(metricFamilies []metric.FamilyInterface) error {
	if testCase.FamilyGens == nil {
		if strings.Contains(testCase.Want, "[STABLE]") {
			return fmt.Errorf("test case expects STABLE metrics but FamilyGens is nil")
		}
		return nil
	}

	genMap := make(map[string]generator.FamilyGenerator, len(testCase.FamilyGens))
	for _, gen := range testCase.FamilyGens {
		genMap[gen.Name] = gen
	}

	for _, f := range metricFamilies {
		var actualFamily metric.Family
		f.Inspect(func(fam metric.Family) {
			actualFamily = fam
		})

		gen, ok := genMap[actualFamily.Name]
		if !ok {
			return fmt.Errorf("metric %q emitted but has no corresponding generator in FamilyGens", actualFamily.Name)
		}

		if gen.StabilityLevel != basemetrics.STABLE {
			continue
		}

		if err := validateFamilyLabels(actualFamily, gen); err != nil {
			return err
		}
	}
	return nil
}

func validateFamilyLabels(family metric.Family, gen generator.FamilyGenerator) error {
	if gen.Labels == nil {
		return fmt.Errorf("STABLE metric %q is missing declared Labels", family.Name)
	}
	if allowedSets, isLegacy := legacyVariableLabelSets[family.Name]; isLegacy {
		return validateLegacyVariableFamilyLabels(family, gen, allowedSets)
	}
	for _, m := range family.Metrics {
		if err := validateSeriesLabels(family.Name, m.LabelKeys, gen.Labels); err != nil {
			return err
		}
	}
	return nil
}

func validateSeriesLabels(metricName string, actualKeys, declaredKeys []string) error {
	if dup, ok := findDuplicateLabel(actualKeys); ok {
		return fmt.Errorf("metric %q output duplicate label %q in %v", metricName, dup, actualKeys)
	}
	for _, actualKey := range actualKeys {
		if slices.Contains(declaredKeys, actualKey) || isDynamicLabelCompatible(actualKey, declaredKeys) {
			continue
		}
		return fmt.Errorf("metric %q output undeclared label %q. Declared labels: %v", metricName, actualKey, declaredKeys)
	}
	for _, declaredKey := range declaredKeys {
		if isDynamicLabel(declaredKey) {
			continue
		}
		if !slices.Contains(actualKeys, declaredKey) {
			return fmt.Errorf("metric %q missing declared label %q. Emitted labels: %v, Declared labels: %v", metricName, declaredKey, actualKeys, declaredKeys)
		}
	}
	return nil
}

func validateLegacyVariableFamilyLabels(family metric.Family, gen generator.FamilyGenerator, allowedSets [][]string) error {
	var union []string
	for _, set := range allowedSets {
		for _, k := range set {
			if !slices.Contains(union, k) {
				union = append(union, k)
			}
		}
	}
	if !equalLabelSet(union, gen.Labels) {
		return fmt.Errorf("metric %q legacy allowed label union %v does not match declared labels %v", family.Name, union, gen.Labels)
	}
	for _, m := range family.Metrics {
		if dup, ok := findDuplicateLabel(m.LabelKeys); ok {
			return fmt.Errorf("metric %q output duplicate label %q in %v", family.Name, dup, m.LabelKeys)
		}
		if !matchesAnyLabelSet(m.LabelKeys, allowedSets) {
			return fmt.Errorf("metric %q emitted label set %v does not match any allowed set in %v", family.Name, m.LabelKeys, allowedSets)
		}
	}
	return nil
}

func matchesAnyLabelSet(actual []string, allowedSets [][]string) bool {
	for _, allowed := range allowedSets {
		if equalLabelSet(actual, allowed) {
			return true
		}
	}
	return false
}

func equalLabelSet(a, b []string) bool {
	for _, k := range a {
		if !slices.Contains(b, k) {
			return false
		}
	}
	for _, k := range b {
		if !slices.Contains(a, k) {
			return false
		}
	}
	return true
}

func findDuplicateLabel(keys []string) (string, bool) {
	for i, k := range keys {
		if slices.Contains(keys[:i], k) {
			return k, true
		}
	}
	return "", false
}

func isDynamicLabel(key string) bool {
	return strings.HasPrefix(key, "label_") || strings.HasPrefix(key, "annotation_")
}

func isDynamicLabelCompatible(key string, declaredLabels []string) bool {
	var prefix string
	switch {
	case strings.HasPrefix(key, "label_"):
		prefix = "label_"
	case strings.HasPrefix(key, "annotation_"):
		prefix = "annotation_"
	default:
		return false
	}

	return slices.ContainsFunc(declaredLabels, func(l string) bool {
		return strings.HasPrefix(l, prefix)
	})
}
