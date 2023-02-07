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
)

// Mock metricFamily instead of importing /pkg/metric to prevent cyclic
// dependency.
type metricFamily struct {
	value []byte
}

// Implement FamilyByteSlicer interface.
func (f *metricFamily) ByteSlice() []byte {
	return f.value
}

func TestObjectsSameNameDifferentNamespaces(t *testing.T) {
	serviceIDS := []string{"a", "b"}

	genFunc := func(obj interface{}) []FamilyByteSlicer {
		o, err := meta.Accessor(obj)
		if err != nil {
			t.Fatal(err)
		}

		metricFamily := metricFamily{
			[]byte(fmt.Sprintf("kube_service_info{uid=\"%v\"} 1", string(o.GetUID()))),
		}

		return []FamilyByteSlicer{&metricFamily}
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

	for _, id := range serviceIDS {
		if !strings.Contains(m, fmt.Sprintf("uid=\"%v\"", id)) {
			t.Fatalf("expected to find metric with uid %v", id)
		}
	}
}
