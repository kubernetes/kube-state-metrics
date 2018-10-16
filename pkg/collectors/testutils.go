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

package collectors

// TODO: Does this file need to be renamed to not be compiled in production?

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"k8s.io/kube-state-metrics/pkg/metrics"
)

type generateMetricsTestCase struct {
	Obj         interface{}
	MetricNames []string
	Want        string
	Func        func(interface{}) []*metrics.Metric
}

func (testCase *generateMetricsTestCase) run() error {
	metrics := testCase.Func(testCase.Obj)
	metrics = filterMetrics(metrics, testCase.MetricNames)

	out := ""

	for _, m := range metrics {
		out += string(*m)
	}

	out = removeUnusedWhitespace(out)
	out = sortByLine(out)

	want := removeUnusedWhitespace(testCase.Want)
	want = sortByLine(want)

	if out != want {
		return fmt.Errorf("expected %v\nbut got %v", want, out)
	}

	return nil
}

func sortByLine(s string) string {
	split := strings.Split(s, "\n")
	sort.Strings(split)
	return strings.Join(split, "\n")
}

func filterMetrics(ms []*metrics.Metric, names []string) []*metrics.Metric {
	// In case the test case is based on all returned metrics, MetricNames does
	// not need to me defined.
	if names == nil {
		return ms
	}
	filtered := []*metrics.Metric{}

	regexps := []*regexp.Regexp{}
	for _, n := range names {
		regexps = append(regexps, regexp.MustCompile(fmt.Sprintf("^%v", n)))
	}

	for _, m := range ms {
		drop := true
		for _, r := range regexps {
			if r.MatchString(string(*m)) {
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
		trimmedLine = strings.TrimSpace(l)

		if len(trimmedLine) > 0 {
			trimmedLines = append(trimmedLines, trimmedLine)
		}
	}

	// The Prometheus metrics representation parser expects an empty line at the
	// end otherwise fails with an unexpected EOF error.
	return strings.Join(trimmedLines, "\n") + "\n"
}
