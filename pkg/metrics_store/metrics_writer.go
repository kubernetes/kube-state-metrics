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
		_, err := w.Write([]byte(help + "\n"))
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
