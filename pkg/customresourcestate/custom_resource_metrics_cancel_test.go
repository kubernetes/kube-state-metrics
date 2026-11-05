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
	"errors"
	"net/http"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

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

func TestCustomResourceMetricsListWatchInFlightListObservesContextCancel(t *testing.T) {
	started := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	cfg := &rest.Config{
		Host: "https://test.invalid",
		ContentConfig: rest.ContentConfig{
			GroupVersion:         &schema.GroupVersion{Group: "example.com", Version: "v1"},
			NegotiatedSerializer: scheme.Codecs,
		},
		Transport: &crsBlockingTransport{onStart: started, onDone: done},
	}
	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	metrics := customResourceMetrics{
		GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"},
		ResourceName:     "foos",
	}
	gvr := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "foos"}
	crClient := dynClient.Resource(gvr)

	genCtx, cancel := context.WithCancel(context.Background())
	lw := metrics.ListWatchWithContext(genCtx, crClient, metav1.NamespaceDefault, "")

	listDone := make(chan error, 1)
	go func() {
		_, err := lw.List(metav1.ListOptions{})
		listDone <- err
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("CRS list request did not start")
	}

	cancel()

	select {
	case err := <-listDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected CRS list canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("CRS list did not return after generation cancel")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("CRS transport did not observe request context cancellation")
	}
}
