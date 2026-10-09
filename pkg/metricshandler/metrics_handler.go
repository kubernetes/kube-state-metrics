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
	// beforeSwap, when set, runs after a generation's sync wait returns and
	// before that generation is installed. Tests use it to publish another
	// rebuild request in that window.
	beforeSwap func()
}

// New creates and returns a new MetricsHandler with the given options.
func New(opts *options.Options, kubeClient kubernetes.Interface, storeBuilder ksmtypes.BuilderInterface, enableGZIPEncoding bool) *MetricsHandler {
	return &MetricsHandler{
		opts:               opts,
		kubeClient:         kubeClient,
		storeBuilder:       storeBuilder,
		enableGZIPEncoding: enableGZIPEncoding,
		mtx:                &sync.RWMutex{},
	}
}

// BuildWriters rebuilds metrics writers after store configuration changes.
// Rebuilds are coalesced. A generation is installed after its stores sync, or
// when the configured sync timeout elapses or a reflector stops before listing.
// Superseded and canceled generations are not installed. A timed-out install
// can expose partial metrics while reflectors that are still running continue
// to fill their stores.
func (m *MetricsHandler) BuildWriters(ctx context.Context) {
	m.rebuildMu.Lock()
	m.rebuildParentCtx = ctx
	if m.rebuildRunning {
		m.pendingRebuild = true
		if m.activeCancel != nil {
			m.activeCancel()
		}
		m.rebuildMu.Unlock()
		return
	}
	// Register cancellation and mark the rebuild running before the goroutine
	// starts, so a request that arrives once the rebuild is visible can cancel it.
	genCtx, genCancel := context.WithCancel(ctx)
	m.activeCancel = genCancel
	m.rebuildRunning = true
	m.rebuildMu.Unlock()

	go m.rebuildLoop(genCtx, genCancel)
}

func (m *MetricsHandler) rebuildLoop(genCtx context.Context, genCancel context.CancelFunc) {
	for {
		writers, waitErr := m.doRebuild(genCtx)
		if m.beforeSwap != nil {
			m.beforeSwap()
		}

		m.rebuildMu.Lock()
		superseded := m.pendingRebuild || genCtx.Err() != nil
		// This generation still owns the handle: a newer request can call it,
		// but cannot replace it until this critical section starts the next one.
		m.activeCancel = nil
		if !superseded && installableSync(waitErr) {
			m.installWriters(writers, genCancel)
			if waitErr != nil {
				klog.InfoS("Installed metrics writers after bounded store sync; live reflectors continue filling stores",
					"writerCount", len(writers),
					"reason", waitErr,
				)
			} else {
				klog.InfoS("Installed metrics writers after store sync", "writerCount", len(writers))
			}
		} else {
			genCancel()
			if !m.pendingRebuild {
				klog.ErrorS(waitErr, "Store sync canceled during metrics writer rebuild; keeping previous writers",
					"writerCount", len(writers),
				)
			}
		}
		if m.pendingRebuild {
			m.pendingRebuild = false
			genCtx, genCancel = context.WithCancel(m.rebuildParentCtx)
			m.activeCancel = genCancel
			m.rebuildMu.Unlock()
			continue
		}
		m.rebuildRunning = false
		m.rebuildMu.Unlock()
		return
	}
}

// installableSync reports whether a WaitForStoresSync result may be installed.
// Only success, the configured sync timeout, and a stopped reflector qualify.
// A canceled or deadline-exceeded generation context does not.
func installableSync(err error) bool {
	return err == nil || errors.Is(err, ksmtypes.ErrStoreSyncTimeout) || errors.Is(err, ksmtypes.ErrReflectorStopped)
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
	syncTimeout := m.opts.StoreSyncTimeout
	if syncTimeout < 0 {
		return writers, fmt.Errorf("invalid store sync timeout %s", syncTimeout)
	}
	// Zero skips the wait and installs immediately, still subject to the
	// generation-context check above and again before installation.
	if syncTimeout == 0 {
		return writers, nil
	}
	syncer, ok := m.storeBuilder.(ksmtypes.StoreSyncBuilder)
	if !ok {
		return writers, nil
	}
	err := syncer.WaitForStoresSync(genCtx, syncTimeout)
	if ctxErr := genCtx.Err(); ctxErr != nil {
		return writers, ctxErr
	}
	return writers, err
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
