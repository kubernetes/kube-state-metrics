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

func TestResolve(t *testing.T) {
	type testcase struct {
		desc      string
		resources map[string][]*DiscoveredResource // map[sourceID] -> []resources
		gvk       schema.GroupVersionKind
		want      []DiscoveredResource
	}
	testcases := []testcase{
		{
			desc: "variable version and kind",
			resources: map[string][]*DiscoveredResource{
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
			resources: map[string][]*DiscoveredResource{
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
			resources: map[string][]*DiscoveredResource{
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
			resources: map[string][]*DiscoveredResource{
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
			resources: map[string][]*DiscoveredResource{
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
			// Populate the discoverer with test data
			for sourceID, resources := range tc.resources {
				discoverer.UpdateSource(sourceID, resources)
			}

			got, err := discoverer.Resolve(tc.gvk)
			if err != nil {
				t.Errorf("got error %v", err)
			}
			// Sort got and tc.want to ensure the order of the elements.
			sort.Slice(got, func(i, j int) bool {
				return got[i].String() < got[j].String()
			})
			sort.Slice(tc.want, func(i, j int) bool {
				return tc.want[i].String() < tc.want[j].String()
			})
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUpdateSourceAndDeleteSource(t *testing.T) {
	discoverer := newTestCRDiscoverer()

	// Add resources for a source
	resources := []*DiscoveredResource{
		{
			GroupVersionKind: schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"},
			Plural:           "testobjects1",
		},
	}
	discoverer.UpdateSource("crd:testobjects.testgroup", resources)
	// Verify resource is present
	got, err := discoverer.Resolve(schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(got))
	}

	// Get stop channel
	stopChan, ok := discoverer.GetStopChan(schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"})
	if !ok {
		t.Fatal("expected stop channel to exist")
	}

	// Delete the source
	discoverer.DeleteSource("crd:testobjects.testgroup")

	// Verify resource is removed
	got, err = discoverer.Resolve(schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 resources, got %d", len(got))
	}

	// Verify stop channel is closed
	select {
	case <-stopChan:
		// expected - channel is closed
	default:
		t.Fatal("expected stop channel to be closed")
	}
}

func TestUpdateSourceNilSkipsUpdate(t *testing.T) {
	discoverer := newTestCRDiscoverer()

	// Add initial resources
	resources := []*DiscoveredResource{
		{
			GroupVersionKind: schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"},
			Plural:           "testobjects1",
		},
	}
	discoverer.UpdateSource("crd:testobjects.testgroup", resources)
	// Update with nil (simulating skipping update)
	discoverer.UpdateSource("crd:testobjects.testgroup", nil)

	// Verify original resource is still present
	got, err := discoverer.Resolve(schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 resource (nil should skip update), got %d", len(got))
	}
}

func TestUpdateSourceEmptyRemovesResources(t *testing.T) {
	discoverer := newTestCRDiscoverer()

	// Add initial resources
	resources := []*DiscoveredResource{
		{
			GroupVersionKind: schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"},
			Plural:           "testobjects1",
		},
	}
	discoverer.UpdateSource("crd:testobjects.testgroup", resources)

	// Update with empty slice (simulating removal)
	discoverer.UpdateSource("crd:testobjects.testgroup", []*DiscoveredResource{})

	// Verify resource is removed
	got, err := discoverer.Resolve(schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 resources, got %d", len(got))
	}
}

func TestCheckAndResetUpdated(t *testing.T) {
	discoverer := newTestCRDiscoverer()

	// Initially not updated
	if discoverer.CheckAndResetUpdated() {
		t.Fatal("expected wasUpdated to be false initially")
	}

	// Add a resource
	discoverer.UpdateSource("crd:testobjects.testgroup", []*DiscoveredResource{
		{
			GroupVersionKind: schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"},
			Plural:           "testobjects1",
		},
	})

	// Should be updated now
	if !discoverer.CheckAndResetUpdated() {
		t.Fatal("expected wasUpdated to be true after UpdateSource")
	}

	// Should be reset
	if discoverer.CheckAndResetUpdated() {
		t.Fatal("expected wasUpdated to be false after CheckAndResetUpdated")
	}
}
