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

	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
)

const (
	infoMarkerName = "Metrics:info"
)

func init() {
	MarkerDefinitions = append(
		MarkerDefinitions,
		must(markers.MakeDefinition(infoMarkerName, markers.DescribesField, infoMarker{})).
			help(infoMarker{}.help()),
		must(markers.MakeDefinition(infoMarkerName, markers.DescribesType, infoMarker{})).
			help(infoMarker{}.help()),
	)
}

// infoMarker implements localGeneratorMarker to generate a info type metric.
type infoMarker struct {
	Name           string
	Help           string              `marker:"help,optional"`
	LabelsFromPath map[string]jsonPath `marker:"labelsFromPath,optional"`
	JSONPath       jsonPath            `marker:"JSONPath,optional"`
	LabelFromKey   string              `marker:"labelFromKey,optional"`
}

var _ LocalGeneratorMarker = &infoMarker{}

func (infoMarker) help() *markers.DefinitionHelp {
	return &markers.DefinitionHelp{
		Category: "Metrics",
		DetailedHelp: markers.DetailedHelp{
			Summary: "Defines a Info metric and uses the implicit path to the field as path for the metric configuration.",
			Details: "",
		},
		FieldHelp: map[string]markers.DetailedHelp{},
	}
}

func (i infoMarker) ToGenerator(basePath ...string) *customresourcestate.Generator {
	return &customresourcestate.Generator{
		Name: i.Name,
		Help: i.Help,
		Each: customresourcestate.Metric{
			Type: customresourcestate.MetricTypeInfo,
			Info: &customresourcestate.MetricInfo{
				MetricMeta:   newMetricMeta(basePath, i.JSONPath, i.LabelsFromPath),
				LabelFromKey: i.LabelFromKey,
			},
		},
	}
}
