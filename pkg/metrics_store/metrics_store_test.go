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

package metricsstore

import (
	"fmt"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
)

func TestObjectsSameNameDifferentNamespaces(t *testing.T) {
	serviceIDs := []string{"a", "b"}

	genFunc := func(obj interface{}) []metric.FamilyInterface {
		o, err := meta.Accessor(obj)
		if err != nil {
			t.Fatal(err)
		}

		metricFamily := metric.Family{
			Name: "kube_service_info",
			Metrics: []*metric.Metric{
				{
					LabelKeys:   []string{"uid"},
					LabelValues: []string{string(o.GetUID())},
					Value:       float64(1),
				},
			},
		}

		return []metric.FamilyInterface{&metricFamily}
	}

	ms := NewMetricsStore([]string{"Information about service."}, genFunc)

	for _, id := range serviceIDs {
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
	mw := NewMetricsWriter("test", ms)
	err := mw.WriteAll(&w)
	if err != nil {
		t.Fatalf("failed to write metrics: %v", err)
	}
	m := w.String()

	for _, id := range serviceIDs {
		if !strings.Contains(m, fmt.Sprintf("uid=\"%v\"", id)) {
			t.Fatalf("expected to find metric with uid %v", id)
		}
	}
}

func TestMetricsStoreResourceVersion(t *testing.T) {
	ms := NewMetricsStore([]string{}, func(_ interface{}) []metric.FamilyInterface { return nil })

	// Test Add
	s1 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			UID:             "uid1",
			ResourceVersion: "123",
		},
	}
	if err := ms.Add(s1); err != nil {
		t.Fatal(err)
	}
	if rv := ms.LastStoreSyncResourceVersion(); rv != "123" {
		t.Fatalf("expected resource version 123, got %v", rv)
	}

	// Test Update
	s1.ResourceVersion = "124"
	if err := ms.Update(s1); err != nil {
		t.Fatal(err)
	}
	if rv := ms.LastStoreSyncResourceVersion(); rv != "124" {
		t.Fatalf("expected resource version 124, got %v", rv)
	}

	// Test Delete
	s1.ResourceVersion = "125"
	if err := ms.Delete(s1); err != nil {
		t.Fatal(err)
	}
	if rv := ms.LastStoreSyncResourceVersion(); rv != "125" {
		t.Fatalf("expected resource version 125, got %v", rv)
	}

	// Test Bookmark
	ms.Bookmark("126")
	if rv := ms.LastStoreSyncResourceVersion(); rv != "126" {
		t.Fatalf("expected resource version 126, got %v", rv)
	}

	// Test Replace
	if err := ms.Replace([]interface{}{}, "127"); err != nil {
		t.Fatal(err)
	}
	s2 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			UID:             "uid2",
			ResourceVersion: "1",
		},
	}
	if err := ms.Replace([]interface{}{s2}, "127"); err != nil {
		t.Fatal(err)
	}
	if rv := ms.LastStoreSyncResourceVersion(); rv != "127" {
		t.Fatalf("expected resource version 127, got %v", rv)
	}
}
