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
	stores []*MetricsStore
}

// NewMetricsWriter creates a new MetricsWriter.
func NewMetricsWriter(stores ...*MetricsStore) *MetricsWriter {
	return &MetricsWriter{
		stores: stores,
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

	for _, s := range m.stores {
		s.mutex.RLock()
		defer func(s *MetricsStore) {
			s.mutex.RUnlock()
		}(s)
	}

	for i, help := range m.stores[0].headers {
		if help != "" && help != "\n" {
			help += "\n"
		}

		if len(m.stores[0].metrics) > 0 {
			_, err := w.Write([]byte(help))
			if err != nil {
				return fmt.Errorf("failed to write help text: %v", err)
			}
		}

		for _, s := range m.stores {
			for _, metricFamilies := range s.metrics {
				_, err := w.Write(metricFamilies[i])
				if err != nil {
					return fmt.Errorf("failed to write metrics family: %v", err)
				}
			}
		}
	}
	return nil
}

// SanitizeHeaders sanitizes the headers of the given MetricsWriterList.
func SanitizeHeaders(contentType string, writers MetricsWriterList) MetricsWriterList {
	var lastHeader string
	for _, writer := range writers {
		if len(writer.stores) > 0 {
			for i, header := range writer.stores[0].headers {

				// Nothing to sanitize.
				if len(header) == 0 {
					continue
				}

				// Removes duplicate headers from the given MetricsWriterList for the same family (generated through CRS).
				// These are expected to be consecutive since G** resolution generates groups of similar metrics with same headers before moving onto the next G** spec in the CRS configuration.
				if header == lastHeader {
					writer.stores[0].headers[i] = ""
				} else if strings.HasPrefix(header, "# HELP") {
					lastHeader = header

					// If the requested content type was proto-based, replace the type with "gauge", as "info" and "statesets" are not recognized by Prometheus' protobuf machinery,
					// else replace them by their respective string representations.
					if strings.HasPrefix(contentType, expfmt.ProtoType) &&
						(strings.HasSuffix(header, metric.InfoN.NString()) || strings.HasSuffix(header, metric.StateSetN.NString())) {
						writer.stores[0].headers[i] = header[:len(header)-1] + string(metric.Gauge)
					}

					// Replace all remaining type enums with their string representations.
					n := int(header[len(header)-1]) - '0'
					if n >= 0 && n < len(metric.TypeNMap) {
						writer.stores[0].headers[i] = header[:len(header)-1] + string(metric.TypeNMap[metric.TypeN(n)])
					}
				}
			}
		}
	}
	return writers
}
