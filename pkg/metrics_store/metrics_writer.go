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
		if help != "" && help != "\n" {
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
	var lastHeader string
	for _, writer := range writers {
		if len(writer.stores) > 0 {
			for i := 0; i < len(writer.stores[0].headers); {
				header := writer.stores[0].headers[i]

				// Removes duplicate headers from the given MetricsWriterList for the same family (generated through CRS).
				// These are expected to be consecutive since G** resolution generates groups of similar metrics with same headers before moving onto the next G** spec in the CRS configuration.
				// Skip this step if we encounter a repeated header, as it will be removed.
				if header != lastHeader && strings.HasPrefix(header, "# HELP") {

					// If the requested content type is text/plain, replace "info" and "statesets" with "gauge", as they are not recognized by Prometheus' plain text machinery.
					// When Prometheus requests proto-based formats, this branch is also used because any requested format that is not OpenMetrics falls back to text/plain in metrics_handler.go
					if contentType.FormatType() == expfmt.TypeTextPlain {
						infoTypeString := string(metric.Info)
						stateSetTypeString := string(metric.StateSet)
						if strings.HasSuffix(header, infoTypeString) {
							header = header[:len(header)-len(infoTypeString)] + string(metric.Gauge)
							writer.stores[0].headers[i] = header
						}
						if strings.HasSuffix(header, stateSetTypeString) {
							header = header[:len(header)-len(stateSetTypeString)] + string(metric.Gauge)
							writer.stores[0].headers[i] = header
						}
					}
				}

				// Nullify duplicate headers after the sanitization to not miss out on any new candidates.
				if header == lastHeader {
					writer.stores[0].headers = append(writer.stores[0].headers[:i], writer.stores[0].headers[i+1:]...)

					// Do not increment the index, as the next header is now at the current index.
					continue
				}

				// Update the last header.
				lastHeader = header

				// Move to the next header.
				i++
			}
		}
	}

	return writers
}
