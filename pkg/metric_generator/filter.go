/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

// FamilyGeneratorFilter represents a filter which decides whether a metric
// family is exposed by the store or not
type FamilyGeneratorFilter interface {

	// Test returns true if it passes the criteria of the filter, otherwise it
	// will return false and the metric family is not exposed by the store
	Test(generator FamilyGenerator) bool
}

// CompositeFamilyGeneratorFilter is composite for combining multiple filters
type CompositeFamilyGeneratorFilter struct {
	filters []FamilyGeneratorFilter
}

// Test tests the generator by passing it through the filters contained within the composite
// and return false if the generator does not match all the filters
func (composite CompositeFamilyGeneratorFilter) Test(generator FamilyGenerator) bool {
	for _, filter := range composite.filters {
		if !filter.Test(generator) {
			return false
		}
	}
	return true
}

// FilterFamilyGenerators filters a given slice of family generators based upon a given filter
// and returns a slice containing the family generators which passed the filter criteria
func FilterFamilyGenerators(filter FamilyGeneratorFilter, families []FamilyGenerator) []FamilyGenerator {
	var filtered []FamilyGenerator

	for _, family := range families {
		if filter.Test(family) {
			filtered = append(filtered, family)
		}
	}

	return filtered
}

// NewCompositeFamilyGeneratorFilter combines multiple family generators filters into one composite filter
func NewCompositeFamilyGeneratorFilter(filters ...FamilyGeneratorFilter) CompositeFamilyGeneratorFilter {
	return CompositeFamilyGeneratorFilter{filters}
}
