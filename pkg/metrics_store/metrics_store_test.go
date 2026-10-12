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
	"fmt"
	"slices"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

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
