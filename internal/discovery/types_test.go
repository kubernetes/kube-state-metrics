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
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TestAppendToMapIdempotency verifies that calling AppendToMap repeatedly with
// the same GVK does not accumulate duplicate kind entries or replace existing
// stop channels.
func TestAppendToMapIdempotency(t *testing.T) {
	const iterations = 10

	gvkp := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{
			Group:   "example.com",
			Version: "v1",
			Kind:    "Foo",
		},
		Plural: "foos",
	}

	r := &CRDiscoverer{}

	r.AppendToMap(gvkp)
	firstCh := r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()]
	if firstCh == nil {
		t.Fatal("expected stop channel to be created on first AppendToMap call")
	}

	for i := 1; i < iterations; i++ {
		r.AppendToMap(gvkp)
	}

	kinds := r.Map[gvkp.Group][gvkp.Version]
	if len(kinds) != 1 {
		t.Errorf("expected exactly 1 kind entry, got %d", len(kinds))
	}

	gotCh := r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()]
	if gotCh != firstCh {
		t.Error("stop channel was replaced on repeated AppendToMap calls")
	}
}

// TestRemoveFromMapClosesChannel verifies that RemoveFromMap closes and removes
// the stop channel for the deleted GVK.
func TestRemoveFromMapClosesChannel(t *testing.T) {
	gvkp := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{
			Group:   "example.com",
			Version: "v1",
			Kind:    "Bar",
		},
		Plural: "bars",
	}

	r := &CRDiscoverer{}
	r.AppendToMap(gvkp)

	ch := r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()]
	if ch == nil {
		t.Fatal("expected stop channel after AppendToMap")
	}

	r.RemoveFromMap(gvkp)

	// Channel must be closed (readable immediately with zero value).
	select {
	case _, open := <-ch:
		if open {
			t.Error("channel should be closed, but received a value")
		}
	default:
		t.Error("channel should be closed but is still blocking")
	}

	// Entry must be removed from the stop channel map.
	if _, exists := r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()]; exists {
		t.Error("stop channel map entry should be deleted after RemoveFromMap")
	}
}

func TestEqualGVKPSet(t *testing.T) {
	a := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"},
		Plural:           "foos",
	}
	b := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Bar"},
		Plural:           "bars",
	}
	a2 := a
	if !equalGVKPSet([]groupVersionKindPlural{a, b}, []groupVersionKindPlural{b, a2}) {
		t.Fatal("expected equal sets ignoring order")
	}
	if equalGVKPSet([]groupVersionKindPlural{a}, []groupVersionKindPlural{b}) {
		t.Fatal("expected different kinds to be unequal")
	}
	aOtherPlural := a
	aOtherPlural.Plural = "fooes"
	if equalGVKPSet([]groupVersionKindPlural{a}, []groupVersionKindPlural{aOtherPlural}) {
		t.Fatal("expected plural changes to be unequal")
	}
}

func TestDiffGVKPsPreservesUnchanged(t *testing.T) {
	keep := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Keep"},
		Plural:           "keeps",
	}
	drop := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Drop"},
		Plural:           "drops",
	}
	add := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v2", Kind: "Keep"},
		Plural:           "keeps",
	}

	removed, added := diffGVKPs(
		[]groupVersionKindPlural{keep, drop},
		[]groupVersionKindPlural{keep, add},
	)
	if len(removed) != 1 || removed[0].Kind != "Drop" {
		t.Fatalf("expected Drop removed, got %#v", removed)
	}
	if len(added) != 1 || added[0].Version != "v2" {
		t.Fatalf("expected v2 Keep added, got %#v", added)
	}
}

// TestCRDUpdateNoopDoesNotCloseStopChannel models the previous bug: status-only
// CRD updates used Remove+Append for identical GVKPs and closed live stop channels.
func TestCRDUpdateNoopDoesNotCloseStopChannel(t *testing.T) {
	gvkp := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"},
		Plural:           "foos",
	}
	r := &CRDiscoverer{}
	if !r.AppendToMap(gvkp) {
		t.Fatal("expected first AppendToMap to change the cache")
	}
	ch := r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()]

	revision := r.cacheRevision
	if r.applyCRDUpdate([]groupVersionKindPlural{gvkp}, []groupVersionKindPlural{gvkp}) {
		t.Fatal("identical GVKPs should be a no-op")
	}
	if r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()] != ch {
		t.Fatal("stop channel changed on no-op diff")
	}
	if r.cacheRevision != revision {
		t.Fatalf("cache revision changed on no-op update: got %d, want %d", r.cacheRevision, revision)
	}
	if r.WasUpdated {
		t.Fatal("WasUpdated set on no-op update")
	}
	select {
	case <-ch:
		t.Fatal("stop channel was closed on no-op update")
	default:
	}
}

func TestCRDUpdateSkipsUnreadableNewGVKPs(t *testing.T) {
	gvkp := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"},
		Plural:           "foos",
	}
	r := &CRDiscoverer{
		GVKToReflectorStopChanMap: make(map[string]chan struct{}),
	}
	if !r.AppendToMap(gvkp) {
		t.Fatal("expected AppendToMap to register GVKP")
	}
	ch := r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()]
	revision := r.cacheRevision

	if r.applyCRDUpdate([]groupVersionKindPlural{gvkp}, nil) {
		t.Fatal("expected unreadable new GVKP set to be ignored")
	}
	if r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()] != ch {
		t.Fatal("stop channel changed on unreadable update")
	}
	if r.cacheRevision != revision {
		t.Fatalf("cache revision changed on unreadable update: got %d, want %d", r.cacheRevision, revision)
	}
	if r.WasUpdated {
		t.Fatal("WasUpdated set on unreadable update")
	}
}

func TestCRDUpdateAddsVersionWithoutClosingKeptChannel(t *testing.T) {
	v1 := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"},
		Plural:           "foos",
	}
	v2 := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v2", Kind: "Foo"},
		Plural:           "foos",
	}
	r := &CRDiscoverer{}
	r.AppendToMap(v1)
	keepCh := r.GVKToReflectorStopChanMap[v1.GroupVersionKind.String()]

	if !r.applyCRDUpdate([]groupVersionKindPlural{v1}, []groupVersionKindPlural{v1, v2}) {
		t.Fatal("expected adding v2 to change the cache")
	}

	if r.GVKToReflectorStopChanMap[v1.GroupVersionKind.String()] != keepCh {
		t.Fatal("v1 stop channel was replaced when adding v2")
	}
	select {
	case <-keepCh:
		t.Fatal("v1 stop channel was closed when adding v2")
	default:
	}
	if r.GVKToReflectorStopChanMap[v2.GroupVersionKind.String()] == nil {
		t.Fatal("expected stop channel for added v2")
	}
	if !r.WasUpdated {
		t.Fatal("expected WasUpdated after real GVK change")
	}
}

func TestCRDDeleteReAddChangesRevision(t *testing.T) {
	gvkp := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"},
		Plural:           "foos",
	}
	r := &CRDiscoverer{}
	r.AppendToMap(gvkp)
	firstRevision := r.cacheRevision
	firstCh := r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()]

	r.RemoveFromMap(gvkp)
	r.AppendToMap(gvkp)

	if r.cacheRevision <= firstRevision {
		t.Fatalf("revision did not advance across delete/re-add: first=%d current=%d", firstRevision, r.cacheRevision)
	}
	if r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()] == firstCh {
		t.Fatal("expected delete/re-add to replace stop channel")
	}
	select {
	case <-firstCh:
	default:
		t.Fatal("expected original stop channel to be closed")
	}
}

func TestAppendToMapReturnsWhetherChanged(t *testing.T) {
	gvkp := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"},
		Plural:           "foos",
	}
	r := &CRDiscoverer{}
	if !r.AppendToMap(gvkp) {
		t.Fatal("first append should report change")
	}
	if r.AppendToMap(gvkp) {
		t.Fatal("duplicate append should not report change")
	}
}

func TestAppendToMapRefreshesPlural(t *testing.T) {
	gvkp := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"},
		Plural:           "foos",
	}
	updated := gvkp
	updated.Plural = "fooes"

	r := &CRDiscoverer{}
	r.AppendToMap(gvkp)
	revision := r.cacheRevision

	if !r.AppendToMap(updated) {
		t.Fatal("plural refresh should report change")
	}
	if got := r.Map[gvkp.Group][gvkp.Version][0].Plural; got != "fooes" {
		t.Fatalf("expected refreshed plural, got %q", got)
	}
	if r.cacheRevision <= revision {
		t.Fatalf("expected revision to advance on plural refresh: before=%d after=%d", revision, r.cacheRevision)
	}
}

func TestRemoveFromMapReturnsWhetherChanged(t *testing.T) {
	gvkp := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"},
		Plural:           "foos",
	}
	r := &CRDiscoverer{}
	r.AppendToMap(gvkp)
	if !r.RemoveFromMap(gvkp) {
		t.Fatal("removing present GVK should report change")
	}
	if r.RemoveFromMap(gvkp) {
		t.Fatal("removing absent GVK should not report change")
	}
}
