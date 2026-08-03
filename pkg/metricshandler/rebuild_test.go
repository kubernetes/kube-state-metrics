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
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	ksmtypes "k8s.io/kube-state-metrics/v2/pkg/builder/types"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

type stubStoreBuilder struct {
	buildWriters metricsstore.MetricsWriterList
	syncOK       atomic.Bool
	buildCount   atomic.Int64
	syncCount    atomic.Int64
	// syncGate, when set, makes WaitForStoresSync block until a result is sent.
	// This lets tests assert mid-retry state without racing the retry timer.
	syncGate chan bool
}

func (s *stubStoreBuilder) WithMetrics(_ prometheus.Registerer)                         {}
func (s *stubStoreBuilder) WithEnabledResources(_ []string) error                       { return nil }
func (s *stubStoreBuilder) WithNamespaces(_ options.NamespaceList)                      {}
func (s *stubStoreBuilder) WithFieldSelectorFilter(_ string)                            {}
func (s *stubStoreBuilder) WithSharding(_ int32, _ int)                                 {}
func (s *stubStoreBuilder) WithContext(_ context.Context)                               {}
func (s *stubStoreBuilder) WithKubeClient(_ clientset.Interface)                        {}
func (s *stubStoreBuilder) WithCustomResourceClients(_ map[string]interface{})          {}
func (s *stubStoreBuilder) WithUsingAPIServerCache(_ bool)                              {}
func (s *stubStoreBuilder) WithFamilyGeneratorFilter(_ generator.FamilyGeneratorFilter) {}
func (s *stubStoreBuilder) WithAllowAnnotations(_ map[string][]string) error            { return nil }
func (s *stubStoreBuilder) WithAllowLabels(_ map[string][]string) error                 { return nil }
func (s *stubStoreBuilder) WithGenerateStoresFunc(_ ksmtypes.BuildStoresFunc)           {}
func (s *stubStoreBuilder) DefaultGenerateStoresFunc() ksmtypes.BuildStoresFunc         { return nil }
func (s *stubStoreBuilder) DefaultGenerateCustomResourceStoresFunc() ksmtypes.BuildCustomResourceStoresFunc {
	return nil
}
func (s *stubStoreBuilder) WithCustomResourceStoreFactories(_ ...customresource.RegistryFactory) {
}
func (s *stubStoreBuilder) BuildStores() [][]cache.Store { return nil }
func (s *stubStoreBuilder) WithGenerateCustomResourceStoresFunc(_ ksmtypes.BuildCustomResourceStoresFunc) {
}

func (s *stubStoreBuilder) Build() metricsstore.MetricsWriterList {
	s.buildCount.Add(1)
	return s.buildWriters
}

func (s *stubStoreBuilder) WaitForStoresSync(_ context.Context, _ time.Duration) bool {
	s.syncCount.Add(1)
	if s.syncGate != nil {
		return <-s.syncGate
	}
	return s.syncOK.Load()
}

// waitForRebuildIdle blocks until no rebuild is in flight, so assertions do not
// race an in-progress swap.
func waitForRebuildIdle(t *testing.T, h *MetricsHandler) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		h.rebuildMu.Lock()
		running := h.rebuildRunning
		h.rebuildMu.Unlock()
		if !running {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("rebuild did not finish before deadline")
}

func TestBuildWriters_KeepsPreviousWritersWhenSyncFails(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters: metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("candidate")},
	}
	opts := options.NewOptions()
	h := New(opts, nil, stub, false)

	existing := metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("stable")}
	h.mtx.Lock()
	h.metricsWriters = existing
	h.mtx.Unlock()

	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	if stub.buildCount.Load() < 1 || stub.syncCount.Load() < 1 {
		t.Fatalf("expected a build and a sync attempt; buildCount=%d syncCount=%d",
			stub.buildCount.Load(), stub.syncCount.Load())
	}

	h.mtx.RLock()
	got := h.metricsWriters
	h.mtx.RUnlock()
	if len(got) != 1 || got[0].ResourceName != "stable" {
		t.Fatalf("expected stable writers to be kept after failed sync, got %+v", got)
	}
}

func TestBuildWriters_SwapsWritersWhenSyncSucceeds(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters: metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("next")},
	}
	stub.syncOK.Store(true)
	opts := options.NewOptions()
	h := New(opts, nil, stub, false)

	h.mtx.Lock()
	h.metricsWriters = metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("prev")}
	h.mtx.Unlock()

	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	h.mtx.RLock()
	got := h.metricsWriters
	h.mtx.RUnlock()
	if len(got) != 1 || got[0].ResourceName != "next" {
		t.Fatalf("expected writers to swap after successful sync, got %+v", got)
	}
}

func TestBuildWriters_RetriesWhenInitialSyncFails(t *testing.T) {
	restore := initialStoreSyncRetryDelay
	initialStoreSyncRetryDelay = time.Millisecond
	defer func() { initialStoreSyncRetryDelay = restore }()

	// Channel-gated sync: first attempt fails immediately; the retry blocks until
	// the test releases a successful result, so mid-retry assertions cannot race
	// the retry timer.
	syncGate := make(chan bool)
	stub := &stubStoreBuilder{
		buildWriters: metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("first")},
		syncGate:     syncGate,
	}
	h := New(options.NewOptions(), nil, stub, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	firstSyncDone := make(chan struct{})
	go func() {
		syncGate <- false
		close(firstSyncDone)
	}()
	// The first rebuild runs inline because no writers exist yet.
	h.BuildWriters(ctx)
	<-firstSyncDone

	// Writers stay empty until we release the retry, even if the retry timer has
	// already fired and is blocked in WaitForStoresSync.
	h.mtx.RLock()
	got := len(h.metricsWriters)
	h.mtx.RUnlock()
	if got != 0 {
		t.Fatalf("expected no writers after failed initial sync, got %d", got)
	}

	// Release the retry sync as success. Send in a goroutine so we do not block
	// if the retry has not entered WaitForStoresSync yet.
	go func() { syncGate <- true }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		h.mtx.RLock()
		writers := h.metricsWriters
		h.mtx.RUnlock()
		if len(writers) == 1 && writers[0].ResourceName == "first" {
			if stub.syncCount.Load() < 2 {
				t.Fatalf("expected retry sync attempt; syncCount=%d", stub.syncCount.Load())
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("expected retry to populate writers; buildCount=%d syncCount=%d",
		stub.buildCount.Load(), stub.syncCount.Load())
}

func TestServeHTTP_WriterSnapshot(t *testing.T) {
	h := New(options.NewOptions(), nil, &stubStoreBuilder{}, false)
	h.mtx.Lock()
	h.metricsWriters = metricsstore.MetricsWriterList{newTestWriter("first"), newTestWriter("second")}
	h.mtx.Unlock()

	// Replace the writer list while ServeHTTP is mid-response: the snapshot it
	// took must still be written out in full.
	rec := httptest.NewRecorder()
	body := &blockingWriter{
		ResponseWriter: rec,
		onFirstWrite: func() {
			h.mtx.Lock()
			h.metricsWriters = metricsstore.MetricsWriterList{newTestWriter("replaced")}
			h.mtx.Unlock()
		},
	}
	// ServeHTTP must not hold mtx across the response body, or onFirstWrite would
	// deadlock instead of failing.
	done := make(chan struct{})
	go func() {
		h.ServeHTTP(body, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ServeHTTP did not complete while writers were replaced")
	}

	out := rec.Body.String()
	for _, want := range []string{"kube_test_first", "kube_test_second"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in response, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "kube_test_replaced") {
		t.Fatalf("response used writers swapped in mid-request:\n%s", out)
	}
}

// blockingWriter runs onFirstWrite once, before the first byte of the response
// body is written, to simulate a rebuild landing during a scrape.
type blockingWriter struct {
	http.ResponseWriter
	onFirstWrite func()
	written      bool
}

func (b *blockingWriter) Write(p []byte) (int, error) {
	if !b.written {
		b.written = true
		b.onFirstWrite()
	}
	return b.ResponseWriter.Write(p)
}

func newTestWriter(name string) *metricsstore.MetricsWriter {
	genFunc := func(obj interface{}) []metric.FamilyInterface {
		o, err := meta.Accessor(obj)
		if err != nil {
			panic(err)
		}
		return []metric.FamilyInterface{&metric.Family{
			Name: "kube_test_" + name,
			Metrics: []*metric.Metric{{
				LabelKeys:   []string{"namespace"},
				LabelValues: []string{o.GetNamespace()},
				Value:       1,
			}},
		}}
	}
	store := metricsstore.NewMetricsStore([]string{"# HELP kube_test_" + name + " test\n"}, genFunc)
	if err := store.Add(&v1.Service{
		ObjectMeta: metav1.ObjectMeta{UID: types.UID(name), Name: name, Namespace: "ns"},
	}); err != nil {
		panic(err)
	}
	return metricsstore.NewMetricsWriter(name, store)
}
