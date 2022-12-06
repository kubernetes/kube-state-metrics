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

// MetricsHandler is a http.Handler that exposes the main kube-state-metrics
// /metrics endpoint. It allows concurrent reconfiguration at runtime.
type MetricsHandler struct {
	opts               *options.Options
	kubeClient         kubernetes.Interface
	storeBuilder       ksmtypes.BuilderInterface
	enableGZIPEncoding bool

	cancel func()

	// mtx protects metricsWriters, curShard, and curTotalShards
	mtx            *sync.RWMutex
	metricsWriters metricsstore.MetricsWriterList
	curShard       int32
	curTotalShards int
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

// ConfigureSharding (re-)configures sharding. Re-configuration can be done
// concurrently.
func (m *MetricsHandler) ConfigureSharding(ctx context.Context, shard int32, totalShards int) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if m.cancel != nil {
		m.cancel()
	}
	if totalShards != 1 {
		klog.InfoS("Configuring sharding of this instance to be shard index (zero-indexed) out of total shards", "shard", shard, "totalShards", totalShards)
	}
	ctx, m.cancel = context.WithCancel(ctx)
	m.storeBuilder.WithSharding(shard, totalShards)
	m.storeBuilder.WithContext(ctx)
	m.metricsWriters = m.storeBuilder.Build()
	m.curShard = shard
	m.curTotalShards = totalShards
}

// Run configures the MetricsHandler's sharding and if autosharding is enabled
// re-configures sharding on re-sharding events. Run should only be called
// once.
func (m *MetricsHandler) Run(ctx context.Context) error {
	autoSharding := len(m.opts.Pod) > 0 && len(m.opts.Namespace) > 0

	if !autoSharding {
		klog.InfoS("Autosharding disabled")
		m.ConfigureSharding(ctx, m.opts.Shard, m.opts.TotalShards)
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

	labelSelectorOptions := func(o *metav1.ListOptions) {
		o.LabelSelector = fields.SelectorFromSet(ss.Labels).String()
	}

	i := cache.NewSharedIndexInformer(
		cache.NewFilteredListWatchFromClient(m.kubeClient.AppsV1().RESTClient(), "statefulsets", m.opts.Namespace, labelSelectorOptions),
		&appsv1.StatefulSet{}, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	i.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(o interface{}) {
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
		UpdateFunc: func(oldo, curo interface{}) {
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

// ServeHTTP implements the http.Handler interface. It writes all generated
// metrics to the response body.
func (m *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	resHeader := w.Header()
	var writer io.Writer = w

	resHeader.Set("Content-Type", `text/plain; version=`+"0.0.4")

	if m.enableGZIPEncoding {
		// Gzip response if requested. Taken from
		// github.com/prometheus/client_golang/prometheus/promhttp.decorateWriter.
		reqHeader := r.Header.Get("Accept-Encoding")
		parts := strings.Split(reqHeader, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "gzip" || strings.HasPrefix(part, "gzip;") {
				writer = gzip.NewWriter(writer)
				resHeader.Set("Content-Encoding", "gzip")
			}
		}
	}

	for _, w := range m.metricsWriters {
		err := w.WriteAll(writer)
		if err != nil {
			klog.ErrorS(err, "Failed to write metrics")
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

	return int32(nominal), nil
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
