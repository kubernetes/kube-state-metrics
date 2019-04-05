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
	"bytes"
	"context"
	"io/ioutil"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	kcoll "k8s.io/kube-state-metrics/internal/collector"
	coll "k8s.io/kube-state-metrics/pkg/collector"
	"k8s.io/kube-state-metrics/pkg/options"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/kube-state-metrics/pkg/whiteblacklist"
)

func BenchmarkKubeStateMetrics(b *testing.B) {
	var collectors []*coll.Collector
	fixtureMultiplier := 1000
	requestCount := 1000

	b.Logf(
		"starting kube-state-metrics benchmark with fixtureMultiplier %v and requestCount %v",
		fixtureMultiplier,
		requestCount,
	)

	kubeClient := fake.NewSimpleClientset()

	if err := injectFixtures(kubeClient, fixtureMultiplier); err != nil {
		b.Errorf("error injecting resources: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := kcoll.NewBuilder(ctx)
	builder.WithEnabledCollectors(options.DefaultCollectors.AsSlice())
	builder.WithKubeClient(kubeClient)
	builder.WithNamespaces(options.DefaultNamespaces)

	l, err := whiteblacklist.New(map[string]struct{}{}, map[string]struct{}{})
	if err != nil {
		b.Fatal(err)
	}
	builder.WithWhiteBlackList(l)

	// This test is not suitable to be compared in terms of time, as it includes
	// a one second wait. Use for memory allocation comparisons, profiling, ...
	b.Run("GenerateMetrics", func(b *testing.B) {
		collectors = builder.Build()

		// Wait for caches to fill
		time.Sleep(time.Second)
	})

	handler := metricHandler{collectors, false}
	req := httptest.NewRequest("GET", "http://localhost:8080/metrics", nil)

	b.Run("MakeRequests", func(b *testing.B) {
		var accumulatedContentLength int

		for i := 0; i < requestCount; i++ {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != 200 {
				b.Fatalf("expected 200 status code but got %v", resp.StatusCode)
			}

			b.StopTimer()
			buf := bytes.Buffer{}
			buf.ReadFrom(resp.Body)
			accumulatedContentLength += buf.Len()
			b.StartTimer()
		}

		b.SetBytes(int64(accumulatedContentLength))
	})
}

// TestFullScrapeCycle is a simple smoke test covering the entire cycle from
// cache filling to scraping.
func TestFullScrapeCycle(t *testing.T) {
	t.Parallel()

	kubeClient := fake.NewSimpleClientset()

	err := pod(kubeClient, 0)
	if err != nil {
		t.Fatalf("failed to insert sample pod %v", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := kcoll.NewBuilder(ctx)
	builder.WithEnabledCollectors(options.DefaultCollectors.AsSlice())
	builder.WithKubeClient(kubeClient)
	builder.WithNamespaces(options.DefaultNamespaces)

	l, err := whiteblacklist.New(map[string]struct{}{}, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	builder.WithWhiteBlackList(l)

	collectors := builder.Build()

	// Wait for caches to fill
	time.Sleep(time.Second)

	handler := metricHandler{collectors, false}
	req := httptest.NewRequest("GET", "http://localhost:8080/metrics", nil)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 status code but got %v", resp.StatusCode)
	}

	body, _ := ioutil.ReadAll(resp.Body)

	expected := `# HELP kube_pod_info Information about pod.
# TYPE kube_pod_info gauge
kube_pod_info{namespace="default",pod="pod0",host_ip="1.1.1.1",pod_ip="1.2.3.4",uid="abc-123-xxx",node="node1",created_by_kind="<none>",created_by_name="<none>",priority_class=""} 1
# HELP kube_pod_start_time Start time in unix timestamp for a pod.
# TYPE kube_pod_start_time gauge
# HELP kube_pod_completion_time Completion time in unix timestamp for a pod.
# TYPE kube_pod_completion_time gauge
# HELP kube_pod_owner Information about the Pod's owner.
# TYPE kube_pod_owner gauge
kube_pod_owner{namespace="default",pod="pod0",owner_kind="<none>",owner_name="<none>",owner_is_controller="<none>"} 1
# HELP kube_pod_labels Kubernetes labels converted to Prometheus labels.
# TYPE kube_pod_labels gauge
kube_pod_labels{namespace="default",pod="pod0"} 1
# HELP kube_pod_created Unix creation timestamp
# TYPE kube_pod_created gauge
kube_pod_created{namespace="default",pod="pod0"} 1.5e+09
# HELP kube_pod_status_scheduled_time Unix timestamp when pod moved into scheduled status
# TYPE kube_pod_status_scheduled_time gauge
# HELP kube_pod_status_phase The pods current phase.
# TYPE kube_pod_status_phase gauge
kube_pod_status_phase{namespace="default",pod="pod0",phase="Pending"} 0
kube_pod_status_phase{namespace="default",pod="pod0",phase="Succeeded"} 0
kube_pod_status_phase{namespace="default",pod="pod0",phase="Failed"} 0
kube_pod_status_phase{namespace="default",pod="pod0",phase="Running"} 1
kube_pod_status_phase{namespace="default",pod="pod0",phase="Unknown"} 0
# HELP kube_pod_status_ready Describes whether the pod is ready to serve requests.
# TYPE kube_pod_status_ready gauge
# HELP kube_pod_status_scheduled Describes the status of the scheduling process for the pod.
# TYPE kube_pod_status_scheduled gauge
# HELP kube_pod_container_info Information about a container in a pod.
# TYPE kube_pod_container_info gauge
kube_pod_container_info{namespace="default",pod="pod0",container="container2",image="k8s.gcr.io/hyperkube2",image_id="docker://sha256:bbb",container_id="docker://cd456"} 1
kube_pod_container_info{namespace="default",pod="pod0",container="container3",image="k8s.gcr.io/hyperkube3",image_id="docker://sha256:ccc",container_id="docker://ef789"} 1
# HELP kube_pod_container_status_waiting Describes whether the container is currently in waiting state.
# TYPE kube_pod_container_status_waiting gauge
kube_pod_container_status_waiting{namespace="default",pod="pod0",container="container2"} 1
kube_pod_container_status_waiting{namespace="default",pod="pod0",container="container3"} 0
# HELP kube_pod_container_status_waiting_reason Describes the reason the container is currently in waiting state.
# TYPE kube_pod_container_status_waiting_reason gauge
kube_pod_container_status_waiting_reason{namespace="default",pod="pod0",container="container2",reason="ContainerCreating"} 0
kube_pod_container_status_waiting_reason{namespace="default",pod="pod0",container="container2",reason="CrashLoopBackOff"} 1
kube_pod_container_status_waiting_reason{namespace="default",pod="pod0",container="container2",reason="CreateContainerConfigError"} 0
kube_pod_container_status_waiting_reason{namespace="default",pod="pod0",container="container2",reason="ErrImagePull"} 0
kube_pod_container_status_waiting_reason{namespace="default",pod="pod0",container="container2",reason="ImagePullBackOff"} 0
kube_pod_container_status_waiting_reason{namespace="default",pod="pod0",container="container3",reason="ContainerCreating"} 0
kube_pod_container_status_waiting_reason{namespace="default",pod="pod0",container="container3",reason="CrashLoopBackOff"} 0
kube_pod_container_status_waiting_reason{namespace="default",pod="pod0",container="container3",reason="CreateContainerConfigError"} 0
kube_pod_container_status_waiting_reason{namespace="default",pod="pod0",container="container3",reason="ErrImagePull"} 0
kube_pod_container_status_waiting_reason{namespace="default",pod="pod0",container="container3",reason="ImagePullBackOff"} 0
# HELP kube_pod_container_status_running Describes whether the container is currently in running state.
# TYPE kube_pod_container_status_running gauge
kube_pod_container_status_running{namespace="default",pod="pod0",container="container2"} 0
kube_pod_container_status_running{namespace="default",pod="pod0",container="container3"} 0
# HELP kube_pod_container_status_terminated Describes whether the container is currently in terminated state.
# TYPE kube_pod_container_status_terminated gauge
kube_pod_container_status_terminated{namespace="default",pod="pod0",container="container2"} 0
kube_pod_container_status_terminated{namespace="default",pod="pod0",container="container3"} 0
# HELP kube_pod_container_status_terminated_reason Describes the reason the container is currently in terminated state.
# TYPE kube_pod_container_status_terminated_reason gauge
kube_pod_container_status_terminated_reason{namespace="default",pod="pod0",container="container2",reason="OOMKilled"} 0
kube_pod_container_status_terminated_reason{namespace="default",pod="pod0",container="container2",reason="Completed"} 0
kube_pod_container_status_terminated_reason{namespace="default",pod="pod0",container="container2",reason="Error"} 0
kube_pod_container_status_terminated_reason{namespace="default",pod="pod0",container="container2",reason="ContainerCannotRun"} 0
kube_pod_container_status_terminated_reason{namespace="default",pod="pod0",container="container3",reason="OOMKilled"} 0
kube_pod_container_status_terminated_reason{namespace="default",pod="pod0",container="container3",reason="Completed"} 0
kube_pod_container_status_terminated_reason{namespace="default",pod="pod0",container="container3",reason="Error"} 0
kube_pod_container_status_terminated_reason{namespace="default",pod="pod0",container="container3",reason="ContainerCannotRun"} 0
# HELP kube_pod_container_status_last_terminated_reason Describes the last reason the container was in terminated state.
# TYPE kube_pod_container_status_last_terminated_reason gauge
kube_pod_container_status_last_terminated_reason{namespace="default",pod="pod0",container="container2",reason="OOMKilled"} 1
kube_pod_container_status_last_terminated_reason{namespace="default",pod="pod0",container="container2",reason="Completed"} 0
kube_pod_container_status_last_terminated_reason{namespace="default",pod="pod0",container="container2",reason="Error"} 0
kube_pod_container_status_last_terminated_reason{namespace="default",pod="pod0",container="container2",reason="ContainerCannotRun"} 0
kube_pod_container_status_last_terminated_reason{namespace="default",pod="pod0",container="container3",reason="OOMKilled"} 0
kube_pod_container_status_last_terminated_reason{namespace="default",pod="pod0",container="container3",reason="Completed"} 0
kube_pod_container_status_last_terminated_reason{namespace="default",pod="pod0",container="container3",reason="Error"} 0
kube_pod_container_status_last_terminated_reason{namespace="default",pod="pod0",container="container3",reason="ContainerCannotRun"} 0
# HELP kube_pod_container_status_ready Describes whether the containers readiness check succeeded.
# TYPE kube_pod_container_status_ready gauge
kube_pod_container_status_ready{namespace="default",pod="pod0",container="container2"} 0
kube_pod_container_status_ready{namespace="default",pod="pod0",container="container3"} 0
# HELP kube_pod_container_status_restarts_total The number of container restarts per container.
# TYPE kube_pod_container_status_restarts_total counter
kube_pod_container_status_restarts_total{namespace="default",pod="pod0",container="container2"} 0
kube_pod_container_status_restarts_total{namespace="default",pod="pod0",container="container3"} 0
# HELP kube_pod_container_resource_requests The number of requested request resource by a container.
# TYPE kube_pod_container_resource_requests gauge
kube_pod_container_resource_requests{namespace="default",pod="pod0",container="pod1_con1",node="node1",resource="nvidia_com_gpu",unit="integer"} 1
kube_pod_container_resource_requests{namespace="default",pod="pod0",container="pod1_con1",node="node1",resource="cpu",unit="core"} 0.2
kube_pod_container_resource_requests{namespace="default",pod="pod0",container="pod1_con1",node="node1",resource="memory",unit="byte"} 1e+08
kube_pod_container_resource_requests{namespace="default",pod="pod0",container="pod1_con1",node="node1",resource="ephemeral_storage",unit="byte"} 3e+08
kube_pod_container_resource_requests{namespace="default",pod="pod0",container="pod1_con1",node="node1",resource="storage",unit="byte"} 4e+08
kube_pod_container_resource_requests{namespace="default",pod="pod0",container="pod1_con2",node="node1",resource="cpu",unit="core"} 0.3
kube_pod_container_resource_requests{namespace="default",pod="pod0",container="pod1_con2",node="node1",resource="memory",unit="byte"} 2e+08
# HELP kube_pod_container_resource_limits The number of requested limit resource by a container.
# TYPE kube_pod_container_resource_limits gauge
kube_pod_container_resource_limits{namespace="default",pod="pod0",container="pod1_con1",node="node1",resource="nvidia_com_gpu",unit="integer"} 1
kube_pod_container_resource_limits{namespace="default",pod="pod0",container="pod1_con1",node="node1",resource="cpu",unit="core"} 0.2
kube_pod_container_resource_limits{namespace="default",pod="pod0",container="pod1_con1",node="node1",resource="memory",unit="byte"} 1e+08
kube_pod_container_resource_limits{namespace="default",pod="pod0",container="pod1_con1",node="node1",resource="ephemeral_storage",unit="byte"} 3e+08
kube_pod_container_resource_limits{namespace="default",pod="pod0",container="pod1_con1",node="node1",resource="storage",unit="byte"} 4e+08
kube_pod_container_resource_limits{namespace="default",pod="pod0",container="pod1_con2",node="node1",resource="memory",unit="byte"} 2e+08
kube_pod_container_resource_limits{namespace="default",pod="pod0",container="pod1_con2",node="node1",resource="cpu",unit="core"} 0.3
# HELP kube_pod_container_resource_requests_cpu_cores The number of requested cpu cores by a container.
# TYPE kube_pod_container_resource_requests_cpu_cores gauge
kube_pod_container_resource_requests_cpu_cores{namespace="default",pod="pod0",container="pod1_con1",node="node1"} 0.2
kube_pod_container_resource_requests_cpu_cores{namespace="default",pod="pod0",container="pod1_con2",node="node1"} 0.3
# HELP kube_pod_container_resource_requests_memory_bytes The number of requested memory bytes by a container.
# TYPE kube_pod_container_resource_requests_memory_bytes gauge
kube_pod_container_resource_requests_memory_bytes{namespace="default",pod="pod0",container="pod1_con1",node="node1"} 1e+08
kube_pod_container_resource_requests_memory_bytes{namespace="default",pod="pod0",container="pod1_con2",node="node1"} 2e+08
# HELP kube_pod_container_resource_limits_cpu_cores The limit on cpu cores to be used by a container.
# TYPE kube_pod_container_resource_limits_cpu_cores gauge
kube_pod_container_resource_limits_cpu_cores{namespace="default",pod="pod0",container="pod1_con1",node="node1"} 0.2
kube_pod_container_resource_limits_cpu_cores{namespace="default",pod="pod0",container="pod1_con2",node="node1"} 0.3
# HELP kube_pod_container_resource_limits_memory_bytes The limit on memory to be used by a container in bytes.
# TYPE kube_pod_container_resource_limits_memory_bytes gauge
kube_pod_container_resource_limits_memory_bytes{namespace="default",pod="pod0",container="pod1_con1",node="node1"} 1e+08
kube_pod_container_resource_limits_memory_bytes{namespace="default",pod="pod0",container="pod1_con2",node="node1"} 2e+08
# HELP kube_pod_spec_volumes_persistentvolumeclaims_info Information about persistentvolumeclaim volumes in a pod.
# TYPE kube_pod_spec_volumes_persistentvolumeclaims_info gauge
# HELP kube_pod_spec_volumes_persistentvolumeclaims_readonly Describes whether a persistentvolumeclaim is mounted read only.
# TYPE kube_pod_spec_volumes_persistentvolumeclaims_readonly gauge`

	expectedSplit := strings.Split(strings.TrimSpace(expected), "\n")
	sort.Strings(expectedSplit)

	gotSplit := strings.Split(strings.TrimSpace(string(body)), "\n")

	gotFiltered := []string{}
	for _, l := range gotSplit {
		if strings.Contains(l, "kube_pod_") {
			gotFiltered = append(gotFiltered, l)
		}
	}

	sort.Strings(gotFiltered)

	if len(expectedSplit) != len(gotFiltered) {
		t.Fatal("expected different output length")
	}

	for i := 0; i < len(expectedSplit); i++ {
		if expectedSplit[i] != gotFiltered[i] {
			t.Fatalf("expected %v, but got %v", expectedSplit[i], gotFiltered[i])
		}
	}
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
			Name:              "pod" + i,
			CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
			Namespace:         "default",
			UID:               "abc-123-xxx",
		},
		Spec: v1.PodSpec{
			NodeName: "node1",
			Containers: []v1.Container{
				{
					Name: "pod1_con1",
					Resources: v1.ResourceRequirements{
						Requests: map[v1.ResourceName]resource.Quantity{
							v1.ResourceCPU:                    resource.MustParse("200m"),
							v1.ResourceMemory:                 resource.MustParse("100M"),
							v1.ResourceEphemeralStorage:       resource.MustParse("300M"),
							v1.ResourceStorage:                resource.MustParse("400M"),
							v1.ResourceName("nvidia.com/gpu"): resource.MustParse("1"),
						},
						Limits: map[v1.ResourceName]resource.Quantity{
							v1.ResourceCPU:                    resource.MustParse("200m"),
							v1.ResourceMemory:                 resource.MustParse("100M"),
							v1.ResourceEphemeralStorage:       resource.MustParse("300M"),
							v1.ResourceStorage:                resource.MustParse("400M"),
							v1.ResourceName("nvidia.com/gpu"): resource.MustParse("1"),
						},
					},
				},
				{
					Name: "pod1_con2",
					Resources: v1.ResourceRequirements{
						Requests: map[v1.ResourceName]resource.Quantity{
							v1.ResourceCPU:    resource.MustParse("300m"),
							v1.ResourceMemory: resource.MustParse("200M"),
						},
						Limits: map[v1.ResourceName]resource.Quantity{
							v1.ResourceCPU:    resource.MustParse("300m"),
							v1.ResourceMemory: resource.MustParse("200M"),
						},
					},
				},
			},
		},
		Status: v1.PodStatus{
			HostIP: "1.1.1.1",
			PodIP:  "1.2.3.4",
			Phase:  v1.PodRunning,
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name:        "container2",
					Image:       "k8s.gcr.io/hyperkube2",
					ImageID:     "docker://sha256:bbb",
					ContainerID: "docker://cd456",
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
					LastTerminationState: v1.ContainerState{
						Terminated: &v1.ContainerStateTerminated{
							Reason: "OOMKilled",
						},
					},
				},
				{
					Name:        "container3",
					Image:       "k8s.gcr.io/hyperkube3",
					ImageID:     "docker://sha256:ccc",
					ContainerID: "docker://ef789",
				},
			},
		},
	}

	_, err := client.CoreV1().Pods(metav1.NamespaceDefault).Create(&pod)
	return err
}
