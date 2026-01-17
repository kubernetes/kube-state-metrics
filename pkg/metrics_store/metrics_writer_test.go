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
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/common/expfmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
)

func TestWriteAllWithSingleStore(t *testing.T) {
	genFunc := func(obj interface{}) []metric.FamilyInterface {
		o, err := meta.Accessor(obj)
		if err != nil {
			t.Fatal(err)
		}

		mf1 := metric.Family{
			Name: "kube_service_info_1",
			Metrics: []*metric.Metric{
				{
					LabelKeys:   []string{"namespace", "uid"},
					LabelValues: []string{o.GetNamespace(), string(o.GetUID())},
					Value:       float64(1),
				},
			},
		}

		mf2 := metric.Family{
			Name: "kube_service_info_2",
			Metrics: []*metric.Metric{
				{
					LabelKeys:   []string{"namespace", "uid"},
					LabelValues: []string{o.GetNamespace(), string(o.GetUID())},
					Value:       float64(1),
				},
			},
		}

		return []metric.FamilyInterface{&mf1, &mf2}
	}
	store := NewMetricsStore([]string{"Info 1 about services", "Info 2 about services"}, genFunc)
	svcs := []v1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       "a1",
				Name:      "service",
				Namespace: "a",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       "a2",
				Name:      "service",
				Namespace: "a",
			},
		},
	}
	for _, s := range svcs {
		svc := s
		if err := store.Add(&svc); err != nil {
			t.Fatal(err)
		}
	}

	multiNsWriter := NewMetricsWriter("test", store)
	w := strings.Builder{}
	err := multiNsWriter.WriteAll(&w)
	if err != nil {
		t.Fatalf("failed to write metrics: %v", err)
	}
	result := w.String()

	resultLines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	if len(resultLines) != 6 {
		t.Fatalf("Invalid number of series, got %d, want %d", len(resultLines), 6)
	}
	if resultLines[0] != "Info 1 about services" {
		t.Fatalf("Invalid metrics header on line 0, got %s, want %s", resultLines[0], "Info 1 about services")
	}
	if resultLines[3] != "Info 2 about services" {
		t.Fatalf("Invalid metrics header on line 3, got %s, want %s", resultLines[3], "Info 2 about services")
	}

	expectedSeries := []string{
		`kube_service_info_1{namespace="a",uid="a1"} 1`,
		`kube_service_info_1{namespace="a",uid="a2"} 1`,
		`kube_service_info_2{namespace="a",uid="a1"} 1`,
		`kube_service_info_2{namespace="a",uid="a2"} 1`,
	}

	for _, series := range expectedSeries {
		if !strings.Contains(result, series) {
			t.Fatalf("Did not find expected series %s", series)
		}
	}
}

func TestWriteAllWithMultipleStores(t *testing.T) {
	genFunc := func(obj interface{}) []metric.FamilyInterface {
		o, err := meta.Accessor(obj)
		if err != nil {
			t.Fatal(err)
		}

		mf1 := metric.Family{
			Name: "kube_service_info_1",
			Metrics: []*metric.Metric{
				{
					LabelKeys:   []string{"namespace", "uid"},
					LabelValues: []string{o.GetNamespace(), string(o.GetUID())},
					Value:       float64(1),
				},
			},
		}

		mf2 := metric.Family{
			Name: "kube_service_info_2",
			Metrics: []*metric.Metric{
				{
					LabelKeys:   []string{"namespace", "uid"},
					LabelValues: []string{o.GetNamespace(), string(o.GetUID())},
					Value:       float64(1),
				},
			},
		}

		return []metric.FamilyInterface{&mf1, &mf2}
	}
	s1 := NewMetricsStore([]string{"Info 1 about services", "Info 2 about services"}, genFunc)
	svcs1 := []v1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       "a1",
				Name:      "service",
				Namespace: "a",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       "a2",
				Name:      "service",
				Namespace: "a",
			},
		},
	}
	for _, s := range svcs1 {
		svc := s
		if err := s1.Add(&svc); err != nil {
			t.Fatal(err)
		}
	}

	svcs2 := []v1.Service{
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       "b1",
				Name:      "service",
				Namespace: "b",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				UID:       "b2",
				Name:      "service",
				Namespace: "b",
			},
		},
	}
	s2 := NewMetricsStore([]string{"Info 1 about services", "Info 2 about services"}, genFunc)
	for _, s := range svcs2 {
		svc := s
		if err := s2.Add(&svc); err != nil {
			t.Fatal(err)
		}
	}

	multiNsWriter := NewMetricsWriter("test", s1, s2)
	w := strings.Builder{}
	err := multiNsWriter.WriteAll(&w)
	if err != nil {
		t.Fatalf("failed to write metrics: %v", err)
	}
	result := w.String()

	resultLines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	if len(resultLines) != 10 {
		t.Fatalf("Invalid number of series, got %d, want %d", len(resultLines), 10)
	}
	if resultLines[0] != "Info 1 about services" {
		t.Fatalf("Invalid metrics header on line 0, got %s, want %s", resultLines[0], "Info 1 about services")
	}
	if resultLines[5] != "Info 2 about services" {
		t.Fatalf("Invalid metrics header on line 0, got %s, want %s", resultLines[5], "Info 2 about services")
	}

	expectedSeries := []string{
		`kube_service_info_1{namespace="a",uid="a1"} 1`,
		`kube_service_info_1{namespace="a",uid="a2"} 1`,
		`kube_service_info_1{namespace="b",uid="b1"} 1`,
		`kube_service_info_1{namespace="b",uid="b2"} 1`,
		`kube_service_info_2{namespace="a",uid="a1"} 1`,
		`kube_service_info_2{namespace="a",uid="a2"} 1`,
		`kube_service_info_2{namespace="b",uid="b1"} 1`,
		`kube_service_info_2{namespace="b",uid="b2"} 1`,
	}

	for _, series := range expectedSeries {
		if !strings.Contains(result, series) {
			t.Fatalf("Did not find expected series %s", series)
		}
	}
}

// TestWriteAllWithEmptyStores checks that nothing is printed if no metrics exist for metric families.
func TestWriteAllWithEmptyStores(t *testing.T) {
	genFunc := func(_ interface{}) []metric.FamilyInterface {
		mf1 := metric.Family{
			Name:    "kube_service_info_1",
			Metrics: []*metric.Metric{},
		}

		mf2 := metric.Family{
			Name:    "kube_service_info_2",
			Metrics: []*metric.Metric{},
		}

		return []metric.FamilyInterface{&mf1, &mf2}
	}
	store := NewMetricsStore([]string{"Info 1 about services", "Info 2 about services"}, genFunc)

	multiNsWriter := NewMetricsWriter("test", store)
	w := strings.Builder{}
	err := multiNsWriter.WriteAll(&w)
	if err != nil {
		t.Fatalf("failed to write metrics: %v", err)
	}
	result := w.String()
	fmt.Println(result)

	if result != "" {
		t.Fatalf("Unexpected output, got %q, want %q", result, "")
	}
}

// No two consecutive headers will be entirely the same. The cases used below are only for their suffixes.
func TestSanitizeHeaders(t *testing.T) {
	testcases := []struct {
		name            string
		contentType     expfmt.Format
		headers         []string
		expectedHeaders []string
	}{
		{
			name:        "OpenMetricsText unique headers",
			contentType: expfmt.NewFormat(expfmt.TypeOpenMetrics),
			headers: []string{
				"",
				"# HELP foo foo_help\n# TYPE foo gauge",
				"# HELP foo foo_help\n# TYPE foo info",
				"# HELP foo foo_help\n# TYPE foo stateset",
				"# HELP foo foo_help\n# TYPE foo counter",
			},
			expectedHeaders: []string{
				"# HELP foo foo_help\n# TYPE foo gauge",
				"# HELP foo foo_help\n# TYPE foo info",
				"# HELP foo foo_help\n# TYPE foo stateset",
				"# HELP foo foo_help\n# TYPE foo counter",
			},
		},
		{
			name:        "OpenMetricsText consecutive duplicate headers",
			contentType: expfmt.NewFormat(expfmt.TypeOpenMetrics),
			headers: []string{
				"",
				"",
				"",
				"# HELP foo foo_help\n# TYPE foo gauge",
				"# HELP foo foo_help\n# TYPE foo gauge",
				"# HELP foo foo_help\n# TYPE foo info",
				"# HELP foo foo_help\n# TYPE foo info",
				"# HELP foo foo_help\n# TYPE foo stateset",
				"# HELP foo foo_help\n# TYPE foo stateset",
				"# HELP foo foo_help\n# TYPE foo counter",
				"# HELP foo foo_help\n# TYPE foo counter",
			},
			expectedHeaders: []string{
				"# HELP foo foo_help\n# TYPE foo gauge",
				"# HELP foo foo_help\n# TYPE foo info",
				"# HELP foo foo_help\n# TYPE foo stateset",
				"# HELP foo foo_help\n# TYPE foo counter",
			},
		},
		{
			name:        "text-format unique headers",
			contentType: expfmt.NewFormat(expfmt.TypeTextPlain),
			headers: []string{
				"",
				"# HELP foo foo_help\n# TYPE foo gauge",
				"# HELP foo foo_help\n# TYPE foo info",
				"# HELP foo foo_help\n# TYPE foo stateset",
				"# HELP foo foo_help\n# TYPE foo counter",
			},
			expectedHeaders: []string{
				"# HELP foo foo_help\n# TYPE foo gauge",
				"# HELP foo foo_help\n# TYPE foo counter",
			},
		},
		{
			name:        "text-format consecutive duplicate headers",
			contentType: expfmt.NewFormat(expfmt.TypeTextPlain),
			headers: []string{
				"",
				"",
				"",
				"# HELP foo foo_help\n# TYPE foo gauge",
				"# HELP foo foo_help\n# TYPE foo gauge",
				"# HELP foo foo_help\n# TYPE foo info",
				"# HELP foo foo_help\n# TYPE foo info",
				"# HELP foo foo_help\n# TYPE foo stateset",
				"# HELP foo foo_help\n# TYPE foo stateset",
				"# HELP foo foo_help\n# TYPE foo counter",
				"# HELP foo foo_help\n# TYPE foo counter",
			},
			expectedHeaders: []string{
				"# HELP foo foo_help\n# TYPE foo gauge",
				"# HELP foo foo_help\n# TYPE foo counter",
			},
		},
	}

	for _, testcase := range testcases {
		writer := NewMetricsWriter("test", NewMetricsStore(testcase.headers, nil))
		t.Run(testcase.name, func(t *testing.T) {
			SanitizeHeaders(testcase.contentType, MetricsWriterList{writer})
			if !reflect.DeepEqual(testcase.expectedHeaders, writer.stores[0].headers) {
				t.Fatalf("(-want, +got):\n%s", cmp.Diff(testcase.expectedHeaders, writer.stores[0].headers))
			}
		})
	}
}

func BenchmarkSanitizeHeaders(b *testing.B) {
	benchmarks := []struct {
		name                      string
		contentType               expfmt.Format
		writersContainsDuplicates bool
	}{
		{
			name:                      "text-format unique headers",
			contentType:               expfmt.NewFormat(expfmt.TypeTextPlain),
			writersContainsDuplicates: false,
		},
		{
			name:                      "text-format duplicate headers",
			contentType:               expfmt.NewFormat(expfmt.TypeTextPlain),
			writersContainsDuplicates: true,
		},
		{
			name:                      "proto-format unique headers",
			contentType:               expfmt.NewFormat(expfmt.TypeProtoText), // Prometheus ProtoFmt is the only proto-based format we check for.
			writersContainsDuplicates: false,
		},
		{
			name:                      "proto-format duplicate headers",
			contentType:               expfmt.NewFormat(expfmt.TypeProtoText), // Prometheus ProtoFmt is the only proto-based format we check for.
			writersContainsDuplicates: true,
		},
	}

	for _, benchmark := range benchmarks {
		headers := []string{}
		for j := 0; j < 10e4; j++ {
			if benchmark.writersContainsDuplicates {
				headers = append(headers, "# HELP foo foo_help\n# TYPE foo info")
			} else {
				headers = append(headers, fmt.Sprintf("# HELP foo_%d foo_help\n# TYPE foo_%d info", j, j))
			}
		}
		writer := NewMetricsWriter("test", NewMetricsStore(headers, nil))
		b.Run(benchmark.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				SanitizeHeaders(benchmark.contentType, MetricsWriterList{writer})
			}
		})
	}
}
