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
type MetricsHandler struct {
	kubeClient   kubernetes.Interface
	storeBuilder ksmtypes.BuilderInterface
	opts         *options.Options
	// storeSyncTimeout is copied in New. A reload decodes YAML into opts while an old rebuild may still run.
	storeSyncTimeout time.Duration

	cancel func()

	// mtx protects metricsWriters, curShard, curTotalShards, and storeBuilder config.
	mtx                *sync.RWMutex
	metricsWriters     metricsstore.MetricsWriterList
	writersInstalled   bool
	curTotalShards     int
	curShard           int32
	enableGZIPEncoding bool

	rebuildMu        sync.Mutex
	rebuildRunning   bool
	pendingRebuild   bool
	rebuildParentCtx context.Context
	activeCancel     context.CancelFunc
	// syncDeadline is the shared end of one rebuild burst. It is not the generation context.
	syncDeadline time.Time
	// progressWait is set while an unpublished candidate waits for one list to finish.
	progressWait bool
	// beforeSwap runs after the sync wait and before install. Tests use it to inject another rebuild.
	beforeSwap func()
}

// New creates and returns a new MetricsHandler with the given options.
func New(opts *options.Options, kubeClient kubernetes.Interface, storeBuilder ksmtypes.BuilderInterface, enableGZIPEncoding bool) *MetricsHandler {
	return &MetricsHandler{
		opts:               opts,
		storeSyncTimeout:   opts.StoreSyncTimeout,
		kubeClient:         kubeClient,
		storeBuilder:       storeBuilder,
		enableGZIPEncoding: enableGZIPEncoding,
		mtx:                &sync.RWMutex{},
	}
}

// BuildWriters returns before the rebuild finishes. Ready stays true after the first install.
// One burst shares a sync deadline. A timeout may serve partial metrics.
func (m *MetricsHandler) BuildWriters(ctx context.Context) {
	m.rebuildMu.Lock()
	m.rebuildParentCtx = ctx
	if m.rebuildRunning {
		m.pendingRebuild = true
		if m.activeCancel != nil && m.cancelActiveGeneration() {
			m.activeCancel()
		}
		m.rebuildMu.Unlock()
		return
	}
	if m.storeSyncTimeout > 0 {
		m.syncDeadline = time.Now().Add(m.storeSyncTimeout)
	}
	// Publish cancel before the goroutine starts, so a later request can cancel this generation.
	genCtx, genCancel := context.WithCancel(ctx)
	m.activeCancel = genCancel
	m.rebuildRunning = true
	m.rebuildMu.Unlock()

	go m.rebuildLoop(genCtx, genCancel)
}

// cancelActiveGeneration reports whether a newer request may cancel the in-flight generation.
// Once the shared deadline has passed, that generation is left to finish.
func (m *MetricsHandler) cancelActiveGeneration() bool {
	if m.storeSyncTimeout == 0 {
		return false
	}
	if m.storeSyncTimeout < 0 {
		return true
	}
	if time.Now().Before(m.syncDeadline) {
		return true
	}
	// A stalled unpublished candidate must not pin a newer configuration.
	return m.progressWait
}

func (m *MetricsHandler) rebuildLoop(genCtx context.Context, genCancel context.CancelFunc) {
	for {
		writers, waitErr := m.doRebuild(genCtx)
		if m.beforeSwap != nil {
			m.beforeSwap()
		}

		m.rebuildMu.Lock()
		canceled := genCtx.Err() != nil
		budgetDue := m.storeSyncTimeout > 0 && !time.Now().Before(m.syncDeadline)
		// Timeout 0 leaves this generation running, so a pending request must not discard it.
		superseded := m.pendingRebuild && !budgetDue && m.storeSyncTimeout != 0
		// Clear the handle while this generation still owns it. The next one is created below.
		m.activeCancel = nil
		installed := false
		if !canceled && !superseded && installableSync(waitErr) {
			m.installWriters(writers, genCancel)
			installed = true
			if waitErr != nil {
				klog.InfoS("Installed metrics writers after bounded store sync; live reflectors continue filling stores",
					"writerCount", len(writers),
					"reason", waitErr,
				)
			} else {
				klog.InfoS("Installed metrics writers after store sync", "writerCount", len(writers))
			}
		} else {
			// Log first. genCancel cancels genCtx and would hide waitErr.
			if !m.pendingRebuild {
				m.logSkippedInstall(genCtx, waitErr, len(writers))
			}
			genCancel()
		}
		if m.pendingRebuild {
			m.pendingRebuild = false
			if installed && m.storeSyncTimeout > 0 {
				m.syncDeadline = time.Now().Add(m.storeSyncTimeout)
			}
			genCtx, genCancel = context.WithCancel(m.rebuildParentCtx)
			m.activeCancel = genCancel
			m.rebuildMu.Unlock()
			continue
		}
		m.syncDeadline = time.Time{}
		m.rebuildRunning = false
		m.rebuildMu.Unlock()
		return
	}
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

// installableSync reports whether err allows installing the generation.
// Cancellation, a parent deadline, and a negative timeout do not.
func installableSync(err error) bool {
	if err == nil || errors.Is(err, ksmtypes.ErrStoreSyncTimeout) {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, errInvalidStoreSyncTimeout) {
		return false
	}
	return true
}

func (m *MetricsHandler) installWriters(writers metricsstore.MetricsWriterList, cancel context.CancelFunc) {
	m.mtx.Lock()
	oldCancel := m.cancel
	m.metricsWriters = writers
	m.cancel = cancel
	m.writersInstalled = true
	m.mtx.Unlock()
	if oldCancel != nil {
		oldCancel()
	}
}

func (m *MetricsHandler) doRebuild(genCtx context.Context) (metricsstore.MetricsWriterList, error) {
	m.mtx.Lock()
	m.storeBuilder.WithContext(genCtx)
	writers := m.storeBuilder.Build()
	m.mtx.Unlock()

	if err := genCtx.Err(); err != nil {
		return writers, err
	}
	configured, remaining := m.syncWaitBudget()
	if configured < 0 {
		return writers, fmt.Errorf("%w: %s", errInvalidStoreSyncTimeout, configured)
	}
	// Zero skips the wait. A canceled generation is still rejected before install.
	if configured == 0 {
		return writers, nil
	}
	if remaining <= 0 {
		if !m.Ready() {
			return writers, ksmtypes.ErrStoreSyncTimeout
		}
		return writers, m.waitUntilStoreProgress(genCtx)
	}
	syncer, ok := m.storeBuilder.(ksmtypes.StoreSyncBuilder)
	if !ok {
		return writers, nil
	}
	err := syncer.WaitForStoresSync(genCtx, remaining)
	if ctxErr := genCtx.Err(); ctxErr != nil {
		return writers, ctxErr
	}
	if errors.Is(err, ksmtypes.ErrStoreSyncTimeout) && m.Ready() {
		return writers, m.waitUntilStoreProgress(genCtx)
	}
	return writers, err
}

// waitUntilStoreProgress blocks until the candidate lists, stops, or is canceled.
// The installed generation keeps serving. A newer build can cancel this wait.
func (m *MetricsHandler) waitUntilStoreProgress(genCtx context.Context) error {
	syncer, ok := m.storeBuilder.(ksmtypes.StoreSyncBuilder)
	if !ok {
		return nil
	}
	m.rebuildMu.Lock()
	// A newer build already arrived after the deadline and before this wait.
	if m.pendingRebuild {
		m.rebuildMu.Unlock()
		return context.Canceled
	}
	m.progressWait = true
	m.rebuildMu.Unlock()
	defer func() {
		m.rebuildMu.Lock()
		m.progressWait = false
		m.rebuildMu.Unlock()
	}()
	err := syncer.WaitForStoreProgress(genCtx)
	if ctxErr := genCtx.Err(); ctxErr != nil {
		return ctxErr
	}
	return err
}

// syncWaitBudget returns the configured timeout and the time left in this burst.
// The deadline is read after Build so a slow build cannot start an unbounded wait.
func (m *MetricsHandler) syncWaitBudget() (configured, remaining time.Duration) {
	m.rebuildMu.Lock()
	defer m.rebuildMu.Unlock()
	configured = m.storeSyncTimeout
	if configured > 0 {
		remaining = time.Until(m.syncDeadline)
	}
	return configured, remaining
}

// Ready reports whether a writer generation has been installed.
func (m *MetricsHandler) Ready() bool {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	return m.writersInstalled
}

// ConfigureStore applies storeBuilder configuration under mtx, then rebuilds writers.
func (m *MetricsHandler) ConfigureStore(ctx context.Context, configure func(ksmtypes.BuilderInterface) error) error {
	if configure == nil {
		m.BuildWriters(ctx)
		return nil
	}
	m.mtx.Lock()
	err := configure(m.storeBuilder)
	m.mtx.Unlock()
	if err != nil {
		return err
	}
	m.BuildWriters(ctx)
	return nil
}

// ConfigureSharding configures sharding. Configuration can be used multiple times and
// concurrently.
func (m *MetricsHandler) ConfigureSharding(ctx context.Context, shard int32, totalShards int) {
	if totalShards != 1 {
		klog.InfoS("Configuring sharding of this instance to be shard index (zero-indexed) out of total shards", "shard", shard, "totalShards", totalShards)
	}
	m.ConfigureStore(ctx, func(b ksmtypes.BuilderInterface) error {
		m.curShard = shard
		m.curTotalShards = totalShards
		b.WithSharding(shard, totalShards)
		return nil
	})
}

// Run configures the MetricsHandler's sharding and if autosharding is enabled
// re-configures sharding on re-sharding events. Run should only be called
// once.
func (m *MetricsHandler) Run(ctx context.Context) error {
	autoSharding := len(m.opts.Pod) > 0 && len(m.opts.Namespace) > 0

	if !autoSharding {
		klog.InfoS("Autosharding disabled")
		m.ConfigureSharding(ctx, m.opts.Shard, m.opts.TotalShards)
		// Wait for context to be done, metrics will be served until then.
		<-ctx.Done()
		return ctx.Err()
	}

	klog.InfoS("Autosharding enabled with pod", "pod", klog.KRef(m.opts.Namespace, m.opts.Pod))
	klog.InfoS("Auto detecting sharding settings")
	ss, err := detectStatefulSet(m.kubeClient, m.opts.Pod, m.opts.Namespace)
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

			m.ConfigureSharding(ctx, shard, totalShards)
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

			m.ConfigureSharding(ctx, shard, totalShards)
		},
	})
	go i.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), i.HasSynced) {
		return errors.New("waiting for informer cache to sync failed")
	}
	<-ctx.Done()
	return ctx.Err()
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
	installed := m.writersInstalled
	writers := m.metricsWriters
	m.mtx.RUnlock()
	if !installed {
		w.WriteHeader(http.StatusServiceUnavailable)
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

func detectStatefulSet(kubeClient kubernetes.Interface, podName, namespaceName string) (*appsv1.StatefulSet, error) {
	p, err := kubeClient.CoreV1().Pods(namespaceName).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("retrieve pod %s for sharding: %w", podName, err)
	}

	owners := p.GetOwnerReferences()
	for _, o := range owners {
		if o.APIVersion != "apps/v1" || o.Kind != "StatefulSet" || o.Controller == nil || !*o.Controller {
			continue
		}

		ss, err := kubeClient.AppsV1().StatefulSets(namespaceName).Get(context.TODO(), o.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("retrieve shard's StatefulSet: %s/%s: %w", namespaceName, o.Name, err)
		}

		return ss, nil
	}

	return nil, fmt.Errorf("no suitable statefulset found for auto detecting sharding for Pod %s/%s", namespaceName, podName)
}
