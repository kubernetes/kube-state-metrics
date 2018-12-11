package metrics

import (
	"strings"
	"testing"
)

func TestFamilyString(t *testing.T) {
	m := Metric{
		Name:        "kube_pod_info",
		LabelKeys:   []string{"namespace"},
		LabelValues: []string{"default"},
		Value:       1,
	}

	f := Family{&m}

	expected := "kube_pod_info{namespace=\"default\"} 1"
	got := strings.TrimSpace(f.String())

	if got != expected {
		t.Fatalf("expected %v but got %v", expected, got)
	}
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
				Name:        "kube_pod_container_info",
				LabelKeys:   []string{"container", "container_id", "image", "image_id", "namespace", "pod"},
				LabelValues: []string{"container2", "docker://cd456", "k8s.gcr.io/hyperkube2", "docker://sha256:bbb", "ns2", "pod2"},
				Value:       float64(1),
			},
			expectedLength: 168,
		},
		{
			testName: "value-35.7",
			metric: Metric{
				Name:        "kube_pod_container_info",
				LabelKeys:   []string{"container", "container_id", "image", "image_id", "namespace", "pod"},
				LabelValues: []string{"container2", "docker://cd456", "k8s.gcr.io/hyperkube2", "docker://sha256:bbb", "ns2", "pod2"},
				Value:       float64(35.7),
			},
			expectedLength: 171,
		},
	}

	for _, test := range tests {
		b.Run(test.testName, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				builder := strings.Builder{}

				test.metric.Write(&builder)

				s := builder.String()

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
