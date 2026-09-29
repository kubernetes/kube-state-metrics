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
