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
	"sync"
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

	if elapsed, err := waitForSync(b, 5*time.Second); err != nil || elapsed > time.Second {
		t.Fatalf("expected a stopped reflector to count as done, err=%v elapsed=%s", err, elapsed)
	}
}

func TestWaitForStoresSync_StoppedBesideSyncedReflector(t *testing.T) {
	b := NewBuilder()
	reflector, liveStop := runReflectorUntilListed(t)
	defer close(liveStop)
	stopped := make(chan struct{})
	close(stopped)
	b.setReflectors(
		startedReflector{reflector: reflector, stopCh: liveStop},
		startedReflector{reflector: newUnstartedReflector(), stopCh: stopped},
	)

	if elapsed, err := waitForSync(b, 5*time.Second); err != nil || elapsed > time.Second {
		t.Fatalf("expected nil when the live reflector has already listed, err=%v elapsed=%s", err, elapsed)
	}
}

func TestWaitForStoresSync_StoppedReflectorWaitsForTimeout(t *testing.T) {
	b := NewBuilder()
	stopped := make(chan struct{})
	close(stopped)
	b.setReflectors(
		startedReflector{reflector: newUnstartedReflector(), stopCh: make(chan struct{})},
		startedReflector{reflector: newUnstartedReflector(), stopCh: stopped},
	)

	const timeout = 200 * time.Millisecond
	elapsed, err := waitForSync(b, timeout)
	if !errors.Is(err, ksmtypes.ErrStoreSyncTimeout) {
		t.Fatalf("expected sync timeout while a reflector is still listing, got %v", err)
	}
	if elapsed < timeout/2 {
		t.Fatalf("stopped reflector ended the wait early: %s", elapsed)
	}
}

func TestWaitForStoresSync_StoppedReflectorWaitsForLiveList(t *testing.T) {
	b := NewBuilder()
	stopped := make(chan struct{})
	close(stopped)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseList := func() { releaseOnce.Do(func() { close(release) }) }
	liveStop := make(chan struct{})
	t.Cleanup(func() {
		releaseList()
		close(liveStop)
	})
	reflector := cache.NewReflectorWithOptions(blockingPodListWatch(release), &v1.Pod{}, cache.NewStore(cache.MetaNamespaceKeyFunc), cache.ReflectorOptions{})
	go reflector.Run(liveStop)
	b.setReflectors(
		startedReflector{reflector: newUnstartedReflector(), stopCh: stopped},
		startedReflector{reflector: reflector, stopCh: liveStop},
	)

	done := make(chan error, 1)
	go func() {
		done <- b.WaitForStoresSync(context.Background(), 5*time.Second)
	}()
	select {
	case err := <-done:
		t.Fatalf("wait returned before the live reflector listed: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	releaseList()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil after the live reflector listed, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("wait did not finish after the live reflector listed")
	}
}

func TestWaitForStoresSync_CanceledContextWinsOverStoppedReflectors(t *testing.T) {
	b := NewBuilder()
	stopCh := make(chan struct{})
	close(stopCh)
	b.setReflectors(startedReflector{reflector: newUnstartedReflector(), stopCh: stopCh})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.beforeSyncPoll = func() { cancel() }

	err := b.WaitForStoresSync(ctx, 5*time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled context to win over stopped reflectors, got %v", err)
	}
}

func TestWaitForStoresSync_StopDuringWait(t *testing.T) {
	b := NewBuilder()
	stopCh := make(chan struct{})
	b.setReflectors(startedReflector{reflector: newUnstartedReflector(), stopCh: stopCh})

	assertSucceedsPromptlyAfter(t, b, func() { close(stopCh) })
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
		assertSucceedsPromptlyAfter(t, b, func() { close(gvkStop) })
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
		assertSucceedsPromptlyAfter(t, b, cancel)
	})
}

func TestWaitForStoresSync_SyncedReflectorIgnoresClosedStop(t *testing.T) {
	b := NewBuilder()
	reflector, stopCh := runReflectorUntilListed(t)
	close(stopCh)
	b.setReflectors(startedReflector{reflector: reflector, stopCh: stopCh})

	if elapsed, err := waitForSync(b, 5*time.Second); err != nil || elapsed > time.Second {
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

// listWatchWithoutWatchList forces the list path. WatchList leaves
// LastSyncResourceVersion empty until the initial-events bookmark.
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

func blockingPodListWatch(release <-chan struct{}) cache.ListerWatcher {
	return listWatchWithoutWatchList{ListWatch: &cache.ListWatch{
		ListFunc: func(metav1.ListOptions) (runtime.Object, error) {
			<-release
			return &v1.PodList{ListMeta: metav1.ListMeta{ResourceVersion: "1"}}, nil
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

func waitForSync(b *Builder, timeout time.Duration) (time.Duration, error) {
	start := time.Now()
	err := b.WaitForStoresSync(context.Background(), timeout)
	return time.Since(start), err
}

func assertSucceedsPromptlyAfter(t *testing.T, b *Builder, closeStop func()) {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- b.WaitForStoresSync(context.Background(), 5*time.Second)
	}()
	time.Sleep(150 * time.Millisecond)
	closeStop()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected stopped reflector to count as done, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("sync did not finish promptly after stop")
	}
}
