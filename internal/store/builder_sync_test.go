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
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func TestWaitForStoresSync_NoReflectors(t *testing.T) {
	b := NewBuilder()
	if !b.WaitForStoresSync(context.Background(), time.Millisecond) {
		t.Fatal("expected sync success with no reflectors")
	}
}

func TestWaitForStoresSync_Timeout(t *testing.T) {
	b := NewBuilder()
	b.setReflectors(startedReflector{reflector: newUnstartedReflector(), stopCh: make(chan struct{})})

	if b.WaitForStoresSync(context.Background(), 50*time.Millisecond) {
		t.Fatal("expected sync to time out before reflector runs")
	}
}

func TestWaitForStoresSync_StoppedBeforeList(t *testing.T) {
	b := NewBuilder()
	stopCh := make(chan struct{})
	close(stopCh)
	b.setReflectors(startedReflector{reflector: newUnstartedReflector(), stopCh: stopCh})

	if ok, elapsed := waitForSync(b, 5*time.Second); ok || elapsed > time.Second {
		t.Fatalf("expected prompt sync failure, ok=%v elapsed=%s", ok, elapsed)
	}
}

func TestWaitForStoresSync_StoppedReflectorAfterOpenOne(t *testing.T) {
	b := NewBuilder()
	stopCh := make(chan struct{})
	close(stopCh)
	b.setReflectors(
		startedReflector{reflector: newUnstartedReflector(), stopCh: make(chan struct{})},
		startedReflector{reflector: newUnstartedReflector(), stopCh: stopCh},
	)

	if ok, elapsed := waitForSync(b, 5*time.Second); ok || elapsed > time.Second {
		t.Fatalf("expected prompt sync failure when a later reflector is stopped, ok=%v elapsed=%s", ok, elapsed)
	}
}

func TestWaitForStoresSync_StopDuringWait(t *testing.T) {
	b := NewBuilder()
	stopCh := make(chan struct{})
	b.setReflectors(startedReflector{reflector: newUnstartedReflector(), stopCh: stopCh})

	assertFailsPromptlyAfter(t, b, func() { close(stopCh) })
}

func TestWaitForStoresSync_CRStopDuringWait(t *testing.T) {
	t.Run("gvk stop", func(t *testing.T) {
		b := NewBuilder()
		ctx := context.Background()
		gvkStop := make(chan struct{})
		b.setReflectors(startedReflector{
			reflector: newUnstartedReflector(),
			stopCh:    newCRReflectorStopCh(ctx, gvkStop),
		})
		assertFailsPromptlyAfter(t, b, func() { close(gvkStop) })
	})

	t.Run("context cancel", func(t *testing.T) {
		b := NewBuilder()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		gvkStop := make(chan struct{})
		b.setReflectors(startedReflector{
			reflector: newUnstartedReflector(),
			stopCh:    newCRReflectorStopCh(ctx, gvkStop),
		})
		assertFailsPromptlyAfter(t, b, cancel)
	})
}

func TestWaitForStoresSync_SyncedReflectorIgnoresClosedStop(t *testing.T) {
	b := NewBuilder()
	reflector, stopCh := runReflectorUntilListed(t)
	close(stopCh)
	b.setReflectors(startedReflector{reflector: reflector, stopCh: stopCh})

	if ok, elapsed := waitForSync(b, 5*time.Second); !ok || elapsed > time.Second {
		t.Fatalf("expected synced reflector to succeed after stop, ok=%v elapsed=%s", ok, elapsed)
	}
}

func newUnstartedReflector() *cache.Reflector {
	return cache.NewReflectorWithOptions(podListWatch(""), &v1.Pod{}, cache.NewStore(cache.MetaNamespaceKeyFunc), cache.ReflectorOptions{})
}

func runReflectorUntilListed(t *testing.T) (*cache.Reflector, chan struct{}) {
	t.Helper()
	reflector := cache.NewReflectorWithOptions(podListWatch("1"), &v1.Pod{}, cache.NewStore(cache.MetaNamespaceKeyFunc), cache.ReflectorOptions{})
	stopCh := make(chan struct{})
	go reflector.Run(stopCh)

	deadline := time.Now().Add(2 * time.Second)
	for reflector.LastSyncResourceVersion() == "" {
		if time.Now().After(deadline) {
			t.Fatal("reflector did not complete its initial list")
		}
		time.Sleep(10 * time.Millisecond)
	}
	return reflector, stopCh
}

// listWatchWithoutWatchList forces the reflector down the list path. client-go
// enables WatchList by default, and that path does not set LastSyncResourceVersion
// until it sees an initial-events bookmark.
type listWatchWithoutWatchList struct {
	*cache.ListWatch
}

func (listWatchWithoutWatchList) IsWatchListSemanticsUnSupported() bool { return true }

func podListWatch(resourceVersion string) cache.ListerWatcher {
	return listWatchWithoutWatchList{ListWatch: &cache.ListWatch{
		ListFunc: func(metav1.ListOptions) (runtime.Object, error) {
			return &v1.PodList{ListMeta: metav1.ListMeta{ResourceVersion: resourceVersion}}, nil
		},
		WatchFunc: func(metav1.ListOptions) (watch.Interface, error) {
			return watch.NewFake(), nil
		},
	}}
}

func (b *Builder) setReflectors(started ...startedReflector) {
	b.reflectorsMu.Lock()
	b.reflectors = started
	b.reflectorsMu.Unlock()
}

func waitForSync(b *Builder, timeout time.Duration) (bool, time.Duration) {
	start := time.Now()
	ok := b.WaitForStoresSync(context.Background(), timeout)
	return ok, time.Since(start)
}

func assertFailsPromptlyAfter(t *testing.T, b *Builder, closeStop func()) {
	t.Helper()
	done := make(chan bool, 1)
	go func() {
		done <- b.WaitForStoresSync(context.Background(), 5*time.Second)
	}()
	time.Sleep(150 * time.Millisecond)
	closeStop()
	select {
	case ok := <-done:
		if ok {
			t.Fatal("expected sync to fail after stop")
		}
	case <-time.After(time.Second):
		t.Fatal("sync did not fail promptly after stop")
	}
}
