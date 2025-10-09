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

package e2e

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil/promlint"
	dto "github.com/prometheus/client_model/go"

	ksmFramework "k8s.io/kube-state-metrics/v2/tests/e2e/framework"
)

var framework *ksmFramework.Framework

func TestMain(m *testing.M) {
	ksmHTTPMetricsURL := flag.String(
		"ksm-http-metrics-url",
		"",
		"url to access the kube-state-metrics service",
	)
	ksmTelemetryURL := flag.String(
		"ksm-telemetry-url",
		"",
		"url to access the kube-state-metrics telemetry endpoint",
	)
	flag.Parse()

	var (
		err      error
		exitCode int
	)

	if framework, err = ksmFramework.New(*ksmHTTPMetricsURL, *ksmTelemetryURL); err != nil {
		log.Fatalf("failed to setup framework: %v\n", err)
	}

	exitCode = m.Run()

	os.Exit(exitCode)
}

func TestIsHealthz(t *testing.T) {
	ok, err := framework.KsmClient.IsHealthz()
	if err != nil {
		t.Fatalf("kube-state-metrics healthz check failed: %v", err)
	}

	if ok == false {
		t.Fatal("kube-state-metrics is unhealthy")
	}
}

func TestLintMetrics(t *testing.T) {
	buf := &bytes.Buffer{}

	err := framework.KsmClient.Metrics(buf)
	if err != nil {
		t.Fatalf("failed to get metrics from kube-state-metrics: %v", err)
	}

	l := promlint.New(buf)
	problems, err := l.Lint()
	if err != nil {
		t.Fatalf("failed to lint: %v", err)
	}

	if len(problems) != 0 {
		t.Fatalf("the problems encountered in Lint are: %v", problems)
	}
}

func TestDocumentation(t *testing.T) {
	labelsDocumentation, err := getLabelsDocumentation()
	if err != nil {
		t.Fatal("Cannot get labels documentation", err)
	}

	metricFamilies, err := framework.ParseMetrics(framework.KsmClient.Metrics)
	if err != nil {
		t.Fatal("Failed to get or decode metrics", err)
	}

	for _, metricFamily := range metricFamilies {
		metric := metricFamily.GetName()

		acceptedLabelNames, ok := labelsDocumentation[metric]
		if !ok {
			t.Errorf("Metric %s not found in documentation.", metric)
			continue
		}
		for _, m := range metricFamily.Metric {
			for _, l := range m.Label {
				labelName := l.GetName()
				labelNameMatched := false
				for _, labelPattern := range acceptedLabelNames {
					re, err := regexp.Compile(labelPattern)
					if err != nil {
						t.Errorf("Cannot compile pattern %s: %v", labelPattern, err)
						continue
					}
					if re.MatchString(labelName) {
						labelNameMatched = true
						break
					}
				}
				if !labelNameMatched {
					t.Errorf("Label %s not found in documentation. Documented labels for metric %s are: %s",
						labelName, metric, strings.Join(acceptedLabelNames, ", "))
				}
			}
		}
	}
}

// getLabelsDocumentation is a helper function that gets metric mabels documentation.
// It returns a map where keys are metric names, and values are slices of label names,
// and an error in case of failure.
// By convention, UPPER_CASE parts in label names denotes wilcard patterns, used for dynamic labels.
func getLabelsDocumentation() (map[string][]string, error) {
	documentedMetrics := map[string][]string{}

	// Match file names such as daemonset-metrics.md
	fileRe := regexp.MustCompile(`^([a-z]*)-metrics.md$`)
	// Match doc lines such as | kube_node_created | Gauge | `node`=&lt;node-address&gt;| STABLE |
	lineRe := regexp.MustCompile(`^\| *(kube_[a-z_]+) *\| *[a-zA-Z]+ *\|(.*)\| *[A-Z]+`)
	// Match label names in label documentation
	labelsRe := regexp.MustCompile("`([a-zA-Z_][a-zA-Z0-9_]*)`")
	// Match wildcard patterns for dynamic labels such as label_CRONJOB_LABEL
	patternRe := regexp.MustCompile(`_[A-Z_]+`)

	err := filepath.WalkDir("../../docs", func(p string, d fs.DirEntry, _ error) error {

		if d.IsDir() || !fileRe.MatchString(d.Name()) {
			// Ignore the entry
			return nil
		}

		f, e := os.Open(filepath.Clean(p))
		if e != nil {
			return fmt.Errorf("cannot read file %s: %w", p, e)
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			params := lineRe.FindStringSubmatch(scanner.Text())
			if len(params) != 3 {
				continue
			}
			metric := params[1]
			labelsDoc := params[2]

			labels := labelsRe.FindAllStringSubmatch(labelsDoc, -1)
			labelPatterns := make([]string, len(labels))
			for i, l := range labels {
				if len(l) <= 1 {
					return fmt.Errorf("label documentation %s did not match regex", labelsDoc)
				}
				labelPatterns[i] = patternRe.ReplaceAllString(l[1], "_.*")
			}

			documentedMetrics[metric] = labelPatterns
		}
		return nil
	})
	if err != nil {
		log.Fatalf("cannot walk the documentation directory: %s", err)
	}

	return documentedMetrics, nil
}

func TestKubeStateMetricsErrorMetrics(t *testing.T) {
	metricFamilies, err := framework.ParseMetrics(framework.KsmClient.TelemetryMetrics)
	if err != nil {
		t.Fatal("Failed to get or decode telemetry metrics", err)
	}

	// This map's keys are the metrics expected in kube-state-metrics telemetry.
	// Its values are booleans, set to true when the metric is found.
	foundMetricFamily := map[string]bool{
		"kube_state_metrics_list_total":  false,
		"kube_state_metrics_watch_total": false,
	}

	for _, metricFamily := range metricFamilies {
		name := metricFamily.GetName()
		if _, expectedMetric := foundMetricFamily[name]; expectedMetric {
			foundMetricFamily[name] = true

			for _, m := range metricFamily.Metric {
				if hasLabelError(m) && m.GetCounter().GetValue() > 0 {
					t.Errorf("Metric %s in telemetry shows a list/watch error", prettyPrintCounter(name, m))
				}
			}
		}
	}

	for metricFamily, found := range foundMetricFamily {
		if !found {
			t.Errorf("Metric family %s was not found in telemetry metrics", metricFamily)
		}
	}
}

func hasLabelError(metric *dto.Metric) bool {
	for _, l := range metric.Label {
		if l.GetName() == "result" && l.GetValue() == "error" {
			return true
		}
	}
	return false
}

func prettyPrintCounter(name string, metric *dto.Metric) string {
	labelStrings := []string{}
	for _, l := range metric.Label {
		labelStrings = append(labelStrings, fmt.Sprintf(`%s="%s"`, l.GetName(), l.GetValue()))
	}
	return fmt.Sprintf("%s{%s} %d", name, strings.Join(labelStrings, ","), int(metric.GetCounter().GetValue()))
}

func TestDefaultCollectorMetricsAvailable(t *testing.T) {
	buf := &bytes.Buffer{}

	err := framework.KsmClient.Metrics(buf)
	if err != nil {
		t.Fatalf("failed to get metrics from kube-state-metrics: %v", err)
	}

	resources := map[string]struct{}{}
	nonDefaultResources := map[string]bool{
		"clusterrole":        true,
		"clusterrolebinding": true,
		"endpointslice":      true,
		"ingressclass":       true,
		"role":               true,
		"rolebinding":        true,
		"serviceaccount":     true,
	}
	nonResources := map[string]bool{
		"builder":   true,
		"metadata":  true,
		"utils":     true,
		"testutils": true,
	}

	files, err := os.ReadDir("../../internal/store/")
	if err != nil {
		t.Fatalf("failed to read dir to get all resources name: %v", err)
	}

	re := regexp.MustCompile(`^([a-z]+).go$`)
	for _, file := range files {
		params := re.FindStringSubmatch(file.Name())
		if len(params) != 2 {
			continue
		}
		if nonResources[params[1]] {
			// Non resource file
			continue
		}
		if nonDefaultResources[params[1]] {
			// Resource disabled by default
			continue
		}
		resources[params[1]] = struct{}{}
	}

	re = regexp.MustCompile(`^kube_([a-z]+)_`)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		params := re.FindStringSubmatch(scanner.Text())
		if len(params) != 2 {
			continue
		}
		delete(resources, params[1])
	}

	err = scanner.Err()
	if err != nil {
		t.Fatalf("failed to scan metrics: %v", err)
	}

	if len(resources) != 0 {
		s := []string{}
		for k := range resources {
			s = append(s, k)
		}
		sort.Strings(s)
		t.Fatalf("failed to find metrics of resources: %s", strings.Join(s, ", "))
	}
}
