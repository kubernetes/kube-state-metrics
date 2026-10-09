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
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	ksmtypes "k8s.io/kube-state-metrics/v2/pkg/builder/types"
)

func TestWaitForStoresSync_NoReflectors(t *testing.T) {
	b := NewBuilder()
	if err := b.WaitForStoresSync(context.Background(), time.Millisecond); err != nil {
		t.Fatal(err)
	}
}

func TestWaitForStoresSync_NoReflectorsCanceled(t *testing.T) {
	b := NewBuilder()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := b.WaitForStoresSync(ctx, time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context with no reflectors, got %v", err)
	}
}

func TestWaitForStoresSync_Timeout(t *testing.T) {
	b := NewBuilder()
	b.setReflectors(startedReflector{reflector: newUnstartedReflector(), stopCh: make(chan struct{})})

	err := b.WaitForStoresSync(context.Background(), 50*time.Millisecond)
	if !errors.Is(err, ksmtypes.ErrStoreSyncTimeout) {
		t.Fatalf("expected sync timeout, got %v", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("configured sync timeout must not be a context deadline")
	}
}

func TestWaitForStoresSync_ParentDeadline(t *testing.T) {
	b := NewBuilder()
	b.setReflectors(startedReflector{reflector: newUnstartedReflector(), stopCh: make(chan struct{})})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	err := b.WaitForStoresSync(ctx, 5*time.Second)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected parent deadline, got %v", err)
	}
	if errors.Is(err, ksmtypes.ErrStoreSyncTimeout) {
		t.Fatal("parent deadline must not be reported as the configured sync timeout")
	}
}

func TestWaitForStoresSync_StoppedBeforeList(t *testing.T) {
	b := NewBuilder()
	stopCh := make(chan struct{})
	close(stopCh)
	b.setReflectors(startedReflector{reflector: newUnstartedReflector(), stopCh: stopCh})

	if err, elapsed := waitForSync(b, 5*time.Second); !errors.Is(err, ksmtypes.ErrReflectorStopped) || elapsed > time.Second {
		t.Fatalf("expected prompt stopped reflector, err=%v elapsed=%s", err, elapsed)
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

	if err, elapsed := waitForSync(b, 5*time.Second); !errors.Is(err, ksmtypes.ErrReflectorStopped) || elapsed > time.Second {
		t.Fatalf("expected prompt stopped reflector when a later reflector is stopped, err=%v elapsed=%s", err, elapsed)
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

	if err, elapsed := waitForSync(b, 5*time.Second); err != nil || elapsed > time.Second {
		t.Fatalf("expected synced reflector to succeed after stop, err=%v elapsed=%s", err, elapsed)
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

func waitForSync(b *Builder, timeout time.Duration) (error, time.Duration) {
	start := time.Now()
	err := b.WaitForStoresSync(context.Background(), timeout)
	return err, time.Since(start)
}

func assertFailsPromptlyAfter(t *testing.T, b *Builder, closeStop func()) {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- b.WaitForStoresSync(context.Background(), 5*time.Second)
	}()
	time.Sleep(150 * time.Millisecond)
	closeStop()
	select {
	case err := <-done:
		if !errors.Is(err, ksmtypes.ErrReflectorStopped) {
			t.Fatalf("expected stopped reflector, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("sync did not fail promptly after stop")
	}
}
