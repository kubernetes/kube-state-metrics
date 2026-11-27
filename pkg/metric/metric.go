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

package metric

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
)

const (
	initialNumBufSize = 24
)

var (
	numBufPool = sync.Pool{
		New: func() any {
			b := make([]byte, 0, initialNumBufSize)
			return &b
		},
	}
)

// Type represents the type of the metric. See https://github.com/prometheus/OpenMetrics/blob/v1.0.0/specification/OpenMetrics.md#metric-types.
type Type string

// Supported metric types.
var (

	// Gauge defines an OpenMetrics gauge.
	Gauge Type = "gauge"

	// Info defines an OpenMetrics info.
	Info Type = "info"

	// StateSet defines an OpenMetrics stateset.
	StateSet Type = "stateset"

	// Counter defines an OpenMetrics counter.
	Counter Type = "counter"
)

// Metric represents a single time series.
type Metric struct {
	// The name of a metric is injected by its family to reduce duplication.
	LabelKeys   []string
	LabelValues []string
	Value       float64
}

func (m *Metric) Write(s *bytes.Buffer) {
	checkLabels(m.LabelKeys, m.LabelValues)

	if len(m.LabelKeys) > 0 {
		writeLabels(s, '{', m.LabelKeys, m.LabelValues)
		s.WriteByte('}')
	}
	s.WriteByte(' ')
	writeFloat(s, m.Value)
	s.WriteByte('\n')
}

// writeAfterPrefix writes the metric following a name and an already opened,
// non-empty label set: its own labels, the closing brace and the value.
func (m *Metric) writeAfterPrefix(s *bytes.Buffer) {
	checkLabels(m.LabelKeys, m.LabelValues)

	writeLabels(s, ',', m.LabelKeys, m.LabelValues)
	s.WriteByte('}')
	s.WriteByte(' ')
	writeFloat(s, m.Value)
	s.WriteByte('\n')
}

func checkLabels(keys, values []string) {
	if len(keys) != len(values) {
		panic(fmt.Sprintf(
			"expected labelKeys %q to be of same length as labelValues %q",
			keys, values,
		))
	}
}

// writeLabels writes keys and values as key="value" pairs, each preceded by
// separator for the first pair and by ',' for the rest. It writes no braces
// of its own beyond the given separator.
func writeLabels(m *bytes.Buffer, separator byte, keys, values []string) {
	for i := range keys {
		m.WriteByte(separator)
		m.WriteString(keys[i])
		m.WriteString("=\"")
		escapeString(m, values[i])
		m.WriteByte('"')
		separator = ','
	}
}

var (
	escapeWithDoubleQuote = strings.NewReplacer("\\", `\\`, "\n", `\n`, "\"", `\"`)
)

// escapeString replaces '\' by '\\', new line character by '\n', and '"' by
// '\"'.
// Taken from github.com/prometheus/common/expfmt/text_create.go.
func escapeString(m *bytes.Buffer, v string) {
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c == '\\' || c == '\n' || c == '"' {
			escapeWithDoubleQuote.WriteString(m, v)
			return
		}
	}
	m.WriteString(v)
}

// writeFloat is equivalent to fmt.Fprint with a float64 argument but hardcodes
// a few common cases for increased efficiency. For non-hardcoded cases, it uses
// strconv.AppendFloat to avoid allocations, similar to writeInt.
// Taken from github.com/prometheus/common/expfmt/text_create.go.
func writeFloat(w *bytes.Buffer, f float64) {
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
