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
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
	buildFn      func(n int64) metricsstore.MetricsWriterList
	syncErr      error
	buildCount   atomic.Int64
	syncCount    atomic.Int64
	// syncGate, when set, makes WaitForStoresSync block until a result is sent.
	syncGate chan error
	// honorTimeout returns ErrStoreSyncTimeout when timeout elapses, or immediately when timeout is not positive.
	honorTimeout bool
	// progressGate, when set, makes WaitForStoreProgress block until a result is sent.
	progressGate  chan error
	progressCount atomic.Int64
	// cancelWait unblocks syncGate when the generation context is done.
	cancelWait bool
	// syncStarted receives when WaitForStoresSync begins. Buffered by the test.
	syncStarted chan struct{}
	// buildGate, when set, makes Build block until a value is received.
	buildGate    chan struct{}
	buildEntered chan struct{}
	ctxMu        sync.Mutex
	buildCtx     context.Context
}

func rebuildOptions() *options.Options {
	opts := options.NewOptions()
	opts.StoreSyncTimeout = time.Minute
	return opts
}

func writerList(name string) metricsstore.MetricsWriterList {
	return metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter(name)}
}

func (s *stubStoreBuilder) WithMetrics(_ prometheus.Registerer)            {}
func (s *stubStoreBuilder) WithEnabledResources(_ []string) error          { return nil }
func (s *stubStoreBuilder) ReplaceEnabledCustomResources(_ []string) error { return nil }
func (s *stubStoreBuilder) WithNamespaces(_ options.NamespaceList)         {}
func (s *stubStoreBuilder) WithFieldSelectorFilter(_ string)               {}
func (s *stubStoreBuilder) WithSharding(_ int32, _ int)                    {}
func (s *stubStoreBuilder) WithContext(ctx context.Context) {
	s.ctxMu.Lock()
	s.buildCtx = ctx
	s.ctxMu.Unlock()
}
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
	n := s.buildCount.Add(1)
	if s.buildEntered != nil {
		select {
		case s.buildEntered <- struct{}{}:
		default:
		}
	}
	if s.buildGate != nil {
		<-s.buildGate
	}
	if s.buildFn != nil {
		return s.buildFn(n)
	}
	return s.buildWriters
}

func (s *stubStoreBuilder) WaitForStoresSync(ctx context.Context, timeout time.Duration) error {
	s.syncCount.Add(1)
	if s.syncStarted != nil {
		select {
		case s.syncStarted <- struct{}{}:
		default:
		}
	}
	if s.honorTimeout {
		if timeout <= 0 {
			return ksmtypes.ErrStoreSyncTimeout
		}
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return ksmtypes.ErrStoreSyncTimeout
		}
	}
	if s.syncGate != nil {
		if s.cancelWait {
			select {
			case err := <-s.syncGate:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return <-s.syncGate
	}
	return s.syncErr
}

func (s *stubStoreBuilder) WaitForStoreProgress(ctx context.Context) error {
	s.progressCount.Add(1)
	if s.progressGate != nil {
		select {
		case err := <-s.progressGate:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (s *stubStoreBuilder) generationContext() context.Context {
	s.ctxMu.Lock()
	defer s.ctxMu.Unlock()
	return s.buildCtx
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

func installedNames(h *MetricsHandler) []string {
	h.mtx.RLock()
	defer h.mtx.RUnlock()
	names := make([]string, len(h.metricsWriters))
	for i, w := range h.metricsWriters {
		names[i] = w.ResourceName
	}
	return names
}

func TestBuildWriters_InstallsWhenSyncFails(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters: writerList("candidate"),
		syncErr:      errors.New("sync failed"),
	}
	h := New(rebuildOptions(), nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = writerList("stable")
	h.writersInstalled = true
	h.mtx.Unlock()

	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	if stub.buildCount.Load() < 1 || stub.syncCount.Load() < 1 {
		t.Fatalf("expected a build and a sync attempt; buildCount=%d syncCount=%d",
			stub.buildCount.Load(), stub.syncCount.Load())
	}
	if got := installedNames(h); len(got) != 1 || got[0] != "candidate" {
		t.Fatalf("expected an unexpected sync error to install candidate writers, got %v", got)
	}
}

func TestBuildWriters_NegativeTimeoutKeepsPreviousWriters(t *testing.T) {
	stub := &stubStoreBuilder{buildWriters: writerList("candidate")}
	opts := rebuildOptions()
	opts.StoreSyncTimeout = -time.Second
	h := New(opts, nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = writerList("stable")
	h.writersInstalled = true
	h.mtx.Unlock()

	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	if got := installedNames(h); len(got) != 1 || got[0] != "stable" {
		t.Fatalf("expected a negative timeout to keep previous writers, got %v", got)
	}
}

func TestBuildWriters_SwapsWritersWhenSyncSucceeds(t *testing.T) {
	stub := &stubStoreBuilder{buildWriters: writerList("next")}
	h := New(rebuildOptions(), nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = writerList("prev")
	h.mtx.Unlock()

	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	if got := installedNames(h); len(got) != 1 || got[0] != "next" {
		t.Fatalf("expected writers to swap after successful sync, got %v", got)
	}
	if !h.Ready() {
		t.Fatal("expected handler to report ready after successful sync")
	}
}

func TestBuildWriters_ReadyAfterEmptyGenerationSyncs(t *testing.T) {
	stub := &stubStoreBuilder{buildWriters: metricsstore.MetricsWriterList{}}
	h := New(rebuildOptions(), nil, stub, false)

	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	if !h.Ready() {
		t.Fatal("expected handler to report ready after a successful sync with zero writers")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected empty installed generation to return 200, got %d", rec.Code)
	}
}

func TestServeHTTP_NotInstalled(t *testing.T) {
	h := New(rebuildOptions(), nil, &stubStoreBuilder{}, false)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 before a generation is installed, got %d", rec.Code)
	}
}

func TestBuildWriters_InstallsOnSyncTimeout(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters: writerList("partial"),
		syncErr:      ksmtypes.ErrStoreSyncTimeout,
	}
	h := New(rebuildOptions(), nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = writerList("stable")
	h.writersInstalled = true
	h.mtx.Unlock()

	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	if got := installedNames(h); len(got) != 1 || got[0] != "partial" {
		t.Fatalf("expected bounded sync to install replacement writers, got %v", got)
	}
	if !h.Ready() {
		t.Fatal("expected handler to be ready after bounded sync")
	}
}

func TestBuildWriters_ParentDeadlineKeepsPreviousWriters(t *testing.T) {
	syncStarted := make(chan struct{}, 1)
	syncGate := make(chan error)
	stub := &stubStoreBuilder{
		buildWriters: writerList("candidate"),
		syncGate:     syncGate,
		syncStarted:  syncStarted,
	}
	h := New(rebuildOptions(), nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = writerList("stable")
	h.writersInstalled = true
	h.mtx.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	h.BuildWriters(ctx)
	<-syncStarted
	<-ctx.Done()
	// The wait itself succeeds. Installation still has to observe the expired
	// generation context and keep the previous writers.
	syncGate <- nil
	waitForRebuildIdle(t, h)

	if got := installedNames(h); len(got) != 1 || got[0] != "stable" {
		t.Fatalf("expected parent deadline to keep previous writers, got %v", got)
	}
}

func TestBuildWriters_ContextDeadlineIsNotSyncTimeout(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters: writerList("candidate"),
		syncErr:      context.DeadlineExceeded,
	}
	h := New(rebuildOptions(), nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = writerList("stable")
	h.writersInstalled = true
	h.mtx.Unlock()

	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	if got := installedNames(h); len(got) != 1 || got[0] != "stable" {
		t.Fatalf("expected a generation deadline to keep previous writers, got %v", got)
	}
}

func TestBuildWriters_CancellationKeepsPreviousWriters(t *testing.T) {
	syncStarted := make(chan struct{}, 1)
	stub := &stubStoreBuilder{
		buildWriters: writerList("candidate"),
		syncGate:     make(chan error),
		cancelWait:   true,
		syncStarted:  syncStarted,
	}
	h := New(rebuildOptions(), nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = writerList("stable")
	h.writersInstalled = true
	h.mtx.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	h.BuildWriters(ctx)
	<-syncStarted
	cancel()
	waitForRebuildIdle(t, h)

	if got := installedNames(h); len(got) != 1 || got[0] != "stable" {
		t.Fatalf("expected cancellation to keep previous writers, got %v", got)
	}
	if stub.syncCount.Load() != 1 {
		t.Fatalf("expected cancellation not to retry, syncCount=%d", stub.syncCount.Load())
	}
}

func TestBuildWriters_ZeroTimeoutInstallsWithoutWaiting(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters: writerList("immediate"),
		syncErr:      errors.New("wait should not be called"),
	}
	opts := options.NewOptions()
	h := New(opts, nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = writerList("stable")
	h.mtx.Unlock()

	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	if stub.syncCount.Load() != 0 {
		t.Fatalf("expected zero timeout to skip the wait, syncCount=%d", stub.syncCount.Load())
	}
	if got := installedNames(h); len(got) != 1 || got[0] != "immediate" {
		t.Fatalf("expected zero timeout to install writers, got %v", got)
	}
}

func TestBuildWriters_ZeroTimeoutCanceledContextKeepsPrevious(t *testing.T) {
	stub := &stubStoreBuilder{buildWriters: writerList("immediate")}
	opts := options.NewOptions()
	h := New(opts, nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = writerList("stable")
	h.writersInstalled = true
	h.mtx.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h.BuildWriters(ctx)
	waitForRebuildIdle(t, h)

	if stub.syncCount.Load() != 0 {
		t.Fatalf("expected canceled zero-timeout rebuild to skip the wait, syncCount=%d", stub.syncCount.Load())
	}
	if got := installedNames(h); len(got) != 1 || got[0] != "stable" {
		t.Fatalf("expected canceled zero-timeout rebuild to keep previous writers, got %v", got)
	}
}

func TestBuildWriters_SupersededDuringBuild(t *testing.T) {
	buildEntered := make(chan struct{}, 1)
	buildGate := make(chan struct{})
	stub := &stubStoreBuilder{
		buildEntered: buildEntered,
		buildGate:    buildGate,
		buildFn: func(n int64) metricsstore.MetricsWriterList {
			if n == 1 {
				return writerList("first")
			}
			return writerList("second")
		},
	}
	h := New(rebuildOptions(), nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = writerList("stable")
	h.writersInstalled = true
	h.mtx.Unlock()

	h.BuildWriters(context.Background())
	<-buildEntered

	h.rebuildMu.Lock()
	registered := h.rebuildRunning && h.activeCancel != nil
	h.rebuildMu.Unlock()
	if !registered {
		t.Fatal("expected the generation cancel to be registered once the rebuild is running")
	}

	h.BuildWriters(context.Background())
	if err := stub.generationContext().Err(); err == nil {
		t.Fatal("expected the second request to cancel the generation that is inside Build")
	}
	close(buildGate)
	waitForRebuildIdle(t, h)

	if got := installedNames(h); len(got) != 1 || got[0] != "second" {
		t.Fatalf("expected the superseded generation to be dropped, got %v", got)
	}
}

func TestBuildWriters_SupersededDuringWait(t *testing.T) {
	syncStarted := make(chan struct{}, 2)
	syncGate := make(chan error)
	stub := &stubStoreBuilder{
		syncGate:    syncGate,
		cancelWait:  true,
		syncStarted: syncStarted,
		buildFn: func(n int64) metricsstore.MetricsWriterList {
			if n == 1 {
				return writerList("first")
			}
			return writerList("second")
		},
	}
	h := New(rebuildOptions(), nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = writerList("stable")
	h.writersInstalled = true
	h.mtx.Unlock()

	h.BuildWriters(context.Background())
	<-syncStarted
	h.BuildWriters(context.Background())
	<-syncStarted
	syncGate <- nil
	waitForRebuildIdle(t, h)

	if got := installedNames(h); len(got) != 1 || got[0] != "second" {
		t.Fatalf("expected the request during the wait to replace the generation, got %v", got)
	}
}

func TestBuildWriters_SupersededBeforeInstall(t *testing.T) {
	stub := &stubStoreBuilder{
		buildFn: func(n int64) metricsstore.MetricsWriterList {
			if n == 1 {
				return writerList("first")
			}
			return writerList("second")
		},
	}
	h := New(rebuildOptions(), nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = writerList("stable")
	h.writersInstalled = true
	h.mtx.Unlock()
	h.beforeSwap = func() {
		h.beforeSwap = nil
		h.BuildWriters(context.Background())
	}

	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	if got := installedNames(h); len(got) != 1 || got[0] != "second" {
		t.Fatalf("expected a request before installation to drop the finished wait, got %v", got)
	}
}

func TestServeHTTP_WriterSnapshot(t *testing.T) {
	h := New(options.NewOptions(), nil, &stubStoreBuilder{}, false)
	h.mtx.Lock()
	h.metricsWriters = metricsstore.MetricsWriterList{newTestWriter("first"), newTestWriter("second")}
	h.writersInstalled = true
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

func TestConfigureStore_AppliesConfigUnderLockThenRebuilds(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters: metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("configured")},
	}
	h := New(rebuildOptions(), nil, stub, false)

	h.mtx.Lock()
	h.metricsWriters = metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("prev")}
	h.mtx.Unlock()

	var configured atomic.Bool
	h.ConfigureStore(context.Background(), func(b ksmtypes.BuilderInterface) error {
		if b != stub {
			t.Fatalf("expected ConfigureStore to pass the handler store builder")
		}
		configured.Store(true)
		return nil
	})
	waitForRebuildIdle(t, h)

	if !configured.Load() {
		t.Fatal("expected ConfigureStore callback to run")
	}
	if stub.buildCount.Load() < 1 {
		t.Fatalf("expected rebuild after ConfigureStore; buildCount=%d", stub.buildCount.Load())
	}

	h.mtx.RLock()
	got := h.metricsWriters
	h.mtx.RUnlock()
	if len(got) != 1 || got[0].ResourceName != "configured" {
		t.Fatalf("expected writers swapped after ConfigureStore rebuild, got %+v", got)
	}
}

func TestConfigureStore_CoalescesWithInFlightRebuild(t *testing.T) {
	syncStarted := make(chan struct{}, 2)
	syncGate := make(chan error)
	stub := &stubStoreBuilder{
		buildWriters: writerList("next"),
		syncGate:     syncGate,
		cancelWait:   true,
		syncStarted:  syncStarted,
	}
	h := New(rebuildOptions(), nil, stub, false)

	h.mtx.Lock()
	h.metricsWriters = writerList("prev")
	h.mtx.Unlock()

	h.BuildWriters(context.Background())
	<-syncStarted

	var configDuringWait atomic.Int64
	h.ConfigureStore(context.Background(), func(ksmtypes.BuilderInterface) error {
		configDuringWait.Add(1)
		return nil
	})

	<-syncStarted
	syncGate <- nil
	waitForRebuildIdle(t, h)

	if configDuringWait.Load() != 1 {
		t.Fatalf("expected ConfigureStore callback once, got %d", configDuringWait.Load())
	}
	if stub.buildCount.Load() < 2 {
		t.Fatalf("expected coalesced rebuild after ConfigureStore during wait; buildCount=%d", stub.buildCount.Load())
	}
	if got := installedNames(h); len(got) != 1 || got[0] != "next" {
		t.Fatalf("expected coalesced rebuild to install writers, got %v", got)
	}
}

func TestConfigureStore_SkipsRebuildOnConfigError(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters: metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("next")},
	}
	h := New(rebuildOptions(), nil, stub, false)

	err := h.ConfigureStore(context.Background(), func(ksmtypes.BuilderInterface) error {
		return fmt.Errorf("config failed")
	})
	if err == nil {
		t.Fatal("expected ConfigureStore to return configuration error")
	}
	if stub.buildCount.Load() != 0 {
		t.Fatalf("expected no rebuild after configuration error; buildCount=%d", stub.buildCount.Load())
	}
}

func TestBuildWriters_RequestBeforeDeadlineInstallsReplacement(t *testing.T) {
	buildGate := make(chan struct{}, 1)
	buildGate <- struct{}{}
	buildEntered := make(chan struct{}, 1)
	syncStarted := make(chan struct{}, 1)
	stub := &stubStoreBuilder{
		honorTimeout: true,
		buildGate:    buildGate,
		buildEntered: buildEntered,
		syncStarted:  syncStarted,
		buildFn: func(n int64) metricsstore.MetricsWriterList {
			return writerList(fmt.Sprintf("gen-%d", n))
		},
	}
	opts := rebuildOptions()
	opts.StoreSyncTimeout = time.Second
	h := New(opts, nil, stub, false)

	h.BuildWriters(context.Background())
	select {
	case <-syncStarted:
	case <-time.After(time.Second):
		t.Fatal("first generation did not start its sync wait")
	}
	select {
	case <-buildEntered:
	default:
	}
	h.rebuildMu.Lock()
	h.syncDeadline = time.Now().Add(time.Second)
	h.rebuildMu.Unlock()
	h.BuildWriters(context.Background())
	select {
	case <-buildEntered:
	case <-time.After(time.Second):
		t.Fatal("replacement generation did not start")
	}
	h.rebuildMu.Lock()
	h.syncDeadline = time.Now().Add(-time.Millisecond)
	h.rebuildMu.Unlock()
	buildGate <- struct{}{}

	deadline := time.Now().Add(time.Second)
	for {
		got := installedNames(h)
		if len(got) == 1 && got[0] == "gen-2" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected the replacement generation to install through the timeout fallback, got %v", got)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if stub.syncCount.Load() != 1 {
		t.Fatalf("expected the exhausted replacement to skip polling, syncCount=%d", stub.syncCount.Load())
	}
}

func TestBuildWriters_SustainedRequestsInstallWithinBudget(t *testing.T) {
	const timeout = 150 * time.Millisecond
	var latest atomic.Int64
	stub := &stubStoreBuilder{
		honorTimeout: true,
		buildFn: func(int64) metricsstore.MetricsWriterList {
			return writerList(fmt.Sprintf("gen-%d", latest.Load()))
		},
	}
	opts := rebuildOptions()
	opts.StoreSyncTimeout = timeout
	h := New(opts, nil, stub, false)

	start := time.Now()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var i int64
		for {
			select {
			case <-stop:
				return
			default:
			}
			i++
			latest.Store(i)
			h.BuildWriters(context.Background())
			select {
			case <-stop:
				return
			case <-time.After(timeout / 8):
			}
		}
	}()

	progressDeadline := start.Add(timeout + 250*time.Millisecond)
	for !h.Ready() {
		if time.Now().After(progressDeadline) {
			close(stop)
			wg.Wait()
			t.Fatal("handler did not install a generation while requests were still arriving")
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(stop)
	wg.Wait()

	want := fmt.Sprintf("gen-%d", latest.Load())
	settleDeadline := time.Now().Add(timeout + 500*time.Millisecond)
	for {
		got := installedNames(h)
		if len(got) == 1 && got[0] == want {
			return
		}
		if time.Now().After(settleDeadline) {
			t.Fatalf("latest pending configuration %s did not install, got %v", want, got)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestBuildWriters_ExhaustedBudgetInstallsOnFirstStart(t *testing.T) {
	var h *MetricsHandler
	stub := &stubStoreBuilder{
		buildFn: func(int64) metricsstore.MetricsWriterList {
			h.rebuildMu.Lock()
			h.syncDeadline = time.Now().Add(-time.Millisecond)
			h.rebuildMu.Unlock()
			return writerList("first")
		},
	}
	opts := rebuildOptions()
	h = New(opts, nil, stub, false)

	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	if !h.Ready() {
		t.Fatal("expected the first generation to install when the budget is already exhausted")
	}
	if got := installedNames(h); len(got) != 1 || got[0] != "first" {
		t.Fatalf("expected the startup generation to install, got %v", got)
	}
	if stub.progressCount.Load() != 0 {
		t.Fatalf("expected startup to skip the progress wait, progressCount=%d", stub.progressCount.Load())
	}
}

func TestBuildWriters_ExhaustedBudgetKeepsServingUntilProgress(t *testing.T) {
	progressGate := make(chan error)
	var h *MetricsHandler
	stub := &stubStoreBuilder{
		progressGate: progressGate,
		buildFn: func(int64) metricsstore.MetricsWriterList {
			h.rebuildMu.Lock()
			h.syncDeadline = time.Now().Add(-time.Millisecond)
			h.rebuildMu.Unlock()
			return writerList("next")
		},
	}
	opts := rebuildOptions()
	h = New(opts, nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = writerList("serving")
	h.writersInstalled = true
	h.mtx.Unlock()

	h.BuildWriters(context.Background())
	deadline := time.Now().Add(time.Second)
	for stub.progressCount.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("expected the replacement to wait for list progress")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := installedNames(h); len(got) != 1 || got[0] != "serving" {
		t.Fatalf("expected the serving generation to stay in place, got %v", got)
	}
	progressGate <- nil
	waitForRebuildIdle(t, h)
	if got := installedNames(h); len(got) != 1 || got[0] != "next" {
		t.Fatalf("expected the candidate to install after list progress, got %v", got)
	}
}

func TestBuildWriters_PendingConfigSkipsStalledProgressWait(t *testing.T) {
	var h *MetricsHandler
	var calls atomic.Int64
	stub := &stubStoreBuilder{
		buildFn: func(int64) metricsstore.MetricsWriterList {
			if calls.Add(1) == 1 {
				h.rebuildMu.Lock()
				h.syncDeadline = time.Now().Add(-time.Millisecond)
				h.pendingRebuild = true
				h.rebuildMu.Unlock()
				return writerList("stalled")
			}
			return writerList("latest")
		},
	}
	h = New(rebuildOptions(), nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = writerList("serving")
	h.writersInstalled = true
	h.mtx.Unlock()

	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	if got := installedNames(h); len(got) != 1 || got[0] != "latest" {
		t.Fatalf("expected the pending configuration to install, got %v", got)
	}
}

func TestBuildWriters_ZeroTimeoutBurstStillInstalls(t *testing.T) {
	const generations = 4
	var latest atomic.Int64
	entered := make(chan int64, generations)
	release := make(chan struct{})
	stop, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	stub := &stubStoreBuilder{
		buildFn: func(n int64) metricsstore.MetricsWriterList {
			entered <- n
			select {
			case <-release:
			case <-stop.Done():
				return writerList("stopped")
			}
			return writerList(fmt.Sprintf("gen-%d", latest.Load()))
		},
	}
	h := New(options.NewOptions(), nil, stub, false)
	// beforeSwap runs after Build releases the handler lock and before this generation installs.
	// From the second build on, it shows whether the overlapped generation was kept.
	var swaps atomic.Int64
	var installedDuringOverlap atomic.Value
	h.beforeSwap = func() {
		if swaps.Add(1) < 2 {
			return
		}
		if names := installedNames(h); len(names) == 1 {
			installedDuringOverlap.Store(names[0])
		}
	}

	latest.Store(1)
	h.BuildWriters(context.Background())
	if got := <-entered; got != 1 {
		t.Fatalf("expected build 1 to start, got %d", got)
	}

	for next := int64(2); next <= generations; next++ {
		latest.Store(next)
		h.BuildWriters(context.Background())
		release <- struct{}{}
		if got := <-entered; got != next {
			t.Fatalf("expected build %d to start, got %d", next, got)
		}
	}
	if installedDuringOverlap.Load() == nil {
		t.Fatal("every overlapped zero-timeout generation was discarded")
	}

	release <- struct{}{}
	waitForRebuildIdle(t, h)
	if got := installedNames(h); len(got) != 1 || got[0] != fmt.Sprintf("gen-%d", generations) {
		t.Fatalf("expected the latest configuration gen-%d to install, got %v", generations, got)
	}
}

func TestBuildWriters_ReloadDoesNotRaceCanceledRebuild(t *testing.T) {
	opts := options.NewOptions()
	opts.StoreSyncTimeout = time.Second
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	stub := &stubStoreBuilder{
		buildEntered: entered,
		buildGate:    release,
	}
	h := New(opts, nil, stub, false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.BuildWriters(ctx)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("build did not start")
	}
	cancel()

	var writes atomic.Int64
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				opts.StoreSyncTimeout = time.Duration(writes.Add(1))
			}
		}
	}()
	for writes.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	close(release)
	waitForRebuildIdle(t, h)
	close(stop)
	wg.Wait()
	if h.Ready() {
		t.Fatal("canceled rebuild installed writers")
	}
}
