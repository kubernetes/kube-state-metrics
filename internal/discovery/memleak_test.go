/*
Copyright 2026 The Kubernetes Authors All rights reserved.
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

package discovery

import (
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func makeGVKPs(n int) []groupVersionKindPlural {
	gvkps := make([]groupVersionKindPlural, n)
	for i := range n {
		gvkps[i] = groupVersionKindPlural{
			GroupVersionKind: schema.GroupVersionKind{
				Group:   fmt.Sprintf("group%d.example.com", i),
				Version: "v1",
				Kind:    fmt.Sprintf("Kind%d", i),
			},
			Plural: fmt.Sprintf("kind%ds", i),
		}
	}
	return gvkps
}

func TestAppendToMapStability(t *testing.T) {
	const (
		numGVKs    = 5
		pollCycles = 500
	)

	gvkps := makeGVKPs(numGVKs)
	d := &CRDiscoverer{}

	for range pollCycles {
		d.AppendToMap(gvkps...)
	}

	kindCount := 0
	for _, versions := range d.Map {
		for _, kinds := range versions {
			kindCount += len(kinds)
		}
	}
	if kindCount != numGVKs {
		t.Errorf("expected exactly %d kind entries after %d poll cycles, got %d", numGVKs, pollCycles, kindCount)
	}
	if got := len(d.GVKToReflectorStopChanMap); got != numGVKs {
		t.Errorf("expected exactly %d stop channels after %d poll cycles, got %d", numGVKs, pollCycles, got)
	}
}
