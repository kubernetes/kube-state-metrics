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
	"k8s.io/kube-state-metrics/v2/internal/discovery"
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

func TestFromConfig_BuiltInResource(t *testing.T) {
	// Test that built-in Kubernetes resources (not CRDs) work with customResourceState
	config := `
kind: CustomResourceStateMetrics
spec:
  resources:
    - groupVersionKind:
        group: storage.k8s.io
        version: v1
        kind: CSINode
      resourcePlural: csinodes
      metricNamePrefix: csi_node
      metrics:
        - name: labels
          help: "CSINode basic information"
          each:
            type: Gauge
            gauge:
              labelsFromPath:
                node: [metadata, name]
              valueFrom: [metadata, name]
`
	// Create an empty CRDiscoverer (no CRDs registered)
	discoverer := &discovery.CRDiscoverer{}

	decoder := yaml.NewDecoder(strings.NewReader(config))
	factoryGenerator, err := FromConfig(decoder, discoverer)
	assert.NoError(t, err)

	factories, err := factoryGenerator()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(factories), "Expected one factory for CSINode")

	factory := factories[0]
	assert.Equal(t, "csinodes", factory.Name(), "Factory name should match configured resourcePlural")
}
