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

	"k8s.io/apimachinery/pkg/runtime/schema"
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
		sort.Slice(got, func(i, j int) bool {
			return got[i].String() < got[j].String()
		})
		sort.Slice(tc.want, func(i, j int) bool {
			return tc.want[i].String() < tc.want[j].String()
		})
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("testcase: %s: got %v, want %v", tc.desc, got, tc.want)
		}
	}
}
