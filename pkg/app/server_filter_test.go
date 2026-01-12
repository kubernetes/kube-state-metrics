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

package app

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/kubernetes/fake"

	"k8s.io/kube-state-metrics/v2/internal/store"
	"k8s.io/kube-state-metrics/v2/pkg/allowdenylist"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	"k8s.io/kube-state-metrics/v2/pkg/metricshandler"
	"k8s.io/kube-state-metrics/v2/pkg/optin"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func TestResourceFiltering(t *testing.T) {
	t.Parallel()

	kubeClient := fake.NewSimpleClientset()

	err := pod(kubeClient, 0)
	if err != nil {
		t.Fatalf("failed to insert sample pod %v", err.Error())
	}
	err = service(kubeClient, 0)
	if err != nil {
		t.Fatalf("failed to insert sample service %v", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reg := prometheus.NewRegistry()
	builder := store.NewBuilder()
	builder.WithMetrics(reg)
	// Enable pods and services
	err = builder.WithEnabledResources([]string{"pods", "services"})
	if err != nil {
		t.Fatal(err)
	}
	builder.WithKubeClient(kubeClient)
	builder.WithNamespaces(options.DefaultNamespaces)
	builder.WithGenerateStoresFunc(builder.DefaultGenerateStoresFunc())

	l, err := allowdenylist.New(map[string]struct{}{}, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	optInMetrics := make(map[string]struct{})
	optInMetricFamilyFilter, err := optin.NewMetricFamilyFilter(optInMetrics)
	if err != nil {
		t.Fatal(err)
	}

	builder.WithFamilyGeneratorFilter(generator.NewCompositeFamilyGeneratorFilter(
		l,
		optInMetricFamilyFilter,
	))
	builder.WithAllowLabels(map[string][]string{})

	handler := metricshandler.New(&options.Options{}, kubeClient, builder, false)
	handler.ConfigureSharding(ctx, 0, 1)

	// Wait for caches to fill
	time.Sleep(time.Second)

	tests := []struct {
		name             string
		query            string
		expectedSubstr   []string
		unexpectedSubstr []string
	}{
		{
			name:             "Only pods",
			query:            "?resources=pods",
			expectedSubstr:   []string{"kube_pod_info"},
			unexpectedSubstr: []string{"kube_service_info"},
		},
		{
			name:             "Only services",
			query:            "?resources=services",
			expectedSubstr:   []string{"kube_service_info"},
			unexpectedSubstr: []string{"kube_pod_info"},
		},
		{
			name:             "Pods and services comma separated",
			query:            "?resources=pods,services",
			expectedSubstr:   []string{"kube_pod_info", "kube_service_info"},
			unexpectedSubstr: []string{},
		},
		{
			name:             "Pods and services multiple params",
			query:            "?resources=pods&resources=services",
			expectedSubstr:   []string{"kube_pod_info", "kube_service_info"},
			unexpectedSubstr: []string{},
		},
		{
			name:             "All resources (no filter)",
			query:            "",
			expectedSubstr:   []string{"kube_pod_info", "kube_service_info"},
			unexpectedSubstr: []string{},
		},
		{
			name:             "Non-existent resource",
			query:            "?resources=nonexistent",
			expectedSubstr:   []string{},
			unexpectedSubstr: []string{"kube_pod_info", "kube_service_info"},
		},
		{
			name:             "Exclude pods",
			query:            "?exclude_resources=pods",
			expectedSubstr:   []string{"kube_service_info"},
			unexpectedSubstr: []string{"kube_pod_info"},
		},
		{
			name:             "Exclude services",
			query:            "?exclude_resources=services",
			expectedSubstr:   []string{"kube_pod_info"},
			unexpectedSubstr: []string{"kube_service_info"},
		},
		{
			name:             "Include pods and services, exclude pods",
			query:            "?resources=pods,services&exclude_resources=pods",
			expectedSubstr:   []string{"kube_service_info"},
			unexpectedSubstr: []string{"kube_pod_info"},
		},
		{
			name:             "Only commas in resources",
			query:            "?resources=,,,,",
			expectedSubstr:   []string{},
			unexpectedSubstr: []string{"kube_pod_info", "kube_service_info"},
		},
		{
			name:             "Multiple commas between resources",
			query:            "?resources=pods,,,services",
			expectedSubstr:   []string{"kube_pod_info", "kube_service_info"},
			unexpectedSubstr: []string{},
		},
		{
			name:             "Only commas in exclude_resources",
			query:            "?exclude_resources=,,,,",
			expectedSubstr:   []string{"kube_pod_info", "kube_service_info"},
			unexpectedSubstr: []string{},
		},
		{
			name:             "Multiple commas between exclude_resources",
			query:            "?exclude_resources=pods,,,services",
			expectedSubstr:   []string{},
			unexpectedSubstr: []string{"kube_pod_info", "kube_service_info"},
		},
		{
			name:             "Multiple exclude_resources params",
			query:            "?exclude_resources=pods&exclude_resources=services",
			expectedSubstr:   []string{},
			unexpectedSubstr: []string{"kube_pod_info", "kube_service_info"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://localhost:8080/metrics"+tc.query, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != 200 {
				t.Fatalf("expected 200 status code but got %v", resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			bodyStr := string(body)

			for _, substr := range tc.expectedSubstr {
				if !strings.Contains(bodyStr, substr) {
					t.Errorf("expected body to contain %q, but it didn't", substr)
				}
			}

			for _, substr := range tc.unexpectedSubstr {
				if strings.Contains(bodyStr, substr) {
					t.Errorf("expected body NOT to contain %q, but it did", substr)
				}
			}
		})
	}
}
