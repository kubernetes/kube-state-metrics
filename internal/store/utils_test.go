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
	"fmt"
	"slices"
	"testing"

	v1 "k8s.io/api/core/v1"

	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func TestIsHugePageSizeFromResourceName(t *testing.T) {
	testCases := []struct {
		resourceName v1.ResourceName
		expectVal    bool
	}{
		{
			resourceName: "pod.alpha.kubernetes.io/opaque-int-resource-foo",
			expectVal:    false,
		},
		{
			resourceName: "hugepages-100m",
			expectVal:    true,
		},
		{
			resourceName: "",
			expectVal:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("resourceName input=%s, expected value=%v", tc.resourceName, tc.expectVal), func(t *testing.T) {
			v := isHugePageResourceName(tc.resourceName)
			if v != tc.expectVal {
				t.Errorf("Got %v but expected %v", v, tc.expectVal)
			}
		})
	}
}

func TestIsAttachableVolumeResourceName(t *testing.T) {
	testCases := []struct {
		resourceName v1.ResourceName
		expectVal    bool
	}{
		{
			resourceName: "pod.alpha.kubernetes.io/opaque-int-resource-foo",
			expectVal:    false,
		},
		{
			resourceName: "attachable-volumes-100m",
			expectVal:    true,
		},
		{
			resourceName: "",
			expectVal:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("resourceName input=%s, expected value=%v", tc.resourceName, tc.expectVal), func(t *testing.T) {
			v := isAttachableVolumeResourceName(tc.resourceName)
			if v != tc.expectVal {
				t.Errorf("Got %v but expected %v", v, tc.expectVal)
			}
		})
	}
}

func TestIsExtendedResourceName(t *testing.T) {
	testCases := []struct {
		resourceName v1.ResourceName
		expectVal    bool
	}{
		{
			resourceName: "pod.alpha.kubernetes.io/opaque-int-resource-foo",
			expectVal:    false,
		},
		{
			resourceName: "kubernetes.io/resource-foo",
			expectVal:    false,
		},
		{
			resourceName: "foo",
			expectVal:    false,
		},
		{
			resourceName: "a/b",
			expectVal:    true,
		},
		{
			resourceName: "requests.foobar",
			expectVal:    false,
		},
		{
			resourceName: "c/d/",
			expectVal:    false,
		},
		{
			resourceName: "",
			expectVal:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("resourceName input=%s, expected value=%v", tc.resourceName, tc.expectVal), func(t *testing.T) {
			v := isExtendedResourceName(tc.resourceName)
			if v != tc.expectVal {
				t.Errorf("Got %v but expected %v", v, tc.expectVal)
			}
		})
	}
}

func TestKubeLabelsToPrometheusLabels(t *testing.T) {
	testCases := []struct {
		kubeLabels   map[string]string
		expectKeys   []string
		expectValues []string
	}{
		{
			kubeLabels: map[string]string{
				"app1": "normal",
			},
			expectKeys:   []string{"label_app1"},
			expectValues: []string{"normal"},
		},
		{
			kubeLabels: map[string]string{
				"0_app3": "starts_with_digit",
			},
			expectKeys:   []string{"label_0_app3"},
			expectValues: []string{"starts_with_digit"},
		},
		{
			kubeLabels: map[string]string{
				"": "empty",
			},
			expectKeys:   []string{"label_"},
			expectValues: []string{"empty"},
		},
		{
			kubeLabels: map[string]string{
				"$app4": "special_char",
			},
			expectKeys:   []string{"label__app4"},
			expectValues: []string{"special_char"},
		},
		{
			kubeLabels: map[string]string{
				"_app5": "starts_with_underscore",
			},
			expectKeys:   []string{"label__app5"},
			expectValues: []string{"starts_with_underscore"},
		},
		{
			kubeLabels: map[string]string{
				"an":    "",
				"order": "",
				"test":  "",
			},
			expectKeys:   []string{"label_an", "label_order", "label_test"},
			expectValues: []string{"", "", ""},
		},
		{
			kubeLabels: map[string]string{
				"conflicting_label1": "underscore",
				"conflicting.label1": "dot",
				"conflicting-label1": "hyphen",

				"conflicting.label2": "dot",
				"conflicting-label2": "hyphen",
				"conflicting_label2": "underscore",

				"conflicting-label3": "hyphen",
				"conflicting_label3": "underscore",
				"conflicting.label3": "dot",
			},
			// keys are sorted alphabetically during sanitization
			expectKeys: []string{
				"label_conflicting_label1_conflict1",
				"label_conflicting_label2_conflict1",
				"label_conflicting_label3_conflict1",
				"label_conflicting_label1_conflict2",
				"label_conflicting_label2_conflict2",
				"label_conflicting_label3_conflict2",
				"label_conflicting_label1_conflict3",
				"label_conflicting_label2_conflict3",
				"label_conflicting_label3_conflict3",
			},
			expectValues: []string{
				"hyphen",
				"hyphen",
				"hyphen",
				"dot",
				"dot",
				"dot",
				"underscore",
				"underscore",
				"underscore",
			},
		},
		{
			kubeLabels: map[string]string{
				"camelCase": "camel_case",
			},
			expectKeys:   []string{"label_camel_case"},
			expectValues: []string{"camel_case"},
		},
		{
			kubeLabels: map[string]string{
				"snake_camelCase": "snake_and_camel_case",
			},
			expectKeys:   []string{"label_snake_camel_case"},
			expectValues: []string{"snake_and_camel_case"},
		},
		{
			kubeLabels: map[string]string{
				"conflicting_camelCase":  "camel_case",
				"conflicting_camel_case": "snake_case",
			},
			expectKeys: []string{
				"label_conflicting_camel_case_conflict1",
				"label_conflicting_camel_case_conflict2",
			},
			expectValues: []string{
				"camel_case",
				"snake_case",
			},
		},
		{
			// A key that sanitizes straight onto the name the conflict suffix
			// would generate. Prometheus rejects a sample carrying the same
			// label name twice, so the suffix has to skip past it.
			kubeLabels: map[string]string{
				"A.b.conflict1": "already_taken",
				"a-b":           "dash",
				"a.b":           "dot",
			},
			expectKeys: []string{
				"label_a_b_conflict1",
				"label_a_b_conflict2",
				"label_a_b_conflict3",
			},
			expectValues: []string{
				"already_taken",
				"dash",
				"dot",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("kubelabels input=%v , expected prometheus keys=%v, expected prometheus values=%v", tc.kubeLabels, tc.expectKeys, tc.expectValues), func(t *testing.T) {
			labelKeys, labelValues := kubeMapToPrometheusLabels("label", tc.kubeLabels)
			if len(labelKeys) != len(tc.expectKeys) {
				t.Errorf("Got Prometheus label keys with len %d but expected %d", len(labelKeys), len(tc.expectKeys))
			}

			if len(labelValues) != len(tc.expectValues) {
				t.Errorf("Got Prometheus label values with len %d but expected %d", len(labelValues), len(tc.expectValues))
			}

			for i := range tc.expectKeys {
				if tc.expectKeys[i] != labelKeys[i] || tc.expectValues[i] != labelValues[i] {
					t.Errorf("Got Prometheus label %q: %q but expected %q: %q", labelKeys[i], labelValues[i], tc.expectKeys[i], tc.expectValues[i])
				}
			}
		})
	}

}

func TestMergeKeyValues(t *testing.T) {
	testCases := []struct {
		name               string
		keyValuePairSlices [][]string
		expectKeys         []string
		expectValues       []string
	}{
		{
			name: "singlePair",
			keyValuePairSlices: [][]string{
				{"keyA", "keyB", "keyC"},
				{"valueA", "valueB", "valueC"},
			},
			expectKeys:   []string{"keyA", "keyB", "keyC"},
			expectValues: []string{"valueA", "valueB", "valueC"},
		},
		{
			name: "evenPair",
			keyValuePairSlices: [][]string{
				{"keyA", "keyB", "keyC"},
				{"valueA", "valueB", "valueC"},
				{"keyX", "keyY", "keyZ"},
				{"valueX", "valueY", "valueZ"},
			},
			expectKeys:   []string{"keyA", "keyB", "keyC", "keyX", "keyY", "keyZ"},
			expectValues: []string{"valueA", "valueB", "valueC", "valueX", "valueY", "valueZ"},
		},
		{
			name: "oddPair",
			keyValuePairSlices: [][]string{
				{"keyA", "keyB", "keyC"},
				{"valueA", "valueB", "valueC"},
				{"keyX", "keyY", "keyZ"},
				{"valueX", "valueY", "valueZ"},
				{"keyM", "keyN", "keyP"},
				{"valueM", "valueN", "valueP"},
			},
			expectKeys:   []string{"keyA", "keyB", "keyC", "keyX", "keyY", "keyZ", "keyM", "keyN", "keyP"},
			expectValues: []string{"valueA", "valueB", "valueC", "valueX", "valueY", "valueZ", "valueM", "valueN", "valueP"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotKeys, gotValues := mergeKeyValues(tc.keyValuePairSlices...)
			if !slices.Equal(gotKeys, tc.expectKeys) {
				t.Errorf("mergeKeyValues() got = %v, want %v", gotKeys, tc.expectKeys)
			}
			if !slices.Equal(gotValues, tc.expectValues) {
				t.Errorf("mergeKeyValues() got1 = %v, want %v", gotValues, tc.expectValues)
			}
		})
	}
}

func TestCreatePrometheusLabelKeysValues(t *testing.T) {
	testCases := []struct {
		name         string
		kubeData     map[string]string
		allowList    []string
		expectKeys   []string
		expectValues []string
	}{
		{
			name: "allMatches",
			kubeData: map[string]string{
				"keyA": "valueA",
				"keyB": "valueB",
			},
			allowList:    []string{"keyA", "keyB"},
			expectKeys:   []string{"metric_key_a", "metric_key_b"},
			expectValues: []string{"valueA", "valueB"},
		},
		{
			name: "additionalAllow",
			kubeData: map[string]string{
				"keyA": "valueA",
				"keyB": "valueB",
			},
			allowList:    []string{"keyA", "keyB", "keyC"},
			expectKeys:   []string{"metric_key_a", "metric_key_b"},
			expectValues: []string{"valueA", "valueB"},
		},
		{
			name: "partialMatches",
			kubeData: map[string]string{
				"keyA": "valueA",
				"keyB": "valueB",
			},
			allowList:    []string{"keyA", "keyC"},
			expectKeys:   []string{"metric_key_a"},
			expectValues: []string{"valueA"},
		},
		{
			name: "wildcardAsSuffix",
			kubeData: map[string]string{
				"keyA":      "valueA",
				"keyB":      "valueB",
				"otherKeyA": "valueC",
				"otherKeyB": "valueD",
			},
			allowList:    []string{"key*"},
			expectKeys:   []string{"metric_key_a", "metric_key_b"},
			expectValues: []string{"valueA", "valueB"},
		},
		{
			name: "wildcardAsPrefix",
			kubeData: map[string]string{
				"keyA":      "valueA",
				"keyB":      "valueB",
				"otherKeyA": "valueC",
				"otherKeyB": "valueD",
			},
			allowList:    []string{"*A"},
			expectKeys:   []string{"metric_key_a", "metric_other_key_a"},
			expectValues: []string{"valueA", "valueC"},
		},
		{
			name: "onlyFullWildcard",
			kubeData: map[string]string{
				"keyA":      "valueA",
				"keyB":      "valueB",
				"otherKeyA": "valueC",
				"otherKeyB": "valueD",
			},
			allowList:    []string{"*"},
			expectKeys:   []string{"metric_key_a", "metric_key_b", "metric_other_key_a", "metric_other_key_b"},
			expectValues: []string{"valueA", "valueB", "valueC", "valueD"},
		},
		{
			name: "additionalFullWildcard",
			kubeData: map[string]string{
				"keyA":      "valueA",
				"keyB":      "valueB",
				"otherKeyA": "valueC",
				"otherKeyB": "valueD",
			},
			allowList:    []string{"keyA", "*"},
			expectKeys:   []string{"metric_key_a"},
			expectValues: []string{"valueA"},
		},
		{
			// "*key*" has two wildcards; without the over-wildcard guard it would be truncated
			// to "^.*key$" and match keys ending in "key" — verify it matches nothing instead.
			name: "multipleWildcards",
			kubeData: map[string]string{
				"keyA":      "valueA",
				"keyB":      "valueB",
				"otherKeyA": "valueC",
				"somekey":   "valueD",
			},
			allowList:    []string{"*key*"},
			expectKeys:   []string{},
			expectValues: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotKeys, gotValues := createPrometheusLabelKeysValues("metric", tc.kubeData, tc.allowList)
			if !slices.Equal(gotKeys, tc.expectKeys) {
				t.Errorf("createPrometheusLabelKeysValues() got = %v, want %v", gotKeys, tc.expectKeys)
			}
			if !slices.Equal(gotValues, tc.expectValues) {
				t.Errorf("createPrometheusLabelKeysValues() got1 = %v, want %v", gotValues, tc.expectValues)
			}
		})
	}
}

func TestCachedCompileAllowListPattern(t *testing.T) {
	t.Run("valid pattern returns non-nil regexp", func(t *testing.T) {
		re := cachedCompileAllowListPattern("app*")
		if re == nil {
			t.Error("expected non-nil regexp for valid pattern")
		}
	})

	t.Run("returned regexp matches correctly", func(t *testing.T) {
		re := cachedCompileAllowListPattern("app*")
		if !re.MatchString("app.kubernetes.io/name") {
			t.Error("expected pattern app* to match app.kubernetes.io/name")
		}
		if re.MatchString("other") {
			t.Error("expected pattern app* not to match 'other'")
		}
	})

	t.Run("same pattern returns same cached regexp instance", func(t *testing.T) {
		pattern := "cached-test-pattern*"
		re1 := cachedCompileAllowListPattern(pattern)
		re2 := cachedCompileAllowListPattern(pattern)
		if re1 != re2 {
			t.Error("expected the same *regexp.Regexp pointer on second call (cache hit)")
		}
	})

	t.Run("pattern exceeding wildcard limit returns nil", func(t *testing.T) {
		// "*key*" has two wildcards; without the rejection guard expandWildcard would silently
		// truncate it to "^.*key$" and match keys ending in "key".
		re := cachedCompileAllowListPattern("*key*")
		if re != nil {
			t.Errorf("expected nil for over-wildcarded pattern, got %v", re)
		}
	})

	t.Run("pre-cached nil entry returns nil without recompiling", func(t *testing.T) {
		pattern := "nil-sentinel-pattern*"
		expanded := expandWildcard(pattern, options.MaxPartialWildcardsPerLabel)
		allowListPatternCache.Store(expanded, nil)
		t.Cleanup(func() { allowListPatternCache.Delete(expanded) })

		re := cachedCompileAllowListPattern(pattern)
		if re != nil {
			t.Error("expected nil for pre-cached failed pattern")
		}
	})
}

func TestExpandWildcard(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
		limit    uint
	}{
		{
			input:    "foo",
			expected: "^foo$",
			limit:    1,
		},
		{
			input:    "foo*",
			expected: "^foo.*$",
			limit:    1,
		},
		{
			input:    "*foo",
			expected: "^.*foo$",
			limit:    1,
		},
		{
			input:    "*foo*",
			expected: "^.*foo$",
			limit:    1,
		},
		{
			input:    "*foo*",
			expected: "^.*foo.*$",
			limit:    2,
		},
		{
			input:    "*foo*",
			expected: "^.*foo.*$",
			limit:    3,
		},
		{
			input:    "*f*o*o*",
			expected: "^.*f.*o.*o$",
			limit:    3,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			got := expandWildcard(tc.input, tc.limit)
			if got != tc.expected {
				t.Errorf("expandWildcard() got = %v, want %v", got, tc.expected)
			}
		})
	}
}

func BenchmarkMapToPrometheusLabels(b *testing.B) {
	for _, n := range []int{4, 8, 32} {
		labels := make(map[string]string, n)
		for i := 0; i < n; i++ {
			labels[fmt.Sprintf("app.kubernetes.io/component-%d", i)] = fmt.Sprintf("value-%d", i)
		}

		b.Run(fmt.Sprintf("labels-%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, _ = mapToPrometheusLabels(labels, "label")
			}
		})
	}

	// Keys that sanitize to the same Prometheus label name exercise the conflict path.
	conflicting := map[string]string{"foo.bar": "a", "foo_bar": "b", "foo-bar": "c", "other": "d"}
	b.Run("conflicts", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = mapToPrometheusLabels(conflicting, "label")
		}
	})
}
