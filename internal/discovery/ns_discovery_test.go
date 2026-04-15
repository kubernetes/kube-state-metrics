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

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_NamespaceDiscoverer_Start_Simple(t *testing.T) {
	client := fake.NewClientset()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	discoverer := NewNamespaceDiscoverer()
	discoverer.Start(ctx, client)

	discoverer.safeRead(func() {
		assert.Empty(t, discoverer.namespaces)
		assert.False(t, discoverer.shouldRebuildMetrics)
	})

	client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
	}, metav1.CreateOptions{})

	assert.Eventually(t, func() bool {
		found := false
		discoverer.safeRead(func() {
			_, found = discoverer.namespaces["default"]
		})
		return found
	}, time.Second, 10*time.Millisecond, "namespace should eventually be added")

	discoverer.safeRead(func() {
		assert.True(t, discoverer.shouldRebuildMetrics)
	})

	client.CoreV1().Namespaces().Delete(ctx, "default", metav1.DeleteOptions{})

	assert.Eventually(t, func() bool {
		empty := true
		discoverer.safeRead(func() {
			empty = len(discoverer.namespaces) == 0
		})
		return empty
	}, time.Second, 10*time.Millisecond, "namespace should eventually be removed")

	discoverer.safeRead(func() {
		assert.True(t, discoverer.shouldRebuildMetrics)
	})
}

func Test_NamespaceDiscoverer_Start_Concurrent(t *testing.T) {
	client := fake.NewClientset()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	discoverer := NewNamespaceDiscoverer()
	discoverer.Start(ctx, client)

	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			}, metav1.CreateOptions{})

			client.CoreV1().Namespaces().Delete(ctx, "default", metav1.DeleteOptions{})
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			client.CoreV1().Namespaces().Delete(ctx, "default", metav1.DeleteOptions{})
		}()
	}

	wg.Wait()

	assert.Eventually(t, func() bool {
		empty := true
		discoverer.safeRead(func() {
			empty = len(discoverer.namespaces) == 0
		})
		return empty
	}, time.Second, 10*time.Millisecond, "namespace should eventually be empty")
}

func Test_NamespaceDiscoverer_Start_LabelSelector(t *testing.T) {
	client := fake.NewClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
				Labels: map[string]string{
					"foo": "bar",
				},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "other",
				Labels: map[string]string{
					"unknown": "label",
				},
			},
		},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	discoverer := NewNamespaceDiscoverer(
		WithLabelSelector("foo=bar"),
	)

	discoverer.Start(ctx, client)

	assert.Eventually(t, func() bool {
		found := false
		discoverer.safeRead(func() {
			_, found = discoverer.namespaces["default"]
		})
		return found
	}, time.Second, 10*time.Millisecond, "namespace should eventually be added")

	// Should now contain only the default namespace labeled with foo=bar
	assert.Equal(t, map[string]struct{}{
		"default": struct{}{},
	}, discoverer.namespaces)

	// TODO: fake client does not seem to support label selectors during watch, only list,
	// this is why we not do explicitly test this scenario here
}

func Test_NamespaceDiscoverer_Start_FieldSelector(t *testing.T) {
	// TODO: there is currently no support for field selectors in the fake client
	// https://github.com/kubernetes-sigs/controller-runtime/issues/1376
}

func Test_NamespaceDiscoverer_PollForCacheUpdates(t *testing.T) {
	discoverer := NewNamespaceDiscoverer()

	// Prepare discoverer for rebuilding metrics
	discoverer.namespaces = map[string]struct{}{"default": struct{}{}}
	discoverer.shouldRebuildMetrics = true

	ctx, cancel := context.WithCancel(context.Background())

	updateChan := discoverer.PollForCacheUpdates(ctx, 1*time.Second)

	namespaces := <-updateChan

	assert.Equal(t, []string{"default"}, namespaces)

	cancel()

	select {
	case _, ok := <-updateChan:
		assert.False(t, ok)
	case <-time.After(3 * time.Second):
		assert.Fail(t, "notification channel did not close in time after context was cancelled")
	}
}
