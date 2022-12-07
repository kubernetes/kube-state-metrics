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

package metricsstore_test

import (
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
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
	store := metricsstore.NewMetricsStore([]string{"Info 1 about services", "Info 2 about services"}, genFunc)
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

	multiNsWriter := metricsstore.NewMetricsWriter(store)
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
	s1 := metricsstore.NewMetricsStore([]string{"Info 1 about services", "Info 2 about services"}, genFunc)
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
	s2 := metricsstore.NewMetricsStore([]string{"Info 1 about services", "Info 2 about services"}, genFunc)
	for _, s := range svcs2 {
		svc := s
		if err := s2.Add(&svc); err != nil {
			t.Fatal(err)
		}
	}

	multiNsWriter := metricsstore.NewMetricsWriter(s1, s2)
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
