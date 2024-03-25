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
	"sigs.k8s.io/controller-tools/pkg/markers"

	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
)

const (
	gaugeMarkerName = "Metrics:gauge"
)

func init() {
	MarkerDefinitions = append(
		MarkerDefinitions,
		must(markers.MakeDefinition(gaugeMarkerName, markers.DescribesField, gaugeMarker{})).
			help(gaugeMarker{}.help()),
		must(markers.MakeDefinition(gaugeMarkerName, markers.DescribesType, gaugeMarker{})).
			help(gaugeMarker{}.help()),
	)
}

// gaugeMarker implements localGeneratorMarker to generate a gauge type metric.
type gaugeMarker struct {
	Name           string
	Help           string              `marker:"help,optional"`
	JSONPath       jsonPath            `marker:"JSONPath,optional"`
	LabelFromKey   string              `marker:"labelFromKey,optional"`
	LabelsFromPath map[string]jsonPath `marker:"labelsFromPath,optional"`
	NilIsZero      bool                `marker:"nilIsZero,optional"`
	ValueFrom      *jsonPath           `marker:"valueFrom,optional"`
}

var _ LocalGeneratorMarker = &gaugeMarker{}

func (gaugeMarker) help() *markers.DefinitionHelp {
	return &markers.DefinitionHelp{
		Category: "Metrics",
		DetailedHelp: markers.DetailedHelp{
			Summary: "Defines a Gauge metric and uses the implicit path to the field joined by the provided JSONPath as path for the metric configuration.",
			Details: "",
		},
		FieldHelp: map[string]markers.DetailedHelp{},
	}
}

func (g gaugeMarker) ToGenerator(basePath ...string) *customresourcestate.Generator {
	var err error
	var valueFrom []string
	if g.ValueFrom != nil {
		valueFrom, err = g.ValueFrom.Parse()
		if err != nil {
			klog.Fatal(err)
		}
	}

	return &customresourcestate.Generator{
		Name: g.Name,
		Help: g.Help,
		Each: customresourcestate.Metric{
			Type: customresourcestate.MetricTypeGauge,
			Gauge: &customresourcestate.MetricGauge{
				NilIsZero:    g.NilIsZero,
				MetricMeta:   newMetricMeta(basePath, g.JSONPath, g.LabelsFromPath),
				LabelFromKey: g.LabelFromKey,
				ValueFrom:    valueFrom,
			},
		},
	}
}
