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

package stable

import (
	"flag"
	"fmt"
	"sort"
	"testing"

	"log"
	"os"
	"strings"

	"github.com/google/go-cmp/cmp"
	prommodel "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"gopkg.in/yaml.v3"

	"github.com/google/go-cmp/cmp/cmpopts"
)

var promText string
var stableYaml string

var skipStableMetrics = []string{}

type Metric struct {
	Name   string `yaml:"name"`
	Help   string `yaml:"help"`
	Type   string
	Labels []string
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
	fatal(err)
	collectedStableMetrics := extractStableMetrics(mf)
	printMetric(collectedStableMetrics)

	expectedStableMetrics, err := readYaml(stableYaml)
	if err != nil {
		t.Fatalf("Can't read stable metrics from file. err = %v", err)
	}

	err = compare(collectedStableMetrics, *expectedStableMetrics, skipStableMetrics)
	if err != nil {
		t.Fatalf("Stable metrics changed: err = %v", err)
	} else {
		fmt.Println("passed")
	}

}

func fatal(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func printMetric(metrics []Metric) {
	yamlData, err := yaml.Marshal(metrics)
	if err != nil {
		fmt.Printf("error while Marshaling. %v", err)
	}
	fmt.Println("---begin YAML file---")
	fmt.Println(string(yamlData))
	fmt.Println("---end YAML file---")
}

func parsePromText(path string) (map[string]*prommodel.MetricFamily, error) {
	reader, err := os.Open(path)
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

func extractStableMetrics(mf map[string]*prommodel.MetricFamily) []Metric {
	metrics := []Metric{}
	for _, v := range mf {
		// Find stable metrics
		if !strings.Contains(*(v.Help), "[STABLE]") {
			continue
		}

		m := Metric{
			Name:    *(v.Name),
			Help:    *(v.Help),
			Type:    (v.Type).String(),
			Buckets: getBuckets(v),
		}
		labels := []string{}
		for _, y := range v.Metric[0].Label {
			labels = append(labels, y.GetName())
		}
		m.Labels = labels
		metrics = append(metrics, m)
	}
	return metrics
}

func readYaml(filename string) (*[]Metric, error) {
	buf, err := os.ReadFile(filename)
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

func convert2Map(metrics []Metric, skipMap map[string]int) map[string]Metric {
	name2Metric := map[string]Metric{}
	for _, v := range metrics {
		if _, ok := skipMap[v.Name]; ok {
			fmt.Printf("skip, metric %s is in skip list\n", v.Name)
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

func compare(collectedStableMetrics []Metric, expectedStableMetrics []Metric, skipStableMetrics []string) error {
	skipMap := map[string]int{}
	for _, v := range skipStableMetrics {
		skipMap[v] = 1
	}
	collected := convert2Map(collectedStableMetrics, skipMap)
	expected := convert2Map(expectedStableMetrics, skipMap)

	var ok bool

	for _, name := range sortedKeys(expected) {
		metric := expected[name]
		var expectedMetric Metric
		if expectedMetric, ok = collected[name]; !ok {
			return fmt.Errorf("not found stable metric %s", name)
		}
		if diff := cmp.Diff(metric, expectedMetric, cmpopts.IgnoreFields(Metric{}, "Help")); diff != "" {
			return fmt.Errorf("stable metric %s mismatch (-want +got):\n%s", name, diff)
		}
	}
	for _, name := range sortedKeys(collected) {
		if _, ok = expected[name]; !ok {
			return fmt.Errorf("detected new stable metric %s which isn't in testdata ", name)
		}
	}
	return nil
}
