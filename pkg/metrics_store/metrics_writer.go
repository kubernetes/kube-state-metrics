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
	"net/http"
	"strings"

	"qoobing.com/gomod/log"
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

func (m MetricsWriter) Push(pushgateway string) error{
	for _, s := range m.stores {
		s.mutex.RLock()
		defer func(s *MetricsStore) {
			s.mutex.RUnlock()
		}(s)
	}

	var builder strings.Builder
	for i, _ := range m.stores[0].headers {
		for _, s := range m.stores {
			for _, metricFamilies := range s.metrics {
				builder.WriteString(string(metricFamilies[i]))
			}
		}
	}
	_, err := http.Post(pushgateway, "text/plain", strings.NewReader(builder.String() + "\n"))
	if err != nil {
		log.Warningf("Error making POST request: %v\n", err)
		return err
	}
	return nil
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

	// If the first store has no headers, but has metrics, we need to write out
	// an empty header to ensure that the metrics are written out correctly.
	if m.stores[0].headers == nil && m.stores[0].metrics != nil {
		m.stores[0].headers = []string{""}
	}
	for i, help := range m.stores[0].headers {
		if help != "" && help != "\n" {
			help += "\n"
		}
		// TODO: This writes out the help text for each metric family, before checking if the metrics for it exist,
		// TODO: which is not ideal, and furthermore, diverges from the OpenMetrics standard.
		_, err := w.Write([]byte(help))
		if err != nil {
			return fmt.Errorf("failed to write help text: %v", err)
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

// SanitizeHeaders removes duplicate headers from the given MetricsWriterList for the same family (generated through CRS).
// These are expected to be consecutive since G** resolution generates groups of similar metrics with same headers before moving onto the next G** spec in the CRS configuration.
func SanitizeHeaders(writers MetricsWriterList) MetricsWriterList {
	var lastHeader string
	for _, writer := range writers {
		for i, header := range writer.stores[0].headers {
			if header == lastHeader {
				writer.stores[0].headers[i] = ""
			} else {
				lastHeader = header
			}
		}
	}
	return writers
}
