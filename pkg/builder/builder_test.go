/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

	"k8s.io/kube-state-metrics/v2/pkg/allowdenylist"
	"k8s.io/kube-state-metrics/v2/pkg/builder"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	dummyMetricList0 = []string{"metric0.1", "metric0.2"}
	dummyMetricList1 = []string{"metric1.1", "metric1.2"}
)

// BuilderInterface and Builder are public, and designed to allow
// injecting custom stores notably when ksm is used as a library.
// This test case ensures we don't break compatibility for external consumers.
func TestBuilderWithCustomStore(t *testing.T) {
	b := builder.NewBuilder()
	b.WithAllowDenyList(&allowdenylist.AllowDenyList{})
	b.WithEnabledResources([]string{"pods"})
	b.WithGenerateStoresFunc(customStore)
	stores := b.BuildStores()

	store0, ok := stores[0][0].(*dummyStore)
	if !ok {
		t.Fatal("Couldn't cast custom metrics store")
	}

	if !reflect.DeepEqual(store0.metrics, dummyMetricList0) {
		t.Fatalf("Unexpected store values: want %v found %v", dummyMetricList0, store0.metrics)
	}

	store1, ok := stores[0][1].(*dummyStore)
	if !ok {
		t.Fatal("Couldn't cast custom metrics store")
	}

	if !reflect.DeepEqual(store1.metrics, dummyMetricList1) {
		t.Fatalf("Unexpected store values: want %v found %v", dummyMetricList1, store1.metrics)
	}
}

func customStore(metricFamilies []generator.FamilyGenerator,
	expectedType interface{},
	listWatchFunc func(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher,
	useAPIServerCache bool,
) []cache.Store {
	stores := make([]cache.Store, 0, 2)
	stores = append(stores, newDummyStore(dummyMetricList0))
	stores = append(stores, newDummyStore(dummyMetricList1))
	return stores
}

func newDummyStore(metrics []string) *dummyStore {
	return &dummyStore{
		metrics: metrics,
	}
}

type dummyStore struct {
	metrics []string
}

func (s *dummyStore) Add(obj interface{}) error {
	return nil
}

func (s *dummyStore) Update(obj interface{}) error {
	return nil
}

func (s *dummyStore) Delete(obj interface{}) error {
	return nil
}

func (s *dummyStore) List() []interface{} {
	return nil
}

func (s *dummyStore) ListKeys() []string {
	return nil
}

func (s *dummyStore) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

func (s *dummyStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

func (s *dummyStore) Replace(list []interface{}, _ string) error {
	return nil
}

func (s *dummyStore) Resync() error {
	return nil
}
