/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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
	_ "embed"
	"flag"
	"os"
	"testing"

	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	update = flag.Bool("update", false, "update the golden files")
)

func filterStableMetrics(familyGenerator []generator.FamilyGenerator) []generator.FamilyGenerator {
	filtered := []generator.FamilyGenerator{}
	for _, f := range familyGenerator {
		if f.StabilityLevel == generator.STABLE {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

func TestStableMetrics(t *testing.T) {
	flag.Parse()
	cases := []generateStableMetricsTestCase{
		{
			Name: "Node stable metrics",
			FilePath: "../../tests/testdata/stable_node_metrics.txt",
			generateMetricsTestCase: generateMetricsTestCase{
				DropHelp: true,
				Obj: &v1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "",
					},
					Status: v1.NodeStatus{
						Conditions: []v1.NodeCondition{
							{Type: v1.NodeNetworkUnavailable},
						},
					},
				},
				Want: ``, // will be replaced by data from filePath
			},
		},
	}

	for i, cs := range cases {
		var err error
		c := cs.generateMetricsTestCase

		metrics := filterStableMetrics(nodeMetricFamilies(nil, nil))
		c.Func = generator.ComposeMetricGenFuncs(metrics)
		c.Headers = generator.ExtractMetricFamilyHeaders(metrics)
		path := cs.FilePath

		if *update {
			writeFile(c.runWithOutput(), path)
		} else {
			c.Want, err = readFile(path)
			if err != nil {
				t.Errorf("Can't open file %v", path)
			}
			err = c.run()
			if err != nil {
				t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
			}
		}
	}
}

func readFile(fileName string) (string, error) {
	b, err := os.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	return string(b), err
}

func writeFile(data string, fileName string) {
	file, err := os.Create(fileName)
	if err != nil {
		return
	}
	defer file.Close()

	file.WriteString(data)
}
