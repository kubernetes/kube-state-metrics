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
