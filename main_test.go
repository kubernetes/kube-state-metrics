/*
Copyright 2015 The Kubernetes Authors All rights reserved.

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

package main

import (
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"k8s.io/kube-state-metrics/pkg/options"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func BenchmarkKubeStateMetrics(t *testing.B) {
	kubeClient := fake.NewSimpleClientset()

	if err := injectFixtures(kubeClient, 1000); err != nil {
		t.Errorf("error injecting resources: %v", err)
	}

	opts := options.NewOptions()
	collectors := options.DefaultCollectors
	namespaces := options.DefaultNamespaces

	registry := prometheus.NewRegistry()
	registerCollectors(registry, kubeClient, collectors, namespaces, opts)
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorLog: promLogger{}})

	req := httptest.NewRequest("GET", "http://localhost:8080/metrics", nil)

	// Wait for informers to sync
	time.Sleep(time.Second)

	amountRequests := 10
	for i := 0; i < amountRequests; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func injectFixtures(client *fake.Clientset, multiplier int) error {
	creators := []func(*fake.Clientset, int) error{
		configMap,
		pod,
	}

	for _, c := range creators {
		for i := 0; i < multiplier; i++ {
			err := c(client, i)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func configMap(client *fake.Clientset, index int) error {
	i := strconv.Itoa(index)

	configMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "configmap" + i,
			ResourceVersion: "123456",
		},
	}
	_, err := client.CoreV1().ConfigMaps(metav1.NamespaceDefault).Create(&configMap)
	return err
}

func pod(client *fake.Clientset, index int) error {
	i := strconv.Itoa(index)

	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod" + i,
		},
		Status: v1.PodStatus{
			ContainerStatuses: []v1.ContainerStatus{
				v1.ContainerStatus{
					Name:        "container1",
					Image:       "k8s.gcr.io/hyperkube1",
					ImageID:     "docker://sha256:aaa",
					ContainerID: "docker://ab123",
				},
			},
		},
	}

	_, err := client.CoreV1().Pods(metav1.NamespaceDefault).Create(&pod)
	return err
}
