/*
Copyright 2020 The Kubernetes Authors All rights reserved.

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

package allow

import (
	"regexp"
)

var defaultMetricLabels = map[*regexp.Regexp][]*regexp.Regexp{
	regexp.MustCompile(".*_labels$"):      {},
	regexp.MustCompile(".*_annotations$"): {},
}

// ParseLabels parses and compiles all of the regexes for Labels
func ParseLabels(l map[string][]string) (Labels, error) {
	labels := make(map[string][]*regexp.Regexp)
	for metric, lbls := range l {
		var labelsRegex []*regexp.Regexp
		for _, lbl := range lbls {
			reg, err := regexp.Compile(lbl)
			if err != nil {
				return nil, err
			}
			labelsRegex = append(labelsRegex, reg)
		}
		labels[metric] = labelsRegex
	}
	return labels, nil
}

// Labels provide a way to allow only specified labels for metrics.
// Falls back to default labels, if your metric doesn't have defaults
// then all labels are allowed.
type Labels map[string][]*regexp.Regexp

// Allowed returns allowed metric keys and values for the metric.
func (a Labels) Allowed(metric string, labels, values []string) ([]string, []string) {
	allowedLabels, ok := a[metric]
	if !ok {
		var defaultsPresent bool
		for metricReg, lbls := range defaultMetricLabels {
			if metricReg.MatchString(metric) {
				allowedLabels = append(allowedLabels, lbls...)
				defaultsPresent = true
			}
		}

		if !defaultsPresent {
			return labels, values
		}
	}

	var finalLabels, finalValues []string
	labelSet := labelSet(allowedLabels)
	alreadyMatched := make(map[string]interface{}) // used to prevent duplicate labels
	for _, allowedLabel := range labelSet {
		for i, label := range labels {
			_, ok := alreadyMatched[label]
			if ok {
				continue
			}
			if allowedLabel.MatchString(label) {
				finalLabels = append(finalLabels, label)
				finalValues = append(finalValues, values[i])
				alreadyMatched[label] = struct{}{}
			}
		}
	}

	return finalLabels, finalValues
}

func labelSet(lists ...[]*regexp.Regexp) []*regexp.Regexp {
	m := make(map[string]interface{})
	var set []*regexp.Regexp
	for _, list := range lists {
		for _, e := range list {
			if _, ok := m[e.String()]; !ok {
				m[e.String()] = struct{}{}
				set = append(set, e)
			}
		}
	}
	return set
}
