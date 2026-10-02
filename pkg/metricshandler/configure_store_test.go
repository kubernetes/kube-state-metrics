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
	"sync/atomic"
	"testing"

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
	buildCount   atomic.Int64
	buildWriters metricsstore.MetricsWriterList
}

func (s *stubStoreBuilder) WithMetrics(prometheus.Registerer) {}
func (s *stubStoreBuilder) WithEnabledResources([]string) error {
	return nil
}
func (s *stubStoreBuilder) WithNamespaces(options.NamespaceList)     {}
func (s *stubStoreBuilder) WithFieldSelectorFilter(string)           {}
func (s *stubStoreBuilder) WithSharding(int32, int)                  {}
func (s *stubStoreBuilder) WithContext(context.Context)              {}
func (s *stubStoreBuilder) WithKubeClient(clientset.Interface)       {}
func (s *stubStoreBuilder) WithCustomResourceClients(map[string]any) {}
func (s *stubStoreBuilder) WithUsingAPIServerCache(bool)             {}
func (s *stubStoreBuilder) WithFamilyGeneratorFilter(generator.FamilyGeneratorFilter) {
}
func (s *stubStoreBuilder) WithAllowAnnotations(map[string][]string) error  { return nil }
func (s *stubStoreBuilder) WithAllowLabels(map[string][]string) error       { return nil }
func (s *stubStoreBuilder) WithGenerateStoresFunc(ksmtypes.BuildStoresFunc) {}
func (s *stubStoreBuilder) DefaultGenerateStoresFunc() ksmtypes.BuildStoresFunc {
	return nil
}
func (s *stubStoreBuilder) DefaultGenerateCustomResourceStoresFunc() ksmtypes.BuildCustomResourceStoresFunc {
	return nil
}
func (s *stubStoreBuilder) WithCustomResourceStoreFactories(...customresource.RegistryFactory) {
}
func (s *stubStoreBuilder) WithGenerateCustomResourceStoresFunc(ksmtypes.BuildCustomResourceStoresFunc) {
}
func (s *stubStoreBuilder) BuildStores() [][]cache.Store { return nil }
func (s *stubStoreBuilder) Build() metricsstore.MetricsWriterList {
	s.buildCount.Add(1)
	return s.buildWriters
}

func TestConfigureStoreAppliesConfigThenRebuilds(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters: metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("configured")},
	}
	h := New(options.NewOptions(), nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("prev")}
	h.mtx.Unlock()

	var configured bool
	if err := h.ConfigureStore(context.Background(), func(b ksmtypes.BuilderInterface) error {
		if b != stub {
			t.Fatalf("expected ConfigureStore to pass the handler store builder")
		}
		configured = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !configured {
		t.Fatal("expected ConfigureStore callback to run")
	}
	if stub.buildCount.Load() != 1 {
		t.Fatalf("expected one rebuild after ConfigureStore, got %d", stub.buildCount.Load())
	}
	h.mtx.RLock()
	got := h.metricsWriters
	h.mtx.RUnlock()
	if len(got) != 1 || got[0].ResourceName != "configured" {
		t.Fatalf("expected writers swapped after ConfigureStore rebuild, got %+v", got)
	}
}

func TestConfigureStoreSkipsRebuildOnConfigError(t *testing.T) {
	stub := &stubStoreBuilder{
		buildWriters: metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("next")},
	}
	h := New(options.NewOptions(), nil, stub, false)
	h.mtx.Lock()
	h.metricsWriters = metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("prev")}
	h.mtx.Unlock()

	err := h.ConfigureStore(context.Background(), func(ksmtypes.BuilderInterface) error {
		return errors.New("bad config")
	})
	if err == nil {
		t.Fatal("expected ConfigureStore to return configuration error")
	}
	if stub.buildCount.Load() != 0 {
		t.Fatalf("expected no rebuild on config error, got %d", stub.buildCount.Load())
	}
}
