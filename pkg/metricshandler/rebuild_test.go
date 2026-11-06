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
	// gateOnlyFirstSync limits syncGate to the first WaitForStoresSync call (0 = every call).
	gateOnlyFirstSync int64
	// honorTimeout returns ErrStoreSyncTimeout when timeout elapses, or immediately when timeout is not positive.
	honorTimeout bool
	// syncTimeoutsBeforeOK returns ErrStoreSyncTimeout for the first N WaitForStoresSync calls, then nil.
	syncTimeoutsBeforeOK int64
	syncTimeoutsMu       sync.Mutex
	syncTimeouts         []time.Duration
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
	opts.StoreSyncTimeout = 50 * time.Millisecond
	return opts
}

func newTestHandler(t *testing.T, opts *options.Options, stub ksmtypes.BuilderInterface) *MetricsHandler {
	t.Helper()
	return New(opts, nil, stub, false)
}

func writerList(name string) metricsstore.MetricsWriterList {
	return metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter(name)}
}

func seedServingWriters(h *MetricsHandler, writers metricsstore.MetricsWriterList) {
	h.mtx.Lock()
	h.metricsWriters = writers
	h.mtx.Unlock()
	h.writersReady.Store(true)
}

func goBuildWriters(h *MetricsHandler, ctx context.Context) {
	go h.BuildWriters(ctx)
}

func waitForInstalledWriter(t *testing.T, h *MetricsHandler, name string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		got := installedNames(h)
		if len(got) == 1 && got[0] == name {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for writer %q, got %v", name, installedNames(h))
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
	s.syncTimeoutsMu.Lock()
	s.syncTimeouts = append(s.syncTimeouts, timeout)
	s.syncTimeoutsMu.Unlock()
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
	if s.syncTimeoutsBeforeOK > 0 {
		if s.syncCount.Load() <= s.syncTimeoutsBeforeOK {
			return ksmtypes.ErrStoreSyncTimeout
		}
		return nil
	}
	if s.syncGate != nil && (s.gateOnlyFirstSync == 0 || s.syncCount.Load() <= s.gateOnlyFirstSync) {
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

func (s *stubStoreBuilder) syncTimeoutCalls() []time.Duration {
	s.syncTimeoutsMu.Lock()
	defer s.syncTimeoutsMu.Unlock()
	return append([]time.Duration(nil), s.syncTimeouts...)
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
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if h.workerQuiescent() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("rebuild worker did not become quiescent")
}

func activeCancelRegistered(h *MetricsHandler) bool {
	h.rebuildMu.Lock()
	defer h.rebuildMu.Unlock()
	return h.activeCancel != nil
}

func activeCancelAndCandidateGen(h *MetricsHandler) (cancel context.CancelFunc, gen uint64) {
	h.rebuildMu.Lock()
	defer h.rebuildMu.Unlock()
	return h.activeCancel, h.activeCandidateGen
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
	h := newTestHandler(t, rebuildOptions(), stub)
	seedServingWriters(h, writerList("stable"))

	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	if stub.buildCount.Load() < 1 || stub.syncCount.Load() < 1 {
		t.Fatalf("expected a build and a sync attempt; buildCount=%d syncCount=%d",
			stub.buildCount.Load(), stub.syncCount.Load())
	}
	if got := installedNames(h); len(got) != 1 || got[0] != "stable" {
		t.Fatalf("expected an unexpected sync error to keep previous writers, got %v", got)
	}
}

func TestBuildWriters_NegativeTimeoutRejectsNew(t *testing.T) {
	opts := rebuildOptions()
	opts.StoreSyncTimeout = -time.Second
	stub := &stubStoreBuilder{buildWriters: writerList("next")}
	_, err := NewValidated(opts, nil, stub, false)
	if err == nil {
		t.Fatal("expected negative store sync timeout to be rejected at construction")
	}
}

func TestNew_PanicsOnNegativeStoreSyncTimeout(t *testing.T) {
	opts := rebuildOptions()
	opts.StoreSyncTimeout = -time.Second
	defer func() {
		if recover() == nil {
			t.Fatal("expected New to panic on negative store sync timeout")
		}
	}()
	_ = New(opts, nil, &stubStoreBuilder{}, false)
}

func TestNew_NilOptionsUsesIndependentDefaults(t *testing.T) {
	stub := &stubStoreBuilder{buildWriters: writerList("w")}
	h1 := New(nil, nil, stub, false)
	h2 := New(nil, nil, stub, false)
	if h1.opts == h2.opts {
		t.Fatal("expected separate options instances for nil args")
	}
	if h1.storeSyncTimeout != options.DefaultStoreSyncTimeout || h2.storeSyncTimeout != options.DefaultStoreSyncTimeout {
		t.Fatalf("expected default store sync timeout %s, got %s and %s", options.DefaultStoreSyncTimeout, h1.storeSyncTimeout, h2.storeSyncTimeout)
	}
}

func TestNewValidated_NilOptionsOK(t *testing.T) {
	stub := &stubStoreBuilder{buildWriters: writerList("w")}
	h, err := NewValidated(nil, nil, stub, false)
	if err != nil {
		t.Fatalf("NewValidated(nil): %v", err)
	}
	if h.opts == nil || h.storeSyncTimeout != options.DefaultStoreSyncTimeout {
		t.Fatalf("unexpected handler: opts=%v timeout=%s", h.opts, h.storeSyncTimeout)
	}
}

func TestNew_ZeroOptionsStructKeepsExplicitZeroTimeout(t *testing.T) {
	opts := &options.Options{}
	if opts.StoreSyncTimeout != 0 {
		t.Fatalf("precondition: zero timeout, got %s", opts.StoreSyncTimeout)
	}
	stub := &stubStoreBuilder{buildWriters: writerList("w")}
	h := New(opts, nil, stub, false)
	if h.opts != opts {
		t.Fatal("expected handler to keep caller-supplied options pointer")
	}
	if h.storeSyncTimeout != 0 {
		t.Fatalf("expected explicit 0 timeout, got %s", h.storeSyncTimeout)
	}
}

func TestBuildWriters_SwapsWritersWhenSyncSucceeds(t *testing.T) {
	stub := &stubStoreBuilder{buildWriters: writerList("next")}
	h := newTestHandler(t, rebuildOptions(), stub)
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
	h := newTestHandler(t, rebuildOptions(), stub)

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
	h := newTestHandler(t, rebuildOptions(), &stubStoreBuilder{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 before a generation is installed, got %d", rec.Code)
	}
}

func TestBuildWriters_WarmSyncTimeoutRetriesUntilSuccess(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters:         writerList("updated"),
		syncTimeoutsBeforeOK: 2,
	}
	h := newTestHandler(t, rebuildOptions(), stub)
	seedServingWriters(h, writerList("stable"))

	h.BuildWriters(context.Background())
	waitForInstalledWriter(t, h, "updated")
	waitForRebuildIdle(t, h)
	if stub.syncCount.Load() < 3 {
		t.Fatalf("expected warm sync timeout to retry, syncCount=%d", stub.syncCount.Load())
	}
}

func TestBuildWriters_WarmSyncTimeoutDoesNotInstallPartial(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters: writerList("partial"),
		syncErr:      ksmtypes.ErrStoreSyncTimeout,
	}
	h := newTestHandler(t, rebuildOptions(), stub)
	seedServingWriters(h, writerList("stable"))

	ctx, cancel := context.WithCancel(context.Background())
	h.BuildWriters(ctx)
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if got := installedNames(h); len(got) == 1 && got[0] == "partial" {
			t.Fatal("warm sync timeout must not install partial writers")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	waitForRebuildIdle(t, h)

	if got := installedNames(h); len(got) != 1 || got[0] != "stable" {
		t.Fatalf("expected warm sync timeout to keep previous writers, got %v", got)
	}
	if stub.syncCount.Load() < 2 {
		t.Fatalf("expected warm sync to keep waiting, syncCount=%d", stub.syncCount.Load())
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
	h := newTestHandler(t, rebuildOptions(), stub)
	seedServingWriters(h, writerList("stable"))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	goBuildWriters(h, ctx)
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
	h := newTestHandler(t, rebuildOptions(), stub)
	seedServingWriters(h, writerList("stable"))

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
	h := newTestHandler(t, rebuildOptions(), stub)
	seedServingWriters(h, writerList("stable"))

	ctx, cancel := context.WithCancel(context.Background())
	goBuildWriters(h, ctx)
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
	opts.StoreSyncTimeout = 0
	h := newTestHandler(t, opts, stub)
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
	opts.StoreSyncTimeout = 0
	h := newTestHandler(t, opts, stub)
	seedServingWriters(h, writerList("stable"))

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

func TestPublishedGenerationNotRegisteredAsActiveCandidateCancel(t *testing.T) {
	stub := &stubStoreBuilder{buildWriters: writerList("installed")}
	h := newTestHandler(t, rebuildOptions(), stub)

	h.BuildWriters(context.Background())
	waitForInstalledWriter(t, h, "installed")
	waitForRebuildIdle(t, h)

	cancel, gen := activeCancelAndCandidateGen(h)
	if cancel != nil || gen != 0 {
		t.Fatalf("activeCancel must be cleared after publication (serving gen uses m.cancel only), got cancel=%v gen=%d", cancel != nil, gen)
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
	h := newTestHandler(t, rebuildOptions(), stub)

	goBuildWriters(h, context.Background())
	<-buildEntered

	if !h.candidateActive.Load() || !activeCancelRegistered(h) {
		t.Fatal("expected the generation cancel to be registered once the rebuild is running")
	}

	go h.BuildWriters(context.Background())
	close(buildGate)
	waitForInstalledWriter(t, h, "second")
}

func TestBuildWriters_SupersededDuringWait(t *testing.T) {
	syncStarted := make(chan struct{}, 2)
	syncGate := make(chan error)
	stub := &stubStoreBuilder{
		syncGate:          syncGate,
		cancelWait:        true,
		syncStarted:       syncStarted,
		gateOnlyFirstSync: 0,
		buildFn: func(n int64) metricsstore.MetricsWriterList {
			if n == 1 {
				return writerList("first")
			}
			return writerList("second")
		},
	}
	h := newTestHandler(t, rebuildOptions(), stub)
	seedServingWriters(h, writerList("stable"))

	goBuildWriters(h, context.Background())
	<-syncStarted
	go h.BuildWriters(context.Background())
	<-syncStarted
	syncGate <- nil
	waitForInstalledWriter(t, h, "second")
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
	h := newTestHandler(t, rebuildOptions(), stub)
	seedServingWriters(h, writerList("stable"))
	h.beforeSwap = func() {
		h.beforeSwap = nil
		// Use async rebuild so the second revision is registered before the first install path proceeds.
		h.BuildWritersAsync(context.Background())
	}

	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	if got := installedNames(h); len(got) != 1 || got[0] != "second" {
		t.Fatalf("expected a request before installation to drop the finished wait, got %v", got)
	}
}

func TestBuildWriters_SupersededRevisionSkipsSyncWait(t *testing.T) {
	buildRelease := make(chan struct{})
	buildEntered := make(chan struct{}, 1)
	syncStarted := make(chan struct{}, 1)
	syncGate := make(chan error)
	stub := &stubStoreBuilder{
		buildEntered: buildEntered,
		buildGate:    buildRelease,
		syncGate:     syncGate,
		cancelWait:   true,
		syncStarted:  syncStarted,
		buildWriters: writerList("latest"),
	}
	h := newTestHandler(t, rebuildOptions(), stub)
	seedServingWriters(h, writerList("stable"))

	h.BuildWritersAsync(context.Background())
	h.BuildWritersAsync(context.Background())
	<-buildEntered
	if stub.buildCount.Load() != 1 {
		t.Fatalf("expected exactly one build after coalescing stale revision, buildCount=%d", stub.buildCount.Load())
	}
	if stub.syncCount.Load() != 0 {
		t.Fatalf("expected build to block before store sync, syncCount=%d", stub.syncCount.Load())
	}
	close(buildRelease)
	<-syncStarted
	syncGate <- nil
	waitForInstalledWriter(t, h, "latest")
	if stub.syncCount.Load() != 1 {
		t.Fatalf("expected exactly one store sync (latest revision only), got %d", stub.syncCount.Load())
	}
}

func TestServeHTTP_WriterSnapshot(t *testing.T) {
	h := newTestHandler(t, options.NewOptions(), &stubStoreBuilder{})
	seedServingWriters(h, metricsstore.MetricsWriterList{newTestWriter("first"), newTestWriter("second")})

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
	h := newTestHandler(t, rebuildOptions(), stub)

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
	h := newTestHandler(t, rebuildOptions(), stub)

	h.mtx.Lock()
	h.metricsWriters = writerList("prev")
	h.mtx.Unlock()

	goBuildWriters(h, context.Background())
	<-syncStarted

	var configDuringWait atomic.Int64
	done := make(chan struct{})
	go func() {
		err := h.ConfigureStore(context.Background(), func(ksmtypes.BuilderInterface) error {
			configDuringWait.Add(1)
			return nil
		})
		if err != nil {
			t.Error(err)
		}
		close(done)
	}()

	<-syncStarted
	syncGate <- nil
	<-done
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
	h := newTestHandler(t, rebuildOptions(), stub)

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
	syncStarted := make(chan struct{}, 2)
	syncGate := make(chan error)
	stub := &stubStoreBuilder{
		syncStarted: syncStarted,
		syncGate:    syncGate,
		cancelWait:  true,
		buildFn: func(n int64) metricsstore.MetricsWriterList {
			return writerList(fmt.Sprintf("gen-%d", n))
		},
	}
	opts := rebuildOptions()
	opts.StoreSyncTimeout = time.Second
	h := newTestHandler(t, opts, stub)

	goBuildWriters(h, context.Background())
	<-syncStarted
	go h.BuildWriters(context.Background())
	<-syncStarted
	syncGate <- nil
	waitForInstalledWriter(t, h, "gen-2")
}

func TestBuildWriters_SustainedRequestsInstallWithinBudget(t *testing.T) {
	const timeout = 150 * time.Millisecond
	var latest atomic.Int64
	stub := &stubStoreBuilder{
		buildFn: func(int64) metricsstore.MetricsWriterList {
			return writerList(fmt.Sprintf("gen-%d", latest.Load()))
		},
	}
	opts := rebuildOptions()
	opts.StoreSyncTimeout = timeout
	h := newTestHandler(t, opts, stub)

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
			h.BuildWritersAsync(context.Background())
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

func TestBuildWriters_StartupTimeoutReturnsUnreadyThenRetries(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters:         writerList("first"),
		syncTimeoutsBeforeOK: 1,
	}
	opts := rebuildOptions()
	opts.StoreSyncTimeout = 50 * time.Millisecond
	h := newTestHandler(t, opts, stub)

	h.BuildWriters(context.Background())
	if h.Ready() {
		t.Fatal("expected startup to remain unready after the first timed-out attempt")
	}
	waitForInstalledWriter(t, h, "first")
}

func TestBuildWriters_WarmPendingInstallsBeforeCoalescedRebuild(t *testing.T) {
	const syncTimeout = 500 * time.Millisecond
	syncGate := make(chan error)
	syncStarted := make(chan struct{}, 2)
	opts := rebuildOptions()
	opts.StoreSyncTimeout = syncTimeout
	stub := &stubStoreBuilder{
		syncGate:          syncGate,
		cancelWait:        true,
		syncStarted:       syncStarted,
		gateOnlyFirstSync: 0,
		buildFn: func(n int64) metricsstore.MetricsWriterList {
			return writerList(fmt.Sprintf("gen-%d", n))
		},
	}
	h := newTestHandler(t, opts, stub)
	seedServingWriters(h, writerList("serving"))

	goBuildWriters(h, context.Background())
	<-syncStarted
	go h.BuildWriters(context.Background())
	<-syncStarted
	syncGate <- nil
	waitForInstalledWriter(t, h, "gen-2")

	calls := stub.syncTimeoutCalls()
	if len(calls) < 2 {
		t.Fatalf("expected at least two sync waits, got %d timeouts %v", len(calls), calls)
	}
	for i, got := range calls {
		if got != syncTimeout {
			t.Fatalf("sync wait %d: expected full timeout %s, got %s", i+1, syncTimeout, got)
		}
	}
}

func TestInstallWriters_SkipsSwapWhenCanceledWhileWaitingForMtx(t *testing.T) {
	stub := &stubStoreBuilder{buildWriters: writerList("candidate")}
	h := newTestHandler(t, rebuildOptions(), stub)

	_, servingCancel := context.WithCancel(context.Background())
	var servingCancelCalled atomic.Bool
	h.mtx.Lock()
	h.metricsWriters = writerList("serving")
	h.cancel = func() {
		servingCancelCalled.Store(true)
		servingCancel()
	}
	h.mtx.Unlock()
	h.writersReady.Store(true)

	ready := make(chan struct{})
	releaseInstall := make(chan struct{})
	var candidateCancelSpy atomic.Int64
	h.beforeInstallWriters = func() {
		h.mtx.Lock()
		close(ready)
		<-releaseInstall
		h.mtx.Unlock()
	}
	h.onCandidateCancel = func() {
		candidateCancelSpy.Add(1)
	}

	ctx, cancelBuild := context.WithCancel(context.Background())
	defer cancelBuild()
	goBuildWriters(h, ctx)
	<-ready
	cancelBuild()
	close(releaseInstall)
	waitForRebuildIdle(t, h)

	if got := installedNames(h); len(got) != 1 || got[0] != "serving" {
		t.Fatalf("expected serving writers to remain, got %v", got)
	}
	if servingCancelCalled.Load() {
		t.Fatal("expected the serving generation cancel not to run")
	}
	if candidateCancelSpy.Load() != 1 {
		t.Fatalf("expected rebuildLoop to clean up the candidate once, got %d", candidateCancelSpy.Load())
	}
}

func TestBuildWriters_SlowBuildUsesFullWarmSyncTimeout(t *testing.T) {
	const syncTimeout = time.Second
	buildRelease := make(chan struct{})
	buildEntered := make(chan struct{}, 1)
	syncStarted := make(chan struct{}, 1)
	opts := rebuildOptions()
	opts.StoreSyncTimeout = syncTimeout
	stub := &stubStoreBuilder{
		buildGate:    buildRelease,
		buildEntered: buildEntered,
		syncStarted:  syncStarted,
		buildWriters: writerList("warm"),
	}
	h := newTestHandler(t, opts, stub)
	seedServingWriters(h, writerList("serving"))

	goBuildWriters(h, context.Background())
	<-buildEntered
	if stub.syncCount.Load() != 0 {
		t.Fatalf("expected sync wait to start only after build, syncCount=%d", stub.syncCount.Load())
	}
	buildRelease <- struct{}{}
	<-syncStarted
	waitForRebuildIdle(t, h)

	calls := stub.syncTimeoutCalls()
	if len(calls) != 1 || calls[0] != syncTimeout {
		t.Fatalf("expected one warm sync with full timeout %s, got %v", syncTimeout, calls)
	}
}

func TestBuildWriters_StartupSupersedesSyncedGenerationBeforeReady(t *testing.T) {
	buildGate := make(chan struct{})
	buildEntered := make(chan struct{}, 2)
	stub := &stubStoreBuilder{
		buildEntered: buildEntered,
		buildGate:    buildGate,
		buildFn: func(n int64) metricsstore.MetricsWriterList {
			if n == 1 {
				return writerList("stalled")
			}
			return writerList("latest")
		},
	}
	opts := rebuildOptions()
	opts.StoreSyncTimeout = time.Second
	h := newTestHandler(t, opts, stub)

	go func() {
		h.BuildWriters(context.Background())
	}()
	<-buildEntered
	go h.BuildWriters(context.Background())
	close(buildGate)
	waitForInstalledWriter(t, h, "latest")
}

func TestBuildWriters_ZeroTimeoutBurstStillInstalls(t *testing.T) {
	const generations = 4
	stub := &stubStoreBuilder{
		buildFn: func(n int64) metricsstore.MetricsWriterList {
			return writerList(fmt.Sprintf("gen-%d", n))
		},
	}
	opts := options.NewOptions()
	opts.StoreSyncTimeout = 0
	h := newTestHandler(t, opts, stub)

	for i := 1; i <= generations; i++ {
		h.BuildWriters(context.Background())
	}
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
	h := newTestHandler(t, opts, stub)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	goBuildWriters(h, ctx)
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

func TestBuildWritersAsync_OwnerCancelDiscardsCandidate(t *testing.T) {
	syncStarted := make(chan struct{}, 1)
	stub := &stubStoreBuilder{
		buildWriters: writerList("async"),
		syncStarted:  syncStarted,
		syncGate:     make(chan error),
		cancelWait:   true,
	}
	h := newTestHandler(t, rebuildOptions(), stub)
	seedServingWriters(h, writerList("stable"))

	ctx, cancel := context.WithCancel(context.Background())
	h.BuildWritersAsync(ctx)
	<-syncStarted
	cancel()
	waitForRebuildIdle(t, h)

	if got := installedNames(h); len(got) != 1 || got[0] != "stable" {
		t.Fatalf("expected async owner cancel to keep previous writers, got %v", got)
	}
}

func TestInstallWriters_RejectsSupersededRevisionAtCommit(t *testing.T) {
	h := newTestHandler(t, rebuildOptions(), &stubStoreBuilder{})
	seedServingWriters(h, writerList("stable"))

	genCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.configRevision.Store(2)

	ok := h.installWriters(genCtx, 1, writerList("stale"), cancel, func() {})
	if ok {
		t.Fatal("expected installWriters to reject when configRevision != attemptRevision")
	}
	if got := installedNames(h); len(got) != 1 || got[0] != "stable" {
		t.Fatalf("expected superseded revision not to install, got %v", got)
	}
}

func TestBuildWriters_SequentialRequestsOneBuildPerRevision(t *testing.T) {
	stub := &stubStoreBuilder{
		buildFn: func(n int64) metricsstore.MetricsWriterList {
			return writerList(fmt.Sprintf("rev-%d", n))
		},
	}
	opts := options.NewOptions()
	opts.StoreSyncTimeout = 0
	h := newTestHandler(t, opts, stub)

	h.BuildWriters(context.Background())
	h.BuildWriters(context.Background())
	waitForRebuildIdle(t, h)

	if stub.buildCount.Load() != 2 {
		t.Fatalf("expected exactly two builds for two revisions, got %d", stub.buildCount.Load())
	}
	if got := installedNames(h); len(got) != 1 || got[0] != "rev-2" {
		t.Fatalf("expected latest revision writers, got %v", got)
	}
}

func TestCooldownExitProcessesNewerRevisionAfterOwnerCancel(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters:         writerList("after-cooldown"),
		syncTimeoutsBeforeOK: 1,
	}
	h := newTestHandler(t, rebuildOptions(), stub)
	seedServingWriters(h, writerList("stable"))

	ctx1, cancel1 := context.WithCancel(context.Background())
	h.BuildWritersAsync(ctx1)

	deadline := time.Now().Add(2 * time.Second)
	for !h.cooldownActive.Load() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !h.cooldownActive.Load() {
		t.Fatal("expected cooldown after first sync timeout")
	}

	done := make(chan struct{})
	go func() {
		h.BuildWriters(context.Background())
		close(done)
	}()

	cancel1()

	waitForInstalledWriter(t, h, "after-cooldown")

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("synchronous BuildWriters did not return")
	}
}

func TestSyncBuildWritersUnblocksEvenIfWorkerFinishesFast(t *testing.T) {
	stub := &stubStoreBuilder{buildWriters: writerList("fast")}
	opts := options.NewOptions()
	opts.StoreSyncTimeout = 0
	h := newTestHandler(t, opts, stub)

	done := make(chan struct{})
	go func() {
		h.BuildWriters(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("BuildWriters did not return after fast install")
	}
}

func TestInstalledGenerationHonorsBuildWritersCtxAfterLibraryUse(t *testing.T) {
	opts := options.NewOptions()
	opts.StoreSyncTimeout = 0
	stub := &stubStoreBuilder{buildWriters: writerList("lib")}
	h := newTestHandler(t, opts, stub)

	ctx, cancel := context.WithCancel(context.Background())
	h.BuildWriters(ctx)
	waitForInstalledWriter(t, h, "lib")

	gen := stub.generationContext()
	if gen.Err() != nil {
		t.Fatalf("expected live generation before cancel, got %v", gen.Err())
	}
	cancel()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if gen.Err() != nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("expected generation context canceled after BuildWriters ctx, still live")
}
