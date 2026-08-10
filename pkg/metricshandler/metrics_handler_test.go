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
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"k8s.io/kube-state-metrics/v2/internal/store"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

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

// "?resources=" names no resource, so it is a request for no filtering rather
// than a request for nothing. Same for its exclude counterpart, and for the
// combination where one is empty and the other is not.
func TestServeHTTPEmptyResourceFilter(t *testing.T) {
	handler := New(&options.Options{}, fake.NewSimpleClientset(), store.NewBuilder(), false)
	handler.metricsWriters = metricsstore.MetricsWriterList{
		metricsstore.NewMetricsWriter("pods", newStoreWithOneMetric(t, "kube_pod_test")),
		metricsstore.NewMetricsWriter("nodes", newStoreWithOneMetric(t, "kube_node_test")),
	}

	for _, tc := range []struct {
		query string
		want  []string
		gone  []string
	}{
		{query: "", want: []string{"kube_pod_test", "kube_node_test"}},
		{query: "?resources=", want: []string{"kube_pod_test", "kube_node_test"}},
		{query: "?exclude_resources=", want: []string{"kube_pod_test", "kube_node_test"}},
		{query: "?resources=&exclude_resources=nodes", want: []string{"kube_pod_test"}, gone: []string{"kube_node_test"}},
		{query: "?resources=pods", want: []string{"kube_pod_test"}, gone: []string{"kube_node_test"}},
	} {
		t.Run(tc.query, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/metrics"+tc.query, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			body := w.Body.String()
			for _, want := range tc.want {
				if !strings.Contains(body, want) {
					t.Errorf("expected %q in the response, got %q", want, body)
				}
			}
			for _, gone := range tc.gone {
				if strings.Contains(body, gone) {
					t.Errorf("did not expect %q in the response, got %q", gone, body)
				}
			}
		})
	}
}

func newStoreWithOneMetric(t *testing.T, name string) *metricsstore.MetricsStore {
	t.Helper()
	s := metricsstore.NewMetricsStore(
		[]string{"# HELP " + name + " help\n# TYPE " + name + " gauge"},
		func(interface{}) []metric.FamilyInterface {
			return []metric.FamilyInterface{&metric.Family{
				Name:    name,
				Metrics: []*metric.Metric{{Value: 1}},
			}}
		},
	)
	if err := s.Add(&v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns", UID: "uid"}}); err != nil {
		t.Fatalf("adding to store: %v", err)
	}
	return s
}
