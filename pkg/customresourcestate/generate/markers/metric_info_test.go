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

func Test_infoMarker_ToGenerator(t *testing.T) {
	tests := []struct {
		name       string
		infoMarker infoMarker
		basePath   []string
		want       *customresourcestate.Generator
	}{
		{
			name:       "Happy path",
			infoMarker: infoMarker{},
			basePath:   []string{},
			want: &customresourcestate.Generator{
				Each: customresourcestate.Metric{
					Type: customresourcestate.MetricTypeInfo,
					Info: &customresourcestate.MetricInfo{
						MetricMeta: customresourcestate.MetricMeta{
							LabelsFromPath: map[string][]string{},
							Path:           []string{},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.infoMarker.ToGenerator(tt.basePath...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("infoMarker.ToGenerator() = %v, want %v", got, tt.want)
			}
		})
	}
}
