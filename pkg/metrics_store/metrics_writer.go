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

package metricsstore

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/prometheus/common/expfmt"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
)

const (
	helpPrefix = "# HELP "
	typePrefix = "# TYPE "
)

var (
	infoTypeString     = string(metric.Info)
	stateSetTypeString = string(metric.StateSet)
	gaugeTypeString    = string(metric.Gauge)
	// gaugeNewline is the pre-built suffix used when rewriting info/stateset TYPE lines.
	// Pre-computing it reduces rewriteTypeLine to a 2-string concat (concatstring2),
	// avoiding the overhead of a 3-string concat (concatstring3) on every rewrite.
	gaugeNewline = gaugeTypeString + "\n"

	seenPool = sync.Pool{
		New: func() interface{} {
			return make(map[string]struct{})
		},
	}

	// snapshotPool recycles the per-scrape slice WriteAll collects each
	// store's rendered objects into, so a large store does not allocate a
	// fresh slice of that size on every scrape.
	snapshotPool = sync.Pool{
		New: func() interface{} {
			s := make([][][]byte, 0, 1024)
			return &s
		},
	}
)

// MetricsWriterList represent a list of MetricsWriter
type MetricsWriterList []*MetricsWriter

// MetricsWriter is a struct that holds multiple MetricsStore(s) and
// implements the MetricsWriter interface.
// It should be used with stores which have the same metric headers.
//
// MetricsWriter writes out metrics from the underlying stores so that
// metrics with the same name coming from different stores end up grouped together.
// It also ensures that the metric headers are only written out once.
type MetricsWriter struct {
	stores       []*MetricsStore
	ResourceName string
}

// NewMetricsWriter creates a new MetricsWriter.
func NewMetricsWriter(resourceName string, stores ...*MetricsStore) *MetricsWriter {
	return &MetricsWriter{
		stores:       stores,
		ResourceName: resourceName,
	}
}

// WriteAll writes out metrics from the underlying stores to the given writer.
//
// WriteAll writes metrics so that the ones with the same name
// are grouped together when written out.
func (m MetricsWriter) WriteAll(w io.Writer) error {
	if len(m.stores) == 0 {
		return nil
	}

	// Collect every store's rendered objects once, up front. The output is
	// grouped by family, so the objects are walked once per header: ranging
	// over the sync.Map that many times re-traverses its hash trie for every
	// family, whereas a flat slice is a sequential scan. The snapshot also
	// keeps one scrape self-consistent -- an object added or removed while
	// the response is being written is either in every family or in none.
	// The rendered byte slices are immutable, so holding on to them is safe.
	snapshot := snapshotPool.Get().(*[][][]byte)
	objects := (*snapshot)[:0]
	defer func() {
		// Drop the references so pooled slices do not keep deleted objects'
		// rendered bytes alive.
		clear(objects)
		*snapshot = objects[:0]
		snapshotPool.Put(snapshot)
	}()
	for _, s := range m.stores {
		s.metrics.Range(func(_ interface{}, value interface{}) bool {
			objects = append(objects, value.([][]byte))
			return true
		})
	}

	// Headers describe the families written below, so they are emitted only
	// when there is at least one object to describe, across all stores: there
	// is one store per namespace, so the first can be empty while a later one
	// holds objects whose families are still written.
	hasMetrics := len(objects) > 0

	for i, help := range m.stores[0].headers {
		// SanitizeHeaders blanks duplicate headers to preserve header/family
		// index alignment. An empty header means suppress the header text but
		// still emit the metric family bytes at this index.
		if help != "" && hasMetrics {
			// Avoid allocating a new string if the header lacks a trailing newline:
			// check once and emit "\n" as a second write if needed.
			needsNewline := help[len(help)-1] != '\n'
			if _, err := io.WriteString(w, help); err != nil {
				return fmt.Errorf("failed to write help text: %w", err)
			}
			if needsNewline {
				if _, err := io.WriteString(w, "\n"); err != nil {
					return fmt.Errorf("failed to write help text: %w", err)
				}
			}
		}

		for _, families := range objects {
			family := families[i]
			// Families with no metrics render to nothing. Skipping them saves
			// a call through the (possibly gzip-wrapped) response writer per
			// object, which adds up for families most objects do not have.
			if len(family) == 0 {
				continue
			}
			if _, err := w.Write(family); err != nil {
				return fmt.Errorf("failed to write metrics family: %w", err)
			}
		}
	}
	return nil
}

// parseHeaderStatic parses and potentially rewrites a header string.
// It returns the extracted metricName (if any) and the normalized/rewritten header string.
func parseHeaderStatic(header string, isTextPlain bool) (metricName string, rewrittenHeader string) {
	rest := header
	// Parse HELP line.
	if len(rest) > len(helpPrefix) && rest[:len(helpPrefix)] == helpPrefix {
		rest = rest[len(helpPrefix):]
		if spaceIdx := strings.IndexByte(rest, ' '); spaceIdx > 0 {
			metricName = rest[:spaceIdx]
		}
		// Advance past the rest of the HELP line.
		nl := strings.IndexByte(rest, '\n')
		if nl == -1 {
			rewrittenHeader = header
			if len(rewrittenHeader) > 0 && rewrittenHeader[len(rewrittenHeader)-1] != '\n' {
				rewrittenHeader += "\n"
			}
			return metricName, rewrittenHeader
		}
		rest = rest[nl+1:]
	}

	needsModification := false
	typeLastSpace := -1
	// Parse TYPE line.
	if len(rest) > len(typePrefix) && rest[:len(typePrefix)] == typePrefix {
		rest = rest[len(typePrefix):]
		if spaceIdx := strings.IndexByte(rest, ' '); spaceIdx > 0 {
			// Record the position of the space before the type suffix within the
			// original header string so rewriteTypeLine can avoid recomputing it.
			typeLastSpace = len(header) - len(rest) + spaceIdx
			if isTextPlain {
				typeSuffix := rest[spaceIdx+1:]
				if l := len(typeSuffix); l > 0 && typeSuffix[l-1] == '\n' {
					typeSuffix = typeSuffix[:l-1]
				}
				needsModification = typeSuffix == infoTypeString || typeSuffix == stateSetTypeString
			}
		}
	}

	if !needsModification {
		rewrittenHeader = header
	} else {
		if rewritten := rewriteTypeLine(header, typeLastSpace); rewritten != "" {
			rewrittenHeader = rewritten
		} else {
			rewrittenHeader = header
		}
	}

	if len(rewrittenHeader) > 0 && rewrittenHeader[len(rewrittenHeader)-1] != '\n' {
		rewrittenHeader += "\n"
	}

	return metricName, rewrittenHeader
}

// rewriteTypeLine replaces an OpenMetrics-only type suffix (info/stateset) with "gauge"
// in the TYPE line of header and returns the updated header.
// lastSpace is the pre-computed byte offset of the space before the type suffix.
func rewriteTypeLine(header string, lastSpace int) string {
	if lastSpace < 0 {
		return ""
	}
	return header[:lastSpace+1] + gaugeNewline
}

// SanitizeHeaders sanitizes the headers of the given MetricsWriterList.
func SanitizeHeaders(contentType expfmt.Format, writers MetricsWriterList) MetricsWriterList {
	isTextPlain := contentType.FormatType() == expfmt.TypeTextPlain
	// A single map replaces the former seenHELP+seenTYPE pair: HELP and TYPE lines
	// always share the same metric name, so one lookup per header suffices.
	seen := seenPool.Get().(map[string]struct{})
	defer func() {
		clear(seen)
		seenPool.Put(seen)
	}()

	clonedWriters := make(MetricsWriterList, 0, len(writers))
	for _, writer := range writers {
		clonedStores := make([]*MetricsStore, 0, len(writer.stores))
		for i, store := range writer.stores {
			clonedStore := &MetricsStore{
				metrics:               store.metrics,
				lastResourceVersion:   store.lastResourceVersion,
				lastResourceVersionMu: store.lastResourceVersionMu,
			}
			if i == 0 {
				clonedHeaders := make([]string, len(store.headers))
				if isTextPlain {
					copy(clonedHeaders, store.headersTextPlain)
				} else {
					copy(clonedHeaders, store.headersOpenMetrics)
				}
				clonedStore.headers = clonedHeaders
			}
			clonedStores = append(clonedStores, clonedStore)
		}

		// Deduplicate and rewrite headers on the cloned store in the same pass.
		if len(clonedStores) > 0 {
			headers := clonedStores[0].headers
			metricNames := writer.stores[0].metricNames
			for i, mName := range metricNames {
				if headers[i] == "" {
					continue
				}
				if mName != "" {
					if _, isSeen := seen[mName]; isSeen {
						headers[i] = ""
						continue
					}
					seen[mName] = struct{}{}
				}
			}
		}

		clonedWriters = append(clonedWriters, &MetricsWriter{stores: clonedStores, ResourceName: writer.ResourceName})
	}

	return clonedWriters
}
