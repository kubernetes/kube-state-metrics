/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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

package optin

import (
	"testing"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		MetricFamily string
		IsOptIn      bool
		OptInMetric  string
		Want         bool
	}{
		{"kube_pod_container_status_running", true, "kube_pod_container_status_.+", true},
		{"kube_pod_container_status_terminated", true, "kube_pod_container_status_running", false},
		{"kube_pod_container_status_reason", true, "kube_pod_container_status_(running|terminated)", false},
		{"kube_node_info", false, "", true},
	}

	for _, test := range tests {
		filter, err := NewMetricFamilyFilter(map[string]struct{}{
			test.OptInMetric: {},
		})
		if err != nil {
			t.Errorf("did not expect NewMetricFamilyFilter to fail, the error is %v", err)
		}

		optInFamilyGenerator := *generator.NewFamilyGenerator(
			test.MetricFamily,
			"",
			metric.Gauge,
			"",
			func(_ interface{}) *metric.Family {
				return nil
			},
		)
		optInFamilyGenerator.OptIn = test.IsOptIn

		result := filter.Test(optInFamilyGenerator)
		if result != test.Want {
			t.Errorf("the metric family did not pass the filter, got: %v, want: %v", result, test.Want)
		}
	}
}

func TestRegexParsing(t *testing.T) {
	t.Run("should fail if an invalid regular expression is passed in", func(t *testing.T) {
		_, err := NewMetricFamilyFilter(map[string]struct{}{"*_pod_info": {}})
		if err == nil {
			t.Errorf("expected NewMetricFamilyFilter to fail for invalid regex pattern")
		}
	})

	t.Run("should succeed when valid regular expressions are passed in", func(t *testing.T) {
		_, err := NewMetricFamilyFilter(map[string]struct{}{"kube_.*_info": {}})
		if err != nil {
			t.Errorf("expected NewMetricFamilyFilter to succeed, but failed : %v", err)
		}
	})
}

func TestStatus(t *testing.T) {
	filter, err := NewMetricFamilyFilter(map[string]struct{}{
		"kube_pod_container_status_running":    {},
		"kube_pod_container_status_terminated": {},
	})
	if err != nil {
		t.Errorf("did not expect NewMetricFamilyFilter to fail, the error is %v", err)
	}

	status := filter.Status()
	if status != "kube_pod_container_status_running, kube_pod_container_status_terminated" {
		t.Errorf("the metric family filter did not return the correct status, got: \"%v\", want: \"%v\"", status, "kube_pod_container_status_running, kube_pod_container_status_terminated")
	}
}

func TestCount(t *testing.T) {
	filter, err := NewMetricFamilyFilter(map[string]struct{}{
		"kube_pod_container_status_running":    {},
		"kube_pod_container_status_terminated": {},
	})
	if err != nil {
		t.Errorf("did not expect NewMetricFamilyFilter to fail, the error is %v", err)
	}

	if filter.Count() != 2 {
		t.Errorf("the metric family filter did not return the correct amount of filters in it, got: %v, want: %v", filter.Count(), 2)
	}
}
