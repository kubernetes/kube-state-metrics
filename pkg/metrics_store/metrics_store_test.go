/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package metricsstore

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	clientfeatures "k8s.io/client-go/features"
	clientfeaturestesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
)

func TestObjectsSameNameDifferentNamespaces(t *testing.T) {
	serviceIDs := []string{"a", "b"}

	genFunc := func(obj any) []metric.FamilyInterface {
		o, err := meta.Accessor(obj)
		if err != nil {
			t.Fatal(err)
		}

		metricFamily := metric.Family{
			Name: "kube_service_info",
			Metrics: []*metric.Metric{
				{
					LabelKeys:   []string{"uid"},
					LabelValues: []string{string(o.GetUID())},
					Value:       float64(1),
				},
			},
		}

		return []metric.FamilyInterface{&metricFamily}
	}

	ms := NewMetricsStore([]string{"Information about service."}, genFunc)

	for _, id := range serviceIDs {
		s := v1.Service{
			Name:      "service",
			Namespace: id,
			UID:       types.UID(id),
		}

		err := ms.Add(&s)
		if err != nil {
			t.Fatal(err)
		}
	}

	w := strings.Builder{}
	mw := NewMetricsWriter("test", ms)
	err := mw.WriteAll(&w)
	if err != nil {
		t.Fatalf("failed to write metrics: %v", err)
	}
	m := w.String()

	for _, id := range serviceIDs {
		if !strings.Contains(m, fmt.Sprintf("uid=\"%v\"", id)) {
			t.Fatalf("expected to find metric with uid %v", id)
		}
	}
}

func TestMetricsStoreResourceVersion(t *testing.T) {
	ms := NewMetricsStore([]string{}, func(_ any) []metric.FamilyInterface { return nil })

	// Test Add
	s1 := &v1.Service{
		UID:             "uid1",
		ResourceVersion: "123",
	}
	if err := ms.Add(s1); err != nil {
		t.Fatal(err)
	}
	if rv := ms.LastStoreSyncResourceVersion(); rv != "123" {
		t.Fatalf("expected resource version 123, got %v", rv)
	}

	// Test Update
	s1.ResourceVersion = "124"
	if err := ms.Update(s1); err != nil {
		t.Fatal(err)
	}
	if rv := ms.LastStoreSyncResourceVersion(); rv != "124" {
		t.Fatalf("expected resource version 124, got %v", rv)
	}

	// Test Delete
	s1.ResourceVersion = "125"
	if err := ms.Delete(s1); err != nil {
		t.Fatal(err)
	}
	if rv := ms.LastStoreSyncResourceVersion(); rv != "125" {
		t.Fatalf("expected resource version 125, got %v", rv)
	}

	// Test Bookmark
	ms.Bookmark("126")
	if rv := ms.LastStoreSyncResourceVersion(); rv != "126" {
		t.Fatalf("expected resource version 126, got %v", rv)
	}

	// Test Replace
	if err := ms.Replace([]any{}, "127"); err != nil {
		t.Fatal(err)
	}
	s2 := &v1.Service{
		UID:             "uid2",
		ResourceVersion: "1",
	}
	if err := ms.Replace([]any{s2}, "127"); err != nil {
		t.Fatal(err)
	}
	if rv := ms.LastStoreSyncResourceVersion(); rv != "127" {
		t.Fatalf("expected resource version 127, got %v", rv)
	}
}

func TestMetricsStoreTransformerDuringWatchList(t *testing.T) {
	clientfeaturestesting.SetFeatureDuringTest(t, clientfeatures.WatchListClient, true)

	var calls atomic.Int64
	ms := NewMetricsStore([]string{"Test info."}, func(obj any) []metric.FamilyInterface {
		calls.Add(1)
		o, _ := meta.Accessor(obj)
		return []metric.FamilyInterface{&metric.Family{
			Name: "kube_test_info",
			Metrics: []*metric.Metric{{
				LabelKeys:   []string{"uid"},
				LabelValues: []string{string(o.GetUID())},
				Value:       1,
			}},
		}}
	})

	w := watch.NewFake()
	lw := &cache.ListWatch{
		ListWithContextFunc: func(context.Context, metav1.ListOptions) (runtime.Object, error) {
			return nil, errors.New("unexpected list, the initial sync should use WatchList")
		},
		WatchFuncWithContext: func(context.Context, metav1.ListOptions) (watch.Interface, error) {
			return w, nil
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	r := cache.NewReflectorWithOptions(lw, &v1.Pod{}, ms, cache.ReflectorOptions{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		r.RunWithContext(ctx)
	}()
	defer func() {
		cancel()
		<-done
	}()

	w.Add(&v1.Pod{Name: "pod", Namespace: "default", UID: "uid-p", ResourceVersion: "1"})

	// The metrics are generated when the object reaches the temporary store,
	// before the initial events end.
	if err := wait.PollUntilContextTimeout(ctx, 10*time.Millisecond, 5*time.Second, true, func(context.Context) (bool, error) {
		return calls.Load() == 1, nil
	}); err != nil {
		t.Fatalf("expected the metrics to be generated before the initial events end: %v", err)
	}

	w.Action(watch.Bookmark, &v1.Pod{
		ResourceVersion: "2",
		Annotations:     map[string]string{metav1.InitialEventsAnnotationKey: "true"},
	})
	if err := wait.PollUntilContextTimeout(ctx, 10*time.Millisecond, wait.ForeverTestTimeout, true, func(context.Context) (bool, error) {
		return ms.LastStoreSyncResourceVersion() == "2", nil
	}); err != nil {
		t.Fatalf("initial sync did not complete: %v", err)
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("expected the metrics to be generated once, got %d calls", n)
	}

	out := strings.Builder{}
	if err := NewMetricsWriter("test", ms).WriteAll(&out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `uid="uid-p"`) {
		t.Errorf("expected the metric of the synced object to be written, got %q", out.String())
	}
}

func TestReplaceRemovesUnlistedObjects(t *testing.T) {
	ms := NewMetricsStore([]string{}, func(_ any) []metric.FamilyInterface { return nil })

	for _, uid := range []types.UID{"keep", "drop"} {
		if err := ms.Add(&v1.Service{UID: uid}); err != nil {
			t.Fatal(err)
		}
	}
	// A WatchList initial sync passes objects already rendered by Transformer.
	list := []any{&v1.Service{UID: "keep"}, &v1.Service{UID: "new"}, &renderedObject{uid: "rendered"}}
	if err := ms.Replace(list, "1"); err != nil {
		t.Fatal(err)
	}

	got := map[types.UID]struct{}{}
	ms.metrics.Range(func(key, _ any) bool {
		got[key.(types.UID)] = struct{}{}
		return true
	})
	want := map[types.UID]struct{}{"keep": {}, "new": {}, "rendered": {}}
	if !maps.Equal(got, want) {
		t.Fatalf("expected store keys %v, got %v", want, got)
	}
}

// A relist of unchanged objects must never expose a partially filled store to
// a concurrent scrape. The metrics generator runs inside Replace for every
// object, so counting the store from there checks each intermediate state.
func TestReplaceDoesNotExposePartialStore(t *testing.T) {
	const nObjects = 100

	var ms *MetricsStore
	checking := false
	ms = NewMetricsStore([]string{}, func(_ any) []metric.FamilyInterface {
		if !checking {
			return nil
		}
		n := 0
		ms.metrics.Range(func(_, _ any) bool {
			n++
			return true
		})
		if n != nObjects {
			t.Fatalf("expected %d objects during relist, observed %d", nObjects, n)
		}
		return nil
	})

	list := make([]any, nObjects)
	for i := range list {
		list[i] = &v1.Service{UID: types.UID(fmt.Sprintf("uid-%d", i))}
	}
	if err := ms.Replace(list, "1"); err != nil {
		t.Fatal(err)
	}

	checking = true
	if err := ms.Replace(list, "2"); err != nil {
		t.Fatal(err)
	}
}

// Objects missing from a relist must be gone before the listed objects are
// upserted, so a concurrent scrape never sees a deleted object next to the ones
// created since.
func TestReplaceDeletesUnlistedObjectsFirst(t *testing.T) {
	var ms *MetricsStore
	checking := false
	ms = NewMetricsStore([]string{}, func(_ any) []metric.FamilyInterface {
		if !checking {
			return nil
		}
		if _, ok := ms.metrics.Load(types.UID("drop")); ok {
			t.Fatal("expected the unlisted object to be deleted before the listed ones are upserted")
		}
		return nil
	})

	for _, uid := range []types.UID{"keep", "drop"} {
		if err := ms.Add(&v1.Service{UID: uid}); err != nil {
			t.Fatal(err)
		}
	}

	checking = true
	if err := ms.Replace([]any{&v1.Service{UID: "keep"}, &v1.Service{UID: "new"}}, "1"); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkAdd(b *testing.B) {
	const nFamilies = 50

	headers := make([]string, nFamilies)
	genFunc := func(obj any) []metric.FamilyInterface {
		o, _ := meta.Accessor(obj)
		families := make([]metric.FamilyInterface, nFamilies)
		for j := range nFamilies {
			families[j] = &metric.Family{
				Name: fmt.Sprintf("kube_bench_metric_%d", j),
				Metrics: []*metric.Metric{
					{
						LabelKeys:   []string{"namespace", "service", "uid"},
						LabelValues: []string{o.GetNamespace(), o.GetName(), string(o.GetUID())},
						Value:       float64(j),
					},
				},
			}
		}
		return families
	}

	store := NewMetricsStore(headers, genFunc)
	svc := &v1.Service{
		UID:       types.UID("uid-0"),
		Name:      "svc-0",
		Namespace: "default",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := store.Add(svc); err != nil {
			b.Fatal(err)
		}
	}
}

func TestMetricsStoreObjectLimit(t *testing.T) {
	genFunc := func(obj any) []metric.FamilyInterface {
		o, err := meta.Accessor(obj)
		if err != nil {
			t.Fatal(err)
		}
		return []metric.FamilyInterface{&metric.Family{
			Name: "kube_service_info",
			Metrics: []*metric.Metric{{
				LabelKeys:   []string{"uid", "resource_version"},
				LabelValues: []string{string(o.GetUID()), o.GetResourceVersion()},
				Value:       1,
			}},
		}}
	}
	service := func(id, rv string) *v1.Service {
		return &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: "default", UID: types.UID(id), ResourceVersion: rv}}
	}
	stored := func(ms *MetricsStore) []string {
		var uids []string
		ms.metrics.Range(func(k, _ any) bool {
			uids = append(uids, string(k.(types.UID)))
			return true
		})
		slices.Sort(uids)
		return uids
	}

	dropped := 0
	ms := NewMetricsStore([]string{"Information about service."}, genFunc, WithObjectLimit(2, func() { dropped++ }))

	// The initial list is capped.
	if err := ms.Replace([]any{service("a", "1"), service("b", "1"), service("c", "1")}, "1"); err != nil {
		t.Fatal(err)
	}
	if got := stored(ms); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("after Replace: got %v, want [a b]", got)
	}

	// New objects from the watch are not stored once the limit is reached, and
	// neither are updates to objects that were dropped.
	for _, obj := range []*v1.Service{service("d", "2"), service("c", "2")} {
		if err := ms.Add(obj); err != nil {
			t.Fatal(err)
		}
	}
	if err := ms.Update(service("c", "3")); err != nil {
		t.Fatal(err)
	}
	if got := stored(ms); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("after watch events: got %v, want [a b]", got)
	}
	if dropped != 4 {
		t.Fatalf("got %d dropped objects, want 4", dropped)
	}

	// Updates to stored objects are still applied.
	if err := ms.Update(service("a", "4")); err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := NewMetricsWriter("services", ms).WriteAll(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `uid="a",resource_version="4"`) {
		t.Fatalf("update to stored object not applied:\n%s", buf.String())
	}

	// Deleting an object frees a slot for a new one.
	if err := ms.Delete(service("b", "5")); err != nil {
		t.Fatal(err)
	}
	if err := ms.Add(service("d", "6")); err != nil {
		t.Fatal(err)
	}
	if got := stored(ms); !slices.Equal(got, []string{"a", "d"}) {
		t.Fatalf("after Delete and Add: got %v, want [a d]", got)
	}

	// Deleting an object that was never stored does not free a slot.
	if err := ms.Delete(service("c", "7")); err != nil {
		t.Fatal(err)
	}
	if err := ms.Add(service("e", "8")); err != nil {
		t.Fatal(err)
	}
	if got := stored(ms); !slices.Equal(got, []string{"a", "d"}) {
		t.Fatalf("after deleting a dropped object: got %v, want [a d]", got)
	}
}

// A relist frees the slots of the objects it no longer contains before it
// admits new ones, so the limit holds the listed objects that were kept.
func TestReplaceObjectLimitAfterChurn(t *testing.T) {
	var dropped int
	ms := NewMetricsStore([]string{}, func(any) []metric.FamilyInterface { return nil }, WithObjectLimit(2, func() { dropped++ }))

	for _, uid := range []types.UID{"a", "b"} {
		if err := ms.Add(&v1.Service{UID: uid}); err != nil {
			t.Fatal(err)
		}
	}
	list := []any{&v1.Service{UID: "b"}, &v1.Service{UID: "c"}, &v1.Service{UID: "d"}}
	if err := ms.Replace(list, "1"); err != nil {
		t.Fatal(err)
	}

	got := map[types.UID]struct{}{}
	ms.metrics.Range(func(key, _ any) bool {
		got[key.(types.UID)] = struct{}{}
		return true
	})
	want := map[types.UID]struct{}{"b": {}, "c": {}}
	if !maps.Equal(got, want) {
		t.Fatalf("expected store keys %v, got %v", want, got)
	}
	if n := ms.objectCount.Load(); n != 2 {
		t.Fatalf("expected an object count of 2, got %d", n)
	}
	if dropped != 1 {
		t.Fatalf("expected 1 dropped object, got %d", dropped)
	}
}
