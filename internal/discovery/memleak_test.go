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

package discovery

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// appendToMapBuggy reproduces the pre-fix AppendToMap behaviour for comparison.
func appendToMapBuggy(r *CRDiscoverer, gvkps ...groupVersionKindPlural) {
	if r.Map == nil {
		r.Map = map[string]map[string][]kindPlural{}
	}
	if r.GVKToReflectorStopChanMap == nil {
		r.GVKToReflectorStopChanMap = map[string]chan struct{}{}
	}
	for _, gvkp := range gvkps {
		if _, ok := r.Map[gvkp.Group]; !ok {
			r.Map[gvkp.Group] = map[string][]kindPlural{}
		}
		if _, ok := r.Map[gvkp.Group][gvkp.Version]; !ok {
			r.Map[gvkp.Group][gvkp.Version] = []kindPlural{}
		}
		r.Map[gvkp.Group][gvkp.Version] = append(r.Map[gvkp.Group][gvkp.Version], kindPlural{Kind: gvkp.Kind, Plural: gvkp.Plural})
		r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()] = make(chan struct{})
	}
}

func makeGVKPs(n int) []groupVersionKindPlural {
	gvkps := make([]groupVersionKindPlural, n)
	for i := range n {
		gvkps[i] = groupVersionKindPlural{
			GroupVersionKind: schema.GroupVersionKind{
				Group:   fmt.Sprintf("group%d.example.com", i),
				Version: "v1",
				Kind:    fmt.Sprintf("Kind%d", i),
			},
			Plural: fmt.Sprintf("kind%ds", i),
		}
	}
	return gvkps
}

func heapKB() uint64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapInuse / 1024
}

// TestMemoryLeakSimulation runs the buggy and fixed AppendToMap side-by-side
// across many poll cycles and reports heap growth and map entry counts.
func TestMemoryLeakSimulation(t *testing.T) {
	const (
		numGVKs    = 5
		pollCycles = 500
	)

	gvkps := makeGVKPs(numGVKs)

	buggyDiscoverer := &CRDiscoverer{}
	heapBefore := heapKB()
	goroutinesBefore := runtime.NumGoroutine()

	for range pollCycles {
		appendToMapBuggy(buggyDiscoverer, gvkps...)
	}

	heapAfterBuggy := heapKB()
	goroutinesAfterBuggy := runtime.NumGoroutine()

	buggyKindCount := 0
	for _, versions := range buggyDiscoverer.Map {
		for _, kinds := range versions {
			buggyKindCount += len(kinds)
		}
	}
	buggyChannelCount := len(buggyDiscoverer.GVKToReflectorStopChanMap)

	fixedDiscoverer := &CRDiscoverer{}
	heapBeforeFixed := heapKB()

	for range pollCycles {
		fixedDiscoverer.AppendToMap(gvkps...)
	}

	heapAfterFixed := heapKB()

	fixedKindCount := 0
	for _, versions := range fixedDiscoverer.Map {
		for _, kinds := range versions {
			fixedKindCount += len(kinds)
		}
	}
	fixedChannelCount := len(fixedDiscoverer.GVKToReflectorStopChanMap)

	t.Logf("Simulation: %d GVKs × %d poll cycles", numGVKs, pollCycles)
	t.Logf("")
	t.Logf("                       │  BUGGY (pre-fix)  │  FIXED (post-fix)")
	t.Logf("  ─────────────────────┼───────────────────┼──────────────────")
	t.Logf("  Kind entries in map  │  %17d  │  %d", buggyKindCount, fixedKindCount)
	t.Logf("  Stop channels live   │  %17d  │  %d", buggyChannelCount, fixedChannelCount)
	t.Logf("  Heap before (KB)     │  %17d  │  %d", heapBefore, heapBeforeFixed)
	t.Logf("  Heap after  (KB)     │  %17d  │  %d", heapAfterBuggy, heapAfterFixed)
	t.Logf("  Heap growth (KB)     │  %17d  │  %d", int64(heapAfterBuggy)-int64(heapBefore), int64(heapAfterFixed)-int64(heapBeforeFixed)) //nolint:gosec
	t.Logf("  Goroutines before    │  %17d  │  (same baseline)", goroutinesBefore)
	t.Logf("  Goroutines after     │  %17d  │  (no goroutines started)", goroutinesAfterBuggy)

	if buggyKindCount != numGVKs*pollCycles {
		t.Errorf("[buggy] expected %d kind entries (linear growth), got %d", numGVKs*pollCycles, buggyKindCount)
	}
	if fixedKindCount != numGVKs {
		t.Errorf("[fixed] expected exactly %d kind entries (stable), got %d", numGVKs, fixedKindCount)
	}
	if fixedChannelCount != numGVKs {
		t.Errorf("[fixed] expected exactly %d stop channels (stable), got %d", numGVKs, fixedChannelCount)
	}

	buggyGrowth := int64(heapAfterBuggy) - int64(heapBefore)      //nolint:gosec
	fixedGrowth := int64(heapAfterFixed) - int64(heapBeforeFixed) //nolint:gosec
	t.Logf("Heap growth comparison is diagnostic only: buggy=%d KB fixed=%d KB", buggyGrowth, fixedGrowth)
}

// TestGoroutineLeakSimulation verifies that goroutines started with the fixed
// pattern (bridge goroutine selecting on both GVK stop channel and context
// cancellation) exit when the context is cancelled.
func TestGoroutineLeakSimulation(t *testing.T) {
	const (
		numGVKs  = 5
		rebuilds = 20
	)

	var wg sync.WaitGroup

	for range rebuilds {
		ctx, cancel := context.WithCancel(context.Background())
		for range numGVKs {
			gvkStopCh := make(chan struct{})
			stopCh := make(chan struct{})
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer close(stopCh)
				select {
				case <-gvkStopCh:
				case <-ctx.Done():
				}
			}()
			_ = stopCh
		}
		cancel()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Logf("All %d bridge goroutines exited after context cancellation", numGVKs*rebuilds)
	case <-time.After(2 * time.Second):
		t.Errorf("goroutines did not exit within 2s after context cancellation")
	}
}
