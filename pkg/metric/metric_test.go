/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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
	"strings"
	"testing"
)

func TestFamilyString(t *testing.T) {
	m := Metric{
		LabelKeys:   []string{"namespace"},
		LabelValues: []string{"default"},
		Value:       1,
	}

	f := Family{
		Name:    "kube_pod_info",
		Metrics: []*Metric{&m},
	}

	expected := "kube_pod_info{namespace=\"default\"} 1"
	got := strings.TrimSpace(string(f.ByteSlice()))

	if got != expected {
		t.Fatalf("expected %v but got %v", expected, got)
	}
}

func TestFamilyDefaultLabels(t *testing.T) {
	tests := []struct {
		name     string
		family   Family
		prefix   []byte
		expected string
	}{
		{
			name: "metrics with and without own labels",
			family: Family{
				Name:               "kube_pod_status",
				DefaultLabelKeys:   []string{"namespace", "pod"},
				DefaultLabelValues: []string{"ns1", "pod1"},
				Metrics: []*Metric{
					{Value: 1},
					{LabelKeys: []string{"phase"}, LabelValues: []string{"Running"}, Value: 1},
					{LabelKeys: []string{"phase", "reason"}, LabelValues: []string{"Failed", "x"}, Value: 0},
				},
			},
			expected: `kube_pod_status{namespace="ns1",pod="pod1"} 1
kube_pod_status{namespace="ns1",pod="pod1",phase="Running"} 1
kube_pod_status{namespace="ns1",pod="pod1",phase="Failed",reason="x"} 0
`,
		},
		{
			name: "default label values are escaped",
			family: Family{
				Name:               "kube_x",
				DefaultLabelKeys:   []string{"name"},
				DefaultLabelValues: []string{"a\"b\nc\\"},
				Metrics: []*Metric{
					{Value: 2},
					{LabelKeys: []string{"k"}, LabelValues: []string{"\""}, Value: 3},
				},
			},
			expected: `kube_x{name="a\"b\nc\\"} 2
kube_x{name="a\"b\nc\\",k="\""} 3
`,
		},
		{
			name: "no metrics renders nothing",
			family: Family{
				Name:               "kube_z",
				DefaultLabelKeys:   []string{"name"},
				DefaultLabelValues: []string{"n"},
			},
			expected: "",
		},
	}

	// Values longer than the size hint assumes make the buffer grow while
	// metrics are written, after the default label prefix has been rendered.
	const longValue = 1.2345678901234567e+300
	grow := Family{
		Name:               "kube_y",
		DefaultLabelKeys:   []string{"name"},
		DefaultLabelValues: []string{"n"},
	}
	var growExpected strings.Builder
	growExpected.WriteString("existing\n")
	for range 100 {
		grow.Metrics = append(grow.Metrics, &Metric{Value: longValue})
		growExpected.WriteString(`kube_y{name="n"} 1.2345678901234567e+300` + "\n")
	}
	tests = append(tests, struct {
		name     string
		family   Family
		prefix   []byte
		expected string
	}{
		name:     "appends after existing bytes and survives buffer growth",
		family:   grow,
		prefix:   []byte("existing\n"),
		expected: growExpected.String(),
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := append([]byte(nil), tc.prefix...)
			if got := string(tc.family.AppendBytes(b)); got != tc.expected {
				t.Errorf("expected:\n%s\ngot:\n%s", tc.expected, got)
			}
		})
	}
}

func TestFamilyDefaultLabelsMismatchPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for mismatched default label keys and values")
		}
	}()

	f := Family{
		Name:               "kube_x",
		DefaultLabelKeys:   []string{"a", "b"},
		DefaultLabelValues: []string{"1"},
		Metrics:            []*Metric{{Value: 1}},
	}
	f.ByteSlice()
}

func BenchmarkMetricWrite(b *testing.B) {
	tests := []struct {
		testName       string
		metric         Metric
		expectedLength int
	}{
		{
			testName: "value-1",
			metric: Metric{
				LabelKeys:   []string{"container", "container_id", "image", "image_id", "namespace", "pod"},
				LabelValues: []string{"container2", "docker://cd456", "k8s.gcr.io/hyperkube2", "docker://sha256:bbb", "ns2", "pod2"},
				Value:       float64(1),
			},
			expectedLength: 145,
		},
		{
			testName: "value-35.7",
			metric: Metric{
				LabelKeys:   []string{"container", "container_id", "image", "image_id", "namespace", "pod"},
				LabelValues: []string{"container2", "docker://cd456", "k8s.gcr.io/hyperkube2", "docker://sha256:bbb", "ns2", "pod2"},
				Value:       35.7,
			},
			expectedLength: 148,
		},
	}

	for _, test := range tests {
		b.Run(test.testName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				builder := bytes.Buffer{}

				test.metric.Write(&builder)

				s := builder.Bytes()

				// Ensuring that the string is actually build, not optimized
				// away by compilation.
				got := len(s)
				if test.expectedLength != got {
					b.Fatalf("expected string of length %v but got %v", test.expectedLength, got)
				}
			}
		})
	}
}
