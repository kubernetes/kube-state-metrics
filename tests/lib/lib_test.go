package lib

import (
	"context"
	"strings"
	"testing"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/collectors"
	"k8s.io/kube-state-metrics/pkg/metrics"
	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"
)

func TestAsLibrary(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()

	service := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "my-service",
			ResourceVersion: "123456",
		},
	}

	_, err := kubeClient.CoreV1().Services(metav1.NamespaceDefault).Create(&service)
	if err != nil {
		t.Fatal(err)
	}

	c := serviceCollector(kubeClient)

	// Wait for informers to sync
	time.Sleep(time.Second)

	w := strings.Builder{}
	c.Collect(&w)
	m := w.String()

	if !strings.Contains(m, service.ObjectMeta.Name) {
		t.Fatal("expected string to contain service name")
	}
}

func serviceCollector(kubeClient clientset.Interface) *collectors.Collector {
	store := metricsstore.NewMetricsStore([]string{"test_metric describes a test metric"}, generateServiceMetrics)

	lw := cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Services(metav1.NamespaceDefault).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Services(metav1.NamespaceDefault).Watch(opts)
		},
	}

	r := cache.NewReflector(&lw, &v1.Service{}, store, 0)

	go r.Run(context.TODO().Done())

	return collectors.NewCollector(store)
}

func generateServiceMetrics(obj interface{}) [][]*metrics.Metric {
	sPointer := obj.(*v1.Service)
	s := *sPointer

	m, err := metrics.NewMetric("test_metric", []string{"name"}, []string{s.Name}, 1)
	if err != nil {
		panic(err)
	}

	ms := []*metrics.Metric{m}

	return [][]*metrics.Metric{ms}
}
