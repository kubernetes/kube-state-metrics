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
	"context"
	// "fmt"
	// "io/ioutil"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"k8s.io/kube-state-metrics/pkg/options"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	kcollectors "k8s.io/kube-state-metrics/pkg/collectors"
)

func BenchmarkKubeStateMetrics(t *testing.B) {
	fixtureMultiplier := 1000
	requestCount := 100

	t.Logf(
		"starting kube-state-metrics benchmark with fixtureMultiplier %v and requestCount %v",
		fixtureMultiplier,
		requestCount,
	)

	kubeClient := fake.NewSimpleClientset()

	if err := injectFixtures(kubeClient, fixtureMultiplier); err != nil {
		t.Errorf("error injecting resources: %v", err)
	}

	opts := options.NewOptions()

	builder := kcollectors.NewBuilder(context.TODO(), opts)
	builder.WithEnabledCollectors(options.DefaultCollectors)
	builder.WithKubeClient(kubeClient)
	builder.WithNamespaces(options.DefaultNamespaces)

	collectors := builder.Build()

	handler := metricHandler{collectors, false}

	req := httptest.NewRequest("GET", "http://localhost:8080/metrics", nil)

	// Wait for informers to sync
	time.Sleep(time.Second)

	var w *httptest.ResponseRecorder
	for i := 0; i < requestCount; i++ {
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// resp := w.Result()
	// body, _ := ioutil.ReadAll(resp.Body)

	// fmt.Println(resp.StatusCode)
	// fmt.Println(resp.Header.Get("Content-Type"))
	// fmt.Println(string(body))
}

func injectFixtures(client *fake.Clientset, multiplier int) error {
	creators := []func(*fake.Clientset, int) error{
		configMap,
		service,
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

func service(client *fake.Clientset, index int) error {
	i := strconv.Itoa(index)

	service := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "service" + i,
			ResourceVersion: "123456",
		},
	}
	_, err := client.CoreV1().Services(metav1.NamespaceDefault).Create(&service)
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
