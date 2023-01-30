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
package generator

import (
	"fmt"
	"sort"

	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-tools/pkg/crd"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/loader"
	ctrlmarkers "sigs.k8s.io/controller-tools/pkg/markers"

	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate/generate/markers"
)

// CustomResourceConfigGenerator implements the Generator interface from controller-tools.
// It uses markers to generate a custom resource configuration for kube-state-metrics from go code.
type CustomResourceConfigGenerator struct{}

var _ genall.Generator = &CustomResourceConfigGenerator{}
var _ genall.NeedsTypeChecking = &CustomResourceConfigGenerator{}

// RegisterMarkers registers all markers needed by this Generator
// into the given registry.
func (g CustomResourceConfigGenerator) RegisterMarkers(into *ctrlmarkers.Registry) error {
	for _, m := range markers.MarkerDefinitions {
		if err := m.Register(into); err != nil {
			return err
		}
	}

	return nil
}

// Generate generates artifacts produced by this marker.
// It's called after RegisterMarkers has been called.
func (g CustomResourceConfigGenerator) Generate(ctx *genall.GenerationContext) error {
	// Create the parser which is specific to the metric generator.
	parser := newParser(
		&crd.Parser{
			Collector: ctx.Collector,
			Checker:   ctx.Checker,
		},
	)

	// Loop over all passed packages.
	for _, pkg := range ctx.Roots {
		// skip packages which don't import metav1 because they can't define a CRD without meta v1.
		metav1 := pkg.Imports()["k8s.io/apimachinery/pkg/apis/meta/v1"]
		if metav1 == nil {
			continue
		}

		// parse the given package to feed crd.FindKubeKinds with Kubernetes Objects.
		parser.NeedPackage(pkg)

		kubeKinds := crd.FindKubeKinds(parser.Parser, metav1)
		if len(kubeKinds) == 0 {
			klog.Fatalf("no objects in the roots")
		}

		// Create metrics for all Custom Resources in this package.
		// This creates the customresourcestate.Resource object which contains all metric
		// definitions for the Custom Resource, if it is part of the package.
		for _, gv := range kubeKinds {
			parser.NeedResourceFor(pkg, gv)
		}
	}

	// Initialize empty customresourcestate configuration file and fill it with the
	// customresourcestate.Resource objects from the parser.
	metrics := customresourcestate.Metrics{
		Spec: customresourcestate.MetricsSpec{
			Resources: []customresourcestate.Resource{},
		},
	}

	for _, resource := range parser.CustomResourceStates {
		if resource == nil {
			continue
		}
		if len(resource.Metrics) > 0 {
			// Sort the metrics to get a deterministic output.
			sort.Slice(resource.Metrics, func(i, j int) bool {
				return resource.Metrics[i].Name < resource.Metrics[j].Name
			})

			metrics.Spec.Resources = append(metrics.Spec.Resources, *resource)
		}
	}

	// Sort the resources by GVK to get a deterministic output.
	sort.Slice(metrics.Spec.Resources, func(i, j int) bool {
		a := metrics.Spec.Resources[i].GroupVersionKind.String()
		b := metrics.Spec.Resources[j].GroupVersionKind.String()
		return a < b
	})

	// Write the rendered yaml to the context which will result in stdout.
	filePath := "metrics.yaml"
	if err := ctx.WriteYAML(filePath, "", []interface{}{metrics}, genall.WithTransform(addCustomResourceStateKind)); err != nil {
		return fmt.Errorf("WriteYAML to %s: %w", filePath, err)
	}

	return nil
}

// CheckFilter indicates the loader.NodeFilter (if any) that should be used
// to prune out unused types/packages when type-checking (nodes for which
// the filter returns true are considered "interesting").  This filter acts
// as a baseline -- all types the pass through this filter will be checked,
// but more than that may also be checked due to other generators' filters.
func (CustomResourceConfigGenerator) CheckFilter() loader.NodeFilter {
	// Re-use controller-tools filter to filter out unrelated nodes that aren't used
	// in CRD generation, like interfaces and struct fields without JSON tag.
	return crd.Generator{}.CheckFilter()
}

// addCustomResourceStateKind adds the correct kind because we don't have a correct
// kubernetes-style object as configuration definition.
func addCustomResourceStateKind(obj map[string]interface{}) error {
	obj["kind"] = "CustomResourceStateMetrics"
	return nil
}
