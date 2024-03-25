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

var (
	// MarkerDefinitions contains all marker definitions defined by this package so
	// they can get used in a generator.
	MarkerDefinitions = []*markerDefinitionWithHelp{
		// GroupName is a marker copied from controller-runtime to identify the API Group.
		// It needs to get added as marker so the parser will be able to read the API
		// which is Group set for a package.
		must(markers.MakeDefinition("groupName", markers.DescribesPackage, "")),
	}
)

// +controllertools:marker:generateHelp:category=CRD

// ResourceMarker is a marker that configures a custom resource.
type ResourceMarker interface {
	// ApplyToCRD applies this marker to the given CRD, in the given version
	// within that CRD.  It's called after everything else in the CRD is populated.
	ApplyToResource(resource *customresourcestate.Resource) error
}

// LocalGeneratorMarker is a marker that creates a custom resource metric generator.
type LocalGeneratorMarker interface {
	// ApplyToCRD applies this marker to the given CRD, in the given version
	// within that CRD.  It's called after everything else in the CRD is populated.
	ToGenerator(basePath ...string) *customresourcestate.Generator
}
