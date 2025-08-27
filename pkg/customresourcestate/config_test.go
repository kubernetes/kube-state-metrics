/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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

package customresourcestate

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"
	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

//go:embed example_config.yaml
var testData string

func Test_Metrics_deserialization(t *testing.T) {
	var m Metrics
	assert.NoError(t, yaml.NewDecoder(strings.NewReader(testData)).Decode(&m))
	configOverrides(&m)
	assert.Equal(t, "active_count", m.Spec.Resources[0].Metrics[0].Name)

	t.Run("can create resource factory", func(t *testing.T) {
		rf, err := NewCustomResourceMetrics(m.Spec.Resources[0])
		assert.NoError(t, err)

		t.Run("labels are merged", func(t *testing.T) {
			assert.Equal(t, map[string]string{
				"name": mustCompilePath(t, "metadata", "name").String(),
			}, toPaths(rf.(*customResourceMetrics).Families[1].LabelFromPath))
		})

		t.Run("errorLogV", func(t *testing.T) {
			assert.Equal(t, klog.Level(5), rf.(*customResourceMetrics).Families[1].ErrorLogV)
		})

		t.Run("resource name", func(t *testing.T) {
			assert.Equal(t, rf.(*customResourceMetrics).ResourceName, "foos")
		})
	})
}

func toPaths(m map[string]valuePath) map[string]string {
	out := make(map[string]string)
	for k, v := range m {
		out[k] = v.String()
	}
	return out
}
