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
	"cmp"
	"slices"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

func TestGVKMapsResolveGVK(t *testing.T) {
	type testcase struct {
		desc    string
		gvkmaps *CRDiscoverer
		gvk     schema.GroupVersionKind
		want    []groupVersionKindPlural
	}
	testcases := []testcase{
		{
			desc: "variable version and kind",
			gvkmaps: &CRDiscoverer{
				Map: map[string]map[string][]kindPlural{
					"apps": {
						"v1": {
							kindPlural{
								Kind:   "Deployment",
								Plural: "deployments",
							},
							kindPlural{
								Kind:   "StatefulSet",
								Plural: "statefulsets",
							},
						},
					},
				},
			},
			gvk: schema.GroupVersionKind{Group: "apps", Version: "*", Kind: "*"},
			want: []groupVersionKindPlural{
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
			gvkmaps: &CRDiscoverer{
				Map: map[string]map[string][]kindPlural{
					"testgroup": {
						"v1": {
							kindPlural{
								Kind:   "TestObject1",
								Plural: "testobjects1",
							},
							kindPlural{
								Kind:   "TestObject2",
								Plural: "testobjects2",
							},
						},
						"v1alpha1": {
							kindPlural{
								Kind:   "TestObject1",
								Plural: "testobjects1",
							},
						},
					},
				},
			},
			gvk: schema.GroupVersionKind{Group: "testgroup", Version: "*", Kind: "TestObject1"},
			want: []groupVersionKindPlural{
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
			gvkmaps: &CRDiscoverer{
				Map: map[string]map[string][]kindPlural{
					"testgroup": {
						"v1": {
							kindPlural{
								Kind:   "TestObject1",
								Plural: "testobjects1",
							},
							kindPlural{
								Kind:   "TestObject2",
								Plural: "testobjects2",
							},
						},
						"v1alpha1": {
							kindPlural{
								Kind:   "TestObject1",
								Plural: "testobjects1",
							},
						},
					},
				},
			},
			gvk: schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "*"},
			want: []groupVersionKindPlural{
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
			gvkmaps: &CRDiscoverer{
				Map: map[string]map[string][]kindPlural{
					"testgroup": {
						"v1": {
							kindPlural{
								Kind:   "TestObject1",
								Plural: "testobjects1",
							},
							kindPlural{
								Kind:   "TestObject2",
								Plural: "testobjects2",
							},
						},
						"v1alpha1": {
							kindPlural{
								Kind:   "TestObject1",
								Plural: "testobjects1",
							},
						},
					},
				},
			},
			gvk: schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"},
			want: []groupVersionKindPlural{
				{
					GroupVersionKind: schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"},
					Plural:           "testobjects1",
				},
			},
		},
		{
			desc: "fixed version and kind, no matching cache entry",
			gvkmaps: &CRDiscoverer{
				Map: map[string]map[string][]kindPlural{
					"testgroup": {
						"v1": {
							kindPlural{
								Kind:   "TestObject2",
								Plural: "testobjects2",
							},
						},
					},
				},
			},
			gvk:  schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"},
			want: nil,
		},
	}
	for _, tc := range testcases {
		got, err := tc.gvkmaps.ResolveGVKToGVKPs(tc.gvk)
		if err != nil {
			t.Errorf("testcase: %s: got error %v", tc.desc, err)
		}
		// Sort got and tc.want to ensure the order of the elements.
		slices.SortFunc(got, func(a, b groupVersionKindPlural) int {
			return cmp.Compare(a.String(), b.String())
		})
		slices.SortFunc(tc.want, func(a, b groupVersionKindPlural) int {
			return cmp.Compare(a.String(), b.String())
		})
		if !slices.Equal(got, tc.want) {
			t.Errorf("testcase: %s: got %v, want %v", tc.desc, got, tc.want)
		}
	}
}

func TestResolveGVKToGVKPsMissingWarnDedup(t *testing.T) {
	missing := schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject1"}
	r := &CRDiscoverer{
		Map: map[string]map[string][]kindPlural{
			"testgroup": {"v1": {kindPlural{Kind: "TestObject2", Plural: "testobjects2"}}},
		},
	}

	// First miss records the warning; the key is tracked.
	if !r.markMissingGVKWarned(missing) {
		t.Fatalf("expected first markMissingGVKWarned to report true")
	}
	// Subsequent misses must not re-warn until the GVK is resolved.
	if r.markMissingGVKWarned(missing) {
		t.Errorf("expected repeated markMissingGVKWarned to report false")
	}

	// Once the CRD appears, resolving it clears the warning state.
	r.SafeWrite(func() {
		r.Map["testgroup"]["v1"] = append(r.Map["testgroup"]["v1"], kindPlural{Kind: "TestObject1", Plural: "testobjects1"})
	})
	if _, err := r.ResolveGVKToGVKPs(missing); err != nil {
		t.Fatalf("unexpected error resolving present GVK: %v", err)
	}
	r.SafeRead(func() {
		if _, ok := r.warnedMissingGVKs[missing.String()]; ok {
			t.Errorf("expected warning to be cleared after successful resolution")
		}
	})

	// After the CRD disappears again, a fresh warning is emitted.
	r.SafeWrite(func() {
		r.Map["testgroup"]["v1"] = []kindPlural{{Kind: "TestObject2", Plural: "testobjects2"}}
	})
	if !r.markMissingGVKWarned(missing) {
		t.Errorf("expected warning to fire again after the GVK disappeared")
	}
}

func TestExtractGVKPs(t *testing.T) {
	// crd builds a minimal CRD object, one entry per version. A nil `served`
	// leaves the field out entirely, mimicking an object that predates it.
	crd := func(versions ...interface{}) *unstructured.Unstructured {
		return &unstructured.Unstructured{Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"group": "testgroup",
				"names": map[string]interface{}{
					"kind":   "TestObject",
					"plural": "testobjects",
				},
				"versions": versions,
			},
		}}
	}
	version := func(name string, served interface{}) interface{} {
		v := map[string]interface{}{"name": name}
		if served != nil {
			v["served"] = served
		}
		return v
	}
	gvkp := func(v string) groupVersionKindPlural {
		return groupVersionKindPlural{
			GroupVersionKind: schema.GroupVersionKind{Group: "testgroup", Version: v, Kind: "TestObject"},
			Plural:           "testobjects",
		}
	}

	testcases := []struct {
		desc string
		obj  interface{}
		want []groupVersionKindPlural
	}{
		{
			desc: "all versions served",
			obj:  crd(version("v1", true), version("v1beta1", true)),
			want: []groupVersionKindPlural{gvkp("v1"), gvkp("v1beta1")},
		},
		{
			desc: "non-served versions are skipped",
			obj:  crd(version("v1", true), version("v1beta1", false)),
			want: []groupVersionKindPlural{gvkp("v1")},
		},
		{
			desc: "no served versions",
			obj:  crd(version("v1alpha1", false), version("v1beta1", false)),
			want: nil,
		},
		{
			desc: "missing served field defaults to served",
			obj:  crd(version("v1", nil)),
			want: []groupVersionKindPlural{gvkp("v1")},
		},
		{
			// served is present but unreadable, so whether the API server would
			// serve this version is unknown. Skip it rather than start a
			// reflector that may never succeed.
			desc: "non-boolean served field is skipped",
			obj:  crd(version("v1", "false"), version("v2", true)),
			want: []groupVersionKindPlural{gvkp("v2")},
		},
		{
			desc: "tombstoned object is unwrapped",
			obj:  cache.DeletedFinalStateUnknown{Key: "testobjects.testgroup", Obj: crd(version("v1", true), version("v1beta1", false))},
			want: []groupVersionKindPlural{gvkp("v1")},
		},
		{
			desc: "unexpected type yields no GVKPs",
			obj:  "not-a-crd",
			want: nil,
		},
		// Every field below is required on apiextensions.k8s.io/v1, so the API
		// server would reject these. They are covered because extractGVKPs runs
		// on the informer goroutine, where a failed type assertion is an
		// unrecovered panic rather than a skipped object.
		{
			desc: "no spec",
			obj:  &unstructured.Unstructured{Object: map[string]interface{}{}},
			want: nil,
		},
		{
			desc: "spec is not an object",
			obj:  &unstructured.Unstructured{Object: map[string]interface{}{"spec": "nope"}},
			want: nil,
		},
		{
			desc: "no group",
			obj: &unstructured.Unstructured{Object: map[string]interface{}{
				"spec": map[string]interface{}{"names": map[string]interface{}{"kind": "K", "plural": "ks"}},
			}},
			want: nil,
		},
		{
			desc: "no names",
			obj: &unstructured.Unstructured{Object: map[string]interface{}{
				"spec": map[string]interface{}{"group": "g"},
			}},
			want: nil,
		},
		{
			desc: "no versions",
			obj: &unstructured.Unstructured{Object: map[string]interface{}{
				"spec": map[string]interface{}{
					"group": "g",
					"names": map[string]interface{}{"kind": "K", "plural": "ks"},
				},
			}},
			want: nil,
		},
		{
			desc: "a malformed version is skipped, the rest are kept",
			obj: &unstructured.Unstructured{Object: map[string]interface{}{
				"spec": map[string]interface{}{
					"group": "testgroup",
					"names": map[string]interface{}{"kind": "TestObject", "plural": "testobjects"},
					"versions": []interface{}{
						"not-an-object",
						map[string]interface{}{"served": true},
						map[string]interface{}{"name": "v1", "served": true},
					},
				},
			}},
			want: []groupVersionKindPlural{gvkp("v1")},
		},
	}
	for _, tc := range testcases {
		got := extractGVKPs(tc.obj)
		if !slices.Equal(got, tc.want) {
			t.Errorf("testcase: %s: got %v, want %v", tc.desc, got, tc.want)
		}
	}
}

// The CRD informer's handlers write the cache while the discovery poll resolves
// configured GVKs against it. Both must go through the lock: a concurrent map
// read and write is a fatal runtime throw, not a recoverable panic. Run under
// -race, which CI already does for unit tests.
func TestResolveGVKToGVKPsIsRaceFree(t *testing.T) {
	r := &CRDiscoverer{}
	gvk := schema.GroupVersionKind{Group: "testgroup", Version: "v1", Kind: "TestObject"}
	gvkp := groupVersionKindPlural{GroupVersionKind: gvk, Plural: "testobjects"}

	const iterations = 200
	var wg sync.WaitGroup

	// Stand in for the informer's event handlers.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			r.SafeWrite(func() { r.AppendToMap(gvkp) })
			r.SafeWrite(func() { r.RemoveFromMap(gvkp) })
		}
	}()

	// Stand in for the discovery poll resolving configured resources, covering
	// the fixed lookup and all three wildcard paths.
	for _, q := range []schema.GroupVersionKind{
		{Group: "testgroup", Version: "v1", Kind: "TestObject"},
		{Group: "testgroup", Version: "v1", Kind: "*"},
		{Group: "testgroup", Version: "*", Kind: "TestObject"},
		{Group: "testgroup", Version: "*", Kind: "*"},
	} {
		wg.Add(1)
		go func(q schema.GroupVersionKind) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				if _, err := r.ResolveGVKToGVKPs(q); err != nil {
					t.Errorf("resolving %v: %v", q, err)
					return
				}
			}
		}(q)
	}

	wg.Wait()
}
