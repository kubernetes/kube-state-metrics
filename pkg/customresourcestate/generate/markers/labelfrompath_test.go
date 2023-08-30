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
package markers

import (
	"reflect"
	"testing"

	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
)

func Test_labelFromPathMarker_ApplyToResource(t *testing.T) {
	type fields struct {
		Name     string
		JSONPath jsonPath
	}
	tests := []struct {
		name         string
		fields       fields
		resource     *customresourcestate.Resource
		wantResource *customresourcestate.Resource
		wantErr      bool
	}{
		{
			name: "happy path",
			fields: fields{
				Name:     "foo",
				JSONPath: ".bar",
			},
			resource: &customresourcestate.Resource{},
			wantResource: &customresourcestate.Resource{
				Labels: customresourcestate.Labels{
					LabelsFromPath: map[string][]string{
						"foo": {"bar"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "label already exists with same path length",
			fields: fields{
				Name:     "foo",
				JSONPath: ".bar",
			},
			resource: &customresourcestate.Resource{
				Labels: customresourcestate.Labels{
					LabelsFromPath: map[string][]string{
						"foo": {"other"},
					},
				},
			},
			wantResource: &customresourcestate.Resource{
				Labels: customresourcestate.Labels{
					LabelsFromPath: map[string][]string{
						"foo": {"other"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "label already exists with different path length",
			fields: fields{
				Name:     "foo",
				JSONPath: ".bar",
			},
			resource: &customresourcestate.Resource{
				Labels: customresourcestate.Labels{
					LabelsFromPath: map[string][]string{
						"foo": {"other", "path"},
					},
				},
			},
			wantResource: &customresourcestate.Resource{
				Labels: customresourcestate.Labels{
					LabelsFromPath: map[string][]string{
						"foo": {"other", "path"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid json path",
			fields: fields{
				Name:     "foo",
				JSONPath: "{.bar}",
			},
			resource:     &customresourcestate.Resource{},
			wantResource: &customresourcestate.Resource{},
			wantErr:      true,
		},
		{
			name: "nil resource",
			fields: fields{
				Name:     "foo",
				JSONPath: "{.bar}",
			},
			resource: nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := labelFromPathMarker{
				Name:     tt.fields.Name,
				JSONPath: tt.fields.JSONPath,
			}
			if err := n.ApplyToResource(tt.resource); (err != nil) != tt.wantErr {
				t.Errorf("labelFromPathMarker.ApplyToResource() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.resource, tt.wantResource) {
				t.Errorf("labelFromPathMarker.ApplyToResource() = %v, want %v", tt.resource, tt.wantResource)
			}

		})
	}
}
