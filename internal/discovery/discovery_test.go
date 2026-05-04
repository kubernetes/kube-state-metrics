/*
	Copyright 2023 The Kubernetes Authors All rights reserved.
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
	"reflect"
	"sort"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// newTestCRDiscoverer creates a CRDiscoverer with no-op metrics for testing.
func newTestCRDiscoverer() *CRDiscoverer {
	return NewCRDiscoverer(
		prometheus.NewCounter(prometheus.CounterOpts{Name: "test_add"}),
		prometheus.NewCounter(prometheus.CounterOpts{Name: "test_update"}),
		prometheus.NewCounter(prometheus.CounterOpts{Name: "test_delete"}),
		prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_count"}),
	)
}

// makeResources is a small helper for building a single-entry resource slice.
func makeResources(group, version, kind, plural string) []DiscoveredResource {
	return []DiscoveredResource{
		{
			GroupVersionKind: schema.GroupVersionKind{Group: group, Version: version, Kind: kind},
			Plural:           plural,
		},
	}
}

func TestResolve(t *testing.T) {
	type testcase struct {
		desc      string
		resources map[string][]DiscoveredResource // map[sourceID] -> []resources
		gvk       schema.GroupVersionKind
		want      []DiscoveredResource
	}
	testcases := []testcase{
		{
			desc: "variable version and kind",
			resources: map[string][]DiscoveredResource{
				"crd:deployments.apps": {
					{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "apps",
							Version: "v1",
							Kind:    "Deployment",
						},
						Plural: "deployments",
					},
					{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "apps",
							Version: "v1",
							Kind:    "StatefulSet",
						},
						Plural: "statefulsets",
					},
				},
			},
			gvk: schema.GroupVersionKind{Group: "apps", Version: "*", Kind: "*"},
			want: []DiscoveredResource{
				{
					GroupVersionKind: schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
					Plural:           "deployments",
				},
				{
					GroupVersionKind: schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"},
					Plural:           "statefulsets",
				},
			},
		},
		{
			desc: "variable version",
			resources: map[string][]DiscoveredResource{
				"crd:testobjects.testgroup": {
					{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "testgroup",
							Version: "v1",
							Kind:    "TestObject1",
						},
						Plural: "testobjects1",
					},
					{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "testgroup",
							Version: "v1",
							Kind:    "TestObject2",
						},
						Plural: "testobjects2",
					},
					{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "testgroup",
							Version: "v1alpha1",
							Kind:    "TestObject1",
						},
						Plural: "testobjects1",
					},
				},
			},
			gvk: schema.GroupVersionKind{Group: "testgroup", Version: "*", Kind: "TestObject1"},
			want: []DiscoveredResource{
				{
					GroupVersionKind: schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"},
					Plural:           "testobjects1",
				},
				{
					GroupVersionKind: schema.GroupVersionKind{Group: "testgroup", Version: "v1alpha1", Kind: "TestObject1"},
					Plural:           "testobjects1",
				},
			},
		},
		{
			desc: "variable kind",
			resources: map[string][]DiscoveredResource{
				"crd:testobjects.testgroup": {
					{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "testgroup",
							Version: "v1",
							Kind:    "TestObject1",
						},
						Plural: "testobjects1",
					},
					{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "testgroup",
							Version: "v1",
							Kind:    "TestObject2",
						},
						Plural: "testobjects2",
					},
					{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "testgroup",
							Version: "v1alpha1",
							Kind:    "TestObject1",
						},
						Plural: "testobjects1",
					},
				},
			},
			gvk: schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "*"},
			want: []DiscoveredResource{
				{
					GroupVersionKind: schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"},
					Plural:           "testobjects1",
				},
				{
					GroupVersionKind: schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject2"},
					Plural:           "testobjects2",
				},
			},
		},
		{
			desc: "fixed version and kind",
			resources: map[string][]DiscoveredResource{
				"crd:testobjects.testgroup": {
					{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "testgroup",
							Version: "v1",
							Kind:    "TestObject1",
						},
						Plural: "testobjects1",
					},
					{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "testgroup",
							Version: "v1",
							Kind:    "TestObject2",
						},
						Plural: "testobjects2",
					},
					{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "testgroup",
							Version: "v1alpha1",
							Kind:    "TestObject1",
						},
						Plural: "testobjects1",
					},
				},
			},
			gvk: schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"},
			want: []DiscoveredResource{
				{
					GroupVersionKind: schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"},
					Plural:           "testobjects1",
				},
			},
		},
		{
			desc: "fixed version and kind, no matching cache entry",
			resources: map[string][]DiscoveredResource{
				"crd:testobjects.testgroup": {
					{
						GroupVersionKind: schema.GroupVersionKind{
							Group:   "testgroup",
							Version: "v1",
							Kind:    "TestObject2",
						},
						Plural: "testobjects2",
					},
				},
			},
			gvk:  schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"},
			want: nil,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			discoverer := newTestCRDiscoverer()
			for sourceID, resources := range tc.resources {
				discoverer.UpdateSource(sourceID, resources)
			}

			got, err := discoverer.Resolve(tc.gvk)
			if err != nil {
				t.Errorf("got error %v", err)
			}
			// Iteration order over the source map is undefined.
			sort.Slice(got, func(i, j int) bool { return got[i].String() < got[j].String() })
			sort.Slice(tc.want, func(i, j int) bool { return tc.want[i].String() < tc.want[j].String() })
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUpdateSourceAndDeleteSource(t *testing.T) {
	d := newTestCRDiscoverer()
	gvk := schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"}

	d.UpdateSource("crd:testobjects.testgroup", makeResources(gvk.Group, gvk.Version, gvk.Kind, "testobjects1"))

	got, err := d.Resolve(gvk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(got))
	}

	stopChan, ok := d.GetStopChan(gvk)
	if !ok {
		t.Fatal("expected stop channel to exist")
	}

	d.DeleteSource("crd:testobjects.testgroup")

	got, err = d.Resolve(gvk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 resources after delete, got %d", len(got))
	}

	select {
	case <-stopChan:
	default:
		t.Fatal("expected stop channel to be closed after DeleteSource")
	}
}

func TestUpdateSourceNilSkipsUpdate(t *testing.T) {
	d := newTestCRDiscoverer()
	gvk := schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"}

	d.UpdateSource("crd:testobjects.testgroup", makeResources(gvk.Group, gvk.Version, gvk.Kind, "testobjects1"))
	d.UpdateSource("crd:testobjects.testgroup", nil)

	got, err := d.Resolve(gvk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 resource (nil should skip update), got %d", len(got))
	}
}

func TestUpdateSourceEmptyRemovesResources(t *testing.T) {
	d := newTestCRDiscoverer()
	gvk := schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"}
	src := "crd:testobjects.testgroup"

	d.UpdateSource(src, makeResources(gvk.Group, gvk.Version, gvk.Kind, "testobjects1"))
	stopChan, _ := d.GetStopChan(gvk)

	d.UpdateSource(src, []DiscoveredResource{})

	got, err := d.Resolve(gvk)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 resources, got %d", len(got))
	}
	// Empty slice should also stop reflectors, just like DeleteSource.
	select {
	case <-stopChan:
	default:
		t.Fatal("expected stop channel to be closed after empty UpdateSource")
	}
}

// TestUpdateSourceEmptyOnExistingSourceCountsAsDelete verifies that
// UpdateSource(id, []) bumps DeleteEvents (not UpdateEvents) when the source
// existed, since semantically the operation is a deletion.
func TestUpdateSourceEmptyOnExistingSourceCountsAsDelete(t *testing.T) {
	d := newTestCRDiscoverer()
	src := "crd:testobjects.testgroup"

	d.UpdateSource(src, makeResources("testgroup", "v1", "TestObject1", "testobjects1"))
	addBefore := testutil.ToFloat64(d.AddEvents)
	updateBefore := testutil.ToFloat64(d.UpdateEvents)
	deleteBefore := testutil.ToFloat64(d.DeleteEvents)

	d.UpdateSource(src, []DiscoveredResource{})

	if got := testutil.ToFloat64(d.AddEvents) - addBefore; got != 0 {
		t.Errorf("AddEvents delta = %v, want 0", got)
	}
	if got := testutil.ToFloat64(d.UpdateEvents) - updateBefore; got != 0 {
		t.Errorf("UpdateEvents delta = %v, want 0", got)
	}
	if got := testutil.ToFloat64(d.DeleteEvents) - deleteBefore; got != 1 {
		t.Errorf("DeleteEvents delta = %v, want 1", got)
	}
}

// TestUpdateSourceReplaceClosesOldStopChans verifies that replacing the
// resource set for an existing source stops the old reflectors before the
// new ones take over.
func TestUpdateSourceReplaceClosesOldStopChans(t *testing.T) {
	d := newTestCRDiscoverer()
	src := "crd:testobjects.testgroup"
	oldGVK := schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "Old"}
	newGVK := schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "New"}

	d.UpdateSource(src, makeResources(oldGVK.Group, oldGVK.Version, oldGVK.Kind, "olds"))
	oldStop, ok := d.GetStopChan(oldGVK)
	if !ok {
		t.Fatal("expected stop channel for old GVK")
	}

	d.UpdateSource(src, makeResources(newGVK.Group, newGVK.Version, newGVK.Kind, "news"))

	select {
	case <-oldStop:
	default:
		t.Fatal("expected old stop channel to be closed after replacement")
	}
	if _, ok := d.GetStopChan(newGVK); !ok {
		t.Fatal("expected stop channel for new GVK")
	}
}

func TestUpdateSourceEmptySourceIDIsNoop(t *testing.T) {
	d := newTestCRDiscoverer()
	d.UpdateSource("", makeResources("g", "v1", "K", "ks"))
	if d.CheckAndResetUpdated() {
		t.Fatal("expected no update from empty sourceID")
	}
}

func TestDeleteSourceEmptySourceIDIsNoop(t *testing.T) {
	d := newTestCRDiscoverer()
	d.DeleteSource("")
	if d.CheckAndResetUpdated() {
		t.Fatal("expected no update from empty sourceID")
	}
}

func TestCheckAndResetUpdated(t *testing.T) {
	d := newTestCRDiscoverer()

	if d.CheckAndResetUpdated() {
		t.Fatal("expected wasUpdated to be false initially")
	}

	d.UpdateSource("crd:testobjects.testgroup", makeResources("testgroup", "v1", "TestObject1", "testobjects1"))

	if !d.CheckAndResetUpdated() {
		t.Fatal("expected wasUpdated to be true after UpdateSource")
	}
	if d.CheckAndResetUpdated() {
		t.Fatal("expected wasUpdated to be false after CheckAndResetUpdated")
	}
}
