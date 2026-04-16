package discovery_test

import (
	"reflect"
	"testing"

	"k8s.io/kube-state-metrics/v2/pkg/discovery"
)

func TestPublicModuleAPI(t *testing.T) {
	// This test ensures that the public API of the library is correct and can be used by external code.
	// It does not test any functionality, but rather that the code compiles and can be used as a library.

	// This is a compile-time check that the NewDiscoverer function can be called with the correct arguments.
	var discoverer discovery.Discoverer = discovery.NewDiscoverer(nil, nil, nil, nil)
	if discoverer == nil {
		t.Fatal("expected discoverer to be non-nil")
	}

	discoverer_type := reflect.TypeOf(discoverer)
	if discoverer_type.Kind() != reflect.Ptr {
		t.Fatal("expected discoverer to be a pointer")
	}

	if discoverer_type.Elem().Name() != "CRDiscoverer" {
		t.Fatalf("expected discoverer to be of type CRDiscoverer, got %s", discoverer_type.Elem().Name())
	}
}
