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
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/prometheus/util/promlint"
)

func TestMain(m *testing.M) {
	ksmurl := flag.String(
		"ksmurl",
		"",
		"url to access the kube-state-metrics service",
	)
	flag.Parse()

	var (
		err      error
		exitCode int
	)

	if framework, err = NewFramework(*ksmurl); err != nil {
		log.Fatalf("failed to setup framework: %v\n", err)
	}

	exitCode = m.Run()

	os.Exit(exitCode)
}

func TestIsHealthz(t *testing.T) {
	ok, err := framework.KsmClient.isHealthz()
	if err != nil {
		t.Fatalf("kube-state-metrics healthz check failed: %v", err)
	}

	if ok == false {
		t.Fatal("kube-state-metrics is unhealthy")
	}
}

func TestLintMetrics(t *testing.T) {
	buf := &bytes.Buffer{}

	err := framework.KsmClient.metrics(buf)
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

	buf := &bytes.Buffer{}

	err = framework.KsmClient.metrics(buf)
	if err != nil {
		t.Fatal("Failed to get metrics from kube-state-metrics", err)
	}

	parser := &expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(buf)
	if err != nil {
		t.Fatal("Cannot decode metrics", err)
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
						t.Errorf("Cannot compile pattern %s: %w", labelPattern, err)
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

	docPath := "../../docs/"
	docFiles, err := ioutil.ReadDir(docPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read documentation directory")
	}

	// Match file names such as daemonset-metrics.md
	fileRe := regexp.MustCompile(`^([a-z]*)-metrics.md$`)
	// Match doc lines such as | kube_node_created | Gauge | `node`=&lt;node-address&gt;| STABLE |
	lineRe := regexp.MustCompile(`^\| *(kube_[a-z_]+) *\| *[a-zA-Z]+ *\|(.*)\| *[A-Z]+`)
	// Match label names in label documentation
	labelsRe := regexp.MustCompile("`([a-zA-Z_][a-zA-Z0-9_]*)`")
	// Match wildcard patterns for dynamic labels such as label_CRONJOB_LABEL
	patternRe := regexp.MustCompile(`_[A-Z_]+`)

	for _, file := range docFiles {
		if file.IsDir() || !fileRe.MatchString(file.Name()) {
			continue
		}

		filePath := path.Join(docPath, file.Name())
		f, err := os.Open(filePath)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot read file %s", filePath)
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
					return nil, errors.Errorf("Label documentation %s did not match regex", labelsDoc)
				}
				labelPatterns[i] = patternRe.ReplaceAllString(l[1], "_.*")
			}

			documentedMetrics[metric] = labelPatterns
		}
	}
	return documentedMetrics, nil
}
