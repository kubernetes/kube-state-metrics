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
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	ksmtypes "k8s.io/kube-state-metrics/v2/pkg/builder/types"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

type stubStoreBuilder struct {
	buildWriters metricsstore.MetricsWriterList
	syncOK       bool
	buildCount   int
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
	s.buildCount++
	return s.buildWriters
}

func (s *stubStoreBuilder) WaitForStoresSync(_ context.Context, _ time.Duration) bool {
	return s.syncOK
}

func TestBuildWriters_KeepsPreviousWritersWhenSyncFails(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters: metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("candidate")},
		syncOK:       false,
	}
	opts := options.NewOptions()
	h := New(opts, nil, stub, false)

	existing := metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("stable")}
	h.mtx.Lock()
	h.metricsWriters = existing
	h.mtx.Unlock()

	h.BuildWriters(context.Background())

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		h.mtx.RLock()
		got := h.metricsWriters
		h.mtx.RUnlock()
		if stub.buildCount >= 1 && len(got) == 1 && got[0].ResourceName == "stable" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected stable writers after failed sync; buildCount=%d", stub.buildCount)
}

func TestBuildWriters_SwapsWritersWhenSyncSucceeds(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters: metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("next")},
		syncOK:       true,
	}
	opts := options.NewOptions()
	h := New(opts, nil, stub, false)

	h.mtx.Lock()
	h.metricsWriters = metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("prev")}
	h.mtx.Unlock()

	h.BuildWriters(context.Background())

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		h.mtx.RLock()
		got := h.metricsWriters
		h.mtx.RUnlock()
		if len(got) == 1 && got[0].ResourceName == "next" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected writers to swap after successful sync")
}

func TestServeHTTP_WriterSnapshot(t *testing.T) {
	h := New(options.NewOptions(), nil, &stubStoreBuilder{}, false)
	w1 := metricsstore.NewMetricsWriter("first")
	w2 := metricsstore.NewMetricsWriter("second")
	h.mtx.Lock()
	h.metricsWriters = metricsstore.MetricsWriterList{w1, w2}
	h.mtx.Unlock()

	h.mtx.RLock()
	snapshot := append(metricsstore.MetricsWriterList(nil), h.metricsWriters...)
	h.mtx.RUnlock()

	h.mtx.Lock()
	h.metricsWriters = metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("replaced")}
	h.mtx.Unlock()

	if len(snapshot) != 2 || snapshot[0].ResourceName != "first" || snapshot[1].ResourceName != "second" {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
}
