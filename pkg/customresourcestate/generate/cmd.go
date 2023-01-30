/*
Copyright 2022 The Kubernetes Authors All rights reserved.

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

package generate

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/genall/help"
	prettyhelp "sigs.k8s.io/controller-tools/pkg/genall/help/pretty"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"

	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate/generate/generator"
)

const (
	generatorName = "metric"
)

var (
	// optionsRegistry contains all the marker definitions used to process command line options
	optionsRegistry = &markers.Registry{}

	generateWhichMarkersFlag bool
)

// GenerateCommand runs the kube-state-metrics custom resource config generator.
var GenerateCommand = &cobra.Command{
	Use:                   "generate [flags] /path/to/package [/path/to/package]",
	Short:                 "Generate custom resource metrics configuration from go-code markers.",
	DisableFlagsInUseLine: true,
	Args:                  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if generateWhichMarkersFlag {
			PrintMarkerDocs()
			return nil
		}

		// Register the metric generator itself as marker so genall.FromOptions is able to initialize the runtime properly.
		// This also registers the markers inside the optionsRegistry so its available to print the marker docs.
		metricGenerator := generator.CustomResourceConfigGenerator{}
		defn := markers.Must(markers.MakeDefinition(generatorName, markers.DescribesPackage, metricGenerator))
		if err := optionsRegistry.Register(defn); err != nil {
			return err
		}

		// Load the passed packages as roots.
		roots, err := loader.LoadRoots(args...)
		if err != nil {
			return fmt.Errorf("loading packages %w", err)
		}

		// Set up the generator runtime using controller-tools and passing our optionsRegistry.
		rt, err := genall.FromOptions(optionsRegistry, []string{generatorName})
		if err != nil {
			return fmt.Errorf("%v", err)
		}

		// Setup the generation context with the loaded roots.
		rt.GenerationContext.Roots = roots
		// Setup the runtime to output to stdout.
		rt.OutputRules = genall.OutputRules{Default: genall.OutputToStdout}

		// Run the generator using the runtime.
		if hadErrs := rt.Run(); hadErrs {
			return fmt.Errorf("generator did not run successfully")
		}

		return nil
	},
	Example: "kube-state-metrics generate ./apis/... > custom-resource-config.yaml",
}

func init() {
	GenerateCommand.Flags().BoolVarP(&generateWhichMarkersFlag, "which-markers", "w", false, "Print out all markers available with the requested generators.")
}

// PrintMarkerDocs prints out marker help for the given generators specified in
// the rawOptions
func PrintMarkerDocs() error {
	// Register the metric generator itself as marker so genall.FromOptions is able to initialize the runtime properly.
	// This also registers the markers inside the optionsRegistry so its available to print the marker docs.
	metricGenerator := generator.CustomResourceConfigGenerator{}
	defn := markers.Must(markers.MakeDefinition(generatorName, markers.DescribesPackage, metricGenerator))
	if err := optionsRegistry.Register(defn); err != nil {
		return err
	}

	// just grab a registry so we don't lag while trying to load roots
	// (like we'd do if we just constructed the full runtime).
	reg, err := genall.RegistryFromOptions(optionsRegistry, []string{generatorName})
	if err != nil {
		return err
	}

	helpInfo := help.ByCategory(reg, help.SortByCategory)

	for _, cat := range helpInfo {
		if cat.Category == "" {
			continue
		}
		contents := prettyhelp.MarkersDetails(false, cat.Category, cat.Markers)
		if err := contents.WriteTo(os.Stderr); err != nil {
			return err
		}
	}
	return nil
}
