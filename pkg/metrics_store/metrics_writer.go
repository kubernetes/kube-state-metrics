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

	for i, help := range m.stores[0].headers {
		var err error

		// SanitizeHeaders blanks duplicate headers to preserve header/family
		// index alignment. An empty header means suppress the header text but
		// still emit the metric family bytes at this index.
		if help != "" {
			// Avoid allocating a new string if the header lacks a trailing newline:
			// check once and emit "\n" as a second write if needed.
			needsNewline := help[len(help)-1] != '\n'
			m.stores[0].metrics.Range(func(_ interface{}, _ interface{}) bool {
				_, err = io.WriteString(w, help)
				if err != nil {
					return false
				}
				if needsNewline {
					_, err = io.WriteString(w, "\n")
				}
				return false
			})
			if err != nil {
				return fmt.Errorf("failed to write help text: %v", err)
			}
		}

		for _, s := range m.stores {
			s.metrics.Range(func(_ interface{}, value interface{}) bool {
				metricFamilies := value.([][]byte)
				_, err = w.Write(metricFamilies[i])
				if err != nil {
					err = fmt.Errorf("failed to write metrics family: %v", err)
					return false
				}
				return true
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// scanHeader inspects a header string and returns deduplication/modification metadata.
// It does not mutate seen — callers must update that map themselves.
// Headers are expected to have the form "# HELP <name> <desc>\n# TYPE <name> <type>\n".
// typeLastSpace is the byte offset of the space before the type suffix in header, or -1.
func scanHeader(header string, seen map[string]struct{}, isTextPlain bool) (shouldRemove, needsModification bool, metricName string, typeLastSpace int) {
	rest := header
	typeLastSpace = -1

	// Parse HELP line.
	if len(rest) > len(helpPrefix) && rest[:len(helpPrefix)] == helpPrefix {
		rest = rest[len(helpPrefix):]
		if spaceIdx := strings.IndexByte(rest, ' '); spaceIdx > 0 {
			metricName = rest[:spaceIdx]
			if _, isSeen := seen[metricName]; isSeen {
				return true, false, metricName, -1
			}
		}
		// Advance past the rest of the HELP line.
		nl := strings.IndexByte(rest, '\n')
		if nl == -1 {
			return false, false, metricName, -1
		}
		rest = rest[nl+1:]
	}

	// Parse TYPE line.
	if len(rest) > len(typePrefix) && rest[:len(typePrefix)] == typePrefix {
		rest = rest[len(typePrefix):]
		if spaceIdx := strings.IndexByte(rest, ' '); spaceIdx > 0 {
			typeName := rest[:spaceIdx]
			if _, isSeen := seen[typeName]; isSeen {
				return true, false, metricName, -1
			}
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

	return false, needsModification, metricName, typeLastSpace
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
	// Pre-count total headers to size the seen map accurately upfront.
	capHint := 0
	for _, writer := range writers {
		if len(writer.stores) > 0 {
			capHint += len(writer.stores[0].headers)
		}
	}

	isTextPlain := contentType.FormatType() == expfmt.TypeTextPlain
	// A single map replaces the former seenHELP+seenTYPE pair: HELP and TYPE lines
	// always share the same metric name, so one lookup per header suffices.
	seen := make(map[string]struct{}, capHint)

	clonedWriters := make(MetricsWriterList, 0, len(writers))
	for _, writer := range writers {
		clonedStores := make([]*MetricsStore, 0, len(writer.stores))
		for i, store := range writer.stores {
			clonedStore := &MetricsStore{
				metrics: store.metrics,
			}
			if i == 0 {
				clonedHeaders := make([]string, len(store.headers))
				copy(clonedHeaders, store.headers)
				clonedStore.headers = clonedHeaders
			}
			clonedStores = append(clonedStores, clonedStore)
		}

		// Deduplicate and rewrite headers on the cloned store in the same pass.
		if len(clonedStores) > 0 {
			headers := clonedStores[0].headers
			for i, header := range headers {
				if header == "" {
					continue
				}

				shouldRemove, needsModification, metricName, lastSpace := scanHeader(header, seen, isTextPlain)

				if shouldRemove {
					headers[i] = ""
					continue
				}

				if metricName != "" {
					seen[metricName] = struct{}{}
				}

				if !needsModification {
					if header[len(header)-1] != '\n' {
						headers[i] = header + "\n"
					}
					continue
				}

				// Surgical replacement: only modify the TYPE line suffix.
				if rewritten := rewriteTypeLine(header, lastSpace); rewritten != "" {
					headers[i] = rewritten
				} else if header[len(header)-1] != '\n' {
					headers[i] = header + "\n"
				}
			}
		}

		clonedWriters = append(clonedWriters, &MetricsWriter{stores: clonedStores, ResourceName: writer.ResourceName})
	}

	return clonedWriters
}
