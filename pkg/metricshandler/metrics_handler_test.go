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
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

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

// A client may send more than one gzip token in Accept-Encoding. Wrapping the
// response writer once per token would leave every inner writer unclosed, so its
// deflate stream is never finalised and the body cannot be decompressed.
func TestServeHTTPGzipWrapsOnce(t *testing.T) {
	// No writers are needed: the response is wrapped before any are consulted,
	// and an empty gzip stream is still a valid one.
	handler := New(&options.Options{}, fake.NewSimpleClientset(), &stubStoreBuilder{}, true)
	handler.BuildWriters(context.Background())

	for _, acceptEncoding := range []string{"gzip", "gzip, gzip", "gzip;q=1.0, gzip"} {
		t.Run(acceptEncoding, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/metrics", nil)
			req.Header.Set("Accept-Encoding", acceptEncoding)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()
			if got := res.Header.Get("Content-Encoding"); got != "gzip" {
				t.Fatalf("Content-Encoding: got %q, want %q", got, "gzip")
			}

			zr, err := gzip.NewReader(res.Body)
			if err != nil {
				t.Fatalf("response is not readable gzip: %v", err)
			}
			defer zr.Close()
			if _, err := io.ReadAll(zr); err != nil {
				t.Fatalf("decompressing response: %v", err)
			}
		})
	}
}

// Until the writers are built, e.g. because autosharding could not derive the
// shard from the StatefulSet, a scrape must fail rather than succeed empty.
func TestServeHTTPUnavailableUntilWritersBuilt(t *testing.T) {
	handler := New(&options.Options{}, fake.NewSimpleClientset(), &stubStoreBuilder{}, false)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("before build: got status %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	handler.BuildWriters(context.Background())

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("after build: got status %d, want %d", w.Code, http.StatusOK)
	}
}

// The StatefulSet lookup must give up when Run's context is cancelled instead
// of blocking shutdown on an unresponsive API server.
func TestRunHonoursContextDuringStatefulSetLookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()
	// Drop a request still hanging here, so a regression fails the test below
	// instead of blocking srv.Close forever.
	defer srv.CloseClientConnections()

	client, err := kubernetes.NewForConfig(&rest.Config{Host: srv.URL})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	handler := New(&options.Options{Pod: "ksm-0", Namespace: "ns"}, client, &stubStoreBuilder{}, false)
	done := make(chan error, 1)
	go func() { done <- handler.Run(ctx) }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected Run to fail when its context ends during the lookup")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after its context was cancelled")
	}
}
