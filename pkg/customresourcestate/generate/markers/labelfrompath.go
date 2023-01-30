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
	"fmt"

	"sigs.k8s.io/controller-tools/pkg/markers"

	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
)

const (
	labelFromPathMarkerName = "Metrics:labelFromPath"
)

func init() {
	MarkerDefinitions = append(
		MarkerDefinitions,
		must(markers.MakeDefinition(labelFromPathMarkerName, markers.DescribesType, labelFromPathMarker{})).
			help(labelFromPathMarker{}.Help()),
		must(markers.MakeDefinition(labelFromPathMarkerName, markers.DescribesField, labelFromPathMarker{})).
			help(labelFromPathMarker{}.Help()),
	)
}

// labelFromPathMarker is the marker to configure a labelFromPath for a gvk.
type labelFromPathMarker struct {
	// +Metrics:labelFromPath:name=<string>,JSONPath=<string> on API type struct
	Name     string
	JSONPath jsonPath `marker:"JSONPath"`
}

var _ ResourceMarker = labelFromPathMarker{}

// Help prints the help information for the LabelFromPathMarker.
func (labelFromPathMarker) Help() *markers.DefinitionHelp {
	return &markers.DefinitionHelp{
		Category: "Metrics",
		DetailedHelp: markers.DetailedHelp{
			Summary: "adds an additional label to all metrics of this field or type with a value from the given JSONPath.",
			Details: "",
		},
		FieldHelp: map[string]markers.DetailedHelp{},
	}
}

func (n labelFromPathMarker) ApplyToResource(resource *customresourcestate.Resource) error {
	if resource.LabelsFromPath == nil {
		resource.LabelsFromPath = map[string][]string{}
	}
	jsonPathElems, err := n.JSONPath.Parse()
	if err != nil {
		return err
	}

	if jsonPath, labelExists := resource.LabelsFromPath[n.Name]; labelExists {
		if len(jsonPathElems) != len(jsonPath) {
			return fmt.Errorf("duplicate definition for label %q", n.Name)
		}
		for i, v := range jsonPath {
			if v != jsonPathElems[i] {
				return fmt.Errorf("duplicate definition for label %q", n.Name)
			}
		}
	}

	resource.LabelsFromPath[n.Name] = jsonPathElems
	return nil
}
