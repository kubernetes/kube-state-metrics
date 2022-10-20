/*
Copyright 2022 The Kubernetes Authors All rights reserved.

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

// To skip this test, put metric name into skippedStableMetrics.
package stable

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	prommodel "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"gopkg.in/yaml.v3"

	"k8s.io/klog/v2"
)

var promText string
var stableYaml string

var skippedStableMetrics = []string{}

type Metric struct {
	Name   string   `yaml:"name"`
	Help   string   `yaml:"help"`
	Type   string   `yaml:"type"`
	Labels []string `yaml:"labels"`
	// Histogram type
	Buckets []float64 `yaml:"buckets,omitempty"`
}

func TestMain(m *testing.M) {
	flag.StringVar(&promText, "collectedMetricsFile", "", "input prometheus metrics text file, text format")
	flag.StringVar(&stableYaml, "stableMetricsFile", "", "expected stable metrics yaml file, yaml format")
	flag.Parse()
	m.Run()
}

func TestStableMetrics(t *testing.T) {
	mf, err := parsePromText(promText)
	if err != nil {
		t.Fatalf("Can't parse collected prometheus metrics text. err = %v", err)
	}
	collectedStableMetrics := extractStableMetrics(mf)
	printMetric(collectedStableMetrics)

	expectedStableMetrics, err := readYaml(stableYaml)
	if err != nil {
		t.Fatalf("Can't read stable metrics from file. err = %v", err)
	}

	err = compare(collectedStableMetrics, *expectedStableMetrics, skippedStableMetrics)
	if err != nil {
		t.Fatalf("Stable metrics changed: err = %v", err)
	} else {
		klog.Infoln("Passed")
	}
}

func compare(collectedStableMetrics []Metric, expectedStableMetrics []Metric, skippedStableMetrics []string) error {
	skipMap := map[string]int{}
	for _, v := range skippedStableMetrics {
		skipMap[v] = 1
	}
	collected := convertToMap(collectedStableMetrics, skipMap)
	expected := convertToMap(expectedStableMetrics, skipMap)

	var ok bool

	for _, name := range sortedKeys(expected) {
		metric := expected[name]
		var expectedMetric Metric
		if expectedMetric, ok = collected[name]; !ok {
			return fmt.Errorf("not found stable metric %s", name)
		}
		// Ingore Labels field due to ordering issue
		if diff := cmp.Diff(metric, expectedMetric, cmpopts.IgnoreFields(Metric{}, "Help", "Labels")); diff != "" {
			return fmt.Errorf("stable metric %s mismatch (-want +got):\n%s", name, diff)
		}
		// Compare Labels field after sorting
		if diff := cmp.Diff(metric.Labels, expectedMetric.Labels, cmpopts.SortSlices(func(l1, l2 string) bool { return l1 > l2 })); diff != "" {
			return fmt.Errorf("stable metric label %s mismatch (-want +got):\n%s", name, diff)
		}
	}
	for _, name := range sortedKeys(collected) {
		if _, ok = expected[name]; !ok {
			return fmt.Errorf("detected new stable metric %s which isn't in testdata ", name)
		}
	}
	return nil
}

func printMetric(metrics []Metric) {
	yamlData, err := yaml.Marshal(metrics)
	if err != nil {
		klog.Errorf("error while Marshaling. %v", err)
	}
	klog.Infoln("---begin YAML file---")
	klog.Infoln(string(yamlData))
	klog.Infoln("---end YAML file---")
}

func parsePromText(path string) (map[string]*prommodel.MetricFamily, error) {
	reader, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		return nil, err
	}
	return mf, nil
}

func getBuckets(v *prommodel.MetricFamily) []float64 {
	buckets := []float64{}
	if v.GetType() == prommodel.MetricType_HISTOGRAM {
		for _, bucket := range v.Metric[0].GetHistogram().GetBucket() {
			buckets = append(buckets, *bucket.UpperBound)
		}
	} else {
		buckets = nil
	}
	return buckets
}

func getLabels(m *prommodel.Metric) []string {
	labels := []string{}
	for _, y := range m.Label {
		labels = append(labels, y.GetName())
	}
	return labels
}

func extractStableMetrics(mf map[string]*prommodel.MetricFamily) []Metric {
	metrics := []Metric{}
	for _, v := range mf {
		// Find stable metrics
		if !strings.Contains(*(v.Help), "[STABLE]") {
			continue
		}
		metrics = append(metrics, Metric{
			Name:    *(v.Name),
			Help:    *(v.Help),
			Type:    (v.Type).String(),
			Buckets: getBuckets(v),
			Labels:  getLabels(v.Metric[0]),
		})
	}
	return metrics
}

func readYaml(filename string) (*[]Metric, error) {
	buf, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}
	c := &[]Metric{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, fmt.Errorf("error %q: %w", filename, err)
	}
	return c, err
}

func convertToMap(metrics []Metric, skipMap map[string]int) map[string]Metric {
	name2Metric := map[string]Metric{}
	for _, v := range metrics {
		if _, ok := skipMap[v.Name]; ok {
			klog.Infof("skip, metric %s is in skip list\n", v.Name)
			continue
		}
		name2Metric[v.Name] = v
	}
	return name2Metric
}

func sortedKeys(m map[string]Metric) []string {
	keys := []string{}
	for name := range m {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}
