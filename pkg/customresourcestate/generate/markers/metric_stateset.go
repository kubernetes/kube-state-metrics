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
	stateSetMarkerName = "Metrics:stateset"
)

func init() {
	MarkerDefinitions = append(
		MarkerDefinitions,
		must(markers.MakeDefinition(stateSetMarkerName, markers.DescribesField, stateSetMarker{})).
			help(stateSetMarker{}.help()),
		must(markers.MakeDefinition(stateSetMarkerName, markers.DescribesType, stateSetMarker{})).
			help(stateSetMarker{}.help()),
	)
}

// stateSetMarker is the marker to generate a stateSet type metric.
type stateSetMarker struct {
	Name           string
	Help           string              `marker:"help,optional"`
	LabelsFromPath map[string]jsonPath `marker:"labelsFromPath,optional"`
	JSONPath       *jsonPath           `marker:"JSONPath,optional"`
	LabelName      string              `marker:"labelName,optional"`
	List           []string            `marker:"list"`
}

var _ LocalGeneratorMarker = &stateSetMarker{}

func (stateSetMarker) help() *markers.DefinitionHelp {
	return &markers.DefinitionHelp{
		Category: "Metrics",
		DetailedHelp: markers.DetailedHelp{
			Summary: "Defines a StateSet metric and uses the implicit path to the field as path for the metric configuration.",
			Details: "",
		},
		FieldHelp: map[string]markers.DetailedHelp{},
	}
}

func (s stateSetMarker) ToGenerator(basePath ...string) *customresourcestate.Generator {
	var valueFrom []string
	var err error
	if s.JSONPath != nil {
		valueFrom, err = s.JSONPath.Parse()
		if err != nil {
			klog.Fatal(err)
		}
	}

	return &customresourcestate.Generator{
		Name: s.Name,
		Help: s.Help,
		Each: customresourcestate.Metric{
			Type: customresourcestate.MetricTypeStateSet,
			StateSet: &customresourcestate.MetricStateSet{
				MetricMeta: newMetricMeta(basePath, "", s.LabelsFromPath),
				List:       s.List,
				LabelName:  s.LabelName,
				ValueFrom:  valueFrom,
			},
		},
	}
}
