/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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
	"strings"

	"k8s.io/kube-state-metrics/pkg/metric"
)

// FamilyGenerator provides everything needed to generate a metric family with a
// Kubernetes object.
type FamilyGenerator struct {
	Name         string
	Help         string
	Type         metric.Type
	GenerateFunc func(obj interface{}) *metric.Family
}

// Generate calls the FamilyGenerator.GenerateFunc and gives the family its
// name. The reasoning behind injecting the name at such a late point in time is
// deduplication in the code, preventing typos made by developers as
// well as saving memory.
func (g *FamilyGenerator) Generate(obj interface{}) *metric.Family {
	family := g.GenerateFunc(obj)
	family.Name = g.Name
	family.Type = g.Type
	return family
}

func (g *FamilyGenerator) generateHeader() string {
	header := strings.Builder{}
	header.WriteString("# HELP ")
	header.WriteString(g.Name)
	header.WriteByte(' ')
	header.WriteString(g.Help)
	header.WriteByte('\n')
	header.WriteString("# TYPE ")
	header.WriteString(g.Name)
	header.WriteByte(' ')
	header.WriteString(string(g.Type))

	return header.String()
}

// ExtractMetricFamilyHeaders takes in a slice of FamilyGenerator metrics and
// returns the extracted headers.
func ExtractMetricFamilyHeaders(families []FamilyGenerator) []string {
	headers := make([]string, len(families))

	for i, f := range families {
		headers[i] = f.generateHeader()
	}

	return headers
}

// ComposeMetricGenFuncs takes a slice of metric families and returns a function
// that composes their metric generation functions into a single one.
func ComposeMetricGenFuncs(familyGens []FamilyGenerator) func(obj interface{}) []metric.FamilyInterface {
	return func(obj interface{}) []metric.FamilyInterface {
		families := make([]metric.FamilyInterface, len(familyGens))

		for i, gen := range familyGens {
			families[i] = gen.Generate(obj)
		}

		return families
	}
}

type allowDenyLister interface {
	IsIncluded(string) bool
	IsExcluded(string) bool
}

// FilterMetricFamilies takes a allow- and a denylist and a slice of metric
// families and returns a filtered slice.
func FilterMetricFamilies(l allowDenyLister, families []FamilyGenerator) []FamilyGenerator {
	filtered := []FamilyGenerator{}

	for _, f := range families {
		if l.IsIncluded(f.Name) {
			filtered = append(filtered, f)
		}
	}

	return filtered
}
