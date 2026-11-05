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
	"errors"
	"net/http"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
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

func TestConfigMapListWatchInFlightListObservesContextCancel(t *testing.T) {
	started := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	client := newCoreClientWithTransport(t, &contextAwareBlockingTransport{onStart: started, onDone: done})

	genCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lw := createConfigMapListWatch(genCtx, client, metav1.NamespaceAll, "")

	listDone := make(chan error, 1)
	go func() {
		_, err := lw.List(metav1.ListOptions{})
		listDone <- err
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("list request did not start")
	}

	cancel()

	select {
	case err := <-listDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected list to fail with context canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("list did not return after generation cancel")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RoundTripper did not observe request context cancellation")
	}
}

func TestBuilderListWatchInFlightListObservesGenerationCancel(t *testing.T) {
	started := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	client := newCoreClientWithTransport(t, &contextAwareBlockingTransport{onStart: started, onDone: done})

	b := NewBuilder()
	genCtx, cancel := context.WithCancel(context.Background())
	b.WithContext(genCtx)

	lw := listWatchFromBuildContext(b, createConfigMapListWatch)(client, metav1.NamespaceAll, "")

	listDone := make(chan error, 1)
	go func() {
		_, err := lw.List(metav1.ListOptions{})
		listDone <- err
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("list request did not start")
	}

	cancel()

	select {
	case err := <-listDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected list to fail with context canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("list did not return after generation cancel")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RoundTripper did not observe request context cancellation")
	}
}

func TestCRListContextInFlightListObservesGVKStop(t *testing.T) {
	started := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	dynClient := newDynamicClientWithTransport(t, &contextAwareBlockingTransport{onStart: started, onDone: done})

	b := NewBuilder()
	gvkStop := make(chan struct{})
	b.GetGVKStopChan = func(string) chan struct{} { return gvkStop }

	genCtx, genCancel := context.WithCancel(context.Background())
	defer genCancel()
	b.WithContext(genCtx)
	listCtx := b.crResourceListContext("example.com/v1, Kind=Foo")

	gvr := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "foos"}
	lw := DynamicNamespacedListWatch(listCtx, dynClient.Resource(gvr), metav1.NamespaceDefault, "")

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

	close(gvkStop)

	select {
	case err := <-listDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected list to fail after GVK stop, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("list did not return after GVK teardown")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RoundTripper did not observe request context cancellation")
	}
}

func TestDynamicNamespacedListWatchUsesRequestContext(t *testing.T) {
	started := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	dynClient := newDynamicClientWithTransport(t, &contextAwareBlockingTransport{onStart: started, onDone: done})

	genCtx, cancel := context.WithCancel(context.Background())
	gvr := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "foos"}
	lw := DynamicNamespacedListWatch(genCtx, dynClient.Resource(gvr), metav1.NamespaceDefault, "")

	listDone := make(chan error, 1)
	go func() {
		_, err := lw.List(metav1.ListOptions{})
		listDone <- err
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("dynamic list did not start")
	}

	cancel()

	select {
	case err := <-listDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected dynamic list canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("dynamic list did not return")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("transport did not observe cancellation")
	}
}