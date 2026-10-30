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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	ksmtypes "k8s.io/kube-state-metrics/v2/pkg/builder/types"
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
	handler.ConfigureSharding(context.Background(), 0, 1)

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

// Until sharding is configured, e.g. because autosharding could not derive the
// shard from the StatefulSet, a scrape must fail rather than succeed empty.
func TestServeHTTPUnavailableUntilShardingConfigured(t *testing.T) {
	handler := New(&options.Options{}, fake.NewSimpleClientset(), &stubStoreBuilder{}, false)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("before sharding: got status %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	handler.ConfigureSharding(context.Background(), 0, 1)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("after sharding: got status %d, want %d", w.Code, http.StatusOK)
	}
}

// A scrape must not see sharding as configured before the first writers are
// built, or it would get an empty 200 instead of a 503.
func TestConfigureShardingBuildsBeforeReady(t *testing.T) {
	stub := &stubStoreBuilder{}
	handler := New(&options.Options{}, fake.NewSimpleClientset(), stub, false)
	// Build runs with mtx held, so the field can be read directly.
	stub.onBuild = func() {
		if handler.shardingConfigured {
			t.Error("sharding marked as configured before the writers were built")
		}
	}

	handler.ConfigureSharding(context.Background(), 0, 1)

	if !handler.shardingConfigured {
		t.Fatal("sharding not marked as configured after ConfigureSharding")
	}
}

// With autosharding, custom resource discovery may reconfigure the store before
// the shard is known. That must not build writers with the static default
// sharding, which would serve every object and duplicate series across shards.
func TestConfigureStoreWaitsForAutosharding(t *testing.T) {
	stub := &stubStoreBuilder{}
	handler := New(&options.Options{Pod: "ksm-0", Namespace: "ns"}, fake.NewSimpleClientset(), stub, false)

	if err := handler.ConfigureStore(context.Background(), func(ksmtypes.BuilderInterface) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if got := stub.buildCount.Load(); got != 0 {
		t.Fatalf("expected no build before sharding is configured, got %d", got)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("before sharding: got status %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	handler.ConfigureSharding(context.Background(), 1, 2)

	if got := stub.buildCount.Load(); got != 1 {
		t.Fatalf("expected one build once sharding is configured, got %d", got)
	}
	if stub.builtShard != 1 || stub.builtTotalShards != 2 {
		t.Fatalf("expected the build to use shard 1 of 2, got shard %d of %d", stub.builtShard, stub.builtTotalShards)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("after sharding: got status %d, want %d", w.Code, http.StatusOK)
	}
}

// The StatefulSet lookup must give up when Run's context is cancelled instead
// of blocking shutdown on an unresponsive API server.
func TestRunHonoursContextDuringStatefulSetLookup(t *testing.T) {
	controller := true
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ksm-0",
			Namespace: "ns",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "apps/v1",
				Kind:       "StatefulSet",
				Name:       "ksm",
				Controller: &controller,
			}},
		},
	}
	statefulSetReached := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/namespaces/ns/pods/ksm-0":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(pod)
		case "/apis/apps/v1/namespaces/ns/statefulsets/ksm":
			close(statefulSetReached)
			<-r.Context().Done()
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	// Drop a request still hanging here, so a regression fails the test below
	// instead of blocking srv.Close forever.
	defer srv.CloseClientConnections()

	client, err := kubernetes.NewForConfig(&rest.Config{Host: srv.URL})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
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
	select {
	case <-statefulSetReached:
	default:
		t.Fatal("expected Run to reach the StatefulSet lookup")
	}
}
