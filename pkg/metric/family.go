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
	for _, m := range f.Metrics {
		buf.WriteString(f.Name)
		m.Write(buf)
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
	size := 0
	for _, m := range f.Metrics {
		// name, '{', '}', the value, the space before it and the newline after
		size += len(f.Name) + 2 + valueSizeHint
		for i, k := range m.LabelKeys {
			// key, '=', '"', value, '"' and the separating ',' or '{'
			size += len(k) + 4
			if i < len(m.LabelValues) {
				size += len(m.LabelValues[i])
			}
		}
	}
	return size
}
