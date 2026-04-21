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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// newCRD builds a minimal CRD-shaped *unstructured.Unstructured for tests.
func newCRD(name, group, kind, plural string, versions ...string) *unstructured.Unstructured {
	versionEntries := make([]interface{}, 0, len(versions))
	for _, v := range versions {
		versionEntries = append(versionEntries, map[string]interface{}{"name": v})
	}
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name": name,
			},
			"spec": map[string]interface{}{
				"group": group,
				"names": map[string]interface{}{
					"kind":   kind,
					"plural": plural,
				},
				"versions": versionEntries,
			},
		},
	}
}

func TestCRDExtractor_SourceID(t *testing.T) {
	e := &crdExtractor{}

	t.Run("valid CRD", func(t *testing.T) {
		u := newCRD("foos.example.com", "example.com", "Foo", "foos", "v1")
		got := e.SourceID(u)
		if got != "crd:foos.example.com" {
			t.Fatalf("SourceID = %q, want %q", got, "crd:foos.example.com")
		}
	})

	t.Run("wrong type returns empty", func(t *testing.T) {
		got := e.SourceID("not an unstructured")
		if got != "" {
			t.Fatalf("SourceID with wrong type = %q, want empty", got)
		}
	})

	t.Run("nil obj returns empty", func(t *testing.T) {
		got := e.SourceID(nil)
		if got != "" {
			t.Fatalf("SourceID(nil) = %q, want empty", got)
		}
	})

	t.Run("empty name returns empty", func(t *testing.T) {
		u := newCRD("", "example.com", "Foo", "foos", "v1")
		got := e.SourceID(u)
		if got != "" {
			t.Fatalf("SourceID with empty name = %q, want empty", got)
		}
	})
}

func TestCRDExtractor_ExtractGVKs(t *testing.T) {
	e := &crdExtractor{}

	t.Run("single version", func(t *testing.T) {
		u := newCRD("foos.example.com", "example.com", "Foo", "foos", "v1")
		got := e.ExtractGVKs(u)
		want := []DiscoveredResource{
			{
				GroupVersionKind: schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"},
				Plural:           "foos",
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ExtractGVKs = %#v, want %#v", got, want)
		}
	})

	t.Run("multiple versions", func(t *testing.T) {
		u := newCRD("foos.example.com", "example.com", "Foo", "foos", "v1", "v1beta1", "v1alpha1")
		got := e.ExtractGVKs(u)
		if len(got) != 3 {
			t.Fatalf("ExtractGVKs returned %d resources, want 3", len(got))
		}
		// Sort for deterministic comparison.
		sort.Slice(got, func(i, j int) bool { return got[i].Version < got[j].Version })
		wantVersions := []string{"v1", "v1alpha1", "v1beta1"}
		for i, r := range got {
			if r.Group != "example.com" || r.Kind != "Foo" || r.Plural != "foos" {
				t.Errorf("entry %d has wrong metadata: %+v", i, r)
			}
			if r.Version != wantVersions[i] {
				t.Errorf("entry %d version = %q, want %q", i, r.Version, wantVersions[i])
			}
		}
	})

	t.Run("wrong top-level type returns nil", func(t *testing.T) {
		for _, in := range []interface{}{"nope", nil} {
			if got := e.ExtractGVKs(in); got != nil {
				t.Errorf("ExtractGVKs(%T) = %v, want nil", in, got)
			}
		}
	})

	// Each case here mutates a freshly-built valid CRD into a malformed one
	// and expects ExtractGVKs to return nil.
	malformed := []struct {
		name   string
		mutate func(*unstructured.Unstructured)
	}{
		{
			name:   "missing spec",
			mutate: func(u *unstructured.Unstructured) { delete(u.Object, "spec") },
		},
		{
			name: "missing spec.group",
			mutate: func(u *unstructured.Unstructured) {
				delete(u.Object["spec"].(map[string]interface{}), "group")
			},
		},
		{
			name: "spec.group wrong type",
			mutate: func(u *unstructured.Unstructured) {
				u.Object["spec"].(map[string]interface{})["group"] = float64(123)
			},
		},
		{
			name: "missing spec.names",
			mutate: func(u *unstructured.Unstructured) {
				delete(u.Object["spec"].(map[string]interface{}), "names")
			},
		},
		{
			name: "missing spec.names.kind",
			mutate: func(u *unstructured.Unstructured) {
				delete(u.Object["spec"].(map[string]interface{})["names"].(map[string]interface{}), "kind")
			},
		},
		{
			name: "missing spec.names.plural",
			mutate: func(u *unstructured.Unstructured) {
				delete(u.Object["spec"].(map[string]interface{})["names"].(map[string]interface{}), "plural")
			},
		},
		{
			name: "missing spec.versions",
			mutate: func(u *unstructured.Unstructured) {
				delete(u.Object["spec"].(map[string]interface{}), "versions")
			},
		},
	}
	for _, tc := range malformed {
		t.Run(tc.name, func(t *testing.T) {
			u := newCRD("foos.example.com", "example.com", "Foo", "foos", "v1")
			tc.mutate(u)
			if got := e.ExtractGVKs(u); got != nil {
				t.Fatalf("ExtractGVKs = %v, want nil", got)
			}
		})
	}

	t.Run("versions with wrong element type are skipped", func(t *testing.T) {
		u := newCRD("foos.example.com", "example.com", "Foo", "foos", "v1")
		spec := u.Object["spec"].(map[string]interface{})
		// Values match what the dynamic informer would produce after JSON
		// decoding (string, float64, bool, []interface{}, map[string]interface{}).
		spec["versions"] = []interface{}{
			"not a map",
			map[string]interface{}{"name": "v1"},
			map[string]interface{}{}, // missing name
			map[string]interface{}{"name": ""},
			map[string]interface{}{"name": float64(7)}, // wrong type
			map[string]interface{}{"name": "v2"},
		}
		got := e.ExtractGVKs(u)
		if len(got) != 2 {
			t.Fatalf("ExtractGVKs returned %d resources, want 2 (only v1 and v2 are valid). got=%#v", len(got), got)
		}
		sort.Slice(got, func(i, j int) bool { return got[i].Version < got[j].Version })
		if got[0].Version != "v1" || got[1].Version != "v2" {
			t.Fatalf("versions = %v, want [v1 v2]", []string{got[0].Version, got[1].Version})
		}
	})
}

// TestCRDExtractor_NoPanicOnArbitraryInput is a smoke test ensuring the
// extractor never panics, including for inputs without an explicit branch.
func TestCRDExtractor_NoPanicOnArbitraryInput(t *testing.T) {
	e := &crdExtractor{}
	weirdInputs := []interface{}{
		nil,
		"",
		42,
		map[string]string{"a": "b"},
		&unstructured.Unstructured{}, // empty Object map
		&unstructured.Unstructured{Object: map[string]interface{}{"spec": "not a map"}},
		&unstructured.Unstructured{Object: map[string]interface{}{"spec": map[string]interface{}{"versions": "not a slice"}}},
	}
	mustNotPanic := func(name string, in interface{}, fn func()) {
		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("%s panicked on input %#v: %v", name, in, rec)
			}
		}()
		fn()
	}
	for _, in := range weirdInputs {
		mustNotPanic("SourceID", in, func() { _ = e.SourceID(in) })
		mustNotPanic("ExtractGVKs", in, func() { _ = e.ExtractGVKs(in) })
	}
}
