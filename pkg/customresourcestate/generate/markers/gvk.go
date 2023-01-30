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
	// GVKMarkerName is the marker for a GVK. Without a set GVKMarkerName the
	// generator will not generate any configuration for this GVK.
	GVKMarkerName = "Metrics:gvk"
)

func init() {
	MarkerDefinitions = append(
		MarkerDefinitions,
		must(markers.MakeDefinition(GVKMarkerName, markers.DescribesType, gvkMarker{})).
			help(gvkMarker{}.Help()),
	)
}

// gvkMarker implements ResourceMarker to opt-in metric generation for a gvk and configure a name prefix.
type gvkMarker struct {
	NamePrefix string `marker:"namePrefix,optional"`
}

var _ ResourceMarker = gvkMarker{}

// Help prints the help information for the gvkMarker.
func (gvkMarker) Help() *markers.DefinitionHelp {
	return &markers.DefinitionHelp{
		Category: "Metrics",
		DetailedHelp: markers.DetailedHelp{
			Summary: "enables the creation of a customresourcestate Resource for the CRD and uses the given prefix for the metrics if configured.",
			Details: "",
		},
		FieldHelp: map[string]markers.DetailedHelp{},
	}
}

func (n gvkMarker) ApplyToResource(resource *customresourcestate.Resource) error {
	resource.MetricNamePrefix = &n.NamePrefix
	return nil
}
