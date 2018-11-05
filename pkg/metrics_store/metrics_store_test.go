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

	metrics := ms.GetAll()

	// Expect 3 lines, HELP + 2 metrics
	if len(metrics) != 3 {
		t.Fatalf("expected 2 metrics but got %v", len(metrics))
	}

	for _, id := range serviceIDS {
		found := false
		for _, m := range metrics {
			if strings.Contains(m, fmt.Sprintf("uid=\"%v\"", id)) {
				found = true
			}
		}

		if !found {
			t.Fatalf("expected to find metric with uid %v", id)
		}
	}
}
