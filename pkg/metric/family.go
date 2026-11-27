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
	"bytes"
)

// FamilyInterface interface for a family
type FamilyInterface interface {
	Inspect(inspect func(Family))
	ByteSlice() []byte
}

// Family represents a set of metrics with the same name and help text.
type Family struct {
	Name    string
	Type    Type
	Metrics []*Metric
	// DefaultLabels are rendered before each metric's own labels, escaped
	// once per family. A pointer keeps Family in the 64-byte size class.
	DefaultLabels *DefaultLabels
}

// maxInlineDefaultLabels is the number of default label values stored inline
// in DefaultLabels. The objects of the built-in stores are identified by at
// most three labels (namespace, name and uid).
const maxInlineDefaultLabels = 3

// DefaultLabels are labels shared by every metric of a Family.
type DefaultLabels struct {
	Keys   []string
	Values []string

	inline [maxInlineDefaultLabels]string
}

// NewDefaultLabels returns default labels with the given keys and values. The
// keys are kept as is, as they are typically shared by all objects of a
// store, and must not be modified afterwards. The values are copied into the
// returned struct, so up to three of them cost no allocation of their own.
func NewDefaultLabels(keys []string, values ...string) *DefaultLabels {
	d := &DefaultLabels{Keys: keys}
	d.Values = append(d.inline[:0], values...)
	return d
}

// SetDefaultLabels sets the family's default labels, see NewDefaultLabels. It
// does nothing for a family without metrics, which renders nothing anyway.
func (f *Family) SetDefaultLabels(keys []string, values ...string) {
	if len(f.Metrics) > 0 {
		f.DefaultLabels = NewDefaultLabels(keys, values...)
	}
}

// Inspect use to inspect the inside of a Family
func (f Family) Inspect(inspect func(Family)) {
	inspect(f)
}

// ByteSlice returns the given Family in its string representation.
func (f Family) ByteSlice() []byte {
	return f.AppendBytes(nil)
}

// AppendBytes appends the family in its string representation to b and returns
// the extended slice. Callers rendering many families for the same object can
// share a single buffer instead of allocating one per family.
func (f Family) AppendBytes(b []byte) []byte {
	// Nothing to render. Returning early also avoids bytes.Buffer.Grow's minimum
	// allocation, which empty families would otherwise pay on every event.
	if len(f.Metrics) == 0 {
		return b
	}

	buf := bytes.NewBuffer(b)
	buf.Grow(f.SizeHint())

	if f.DefaultLabels == nil || len(f.DefaultLabels.Keys) == 0 {
		for _, m := range f.Metrics {
			buf.WriteString(f.Name)
			m.Write(buf)
		}
		return buf.Bytes()
	}

	keys, values := f.DefaultLabels.Keys, f.DefaultLabels.Values
	checkLabels(keys, values)

	// Render the name and the default labels, without the closing brace, once.
	// Every following metric copies these bytes instead of escaping the values
	// again. The slice stays valid if buf grows: growing copies the contents to
	// a new array and leaves the old one untouched.
	start := buf.Len()
	buf.WriteString(f.Name)
	writeLabels(buf, '{', keys, values)
	prefix := buf.Bytes()[start:]

	for i, m := range f.Metrics {
		if i > 0 {
			buf.Write(prefix)
		}
		m.writeAfterPrefix(buf)
	}

	return buf.Bytes()
}

// valueSizeHint is the assumed rendered length of a metric value plus the
// separating space and trailing newline.
const valueSizeHint = 16

// SizeHint estimates the rendered size of the family so callers can size their
// buffer up front instead of growing (and copying) it repeatedly. It is an
// estimate, not a bound: the value length is assumed rather than computed.
func (f Family) SizeHint() int {
	defaultSize := 0
	if d := f.DefaultLabels; d != nil {
		defaultSize = labelsSizeHint(d.Keys, d.Values)
	}

	size := 0
	for _, m := range f.Metrics {
		// name, '{', '}', the value, the space before it and the newline after
		size += len(f.Name) + 2 + valueSizeHint + defaultSize + labelsSizeHint(m.LabelKeys, m.LabelValues)
	}
	return size
}

// labelsSizeHint returns the rendered size of a label set without its braces.
func labelsSizeHint(keys, values []string) int {
	size := 0
	for i, k := range keys {
		// key, '=', '"', value, '"' and the separating ',' or '{'
		size += len(k) + 4
		if i < len(values) {
			size += len(values[i])
		}
	}
	return size
}
