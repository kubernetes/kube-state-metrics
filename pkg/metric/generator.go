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

package metric

import (
	"strings"

	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"
)

// FamilyGenerator provides everything needed to generate a metric family with a
// Kubernetes object.
type FamilyGenerator struct {
	Name         string
	Help         string
	Type         MetricType
	GenerateFunc func(obj interface{}) Family
}

func (f *FamilyGenerator) generateHeader() string {
	header := strings.Builder{}
	header.WriteString("# HELP ")
	header.WriteString(f.Name)
	header.WriteByte(' ')
	header.WriteString(f.Help)
	header.WriteByte('\n')
	header.WriteString("# TYPE ")
	header.WriteString(f.Name)
	header.WriteByte(' ')
	header.WriteString(string(f.Type))

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
func ComposeMetricGenFuncs(familyGens []FamilyGenerator) func(obj interface{}) []metricsstore.FamilyStringer {
	return func(obj interface{}) []metricsstore.FamilyStringer {
		families := make([]metricsstore.FamilyStringer, len(familyGens))

		for i, gen := range familyGens {
			family := gen.GenerateFunc(obj)
			// Make family aware of its name.
			family.Name = gen.Name
			families[i] = &family
		}

		return families
	}
}

type whiteBlackLister interface {
	IsIncluded(string) bool
	IsExcluded(string) bool
}

// FilterMetricFamilies takes a white- and a blacklist and a slice of metric
// families and returns a filtered slice.
func FilterMetricFamilies(l whiteBlackLister, families []FamilyGenerator) []FamilyGenerator {
	filtered := []FamilyGenerator{}

	for _, f := range families {
		if l.IsIncluded(f.Name) {
			filtered = append(filtered, f)
		}
	}

	return filtered
}
