/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/common/expfmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	ksmtypes "k8s.io/kube-state-metrics/v2/pkg/builder/types"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

var errInvalidStoreSyncTimeout = errors.New("invalid store sync timeout")

// negotiableFormats lists the exposition formats considered during content
// negotiation, in order of preference. It matches the list used by the
// deprecated expfmt.NegotiateIncludingOpenMetrics so negotiation is unchanged;
// anything other than OpenMetrics is served as plain text below.
var negotiableFormats = func() []expfmt.Format {
	openMetrics001, _ := expfmt.NewOpenMetricsFormat(expfmt.OpenMetricsVersion_0_0_1)
	return []expfmt.Format{
		expfmt.NewFormat(expfmt.TypeOpenMetrics),
		openMetrics001,
		expfmt.NewFormat(expfmt.TypeProtoDelim),
		expfmt.NewFormat(expfmt.TypeProtoText),
		expfmt.NewFormat(expfmt.TypeProtoCompact),
		expfmt.NewFormat(expfmt.TypeTextPlain),
	}
}()

// MetricsHandler is a http.Handler that exposes the main kube-state-metrics
// /metrics endpoint. It allows concurrent reconfiguration at runtime.
//
// Synchronous rebuild contract (BuildWriters, ConfigureStore, ConfigureSharding):
// each call waits for one attempt on its revision—success, discard (incl. sync
// failure), supersession, or caller cancellation—not for automatic retries after
// a warm sync timeout. Publication/install for that attempt completes before the
// synchronous wait returns; a timed-out warm attempt keeps prior writers while
// retries run asynchronously. Startup may return before any generation is ready;
// /metrics and readiness stay 503 until writers install. Ready() means some
// generation is installed, not that it matches the latest config. Discovery and
// autosharding use the asynchronous entry points.
type MetricsHandler struct {
	kubeClient   kubernetes.Interface
	storeBuilder ksmtypes.BuilderInterface
	opts         *options.Options
	// storeSyncTimeout is copied in New. A reload decodes YAML into opts while an old rebuild may still run.
	storeSyncTimeout time.Duration

	cancel                 func()
	stopInstalledGenBridge func()

	// mtx protects metricsWriters, shardingConfigured, curShard, curTotalShards, and storeBuilder config.
	mtx            *sync.RWMutex
	metricsWriters metricsstore.MetricsWriterList
	// writersReady is set when a generation is installed (including an intentionally empty list).
	writersReady atomic.Bool
	// shardingConfigured is set once ConfigureSharding has run for the first time.
	// Until then, e.g. because autosharding could not derive the shard from the
	// StatefulSet, there is nothing meaningful to serve.
	shardingConfigured bool
	curTotalShards     int
	curShard           int32
	enableGZIPEncoding bool

	rebuildMu          sync.Mutex
	configRevision     atomic.Uint64
	lastHandledRev     atomic.Uint64
	activeCandidateGen uint64
	// requestOwnerByRev maps a pending config revision to the caller context that registered it
	// (sync or async). Removed when the worker finishes that revision (publish, discard, or supersede).
	requestOwnerByRev map[uint64]context.Context
	syncWaiters       []*syncWaiter
	activeCancel      context.CancelFunc
	workerWake        chan struct{}
	workerStop        chan struct{}
	workerStopped     sync.Once
	workerParentCtx   context.Context
	workerWG          sync.WaitGroup
	candidateActive   atomic.Bool
	cooldownActive    atomic.Bool
	runCtxMu          sync.Mutex

	// beforeSwap runs after the sync wait and before install. Tests use it to inject another rebuild.
	beforeSwap func()
	// beforeInstallWriters runs at the start of installWriters, before mtx is acquired. Tests only.
	beforeInstallWriters func()
	// onCandidateCancel runs when rebuildLoop calls genCancel after a failed publish. Tests only.
	onCandidateCancel func()
}

// New creates and returns a new MetricsHandler with the given options.
// opts may be nil, in which case a fresh [options.NewOptions] is used (handlers do not share that instance).
// A zero-valued *options.Options is not treated as nil; explicit zero fields (including store_sync_timeout 0) are kept.
// It panics when store_sync_timeout is negative; use NewValidated for an error return.
func New(opts *options.Options, kubeClient kubernetes.Interface, storeBuilder ksmtypes.BuilderInterface, enableGZIPEncoding bool) *MetricsHandler {
	effective, err := effectiveHandlerOpts(opts)
	if err != nil {
		panic(err)
	}
	return mustNewHandler(effective, kubeClient, storeBuilder, enableGZIPEncoding)
}

// NewValidated is like New but returns an error instead of panicking on invalid store_sync_timeout.
func NewValidated(opts *options.Options, kubeClient kubernetes.Interface, storeBuilder ksmtypes.BuilderInterface, enableGZIPEncoding bool) (*MetricsHandler, error) {
	effective, err := effectiveHandlerOpts(opts)
	if err != nil {
		return nil, err
	}
	return mustNewHandler(effective, kubeClient, storeBuilder, enableGZIPEncoding), nil
}

func effectiveHandlerOpts(opts *options.Options) (*options.Options, error) {
	if opts == nil {
		def := options.NewOptions()
		return def, nil
	}
	if opts.StoreSyncTimeout < 0 {
		return nil, fmt.Errorf("%w: %s", errInvalidStoreSyncTimeout, opts.StoreSyncTimeout)
	}
	return opts, nil
}

func mustNewHandler(opts *options.Options, kubeClient kubernetes.Interface, storeBuilder ksmtypes.BuilderInterface, enableGZIPEncoding bool) *MetricsHandler {
	parentCtx := context.Background()
	h := &MetricsHandler{
		opts:               opts,
		storeSyncTimeout:   opts.StoreSyncTimeout,
		kubeClient:         kubeClient,
		storeBuilder:       storeBuilder,
		enableGZIPEncoding: enableGZIPEncoding,
		mtx:                &sync.RWMutex{},
		requestOwnerByRev:  make(map[uint64]context.Context),
		workerWake:         make(chan struct{}, 1),
		workerStop:         make(chan struct{}),
		workerParentCtx:    parentCtx,
	}
	h.startRebuildWorker()
	return h
}

// BuildWriters waits for one rebuild attempt to finish, be superseded, or be canceled by ctx.
// Writers install only after successful store sync; store_sync_timeout 0 skips the wait.
// Automatic retries after a timed-out attempt continue asynchronously without blocking the caller.
func (m *MetricsHandler) BuildWriters(ctx context.Context) {
	m.enqueueRebuild(ctx, true)
}

// BuildWritersAsync schedules a rebuild without waiting for the attempt to finish.
func (m *MetricsHandler) BuildWritersAsync(ctx context.Context) {
	m.enqueueRebuild(ctx, false)
}

// cancelActiveGeneration reports whether a newer request may cancel the in-flight candidate.
func (m *MetricsHandler) cancelActiveGeneration() bool {
	return m.storeSyncTimeout != 0
}

func (m *MetricsHandler) logSkippedInstall(genCtx context.Context, waitErr error, writerCount int) {
	reason := waitErr
	if genCtx.Err() != nil {
		reason = genCtx.Err()
	}
	if reason == nil || errors.Is(reason, context.Canceled) || errors.Is(reason, context.DeadlineExceeded) {
		klog.InfoS("Store sync ended before writer install; keeping previous writers",
			"reason", reason,
			"writerCount", writerCount,
		)
		return
	}
	klog.ErrorS(reason, "Store sync failed during metrics writer rebuild; keeping previous writers",
		"writerCount", writerCount,
	)
}

func installableSync(err error) bool {
	return err == nil
}

func (m *MetricsHandler) installWriters(genCtx context.Context, attemptRevision uint64, writers metricsstore.MetricsWriterList, cancel context.CancelFunc, stopGenOwnerBridge func()) bool {
	if m.beforeInstallWriters != nil {
		m.beforeInstallWriters()
	}
	if m.requestOwnerCanceled(attemptRevision) {
		return false
	}
	// Lock rebuildMu before mtx so enqueueRebuild cannot call activeCancel while this path is
	// between publishing checks and clearing activeCancel (P1 handoff race).
	m.rebuildMu.Lock()
	if genCtx.Err() != nil {
		m.rebuildMu.Unlock()
		return false
	}
	if m.configRevision.Load() != attemptRevision {
		m.rebuildMu.Unlock()
		return false
	}
	m.mtx.Lock()
	if genCtx.Err() != nil {
		m.mtx.Unlock()
		m.rebuildMu.Unlock()
		return false
	}
	if m.configRevision.Load() != attemptRevision {
		m.mtx.Unlock()
		m.rebuildMu.Unlock()
		return false
	}
	oldCancel := m.cancel
	oldStopBridge := m.stopInstalledGenBridge
	m.metricsWriters = writers
	m.cancel = cancel
	m.stopInstalledGenBridge = stopGenOwnerBridge
	m.writersReady.Store(true)
	m.shardingConfigured = true
	// Clear candidate slot only after publish; if mtx checks failed we still cancel via runOneCandidate.
	if m.activeCandidateGen == attemptRevision {
		m.activeCancel = nil
		m.activeCandidateGen = 0
	}
	m.mtx.Unlock()
	m.rebuildMu.Unlock()
	if oldCancel != nil {
		oldCancel()
	}
	if oldStopBridge != nil {
		oldStopBridge()
	}
	return true
}

func (m *MetricsHandler) doRebuild(genCtx context.Context) (metricsstore.MetricsWriterList, error) {
	m.mtx.Lock()
	m.storeBuilder.WithContext(genCtx)
	writers := m.storeBuilder.Build()
	m.mtx.Unlock()

	if err := genCtx.Err(); err != nil {
		return writers, err
	}
	configured := m.storeSyncTimeout
	if configured < 0 {
		return writers, fmt.Errorf("%w: %s", errInvalidStoreSyncTimeout, configured)
	}
	if configured == 0 {
		return writers, nil
	}
	syncer, ok := m.storeBuilder.(ksmtypes.StoreSyncBuilder)
	if !ok {
		return writers, nil
	}
	err := syncer.WaitForStoresSync(genCtx, configured)
	if ctxErr := genCtx.Err(); ctxErr != nil {
		return writers, ctxErr
	}
	return writers, err
}

// Ready reports whether a writer generation has been installed.
func (m *MetricsHandler) Ready() bool {
	return m.writersReady.Load()
}

// ConfigureStore applies storeBuilder configuration under mtx, then rebuilds writers synchronously.
// With autosharding, the rebuild is skipped until sharding is configured, as the builder still
// carries the static default sharding; ConfigureSharding builds once the shard is known.
func (m *MetricsHandler) ConfigureStore(ctx context.Context, configure func(ksmtypes.BuilderInterface) error) error {
	if configure != nil {
		m.mtx.Lock()
		err := configure(m.storeBuilder)
		m.mtx.Unlock()
		if err != nil {
			return err
		}
	}
	m.mtx.RLock()
	skip := m.autoSharding() && !m.shardingConfigured
	m.mtx.RUnlock()
	if skip {
		return nil
	}
	m.BuildWriters(ctx)
	return nil
}

// ConfigureStoreAsync applies configuration and schedules a rebuild without blocking on sync.
func (m *MetricsHandler) ConfigureStoreAsync(ctx context.Context, configure func(ksmtypes.BuilderInterface) error) error {
	if configure != nil {
		m.mtx.Lock()
		err := configure(m.storeBuilder)
		m.mtx.Unlock()
		if err != nil {
			return err
		}
	}
	m.mtx.RLock()
	skip := m.autoSharding() && !m.shardingConfigured
	m.mtx.RUnlock()
	if skip {
		return nil
	}
	if configure == nil {
		m.BuildWritersAsync(ctx)
		return nil
	}
	m.enqueueRebuild(ctx, false)
	return nil
}

// ConfigureSharding configures sharding and waits for one rebuild attempt to finish.
func (m *MetricsHandler) ConfigureSharding(ctx context.Context, shard int32, totalShards int) {
	if totalShards != 1 {
		klog.InfoS("Configuring sharding of this instance to be shard index (zero-indexed) out of total shards", "shard", shard, "totalShards", totalShards)
	}
	m.mtx.Lock()
	m.curShard = shard
	m.curTotalShards = totalShards
	m.storeBuilder.WithSharding(shard, totalShards)
	m.mtx.Unlock()
	m.BuildWriters(ctx)
}

// ConfigureShardingAsync schedules a sharding update without blocking on store sync.
func (m *MetricsHandler) ConfigureShardingAsync(ctx context.Context, shard int32, totalShards int) {
	if totalShards != 1 {
		klog.InfoS("Configuring sharding of this instance to be shard index (zero-indexed) out of total shards", "shard", shard, "totalShards", totalShards)
	}
	m.mtx.Lock()
	m.curShard = shard
	m.curTotalShards = totalShards
	m.storeBuilder.WithSharding(shard, totalShards)
	m.mtx.Unlock()
	m.BuildWritersAsync(ctx)
}

// Run configures the MetricsHandler's sharding and if autosharding is enabled
// re-configures sharding on re-sharding events. Run should only be called
// once.
func (m *MetricsHandler) Run(ctx context.Context) error {
	m.runCtxMu.Lock()
	m.workerParentCtx = ctx
	m.runCtxMu.Unlock()
	defer m.stopRebuildWorker()

	if !m.autoSharding() {
		klog.InfoS("Autosharding disabled")
		m.ConfigureSharding(ctx, m.opts.Shard, m.opts.TotalShards)
		// Wait for context to be done, metrics will be served until then.
		<-ctx.Done()
		return ctx.Err()
	}

	klog.InfoS("Autosharding enabled with pod", "pod", klog.KRef(m.opts.Namespace, m.opts.Pod))
	klog.InfoS("Auto detecting sharding settings")
	ss, err := detectStatefulSet(ctx, m.kubeClient, m.opts.Pod, m.opts.Namespace)
	if err != nil {
		return fmt.Errorf("detect StatefulSet: %w", err)
	}
	statefulSetName := ss.Name

	fieldSelectorOptions := func(o *metav1.ListOptions) {
		o.FieldSelector = fields.OneTermEqualSelector("metadata.name", statefulSetName).String()
	}

	i := cache.NewSharedIndexInformer(
		cache.NewFilteredListWatchFromClient(m.kubeClient.AppsV1().RESTClient(), "statefulsets", m.opts.Namespace, fieldSelectorOptions),
		&appsv1.StatefulSet{}, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	i.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(o any) {
			ss := o.(*appsv1.StatefulSet)
			if ss.Name != statefulSetName {
				return
			}

			shard, totalShards, err := shardingSettingsFromStatefulSet(ss, m.opts.Pod)
			if err != nil {
				klog.ErrorS(err, "Detected sharding settings from StatefulSet")
				return
			}

			m.mtx.RLock()
			shardingUnchanged := m.curShard == shard && m.curTotalShards == totalShards
			m.mtx.RUnlock()

			if shardingUnchanged {
				return
			}

			m.ConfigureShardingAsync(ctx, shard, totalShards)
		},
		UpdateFunc: func(oldo, curo any) {
			old := oldo.(*appsv1.StatefulSet)
			cur := curo.(*appsv1.StatefulSet)
			if cur.Name != statefulSetName {
				return
			}

			if old.ResourceVersion == cur.ResourceVersion {
				return
			}

			shard, totalShards, err := shardingSettingsFromStatefulSet(cur, m.opts.Pod)
			if err != nil {
				klog.ErrorS(err, "Detected sharding settings from StatefulSet")
				return
			}

			m.mtx.RLock()
			shardingUnchanged := m.curShard == shard && m.curTotalShards == totalShards
			m.mtx.RUnlock()

			if shardingUnchanged {
				return
			}

			m.ConfigureShardingAsync(ctx, shard, totalShards)
		},
	})
	go i.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), i.HasSynced) {
		return errors.New("waiting for informer cache to sync failed")
	}
	<-ctx.Done()
	return ctx.Err()
}

func (m *MetricsHandler) autoSharding() bool {
	return len(m.opts.Pod) > 0 && len(m.opts.Namespace) > 0
}

// ServeHTTP implements the http.Handler interface. It writes all generated metrics to the response body.
// Note that all operations defined within this procedure are performed at every request.
func (m *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Snapshot the writers instead of holding the lock for the whole response.
	// BuildWriters and ConfigureSharding need the write lock, so a slow client
	// would otherwise delay a re-shard -- and since a waiting writer blocks new
	// readers, every scrape queued behind it too. Writers are replaced wholesale
	// rather than mutated, so a snapshot stays readable and self-consistent even
	// if it is rebuilt mid-response.
	m.mtx.RLock()
	installed := m.writersReady.Load()
	writers := m.metricsWriters
	configured := m.shardingConfigured
	m.mtx.RUnlock()
	if !installed {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// An empty 200 would be recorded as every series of this instance going
	// stale; a failed scrape makes the problem visible as up == 0 instead.
	if !configured {
		http.Error(w, "metrics are not available yet: sharding is not configured", http.StatusServiceUnavailable)
		return
	}

	resHeader := w.Header()
	var writer io.Writer = w

	contentType := expfmt.NegotiateAccept(r.Header, negotiableFormats...)

	// We do not support protobuf at the moment. Fall back to FmtText if the negotiated exposition format is not FmtOpenMetrics See: https://github.com/kubernetes/kube-state-metrics/issues/2022.

	if contentType.FormatType() != expfmt.TypeOpenMetrics {
		contentType = expfmt.NewFormat(expfmt.TypeTextPlain)
	}
	resHeader.Set("Content-Type", string(contentType))

	if m.enableGZIPEncoding {
		// Gzip response if requested. Taken from
		// github.com/prometheus/client_golang/prometheus/promhttp.decorateWriter.
		reqHeader := r.Header.Get("Accept-Encoding")
		for part := range strings.SplitSeq(reqHeader, ",") {
			part = strings.TrimSpace(part)
			if part == "gzip" || strings.HasPrefix(part, "gzip;") {
				writer = gzip.NewWriter(writer)
				resHeader.Set("Content-Encoding", "gzip")
				// Stop at the first match, as the upstream implementation does.
				// Wrapping twice would leave the inner writer unclosed, so its
				// stream would never be finalised and the body would be unusable.
				break
			}
		}
	}

	requestedResources := parseResources(r.URL.Query()["resources"])
	excludedResources := parseResources(r.URL.Query()["exclude_resources"])

	// Filter writers before sanitizing so that SanitizeHeaders only
	// deduplicates across the writers that will actually be written.
	// Sanitizing first can suppress HELP/TYPE headers for metrics whose
	// only active writer is later in the list but its earlier same-named
	// counterpart was filtered out.
	activeWriters := writers
	if requestedResources != nil || excludedResources != nil {
		activeWriters = make(metricsstore.MetricsWriterList, 0, len(writers))
		for _, mw := range writers {
			if requestedResources != nil {
				if _, ok := requestedResources[mw.ResourceName]; !ok {
					continue
				}
			}
			if excludedResources != nil {
				if _, ok := excludedResources[mw.ResourceName]; ok {
					continue
				}
			}
			activeWriters = append(activeWriters, mw)
		}
	}

	sanitizedWriters := metricsstore.SanitizeHeaders(contentType, activeWriters)

	for _, w := range sanitizedWriters {
		err := w.WriteAll(writer)
		if err != nil {
			klog.ErrorS(err, "Failed to write metrics")
		}
	}

	// OpenMetrics spec requires that we end with an EOF directive.
	if contentType.FormatType() == expfmt.TypeOpenMetrics {
		_, err := writer.Write([]byte("# EOF\n"))
		if err != nil {
			klog.ErrorS(err, "Failed to write EOF directive")
		}
	}

	// In case we gzipped the response, we have to close the writer.
	if closer, ok := writer.(io.Closer); ok {
		err := closer.Close()
		if err != nil {
			klog.ErrorS(err, "Failed to close the writer")
		}
	}
}

func parseResources(params []string) map[string]struct{} {
	if params == nil {
		return nil
	}
	resMap := make(map[string]struct{})
	for _, p := range params {
		for res := range strings.SplitSeq(p, ",") {
			res = strings.TrimSpace(res)
			if res != "" {
				resMap[res] = struct{}{}
			}
		}
	}
	return resMap
}

func shardingSettingsFromStatefulSet(ss *appsv1.StatefulSet, podName string) (nominal int32, totalReplicas int, err error) {
	nominal, err = detectNominalFromPod(ss.Name, podName)
	if err != nil {
		return 0, 0, fmt.Errorf("detecting Pod nominal: %w", err)
	}

	totalReplicas = 1
	replicas := ss.Spec.Replicas
	if replicas != nil {
		totalReplicas = int(*replicas)
	}

	return nominal, totalReplicas, nil
}

func detectNominalFromPod(statefulSetName, podName string) (int32, error) {
	nominalString := strings.TrimPrefix(podName, statefulSetName+"-")
	nominal, err := strconv.ParseInt(nominalString, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to detect shard index for Pod %s of StatefulSet %s, parsed %s: %w", podName, statefulSetName, nominalString, err)
	}

	return int32(nominal), nil //nolint:gosec
}

func detectStatefulSet(ctx context.Context, kubeClient kubernetes.Interface, podName, namespaceName string) (*appsv1.StatefulSet, error) {
	p, err := kubeClient.CoreV1().Pods(namespaceName).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("retrieve pod %s for sharding: %w", podName, err)
	}

	owners := p.GetOwnerReferences()
	for _, o := range owners {
		if o.APIVersion != "apps/v1" || o.Kind != "StatefulSet" || o.Controller == nil || !*o.Controller {
			continue
		}

		ss, err := kubeClient.AppsV1().StatefulSets(namespaceName).Get(ctx, o.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("retrieve shard's StatefulSet: %s/%s: %w", namespaceName, o.Name, err)
		}

		return ss, nil
	}

	return nil, fmt.Errorf("no suitable statefulset found for auto detecting sharding for Pod %s/%s", namespaceName, podName)
}
