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

func Test_stateSetMarker_ToGenerator(t *testing.T) {
	tests := []struct {
		name           string
		stateSetMarker stateSetMarker
		basePath       []string
		want           *customresourcestate.Generator
	}{
		{
			name: "Happy path",
			stateSetMarker: stateSetMarker{
				JSONPath: jsonPathPointer(".foo"),
			},
			basePath: []string{},
			want: &customresourcestate.Generator{
				Each: customresourcestate.Metric{
					Type: customresourcestate.MetricTypeStateSet,
					StateSet: &customresourcestate.MetricStateSet{
						MetricMeta: customresourcestate.MetricMeta{
							LabelsFromPath: map[string][]string{},
							Path:           []string{},
						},
						ValueFrom: []string{"foo"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.stateSetMarker.ToGenerator(tt.basePath...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stateSetMarker.ToGenerator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func jsonPathPointer(s string) *jsonPath {
	j := jsonPath(s)
	return &j
}
