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
	"context"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/goleak"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/internal/store"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

type customWatch struct {
	watch.Interface
	onStop func()
}

func (w *customWatch) Stop() {
	w.Interface.Stop()
	w.onStop()
}

type dummyFactory struct {
	gvk       schema.GroupVersionKind
	watchOnce sync.Once
	stopOnce  sync.Once
	watchRun  chan struct{}
	watchStop chan struct{}
}

func (f *dummyFactory) Name() string {
	return "foos"
}

func (f *dummyFactory) CreateClient(_ *rest.Config) (interface{}, error) {
	return nil, nil
}

func (f *dummyFactory) MetricFamilyGenerators() []generator.FamilyGenerator {
	return nil
}

func (f *dummyFactory) ExpectedType() interface{} {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(f.gvk)
	return u
}

func (f *dummyFactory) ListWatch(_ interface{}, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(_ metav1.ListOptions) (runtime.Object, error) {
			return &unstructured.UnstructuredList{}, nil
		},
		WatchFunc: func(_ metav1.ListOptions) (watch.Interface, error) {
			f.watchOnce.Do(func() { close(f.watchRun) })
			w := watch.NewFake()
			return &customWatch{
				Interface: w,
				onStop: func() {
					f.stopOnce.Do(func() { close(f.watchStop) })
				},
			}, nil
		},
	}
}

func TestCRReflectorLifecycleLeak(t *testing.T) {
	// Call VerifyNone at the very start to establish baseline
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gvkp := groupVersionKindPlural{
		GroupVersionKind: schema.GroupVersionKind{
			Group:   "example.com",
			Version: "v1",
			Kind:    "Foo",
		},
		Plural: "foos",
	}

	d := &CRDiscoverer{}

	// 1. Initial GVK discovery
	d.AppendToMap(gvkp)

	b := store.NewBuilder()
	b.WithContext(ctx)
	b.WithNamespaces(options.NamespaceList{metav1.NamespaceAll})
	b.WithFamilyGeneratorFilter(generator.NewCompositeFamilyGeneratorFilter())
	b.WithMetrics(prometheus.NewRegistry())
	// Wire the discoverer stop channels to the builder
	b.GetGVKStopChan = d.GetStopChanForGVK

	// Instantiate dummyFactory and wire sync channels
	watchRun := make(chan struct{})
	watchStop := make(chan struct{})
	df := &dummyFactory{
		gvk:       gvkp.GroupVersionKind,
		watchRun:  watchRun,
		watchStop: watchStop,
	}

	// Register factory and enable custom resource
	b.WithCustomResourceStoreFactories(df)
	b.WithCustomResourceClients(map[string]interface{}{"example.com/v1, Resource=foos": nil})
	if err := b.WithEnabledResources([]string{"example.com/v1, Resource=foos"}); err != nil {
		t.Fatal(err)
	}
	b.WithGenerateCustomResourceStoresFunc(b.DefaultGenerateCustomResourceStoresFunc())

	// Build the stores to start the custom resource reflector
	b.BuildStores()

	// Wait until the reflector has actually started Watch
	select {
	case <-watchRun:
	case <-time.After(2 * time.Second):
		t.Fatal("reflector watch did not start in time")
	}

	// Capture the initial stop channel
	stopCh := d.GetStopChanForGVK(gvkp.GroupVersionKind.String())
	if stopCh == nil {
		t.Fatal("expected stop channel to exist for GVK")
	}

	// 2. Simulate repeated discovery event (the bug path)
	d.AppendToMap(gvkp)

	// Assert that the stop channel did not get replaced/overwritten on repeated discovery
	gotCh := d.GetStopChanForGVK(gvkp.GroupVersionKind.String())
	if gotCh != stopCh {
		t.Fatal("stop channel was replaced on repeated AppendToMap calls")
	}

	// 3. Exercise deletion-driven stop-channel cleanup
	d.RemoveFromMap(gvkp)

	// Verify that the stop channel is closed
	select {
	case <-stopCh:
		// Passed, channel is closed
	case <-time.After(2 * time.Second):
		t.Fatal("stop channel was not closed on RemoveFromMap")
	}

	// Verify that the channel is removed from the discoverer map
	if gotAfterRemove := d.GetStopChanForGVK(gvkp.GroupVersionKind.String()); gotAfterRemove != nil {
		t.Fatal("expected stop channel to be nil in discoverer after RemoveFromMap")
	}

	// 4. Bounded wait for reflector teardown
	select {
	case <-watchStop:
		// Passed, watch was stopped
	case <-time.After(2 * time.Second):
		t.Fatal("reflector watch did not stop in time")
	}
}
