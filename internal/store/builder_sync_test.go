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

package store

import (
	"context"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

func TestWaitForStoresSync_NoReflectors(t *testing.T) {
	b := NewBuilder()
	if !b.WaitForStoresSync(context.Background(), time.Millisecond) {
		t.Fatal("expected sync success with no reflectors")
	}
}

func TestWaitForStoresSync_Timeout(t *testing.T) {
	b := NewBuilder()
	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	lw := &cache.ListWatch{
		ListFunc: func(_ metav1.ListOptions) (runtime.Object, error) {
			return &v1.PodList{}, nil
		},
	}
	reflector := cache.NewReflectorWithOptions(lw, &v1.Pod{}, store, cache.ReflectorOptions{})
	b.reflectorsMu.Lock()
	b.reflectors = []*cache.Reflector{reflector}
	b.reflectorsMu.Unlock()

	if b.WaitForStoresSync(context.Background(), 50*time.Millisecond) {
		t.Fatal("expected sync to time out before reflector runs")
	}
}
