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
			for i, header := range writer.stores[0].headers {
				// If the requested content type is text/plain, replace "info" and "statesets" with "gauge", as they are not recognized by Prometheus' plain text machinery.
				// When Prometheus requests proto-based formats, this branch is also used because any requested format that is not OpenMetrics falls back to text/plain in metrics_handler.go.
				if strings.HasPrefix(header, "# HELP") && contentType.FormatType() == expfmt.TypeTextPlain {
					infoTypeString := string(metric.Info)
					stateSetTypeString := string(metric.StateSet)
					if strings.HasSuffix(header, infoTypeString) {
						header = header[:len(header)-len(infoTypeString)] + string(metric.Gauge)
					}
					if strings.HasSuffix(header, stateSetTypeString) {
						header = header[:len(header)-len(stateSetTypeString)] + string(metric.Gauge)
					}
				}

				// Keep header indexing stable with metric families. Duplicate headers are blanked instead of removed.
				if header == "" || header == "\n" || header == lastHeader {
					writer.stores[0].headers[i] = ""
					continue
				}

				writer.stores[0].headers[i] = header
				lastHeader = header
			}
		}
	}

	return writers
}
