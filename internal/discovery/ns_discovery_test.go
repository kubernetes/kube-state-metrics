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

	discoverer := NewNamespaceDiscoverer()
	discoverer.Start(context.TODO(), client)

	discoverer.safeRead(func() {
		// There should be no namespaces at start time
		assert.Empty(t, discoverer.namespaces)

		// There should be no need to rebuild metrics at this time
		assert.False(t, discoverer.shouldRebuildMetrics)
	})

	client.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
	}, metav1.CreateOptions{})

	time.Sleep(10 * time.Millisecond)

	discoverer.safeRead(func() {
		// Should now contain the namespace added above.
		assert.Equal(t, map[string]struct{}{
			"default": struct{}{},
		}, discoverer.namespaces)

		// Should warrant the rebuilding of metrics to add the namespace from the store
		assert.True(t, discoverer.shouldRebuildMetrics)
	})

	client.CoreV1().Namespaces().Delete(context.TODO(), "default", metav1.DeleteOptions{})

	time.Sleep(10 * time.Millisecond)

	discoverer.safeRead(func() {
		// Should not contain the namespace deleted above.
		assert.Empty(t, discoverer.namespaces)

		// Should warrant the rebuilding of metrics to remove the namespace from the store
		assert.True(t, discoverer.shouldRebuildMetrics)
	})
}

func Test_NamespaceDiscoverer_Start_Concurrent(t *testing.T) {
	client := fake.NewClientset()

	discoverer := NewNamespaceDiscoverer()
	discoverer.Start(context.TODO(), client)

	var wg sync.WaitGroup

	for i := 0; i <= 10000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			client.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			}, metav1.CreateOptions{})

			client.CoreV1().Namespaces().Delete(context.TODO(), "default", metav1.DeleteOptions{})
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			client.CoreV1().Namespaces().Delete(context.TODO(), "default", metav1.DeleteOptions{})
		}()
	}

	wg.Wait()

	time.Sleep(10 * time.Millisecond)

	// Should not contain the namespace deleted above.
	assert.Empty(t, discoverer.namespaces)
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

	discoverer := NewNamespaceDiscoverer(
		WithLabelSelector("foo=bar"),
	)

	discoverer.Start(context.TODO(), client)

	time.Sleep(10 * time.Millisecond)

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

	ctx, cancel := context.WithCancel(context.TODO())

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
