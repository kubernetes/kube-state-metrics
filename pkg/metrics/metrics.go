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

package metrics

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"k8s.io/kube-state-metrics/pkg/options"
)

// Metric represents a single line entry in the /metrics export format
type Metric string

// NewMetric returns a new Metric
func NewMetric(name string, labelKeys []string, labelValues []string, value float64) (*Metric, error) {
	if len(labelKeys) != len(labelValues) {
		return nil, errors.New("expected labelKeys to be of same length as labelValues")
	}

	m := ""

	m = m + name

	m = m + labelsToString(labelKeys, labelValues)

	m = m + fmt.Sprintf(" %v", value)

	m = m + "\n"

	metric := Metric(m)

	return &metric, nil
}

func labelsToString(keys, values []string) string {
	if len(keys) > 0 {
		labels := []string{}
		for i := 0; i < len(keys); i++ {
			labels = append(
				labels,
				fmt.Sprintf(`%s="%s"`, keys[i], escapeString(values[i])),
			)
		}
		// TODO: Do labels need to be sorted. As of now, this is only needed to
		// make output deterministic between test runs.
		sort.Strings(labels)
		return "{" + strings.Join(labels, ",") + "}"
	}

	return ""
}

var (
	escapeWithDoubleQuote = strings.NewReplacer("\\", `\\`, "\n", `\n`, "\"", `\"`)
)

// escapeString replaces '\' by '\\', new line character by '\n', and - if
// includeDoubleQuote is true - '"' by '\"'.
// TODO: Taken from github.com/prometheus/common/expfmt/text_create.go, should be better referenced?
func escapeString(v string) string {
	return escapeWithDoubleQuote.Replace(v)
}

// MetricFamilyDesc represents the HELP and TYPE string above a metric family list
type MetricFamilyDesc string

type gathererFunc func() ([]*dto.MetricFamily, error)

func (f gathererFunc) Gather() ([]*dto.MetricFamily, error) {
	return f()
}

// FilteredGatherer wraps a prometheus.Gatherer to filter metrics based on a
// white or blacklist. Whitelist and blacklist are mutually exclusive.
// TODO: Bring white and blacklisting back
func FilteredGatherer(r prometheus.Gatherer, whitelist options.MetricSet, blacklist options.MetricSet) prometheus.Gatherer {
	whitelistEnabled := !whitelist.IsEmpty()
	blacklistEnabled := !blacklist.IsEmpty()

	if whitelistEnabled {
		return gathererFunc(func() ([]*dto.MetricFamily, error) {
			metricFamilies, err := r.Gather()
			if err != nil {
				return nil, err
			}

			newMetricFamilies := []*dto.MetricFamily{}
			for _, metricFamily := range metricFamilies {
				// deferencing this string may be a performance bottleneck
				name := *metricFamily.Name
				_, onWhitelist := whitelist[name]
				if onWhitelist {
					newMetricFamilies = append(newMetricFamilies, metricFamily)
				}
			}

			return newMetricFamilies, nil
		})
	}

	if blacklistEnabled {
		return gathererFunc(func() ([]*dto.MetricFamily, error) {
			metricFamilies, err := r.Gather()
			if err != nil {
				return nil, err
			}

			newMetricFamilies := []*dto.MetricFamily{}
			for _, metricFamily := range metricFamilies {
				name := *metricFamily.Name
				_, onBlacklist := blacklist[name]
				if onBlacklist {
					continue
				}
				newMetricFamilies = append(newMetricFamilies, metricFamily)
			}

			return newMetricFamilies, nil
		})
	}

	return r
}
