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
		// Skip empty headers (set by SanitizeHeaders for duplicates)
		if help == "" {
			continue
		}

		if help != "\n" {
			help += "\n"
		}

		var err error
		m.stores[0].metrics.Range(func(_ interface{}, _ interface{}) bool {
			_, err = w.Write([]byte(help))
			if err != nil {
				err = fmt.Errorf("failed to write help text: %v", err)
			}
			return false
		})
		if err != nil {
			return err
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

// SanitizeHeaders sanitizes the headers of the given MetricsWriterList.
func SanitizeHeaders(contentType expfmt.Format, writers MetricsWriterList) MetricsWriterList {
	clonedWriters := make(MetricsWriterList, 0, len(writers))
	for _, writer := range writers {
		clonedStores := make([]*MetricsStore, 0, len(writer.stores))
		for _, store := range writer.stores {
			clonedHeaders := make([]string, len(store.headers))
			copy(clonedHeaders, store.headers)
			clonedStore := &MetricsStore{
				headers: clonedHeaders,
			}
			// Share the metrics backing storage by sharing the pointer.
			clonedStore.metrics = store.metrics
			clonedStores = append(clonedStores, clonedStore)
		}
		clonedWriters = append(clonedWriters, &MetricsWriter{stores: clonedStores, ResourceName: writer.ResourceName})
	}

	// Deduplicate by metric name across all writers to handle non-consecutive duplicates during CRS reload.
	seenHELP := make(map[string]struct{})
	seenTYPE := make(map[string]struct{})
	for _, writer := range clonedWriters {
		if len(writer.stores) > 0 {
			for i := 0; i < len(writer.stores[0].headers); i++ {
				header := writer.stores[0].headers[i]
				lines := strings.Split(header, "\n")
				shouldRemove := false
				modifiedLines := make([]string, 0, len(lines))

				for _, line := range lines {
					switch {
					case strings.HasPrefix(line, "# HELP "):
						fields := strings.Fields(line)
						if len(fields) >= 3 {
							metricName := fields[2]
							if _, seen := seenHELP[metricName]; seen {
								shouldRemove = true
								break
							}
							seenHELP[metricName] = struct{}{}
							modifiedLines = append(modifiedLines, line)
						} else {
							modifiedLines = append(modifiedLines, line)
						}
					case strings.HasPrefix(line, "# TYPE "):
						if shouldRemove {
							break
						}
						fields := strings.Fields(line)
						if len(fields) >= 3 {
							metricName := fields[2]
							modifiedLine := line
							if contentType.FormatType() == expfmt.TypeTextPlain {
								infoTypeString := string(metric.Info)
								stateSetTypeString := string(metric.StateSet)
								if strings.HasSuffix(line, infoTypeString) {
									modifiedLine = line[:len(line)-len(infoTypeString)] + string(metric.Gauge)
								} else if strings.HasSuffix(line, stateSetTypeString) {
									modifiedLine = line[:len(line)-len(stateSetTypeString)] + string(metric.Gauge)
								}
							}
							if _, seen := seenTYPE[metricName]; seen {
								shouldRemove = true
								break
							}
							seenTYPE[metricName] = struct{}{}
							modifiedLines = append(modifiedLines, modifiedLine)
						} else {
							modifiedLines = append(modifiedLines, line)
						}
					default:
						modifiedLines = append(modifiedLines, line)
					}
				}

				if shouldRemove {
					writer.stores[0].headers[i] = ""
				} else if len(modifiedLines) > 0 {
					writer.stores[0].headers[i] = strings.Join(modifiedLines, "\n")
				}
			}
		}
	}

	return clonedWriters
}
