package metricsstore

import (
	"fmt"
	"strings"
	"testing"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kube-state-metrics/pkg/metrics"
)

func TestObjectsSameNameDifferentNamespaces(t *testing.T) {
	serviceIDS := []string{"a", "b"}

	genFunc := func(obj interface{}) [][]*metrics.Metric {
		o, err := meta.Accessor(obj)
		if err != nil {
			t.Fatal(err)
		}

		metric := metrics.Metric(fmt.Sprintf("kube_service_info{uid=\"%v\"} 1\n", o.GetUID()))
		metricFamily := []*metrics.Metric{&metric}

		return [][]*metrics.Metric{metricFamily}
	}

	ms := NewMetricsStore([]string{"Information about service."}, genFunc)

	for _, id := range serviceIDS {
		s := v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "service",
				Namespace: id,
				UID:       types.UID(id),
			},
		}

		err := ms.Add(&s)
		if err != nil {
			t.Fatal(err)
		}
	}

	w := strings.Builder{}
	ms.WriteAll(&w)
	m := w.String()

	// Expect 4 lines, HELP + 2 metrics + newline
	if len(strings.Split(m, "\n")) != 4 {
		t.Fatalf("expected 2 metrics but got %v", len(strings.Split(m, "\n")))
	}

	for _, id := range serviceIDS {
		if !strings.Contains(m, fmt.Sprintf("uid=\"%v\"", id)) {

			t.Fatalf("expected to find metric with uid %v", id)
		}
	}
}
