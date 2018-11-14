package metrics

import (
	"strings"
	"testing"
)

func TestFamilyString(t *testing.T) {
	m := Metric{
		Name:        "kube_service_info",
		LabelKeys:   []string{"name"},
		LabelValues: []string{"a"},
		Value:       1,
	}

	f := Family{&m}

	expected := "kube_service_info{name=\"a\"} 1"
	s := strings.TrimSpace(f.String())

	if expected != s {
		t.Fatalf("expected %v but got %v", expected, s)
	}
}

func BenchmarkNewMetric(b *testing.B) {
	tests := []struct {
		testName    string
		metricName  string
		labelKeys   []string
		labelValues []string
		value       float64
	}{
		{
			testName:    "value-1",
			metricName:  "kube_pod_container_info",
			labelKeys:   []string{"container", "container_id", "image", "image_id", "namespace", "pod"},
			labelValues: []string{"container2", "docker://cd456", "k8s.gcr.io/hyperkube2", "docker://sha256:bbb", "ns2", "pod2"},
			value:       float64(1),
		},
		{
			testName:    "value-35.7",
			metricName:  "kube_pod_container_info",
			labelKeys:   []string{"container", "container_id", "image", "image_id", "namespace", "pod"},
			labelValues: []string{"container2", "docker://cd456", "k8s.gcr.io/hyperkube2", "docker://sha256:bbb", "ns2", "pod2"},
			value:       float64(35.7),
		},
	}

	for _, test := range tests {
		b.Run(test.testName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				m := Metric{test.metricName, test.labelKeys, test.labelValues, test.value}
				m.String()
			}
		})
	}
}
