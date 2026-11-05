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

package customresourcestate

import (
	"context"
	_ "embed"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	"k8s.io/kube-state-metrics/v2/internal/store"
	"k8s.io/kube-state-metrics/v2/pkg/allowdenylist"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

//go:embed example_config.yaml
var exampleConfigYAML string

type crsBlockingTransport struct {
	onStart chan struct{}
	onDone  chan struct{}
}

func (t *crsBlockingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.onStart != nil {
		t.onStart <- struct{}{}
	}
	<-req.Context().Done()
	if t.onDone != nil {
		t.onDone <- struct{}{}
	}
	return nil, req.Context().Err()
}

func TestCustomResourceMetrics_BuilderPathInFlightListObservesGenerationCancel(t *testing.T) {
	var metrics Metrics
	if err := yaml.NewDecoder(strings.NewReader(exampleConfigYAML)).Decode(&metrics); err != nil {
		t.Fatal(err)
	}
	configOverrides(&metrics)
	factory, err := NewCustomResourceMetrics(metrics.Spec.Resources[0])
	if err != nil {
		t.Fatal(err)
	}

	started := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	cfg := &rest.Config{
		Host: "https://test.invalid",
		ContentConfig: rest.ContentConfig{
			GroupVersion:         &schema.GroupVersion{Group: "myteam.io", Version: "v1"},
			NegotiatedSerializer: scheme.Codecs,
		},
		Transport: &crsBlockingTransport{onStart: started, onDone: done},
	}
	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	gvr := schema.GroupVersionResource{Group: "myteam.io", Version: "v1", Resource: "foos"}

	l, err := allowdenylist.New(map[string]struct{}{}, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	genCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b := store.NewBuilder()
	b.WithMetrics(prometheus.NewRegistry())
	b.WithKubeClient(fake.NewSimpleClientset())
	b.WithContext(genCtx)
	b.WithNamespaces(options.DefaultNamespaces)
	b.WithFamilyGeneratorFilter(l)
	b.WithGenerateStoresFunc(b.DefaultGenerateStoresFunc())
	b.WithGenerateCustomResourceStoresFunc(b.DefaultGenerateCustomResourceStoresFunc())
	b.WithCustomResourceStoreFactories(factory)
	b.WithCustomResourceClients(map[string]any{
		gvr.String(): dynClient.Resource(gvr),
	})
	b.GetGVKStopChan = func(string) chan struct{} { return make(chan struct{}) }
	if err := b.WithEnabledResources([]string{gvr.String()}); err != nil {
		t.Fatal(err)
	}
	b.Build()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("CRS list request did not start")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("CRS transport did not observe request context cancellation")
	}
}
