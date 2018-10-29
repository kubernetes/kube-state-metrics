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
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"k8s.io/kube-state-metrics/pkg/options"
)

const (
	initialNumBufSize = 24
)

var (
	numBufPool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 0, initialNumBufSize)
			return &b
		},
	}
)

// Metric represents a single line entry in the /metrics export format
type Metric string

// NewMetric returns a new Metric
func NewMetric(name string, labelKeys []string, labelValues []string, value float64) (*Metric, error) {
	if len(labelKeys) != len(labelValues) {
		return nil, errors.New("expected labelKeys to be of same length as labelValues")
	}

	m := strings.Builder{}

	m.WriteString(name)

	labelsToString(&m, labelKeys, labelValues)

	m.WriteByte(' ')

	writeFloat(&m, value)

	m.WriteByte('\n')

	metric := Metric(m.String())

	return &metric, nil
}

func labelsToString(m *strings.Builder, keys, values []string) {
	if len(keys) > 0 {
		var separator byte = '{'

		for i := 0; i < len(keys); i++ {
			m.WriteByte(separator)
			m.WriteString(keys[i])
			m.WriteString("=\"")
			escapeString(m, values[i])
			m.WriteByte('"')
			separator = ','
		}

		m.WriteByte('}')
	}
}

var (
	escapeWithDoubleQuote = strings.NewReplacer("\\", `\\`, "\n", `\n`, "\"", `\"`)
)

// escapeString replaces '\' by '\\', new line character by '\n', and '"' by
// '\"'.
// Taken from github.com/prometheus/common/expfmt/text_create.go.
func escapeString(m *strings.Builder, v string) {
	escapeWithDoubleQuote.WriteString(m, v)
}

// writeFloat is equivalent to fmt.Fprint with a float64 argument but hardcodes
// a few common cases for increased efficiency. For non-hardcoded cases, it uses
// strconv.AppendFloat to avoid allocations, similar to writeInt.
// Taken from github.com/prometheus/common/expfmt/text_create.go.
func writeFloat(w *strings.Builder, f float64) {
	switch {
	case f == 1:
		w.WriteByte('1')
	case f == 0:
		w.WriteByte('0')
	case f == -1:
		w.WriteString("-1")
	case math.IsNaN(f):
		w.WriteString("NaN")
	case math.IsInf(f, +1):
		w.WriteString("+Inf")
	case math.IsInf(f, -1):
		w.WriteString("-Inf")
	default:
		bp := numBufPool.Get().(*[]byte)
		*bp = strconv.AppendFloat((*bp)[:0], f, 'g', -1, 64)
		w.Write(*bp)
		numBufPool.Put(bp)
	}
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
