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

package store

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/allowdenylist"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

// contextAwareBlockingTransport blocks until the request context is canceled.
type contextAwareBlockingTransport struct {
	onStart chan struct{}
	onDone  chan struct{}
}

func (t *contextAwareBlockingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.onStart != nil {
		t.onStart <- struct{}{}
	}
	<-req.Context().Done()
	if t.onDone != nil {
		t.onDone <- struct{}{}
	}
	return nil, req.Context().Err()
}

func newCoreClientWithTransport(t *testing.T, rt http.RoundTripper) kubernetes.Interface {
	t.Helper()
	cfg := &rest.Config{
		Host: "https://test.invalid",
		ContentConfig: rest.ContentConfig{
			GroupVersion:         &schema.GroupVersion{Version: "v1"},
			NegotiatedSerializer: scheme.Codecs,
		},
		Transport: rt,
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func newDynamicClientWithTransport(t *testing.T, rt http.RoundTripper) dynamic.Interface {
	t.Helper()
	cfg := &rest.Config{
		Host: "https://test.invalid",
		ContentConfig: rest.ContentConfig{
			GroupVersion:         &schema.GroupVersion{Group: "example.com", Version: "v1"},
			NegotiatedSerializer: scheme.Codecs,
		},
		Transport: rt,
	}
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func newIntegrationBuilder(t *testing.T, client kubernetes.Interface, genCtx context.Context) *Builder {
	t.Helper()
	l, err := allowdenylist.New(map[string]struct{}{}, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	b := NewBuilder()
	b.WithMetrics(prometheus.NewRegistry())
	b.WithKubeClient(client)
	b.WithContext(genCtx)
	b.WithNamespaces(options.DefaultNamespaces)
	b.WithFamilyGeneratorFilter(l)
	b.WithGenerateStoresFunc(b.DefaultGenerateStoresFunc())
	b.WithGenerateCustomResourceStoresFunc(b.DefaultGenerateCustomResourceStoresFunc())
	return b
}

func waitInFlightCancel(t *testing.T, started, done <-chan struct{}, cancel func()) {
	t.Helper()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("list request did not start")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RoundTripper did not observe request context cancellation")
	}
}

func TestBuilderBuildConfigMapsInFlightListObservesGenerationCancel(t *testing.T) {
	started := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	client := newCoreClientWithTransport(t, &contextAwareBlockingTransport{onStart: started, onDone: done})

	genCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b := newIntegrationBuilder(t, client, genCtx)
	if err := b.WithEnabledResources([]string{"configmaps"}); err != nil {
		t.Fatal(err)
	}
	b.Build()

	waitInFlightCancel(t, started, done, cancel)
}

const integrationCRResource = "example.com/v1, Resource=foos"

var (
	integrationCRGVK = schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"}
	integrationCRGVR = schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "foos"}
)

type integrationCRFactory struct {
	client dynamic.NamespaceableResourceInterface
}

var (
	_ customresource.RegistryFactory         = integrationCRFactory{}
	_ customresource.ContextListWatchFactory = integrationCRFactory{}
)

func (integrationCRFactory) Name() string { return "foos" }

func (f integrationCRFactory) CreateClient(*rest.Config) (any, error) {
	return f.client, nil
}

func (integrationCRFactory) MetricFamilyGenerators() []generator.FamilyGenerator {
	return []generator.FamilyGenerator{{
		Name: "kube_cr_integration_test",
		Type: metric.Gauge,
		Help: "integration test",
		GenerateFunc: func(any) *metric.Family {
			return &metric.Family{}
		},
	}}
}

func (integrationCRFactory) ExpectedType() any {
	u := unstructured.Unstructured{}
	u.SetGroupVersionKind(integrationCRGVK)
	return &u
}

func (f integrationCRFactory) ListWatchWithContext(ctx context.Context, customResourceClient any, ns string, fieldSelector string) cache.ListerWatcher {
	return DynamicNamespacedListWatch(ctx, customResourceClient.(dynamic.NamespaceableResourceInterface), ns, fieldSelector)
}

func (f integrationCRFactory) ListWatch(customResourceClient any, ns string, fieldSelector string) cache.ListerWatcher {
	return f.ListWatchWithContext(context.Background(), customResourceClient, ns, fieldSelector)
}

func TestBuilderBuildCustomResourceInFlightListObservesGenerationCancel(t *testing.T) {
	started := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	dynClient := newDynamicClientWithTransport(t, &contextAwareBlockingTransport{onStart: started, onDone: done})

	genCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b := newIntegrationBuilder(t, fake.NewSimpleClientset(), genCtx)
	factory := integrationCRFactory{client: dynClient.Resource(integrationCRGVR)}
	b.WithCustomResourceStoreFactories(factory)
	b.WithCustomResourceClients(map[string]any{
		integrationCRGVR.String(): dynClient.Resource(integrationCRGVR),
	})
	b.GetGVKStopChan = func(string) chan struct{} { return make(chan struct{}) }
	if err := b.WithEnabledResources([]string{integrationCRResource}); err != nil {
		t.Fatal(err)
	}
	b.Build()

	waitInFlightCancel(t, started, done, cancel)
}

func TestBuilderBuildCustomResourceInFlightListObservesGVKStop(t *testing.T) {
	started := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	dynClient := newDynamicClientWithTransport(t, &contextAwareBlockingTransport{onStart: started, onDone: done})

	gvkStop := make(chan struct{})
	genCtx, genCancel := context.WithCancel(context.Background())
	defer genCancel()

	b := newIntegrationBuilder(t, fake.NewSimpleClientset(), genCtx)
	factory := integrationCRFactory{client: dynClient.Resource(integrationCRGVR)}
	b.WithCustomResourceStoreFactories(factory)
	b.WithCustomResourceClients(map[string]any{
		integrationCRGVR.String(): dynClient.Resource(integrationCRGVR),
	})
	b.GetGVKStopChan = func(string) chan struct{} { return gvkStop }
	if err := b.WithEnabledResources([]string{integrationCRResource}); err != nil {
		t.Fatal(err)
	}
	b.Build()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("CRS list request did not start")
	}
	close(gvkStop)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RoundTripper did not observe request context cancellation after GVK stop")
	}
}
