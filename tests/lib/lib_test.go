/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package lib

import (
	"context"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
)

func TestAsLibrary(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()

	service := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "my-service",
			ResourceVersion: "123456",
		},
	}

	_, err := kubeClient.CoreV1().Services(metav1.NamespaceDefault).Create(context.TODO(), &service, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	c := serviceCollector(kubeClient)

	// Wait for informers to sync
	time.Sleep(time.Second)

	w := strings.Builder{}
	mw := metricsstore.NewMetricsWriter("test", c)
	err = mw.WriteAll(&w)
	if err != nil {
		t.Fatalf("failed to write metrics: %v", err)
	}
	m := w.String()

	if !strings.Contains(m, service.Name) {
		t.Fatal("expected string to contain service name")
	}
}

func serviceCollector(kubeClient clientset.Interface) *metricsstore.MetricsStore {
	store := metricsstore.NewMetricsStore([]string{"test_metric describes a test metric"}, generateServiceMetrics)

	lw := cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Services(metav1.NamespaceDefault).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Services(metav1.NamespaceDefault).Watch(context.TODO(), opts)
		},
	}

	r := cache.NewReflectorWithOptions(&lw, &v1.Service{}, store, cache.ReflectorOptions{ResyncPeriod: 0})

	go r.Run(context.TODO().Done())

	return store
}

func generateServiceMetrics(obj interface{}) []metric.FamilyInterface {
	sPointer := obj.(*v1.Service)
	s := *sPointer

	m := metric.Metric{
		LabelKeys:   []string{"name"},
		LabelValues: []string{s.Name},
		Value:       1,
	}

	family := metric.Family{
		Name:    "test_metric",
		Metrics: []*metric.Metric{&m},
	}

	return []metric.FamilyInterface{&family}
}
