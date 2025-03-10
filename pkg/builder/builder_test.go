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

package builder_test

import (
	"reflect"
	"testing"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/builder"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	fakeMetricLists = [][]string{
		{"metric0.1", "metric0.2"},
		{"metric1.1", "metric1.2"},
	}
)

// BuilderInterface and Builder are public, and designed to allow
// injecting custom stores notably when ksm is used as a library.
// This test case ensures we don't break compatibility for external consumers.
func TestBuilderWithCustomStore(t *testing.T) {
	b := builder.NewBuilder()
	b.WithFamilyGeneratorFilter(generator.NewCompositeFamilyGeneratorFilter())
	err := b.WithEnabledResources([]string{"pods"})
	if err != nil {
		t.Fatal(err)
	}

	b.WithGenerateStoresFunc(customStore)
	var fStores []*fakeStore
	for _, stores := range b.BuildStores() {
		for _, store := range stores {
			fStores = append(fStores, store.(*fakeStore))
		}
	}

	for i, fStore := range fStores {
		metrics := fStore.List()
		for j, m := range metrics {
			if !reflect.DeepEqual(m, fakeMetricLists[i][j]) {
				t.Fatalf("Unexpected store values: want %v found %v", fakeMetricLists[i], metrics)
			}
		}
	}
}

func customStore(_ []generator.FamilyGenerator,
	_ interface{},
	_ func(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher,
	_ bool,
	_ int64,
) []cache.Store {
	stores := make([]cache.Store, 0, 2)
	stores = append(stores, newFakeStore(fakeMetricLists[0]))
	stores = append(stores, newFakeStore(fakeMetricLists[1]))
	return stores
}

func newFakeStore(metrics []string) *fakeStore {
	return &fakeStore{
		metrics: metrics,
	}
}

type fakeStore struct {
	metrics []string
}

func (s *fakeStore) Add(_ interface{}) error {
	return nil
}

func (s *fakeStore) Update(_ interface{}) error {
	return nil
}

func (s *fakeStore) Delete(_ interface{}) error {
	return nil
}

func (s *fakeStore) List() []interface{} {
	metrics := make([]interface{}, len(s.metrics))
	for i, m := range s.metrics {
		metrics[i] = m
	}

	return metrics
}

func (s *fakeStore) ListKeys() []string {
	return nil
}

func (s *fakeStore) Get(_ interface{}) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

func (s *fakeStore) GetByKey(_ string) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

func (s *fakeStore) Replace(_ []interface{}, _ string) error {
	return nil
}

func (s *fakeStore) Resync() error {
	return nil
}
