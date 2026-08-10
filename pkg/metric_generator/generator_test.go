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

package generator

import (
	"strings"
	"testing"

	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
)

// The help text of a custom resource metric comes from user-supplied YAML, so it
// can contain characters the exposition format reserves. A newline ends the HELP
// line early and a backslash starts an escape sequence, either of which makes the
// whole exposition unparseable rather than just the offending family.
func TestGenerateHeaderEscapesHelp(t *testing.T) {
	for _, tc := range []struct {
		name string
		help string
		want string
	}{
		{name: "plain", help: "Information about a thing.", want: "Information about a thing."},
		{name: "backslash", help: `matches \d+ digits`, want: `matches \\d+ digits`},
		{name: "trailing backslash", help: `ends with \`, want: `ends with \\`},
		{name: "newline", help: "line one\nline two", want: `line one\nline two`},
		{name: "carriage return is not reserved", help: "a\rb", want: "a\rb"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := NewFamilyGeneratorWithStability(
				"kube_test_metric", tc.help, metric.Gauge, basemetrics.ALPHA, "",
				func(interface{}) *metric.Family { return &metric.Family{} },
			)

			got := g.generateHeader()
			want := "# HELP kube_test_metric " + tc.want + "\n# TYPE kube_test_metric gauge\n"
			if got != want {
				t.Errorf("got header %q, want %q", got, want)
			}
			// The HELP line must stay a single line, otherwise the TYPE line that
			// follows is no longer where every reader of this header expects it.
			if lines := strings.Count(got, "\n"); lines != 2 {
				t.Errorf("header spans %d lines, want 2: %q", lines, got)
			}
		})
	}
}

// STABLE metrics get their stability level prefixed onto the help text, which is
// a separate write path and needs the same escaping.
func TestGenerateHeaderEscapesHelpForStableMetrics(t *testing.T) {
	g := NewFamilyGeneratorWithStability(
		"kube_test_metric", `matches \d+ digits`, metric.Gauge, basemetrics.STABLE, "",
		func(interface{}) *metric.Family { return &metric.Family{} },
	)

	got := g.generateHeader()
	want := `# HELP kube_test_metric [STABLE] matches \\d+ digits` + "\n# TYPE kube_test_metric gauge\n"
	if got != want {
		t.Errorf("got header %q, want %q", got, want)
	}
}
