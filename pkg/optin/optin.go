/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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

package optin

import (
	"regexp"
	"sort"
	"strings"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

// MetricFamilyFilter filters metric families which are defined as opt-in by their generator.FamilyGenerator
type MetricFamilyFilter struct {
	metrics []*regexp.Regexp
}

// Test tests if a given generator is an opt-in metric family and was passed as an opt-in metric family at startup
func (filter MetricFamilyFilter) Test(generator generator.FamilyGenerator) bool {
	if !generator.OptIn {
		return true
	}
	for _, metric := range filter.metrics {
		if metric.MatchString(generator.Name) {
			return true
		}
	}
	return false
}

// Status returns the metrics contained within the filter as a comma-separated string
func (filter MetricFamilyFilter) Status() string {
	asStrings := make([]string, 0)
	for _, metric := range filter.metrics {
		asStrings = append(asStrings, metric.String())
	}
	// sort the strings for the sake of ux such that the resulting status is consistent
	sort.Strings(asStrings)
	return strings.Join(asStrings, ", ")
}

// Count returns the amount of metrics contained within the filter
func (filter MetricFamilyFilter) Count() int {
	return len(filter.metrics)
}

// NewMetricFamilyFilter creates new MetricFamilyFilter instances.
func NewMetricFamilyFilter(metrics map[string]struct{}) (*MetricFamilyFilter, error) {
	regexes := make([]*regexp.Regexp, 0)
	for metric := range metrics {
		regex, err := regexp.Compile(metric)
		if err != nil {
			return nil, err
		}
		regexes = append(regexes, regex)
	}
	return &MetricFamilyFilter{regexes}, nil
}
