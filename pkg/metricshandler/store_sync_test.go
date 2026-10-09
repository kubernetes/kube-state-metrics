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

package metricshandler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"k8s.io/kube-state-metrics/v2/internal/store"
	"k8s.io/kube-state-metrics/v2/pkg/allowdenylist"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	"k8s.io/kube-state-metrics/v2/pkg/optin"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func TestRealBuilder_ForbiddenSecretInstallsPodMetrics(t *testing.T) {
	pod := samplePod("original")
	kubeClient := fake.NewSimpleClientset(pod)
	kubeClient.PrependReactor("list", "secrets", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "secrets"}, "", fmt.Errorf("forbidden"))
	})

	opts := options.NewOptions()
	opts.StoreSyncTimeout = 500 * time.Millisecond
	handler := newRealHandler(t, kubeClient, []string{"pods", "secrets"}, opts)

	handler.BuildWriters(context.Background())
	waitUntil(t, 5*time.Second, handler.Ready, "handler did not become ready after the sync timeout")

	body := scrape(t, handler)
	if !strings.Contains(body, "kube_pod_info") || !strings.Contains(body, "original") {
		t.Fatalf("expected pod metrics after the secrets list was forbidden, got:\n%s", body)
	}
}

func TestRealBuilder_SlowReplacementKeepsServingUntilList(t *testing.T) {
	kubeClient := fake.NewSimpleClientset(samplePod("original"))
	var block atomic.Bool
	var once sync.Once
	entered := make(chan struct{})
	release := make(chan struct{})
	kubeClient.PrependReactor("list", "pods", func(clienttesting.Action) (bool, runtime.Object, error) {
		if !block.Load() {
			return false, nil, nil
		}
		once.Do(func() { close(entered) })
		<-release
		return false, nil, nil
	})

	opts := options.NewOptions()
	opts.StoreSyncTimeout = 30 * time.Second
	handler := newRealHandler(t, kubeClient, []string{"pods"}, opts)
	handler.BuildWriters(context.Background())
	waitUntil(t, 5*time.Second, handler.Ready, "initial generation did not become ready")

	installed := writerIdentity(handler)
	body := scrape(t, handler)
	if !strings.Contains(body, "original") {
		t.Fatalf("expected the installed generation to serve the original pod, got:\n%s", body)
	}

	block.Store(true)
	handler.BuildWriters(context.Background())
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("replacement list did not block")
	}

	if got := writerIdentity(handler); got != installed {
		t.Fatalf("replacement swapped before its list finished: was %s now %s", installed, got)
	}
	body = scrape(t, handler)
	if !strings.Contains(body, "original") {
		t.Fatalf("expected the installed generation to keep serving during the blocked list, got:\n%s", body)
	}

	close(release)
	waitUntil(t, 5*time.Second, func() bool {
		return writerIdentity(handler) != installed
	}, "replacement was not installed after its list finished")
}

func newRealHandler(t *testing.T, kubeClient *fake.Clientset, resources []string, opts *options.Options) *MetricsHandler {
	t.Helper()
	builder := store.NewBuilder()
	builder.WithMetrics(prometheus.NewRegistry())
	if err := builder.WithEnabledResources(resources); err != nil {
		t.Fatal(err)
	}
	builder.WithKubeClient(kubeClient)
	builder.WithNamespaces(options.DefaultNamespaces)
	builder.WithSharding(0, 1)
	builder.WithGenerateStoresFunc(builder.DefaultGenerateStoresFunc())

	allowDeny, err := allowdenylist.New(map[string]struct{}{}, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	optIn, err := optin.NewMetricFamilyFilter(map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	builder.WithFamilyGeneratorFilter(generator.NewCompositeFamilyGeneratorFilter(allowDeny, optIn))
	if err := builder.WithAllowLabels(map[string][]string{}); err != nil {
		t.Fatal(err)
	}

	return New(opts, kubeClient, builder, false)
}

func samplePod(name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       types.UID("uid-" + name),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "c", Image: "img"}},
		},
	}
}

func writerIdentity(h *MetricsHandler) string {
	h.mtx.RLock()
	defer h.mtx.RUnlock()
	if len(h.metricsWriters) == 0 {
		return ""
	}
	return fmt.Sprintf("%p", h.metricsWriters[0])
}

func scrape(t *testing.T, h *MetricsHandler) string {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from an installed generation, got %d", rec.Code)
	}
	return rec.Body.String()
}

func waitUntil(t *testing.T, timeout time.Duration, done func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !done() {
		if time.Now().After(deadline) {
			t.Fatal(message)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
