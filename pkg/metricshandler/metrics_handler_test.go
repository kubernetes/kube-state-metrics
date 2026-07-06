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

package metricshandler

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/prometheus/common/expfmt"
)

func TestNegotiateSupportedContentType(t *testing.T) {
	prometheusDefaultAccept := "application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.6," +
		"application/openmetrics-text;version=1.0.0;escaping=allow-utf-8;q=0.5," +
		"application/openmetrics-text;version=0.0.1;q=0.4," +
		"text/plain;version=1.0.0;escaping=allow-utf-8;q=0.3," +
		"text/plain;version=0.0.4;q=0.2," +
		"*/*;q=0.1"

	tests := []struct {
		name     string
		accept   string
		expected expfmt.Format
	}{
		{
			name:     "prometheus default accept prefers openmetrics over text/plain",
			accept:   prometheusDefaultAccept,
			expected: expfmt.FmtOpenMetrics_1_0_0 + "; escaping=allow-utf-8",
		},
		{
			name:     "openmetrics only",
			accept:   "application/openmetrics-text;version=1.0.0",
			expected: expfmt.FmtOpenMetrics_1_0_0 + "; escaping=values",
		},
		{
			name:     "text plain only",
			accept:   "text/plain;version=0.0.4",
			expected: expfmt.FmtText,
		},
		{
			name:     "protobuf only falls back to text plain",
			accept:   "application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.6",
			expected: expfmt.FmtText,
		},
		{
			name:     "empty accept falls back to text plain",
			accept:   "",
			expected: expfmt.FmtText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.Header{}
			if tt.accept != "" {
				h.Set("Accept", tt.accept)
			}
			got := negotiateSupportedContentType(h)
			if got.FormatType() != tt.expected.FormatType() {
				t.Errorf("negotiateSupportedContentType() format type = %v, want %v (got %q, want %q)", got.FormatType(), tt.expected.FormatType(), got, tt.expected)
			}
			if tt.name == "prometheus default accept prefers openmetrics over text/plain" {
				if got != tt.expected {
					t.Errorf("negotiateSupportedContentType() = %q, want %q", got, tt.expected)
				}
			}
		})
	}
}

func TestFilterProtoFromAccept(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   string
	}{
		{
			name:   "removes protobuf entry",
			accept: "application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.6,text/plain;version=0.0.4;q=0.2",
			want:   "text/plain;version=0.0.4;q=0.2",
		},
		{
			name:   "empty accept",
			accept: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterProtoFromAccept(tt.accept)
			if got != tt.want {
				t.Errorf("filterProtoFromAccept() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseResources(t *testing.T) {
	tests := []struct {
		name     string
		params   []string
		expected map[string]struct{}
	}{
		{
			name:     "nil params",
			params:   nil,
			expected: nil,
		},
		{
			name:     "empty params",
			params:   []string{},
			expected: map[string]struct{}{},
		},
		{
			name:     "single resource",
			params:   []string{"pods"},
			expected: map[string]struct{}{"pods": {}},
		},
		{
			name:     "comma separated resources",
			params:   []string{"pods,deployments"},
			expected: map[string]struct{}{"pods": {}, "deployments": {}},
		},
		{
			name:     "multiple params strings",
			params:   []string{"pods", "deployments"},
			expected: map[string]struct{}{"pods": {}, "deployments": {}},
		},
		{
			name:     "mixed comma and multiple params",
			params:   []string{"pods,services", "deployments"},
			expected: map[string]struct{}{"pods": {}, "services": {}, "deployments": {}},
		},
		{
			name:     "whitespace handling",
			params:   []string{" pods , deployments "},
			expected: map[string]struct{}{"pods": {}, "deployments": {}},
		},
		{
			name:     "empty strings in split",
			params:   []string{"pods,,deployments"},
			expected: map[string]struct{}{"pods": {}, "deployments": {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseResources(tt.params)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseResources() = %v, want %v", got, tt.expected)
			}
		})
	}
}
