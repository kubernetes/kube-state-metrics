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

func Test_jsonPath_Parse(t *testing.T) {
	tests := []struct {
		name    string
		j       jsonPath
		want    []string
		wantErr bool
	}{
		{
			name:    "empty input",
			j:       "",
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "dot input",
			j:       ".",
			want:    []string{""},
			wantErr: false,
		},
		{
			name:    "some path input",
			j:       ".foo.bar",
			want:    []string{"foo", "bar"},
			wantErr: false,
		},
		{
			name:    "invalid character ,",
			j:       ".foo,.bar",
			wantErr: true,
		},
		{
			name:    "invalid closure",
			j:       "{.foo}",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.j.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("jsonPath.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("jsonPath.Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newMetricMeta(t *testing.T) {
	tests := []struct {
		name               string
		basePath           []string
		j                  jsonPath
		jsonLabelsFromPath map[string]jsonPath
		want               customresourcestate.MetricMeta
	}{
		{
			name:               "with basePath and jsonpath, without jsonLabelsFromPath",
			basePath:           []string{"foo"},
			j:                  jsonPath(".bar"),
			jsonLabelsFromPath: map[string]jsonPath{},
			want: customresourcestate.MetricMeta{
				Path:           []string{"foo", "bar"},
				LabelsFromPath: map[string][]string{},
			},
		},
		{
			name:               "with basePath, jsonpath and jsonLabelsFromPath",
			basePath:           []string{"foo"},
			j:                  jsonPath(".bar"),
			jsonLabelsFromPath: map[string]jsonPath{"some": ".label.from.path"},
			want: customresourcestate.MetricMeta{
				Path: []string{"foo", "bar"},
				LabelsFromPath: map[string][]string{
					"some": {"label", "from", "path"},
				},
			},
		},
		{
			name:               "no basePath, jsonpath and jsonLabelsFromPath",
			basePath:           []string{},
			j:                  jsonPath(""),
			jsonLabelsFromPath: map[string]jsonPath{},
			want: customresourcestate.MetricMeta{
				Path:           []string{},
				LabelsFromPath: map[string][]string{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newMetricMeta(tt.basePath, tt.j, tt.jsonLabelsFromPath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newMetricMeta() = %v, want %v", got, tt.want)
			}
		})
	}
}
