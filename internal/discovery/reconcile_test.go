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
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	ksmtypes "k8s.io/kube-state-metrics/v2/pkg/builder/types"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	applicationV1 = groupVersionKindPlural{Group: "example.com", Version: "v1", Kind: "Application", Plural: "applications"}
	widgetV1      = groupVersionKindPlural{Group: "unrelated.example.org", Version: "v1", Kind: "Widget", Plural: "widgets"}
)

// fakeFactory is a customresource.RegistryFactory for a resolved GVKP.
type fakeFactory struct {
	gvkp groupVersionKindPlural
}

func (f fakeFactory) Name() string { return f.gvkp.Plural }

func (f fakeFactory) CreateClient(*rest.Config) (any, error) { return nil, nil }

func (f fakeFactory) MetricFamilyGenerators() []generator.FamilyGenerator { return nil }

func (f fakeFactory) ExpectedType() any {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(f.gvkp.GroupVersionKind)
	return u
}

func (f fakeFactory) ListWatch(any, string, string) cache.ListerWatcher { return nil }

// fakeStore counts rebuilds instead of building stores. onConfigure, if set,
// runs where the metrics handler would build the stores.
type fakeStore struct {
	rebuilds    int
	onConfigure func()
}

func (s *fakeStore) ConfigureStore(context.Context, func(ksmtypes.BuilderInterface) error) error {
	s.rebuilds++
	if s.onConfigure != nil {
		s.onConfigure()
	}
	return nil
}

// newTestReconciler returns a reconciler that resolves the configured GVKs
// against the discoverer's cache, as the custom resource state config does.
func newTestReconciler(r *CRDiscoverer, store *fakeStore, configured ...schema.GroupVersionKind) *cacheReconciler {
	return &cacheReconciler{
		discoverer: r,
		store:      store,
		factoryGenerator: func() ([]customresource.RegistryFactory, error) {
			var factories []customresource.RegistryFactory
			for _, gvk := range configured {
				resolved, err := r.ResolveGVKToGVKPs(gvk)
				if err != nil {
					return nil, err
				}
				for _, gvkp := range resolved {
					factories = append(factories, fakeFactory{gvkp: gvkp})
				}
			}
			return factories, nil
		},
		createClients: func(...customresource.RegistryFactory) (map[string]any, error) {
			return map[string]any{}, nil
		},
	}
}

// addCRD and deleteCRD change the cache as the CRD informer's event handlers
// do.
func addCRD(r *CRDiscoverer, gvkps ...groupVersionKindPlural) {
	r.SafeWrite(func() {
		if r.AppendToMap(gvkps...) {
			r.WasUpdated = true
		}
	})
}

func deleteCRD(r *CRDiscoverer, gvkps ...groupVersionKindPlural) {
	r.SafeWrite(func() {
		if r.RemoveFromMap(gvkps...) {
			r.WasUpdated = true
		}
	})
}

func wasUpdated(r *CRDiscoverer) bool {
	var updated bool
	r.SafeRead(func() {
		updated = r.WasUpdated
	})
	return updated
}

func TestReconcileSkipsUnrelatedCRDs(t *testing.T) {
	r := &CRDiscoverer{}
	store := &fakeStore{}
	c := newTestReconciler(r, store, applicationV1.GroupVersionKind)
	addCRD(r, widgetV1)
	c.reconcile(t.Context())
	if store.rebuilds != 0 {
		t.Fatalf("got %d rebuilds before the configured CRD exists, want 0", store.rebuilds)
	}
	if wasUpdated(r) {
		t.Fatal("WasUpdated is still set after skipping the rebuild")
	}

	addCRD(r, applicationV1)
	c.reconcile(t.Context())
	if store.rebuilds != 1 {
		t.Fatalf("got %d rebuilds after the configured CRD was added, want 1", store.rebuilds)
	}

	for _, change := range []struct {
		name  string
		apply func()
	}{
		{name: "delete", apply: func() { deleteCRD(r, widgetV1) }},
		{name: "add", apply: func() { addCRD(r, widgetV1) }},
	} {
		change.apply()
		if !wasUpdated(r) {
			t.Fatalf("%s: expected the unrelated CRD to update the cache", change.name)
		}
		c.reconcile(t.Context())
		if store.rebuilds != 1 {
			t.Fatalf("%s: got %d rebuilds after an unrelated CRD changed, want 1", change.name, store.rebuilds)
		}
		if wasUpdated(r) {
			t.Fatalf("%s: WasUpdated is still set after skipping the rebuild", change.name)
		}
	}
}

// TestReconcileRebuildsWhenConfiguredCRDIsRecreated covers a configured CRD
// that is deleted and re-added between ticks: it resolves to the same
// resources, but its reflectors were stopped with the old stop channel.
func TestReconcileRebuildsWhenConfiguredCRDIsRecreated(t *testing.T) {
	r := &CRDiscoverer{}
	store := &fakeStore{}
	c := newTestReconciler(r, store, applicationV1.GroupVersionKind)
	addCRD(r, applicationV1)
	c.reconcile(t.Context())

	deleteCRD(r, applicationV1)
	addCRD(r, applicationV1)
	c.reconcile(t.Context())
	if store.rebuilds != 2 {
		t.Fatalf("got %d rebuilds, want 2", store.rebuilds)
	}
}

// TestReconcileRebuildsWhenStopChannelIsReplacedDuringRebuild covers a CRD
// that is deleted and re-added while the stores are being built: the
// reflectors hold the closed stop channel, so the next tick must rebuild even
// though the cache looks the same as after the rebuild.
func TestReconcileRebuildsWhenStopChannelIsReplacedDuringRebuild(t *testing.T) {
	r := &CRDiscoverer{}
	store := &fakeStore{}
	c := newTestReconciler(r, store, applicationV1.GroupVersionKind)
	addCRD(r, applicationV1)
	store.onConfigure = func() {
		store.onConfigure = nil
		deleteCRD(r, applicationV1)
		addCRD(r, applicationV1)
	}
	c.reconcile(t.Context())
	if !wasUpdated(r) {
		t.Fatal("WasUpdated was cleared although the cache changed during the rebuild")
	}

	c.reconcile(t.Context())
	if store.rebuilds != 2 {
		t.Fatalf("got %d rebuilds, want 2", store.rebuilds)
	}
}
