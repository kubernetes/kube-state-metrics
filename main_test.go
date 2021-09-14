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
	"fmt"
	"io"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"

	"k8s.io/kube-state-metrics/v2/internal/store"
	"k8s.io/kube-state-metrics/v2/pkg/allowdenylist"
	"k8s.io/kube-state-metrics/v2/pkg/metricshandler"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func BenchmarkKubeStateMetrics(b *testing.B) {
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
	reg := prometheus.NewRegistry()

	builder := store.NewBuilder()
	builder.WithMetrics(reg)
	builder.WithEnabledResources(options.DefaultResources.AsSlice())
	builder.WithKubeClient(kubeClient)
	builder.WithSharding(0, 1)
	builder.WithContext(ctx)
	builder.WithNamespaces(options.DefaultNamespaces)
	builder.WithGenerateStoresFunc(builder.DefaultGenerateStoresFunc(), false)

	l, err := allowdenylist.New(map[string]struct{}{}, map[string]struct{}{})
	if err != nil {
		b.Fatal(err)
	}
	builder.WithAllowDenyList(l)

	builder.WithAllowAnnotations(map[string][]string{})
	builder.WithAllowLabels(map[string][]string{})

	// This test is not suitable to be compared in terms of time, as it includes
	// a one second wait. Use for memory allocation comparisons, profiling, ...
	handler := metricshandler.New(&options.Options{}, kubeClient, builder, false)
	b.Run("GenerateMetrics", func(b *testing.B) {
		handler.ConfigureSharding(ctx, 0, 1)

		// Wait for caches to fill
		time.Sleep(time.Second)
	})

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
	reg := prometheus.NewRegistry()
	builder := store.NewBuilder()
	builder.WithMetrics(reg)
	builder.WithEnabledResources(options.DefaultResources.AsSlice())
	builder.WithKubeClient(kubeClient)
	builder.WithNamespaces(options.DefaultNamespaces)
	builder.WithGenerateStoresFunc(builder.DefaultGenerateStoresFunc(), false)

	l, err := allowdenylist.New(map[string]struct{}{}, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	builder.WithAllowDenyList(l)
	builder.WithAllowLabels(map[string][]string{
		"kube_pod_labels": {
			"namespace",
			"pod",
			"uid",
		},
	})

	handler := metricshandler.New(&options.Options{}, kubeClient, builder, false)
	handler.ConfigureSharding(ctx, 0, 1)

	// Wait for caches to fill
	time.Sleep(time.Second)

	req := httptest.NewRequest("GET", "http://localhost:8080/metrics", nil)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 status code but got %v", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)

	expected := `# HELP kube_pod_annotations Kubernetes annotations converted to Prometheus labels.
# HELP kube_pod_completion_time Completion time in unix timestamp for a pod.
# HELP kube_pod_container_info Information about a container in a pod.
# HELP kube_pod_container_resource_limits The number of requested limit resource by a container.
# HELP kube_pod_container_resource_requests The number of requested request resource by a container.
# HELP kube_pod_container_state_started Start time in unix timestamp for a pod container.
# HELP kube_pod_container_status_last_terminated_reason Describes the last reason the container was in terminated state.
# HELP kube_pod_container_status_ready Describes whether the containers readiness check succeeded.
# HELP kube_pod_container_status_restarts_total The number of container restarts per container.
# HELP kube_pod_container_status_running Describes whether the container is currently in running state.
# HELP kube_pod_container_status_terminated Describes whether the container is currently in terminated state.
# HELP kube_pod_container_status_terminated_reason Describes the reason the container is currently in terminated state.
# HELP kube_pod_container_status_waiting Describes whether the container is currently in waiting state.
# HELP kube_pod_container_status_waiting_reason Describes the reason the container is currently in waiting state.
# HELP kube_pod_created Unix creation timestamp
# HELP kube_pod_deletion_timestamp Unix deletion timestamp
# HELP kube_pod_info Information about pod.
# HELP kube_pod_init_container_info Information about an init container in a pod.
# HELP kube_pod_init_container_resource_limits The number of requested limit resource by an init container.
# HELP kube_pod_init_container_resource_limits_cpu_cores The number of CPU cores requested limit by an init container.
# HELP kube_pod_init_container_resource_limits_ephemeral_storage_bytes Bytes of ephemeral-storage requested limit by an init container.
# HELP kube_pod_init_container_resource_limits_memory_bytes Bytes of memory requested limit by an init container.
# HELP kube_pod_init_container_resource_limits_storage_bytes Bytes of storage requested limit by an init container.
# HELP kube_pod_init_container_resource_requests The number of requested request resource by an init container.
# HELP kube_pod_init_container_resource_requests_cpu_cores The number of CPU cores requested by an init container.
# HELP kube_pod_init_container_resource_requests_ephemeral_storage_bytes Bytes of ephemeral-storage requested by an init container.
# HELP kube_pod_init_container_resource_requests_memory_bytes Bytes of memory requested by an init container.
# HELP kube_pod_init_container_resource_requests_storage_bytes Bytes of storage requested by an init container.
# HELP kube_pod_init_container_status_last_terminated_reason Describes the last reason the init container was in terminated state.
# HELP kube_pod_init_container_status_ready Describes whether the init containers readiness check succeeded.
# HELP kube_pod_init_container_status_restarts_total The number of restarts for the init container.
# HELP kube_pod_init_container_status_running Describes whether the init container is currently in running state.
# HELP kube_pod_init_container_status_terminated Describes whether the init container is currently in terminated state.
# HELP kube_pod_init_container_status_terminated_reason Describes the reason the init container is currently in terminated state.
# HELP kube_pod_init_container_status_waiting Describes whether the init container is currently in waiting state.
# HELP kube_pod_init_container_status_waiting_reason Describes the reason the init container is currently in waiting state.
# HELP kube_pod_labels Kubernetes labels converted to Prometheus labels.
# HELP kube_pod_overhead_cpu_cores The pod overhead in regards to cpu cores associated with running a pod.
# HELP kube_pod_overhead_memory_bytes The pod overhead in regards to memory associated with running a pod.
# HELP kube_pod_runtimeclass_name_info The runtimeclass associated with the pod.
# HELP kube_pod_owner Information about the Pod's owner.
# HELP kube_pod_restart_policy Describes the restart policy in use by this pod.
# HELP kube_pod_spec_volumes_persistentvolumeclaims_info Information about persistentvolumeclaim volumes in a pod.
# HELP kube_pod_spec_volumes_persistentvolumeclaims_readonly Describes whether a persistentvolumeclaim is mounted read only.
# HELP kube_pod_start_time Start time in unix timestamp for a pod.
# HELP kube_pod_status_phase The pods current phase.
# HELP kube_pod_status_ready Describes whether the pod is ready to serve requests.
# HELP kube_pod_status_reason The pod status reasons
# HELP kube_pod_status_scheduled Describes the status of the scheduling process for the pod.
# HELP kube_pod_status_scheduled_time Unix timestamp when pod moved into scheduled status
# HELP kube_pod_status_unschedulable Describes the unschedulable status for the pod.
# TYPE kube_pod_annotations gauge
# TYPE kube_pod_completion_time gauge
# TYPE kube_pod_container_info gauge
# TYPE kube_pod_container_resource_limits gauge
# TYPE kube_pod_container_resource_requests gauge
# TYPE kube_pod_container_state_started gauge
# TYPE kube_pod_container_status_last_terminated_reason gauge
# TYPE kube_pod_container_status_ready gauge
# TYPE kube_pod_container_status_restarts_total counter
# TYPE kube_pod_container_status_running gauge
# TYPE kube_pod_container_status_terminated gauge
# TYPE kube_pod_container_status_terminated_reason gauge
# TYPE kube_pod_container_status_waiting gauge
# TYPE kube_pod_container_status_waiting_reason gauge
# TYPE kube_pod_created gauge
# TYPE kube_pod_deletion_timestamp gauge
# TYPE kube_pod_info gauge
# TYPE kube_pod_init_container_info gauge
# TYPE kube_pod_init_container_resource_limits gauge
# TYPE kube_pod_init_container_resource_limits_cpu_cores gauge
# TYPE kube_pod_init_container_resource_limits_ephemeral_storage_bytes gauge
# TYPE kube_pod_init_container_resource_limits_memory_bytes gauge
# TYPE kube_pod_init_container_resource_limits_storage_bytes gauge
# TYPE kube_pod_init_container_resource_requests gauge
# TYPE kube_pod_init_container_resource_requests_cpu_cores gauge
# TYPE kube_pod_init_container_resource_requests_ephemeral_storage_bytes gauge
# TYPE kube_pod_init_container_resource_requests_memory_bytes gauge
# TYPE kube_pod_init_container_resource_requests_storage_bytes gauge
# TYPE kube_pod_init_container_status_last_terminated_reason gauge
# TYPE kube_pod_init_container_status_ready gauge
# TYPE kube_pod_init_container_status_restarts_total counter
# TYPE kube_pod_init_container_status_running gauge
# TYPE kube_pod_init_container_status_terminated gauge
# TYPE kube_pod_init_container_status_terminated_reason gauge
# TYPE kube_pod_init_container_status_waiting gauge
# TYPE kube_pod_init_container_status_waiting_reason gauge
# TYPE kube_pod_labels gauge
# TYPE kube_pod_overhead_cpu_cores gauge
# TYPE kube_pod_overhead_memory_bytes gauge
# TYPE kube_pod_runtimeclass_name_info gauge
# TYPE kube_pod_owner gauge
# TYPE kube_pod_restart_policy gauge
# TYPE kube_pod_spec_volumes_persistentvolumeclaims_info gauge
# TYPE kube_pod_spec_volumes_persistentvolumeclaims_readonly gauge
# TYPE kube_pod_start_time gauge
# TYPE kube_pod_status_phase gauge
# TYPE kube_pod_status_ready gauge
# TYPE kube_pod_status_reason gauge
# TYPE kube_pod_status_scheduled gauge
# TYPE kube_pod_status_scheduled_time gauge
# TYPE kube_pod_status_unschedulable gauge
kube_pod_annotations{namespace="default",pod="pod0",uid="abc-0"} 1
kube_pod_container_info{namespace="default",pod="pod0",uid="abc-0",container="container2",image="k8s.gcr.io/hyperkube2",image_id="docker://sha256:bbb",container_id="docker://cd456"} 1
kube_pod_container_info{namespace="default",pod="pod0",uid="abc-0",container="container3",image="k8s.gcr.io/hyperkube3",image_id="docker://sha256:ccc",container_id="docker://ef789"} 1
kube_pod_container_resource_limits{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1",node="node1",resource="cpu",unit="core"} 0.2
kube_pod_container_resource_limits{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1",node="node1",resource="ephemeral_storage",unit="byte"} 3e+08
kube_pod_container_resource_limits{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1",node="node1",resource="memory",unit="byte"} 1e+08
kube_pod_container_resource_limits{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1",node="node1",resource="nvidia_com_gpu",unit="integer"} 1
kube_pod_container_resource_limits{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1",node="node1",resource="storage",unit="byte"} 4e+08
kube_pod_container_resource_limits{namespace="default",pod="pod0",uid="abc-0",container="pod1_con2",node="node1",resource="cpu",unit="core"} 0.3
kube_pod_container_resource_limits{namespace="default",pod="pod0",uid="abc-0",container="pod1_con2",node="node1",resource="memory",unit="byte"} 2e+08
kube_pod_container_resource_requests{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1",node="node1",resource="cpu",unit="core"} 0.2
kube_pod_container_resource_requests{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1",node="node1",resource="ephemeral_storage",unit="byte"} 3e+08
kube_pod_container_resource_requests{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1",node="node1",resource="memory",unit="byte"} 1e+08
kube_pod_container_resource_requests{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1",node="node1",resource="nvidia_com_gpu",unit="integer"} 1
kube_pod_container_resource_requests{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1",node="node1",resource="storage",unit="byte"} 4e+08
kube_pod_container_resource_requests{namespace="default",pod="pod0",uid="abc-0",container="pod1_con2",node="node1",resource="cpu",unit="core"} 0.3
kube_pod_container_resource_requests{namespace="default",pod="pod0",uid="abc-0",container="pod1_con2",node="node1",resource="memory",unit="byte"} 2e+08
kube_pod_container_status_last_terminated_reason{namespace="default",pod="pod0",uid="abc-0",container="container2",reason="OOMKilled"} 1
kube_pod_container_status_ready{namespace="default",pod="pod0",uid="abc-0",container="container2"} 0
kube_pod_container_status_ready{namespace="default",pod="pod0",uid="abc-0",container="container3"} 0
kube_pod_container_status_restarts_total{namespace="default",pod="pod0",uid="abc-0",container="container2"} 0
kube_pod_container_status_restarts_total{namespace="default",pod="pod0",uid="abc-0",container="container3"} 0
kube_pod_container_status_running{namespace="default",pod="pod0",uid="abc-0",container="container2"} 0
kube_pod_container_status_running{namespace="default",pod="pod0",uid="abc-0",container="container3"} 0
kube_pod_container_status_terminated{namespace="default",pod="pod0",uid="abc-0",container="container2"} 0
kube_pod_container_status_terminated{namespace="default",pod="pod0",uid="abc-0",container="container3"} 0
kube_pod_container_status_waiting_reason{namespace="default",pod="pod0",uid="abc-0",container="container2",reason="CrashLoopBackOff"} 1
kube_pod_container_status_waiting{namespace="default",pod="pod0",uid="abc-0",container="container2"} 1
kube_pod_container_status_waiting{namespace="default",pod="pod0",uid="abc-0",container="container3"} 0
kube_pod_created{namespace="default",pod="pod0",uid="abc-0"} 1.5e+09
kube_pod_info{namespace="default",pod="pod0",uid="abc-0",host_ip="1.1.1.1",pod_ip="1.2.3.4",node="node1",created_by_kind="<none>",created_by_name="<none>",priority_class="",host_network="false"} 1
kube_pod_labels{namespace="default",pod="pod0",uid="abc-0"} 1
kube_pod_owner{namespace="default",pod="pod0",uid="abc-0",owner_kind="<none>",owner_name="<none>",owner_is_controller="<none>"} 1
kube_pod_restart_policy{namespace="default",pod="pod0",uid="abc-0",type="Always"} 1
kube_pod_status_phase{namespace="default",pod="pod0",uid="abc-0",phase="Failed"} 0
kube_pod_status_phase{namespace="default",pod="pod0",uid="abc-0",phase="Pending"} 0
kube_pod_status_phase{namespace="default",pod="pod0",uid="abc-0",phase="Running"} 1
kube_pod_status_phase{namespace="default",pod="pod0",uid="abc-0",phase="Succeeded"} 0
kube_pod_status_phase{namespace="default",pod="pod0",uid="abc-0",phase="Unknown"} 0
kube_pod_status_reason{namespace="default",pod="pod0",uid="abc-0",reason="Evicted"} 0
kube_pod_status_reason{namespace="default",pod="pod0",uid="abc-0",reason="NodeAffinity"} 0
kube_pod_status_reason{namespace="default",pod="pod0",uid="abc-0",reason="NodeLost"} 0
kube_pod_status_reason{namespace="default",pod="pod0",uid="abc-0",reason="Shutdown"} 0
kube_pod_status_reason{namespace="default",pod="pod0",uid="abc-0",reason="UnexpectedAdmissionError"} 0
`

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
		fmt.Println(len(expectedSplit))
		fmt.Println(len(gotFiltered))
		t.Fatalf("expected different output length, expected \n\n%s\n\ngot\n\n%s", expected, strings.Join(gotFiltered, "\n"))
	}

	for i := 0; i < len(expectedSplit); i++ {
		if expectedSplit[i] != gotFiltered[i] {
			t.Fatalf("expected:\n\n%v\n, but got:\n\n%v", expectedSplit[i], gotFiltered[i])
		}
	}

	telemetryMux := buildTelemetryServer(reg)

	req2 := httptest.NewRequest("GET", "http://localhost:8081/metrics", nil)

	w2 := httptest.NewRecorder()
	telemetryMux.ServeHTTP(w2, req2)

	resp2 := w2.Result()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 status code but got %v", resp.StatusCode)
	}

	body2, _ := io.ReadAll(resp2.Body)

	expected2 := `# HELP kube_state_metrics_shard_ordinal Current sharding ordinal/index of this instance
# HELP kube_state_metrics_total_shards Number of total shards this instance is aware of
# TYPE kube_state_metrics_shard_ordinal gauge
# TYPE kube_state_metrics_total_shards gauge
kube_state_metrics_shard_ordinal{shard_ordinal="0"} 0
kube_state_metrics_total_shards 1
`

	expectedSplit2 := strings.Split(strings.TrimSpace(expected2), "\n")
	sort.Strings(expectedSplit2)

	gotSplit2 := strings.Split(strings.TrimSpace(string(body2)), "\n")

	gotFiltered2 := []string{}
	for _, l := range gotSplit2 {
		if strings.Contains(l, "_shard") {
			gotFiltered2 = append(gotFiltered2, l)
		}
	}

	sort.Strings(gotFiltered2)

	if len(expectedSplit2) != len(gotFiltered2) {
		fmt.Println(len(expectedSplit2))
		fmt.Println(len(gotFiltered2))
		t.Fatalf("expected different output length, expected \n\n%s\n\ngot\n\n%s", expected2, strings.Join(gotFiltered2, "\n"))
	}

	for i := 0; i < len(expectedSplit2); i++ {
		if expectedSplit2[i] != gotFiltered2[i] {
			t.Fatalf("expected:\n\n%v\n, but got:\n\n%v", expectedSplit2[i], gotFiltered2[i])
		}
	}
}

// TestShardingEquivalenceScrapeCycle is a simple smoke test covering the entire cycle from
// cache filling to scraping comparing a sharded with an unsharded setup.
func TestShardingEquivalenceScrapeCycle(t *testing.T) {
	t.Parallel()

	kubeClient := fake.NewSimpleClientset()

	for i := 0; i < 10; i++ {
		err := pod(kubeClient, i)
		if err != nil {
			t.Fatalf("failed to insert sample pod %v", err.Error())
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l, err := allowdenylist.New(map[string]struct{}{}, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	reg := prometheus.NewRegistry()
	unshardedBuilder := store.NewBuilder()
	unshardedBuilder.WithMetrics(reg)
	unshardedBuilder.WithEnabledResources(options.DefaultResources.AsSlice())
	unshardedBuilder.WithKubeClient(kubeClient)
	unshardedBuilder.WithNamespaces(options.DefaultNamespaces)
	unshardedBuilder.WithAllowDenyList(l)
	unshardedBuilder.WithAllowLabels(map[string][]string{})
	unshardedBuilder.WithGenerateStoresFunc(unshardedBuilder.DefaultGenerateStoresFunc(), false)

	unshardedHandler := metricshandler.New(&options.Options{}, kubeClient, unshardedBuilder, false)
	unshardedHandler.ConfigureSharding(ctx, 0, 1)

	regShard1 := prometheus.NewRegistry()
	shardedBuilder1 := store.NewBuilder()
	shardedBuilder1.WithMetrics(regShard1)
	shardedBuilder1.WithEnabledResources(options.DefaultResources.AsSlice())
	shardedBuilder1.WithKubeClient(kubeClient)
	shardedBuilder1.WithNamespaces(options.DefaultNamespaces)
	shardedBuilder1.WithAllowDenyList(l)
	shardedBuilder1.WithAllowLabels(map[string][]string{})
	shardedBuilder1.WithGenerateStoresFunc(shardedBuilder1.DefaultGenerateStoresFunc(), false)

	shardedHandler1 := metricshandler.New(&options.Options{}, kubeClient, shardedBuilder1, false)
	shardedHandler1.ConfigureSharding(ctx, 0, 2)

	regShard2 := prometheus.NewRegistry()
	shardedBuilder2 := store.NewBuilder()
	shardedBuilder2.WithMetrics(regShard2)
	shardedBuilder2.WithEnabledResources(options.DefaultResources.AsSlice())
	shardedBuilder2.WithKubeClient(kubeClient)
	shardedBuilder2.WithNamespaces(options.DefaultNamespaces)
	shardedBuilder2.WithAllowDenyList(l)
	shardedBuilder2.WithAllowLabels(map[string][]string{})
	shardedBuilder2.WithGenerateStoresFunc(shardedBuilder2.DefaultGenerateStoresFunc(), false)

	shardedHandler2 := metricshandler.New(&options.Options{}, kubeClient, shardedBuilder2, false)
	shardedHandler2.ConfigureSharding(ctx, 1, 2)

	// Wait for caches to fill
	time.Sleep(time.Second)

	// unsharded request as the controlled environment
	req := httptest.NewRequest("GET", "http://localhost:8080/metrics", nil)

	w := httptest.NewRecorder()
	unshardedHandler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 status code but got %v", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	expected := string(body)

	// sharded requests
	//
	// request first shard
	req = httptest.NewRequest("GET", "http://localhost:8080/metrics", nil)

	w = httptest.NewRecorder()
	shardedHandler1.ServeHTTP(w, req)

	resp = w.Result()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 status code but got %v", resp.StatusCode)
	}

	body, _ = io.ReadAll(resp.Body)
	got1 := string(body)

	// request second shard
	req = httptest.NewRequest("GET", "http://localhost:8080/metrics", nil)

	w = httptest.NewRecorder()
	shardedHandler2.ServeHTTP(w, req)

	resp = w.Result()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 status code but got %v", resp.StatusCode)
	}

	body, _ = io.ReadAll(resp.Body)
	got2 := string(body)

	// normalize results:

	expectedSplit := strings.Split(strings.TrimSpace(expected), "\n")
	sort.Strings(expectedSplit)

	expectedFiltered := []string{}
	for _, l := range expectedSplit {
		if strings.HasPrefix(l, "kube_pod_") {
			expectedFiltered = append(expectedFiltered, l)
		}
	}

	got1Split := strings.Split(strings.TrimSpace(got1), "\n")
	sort.Strings(got1Split)

	got1Filtered := []string{}
	for _, l := range got1Split {
		if strings.HasPrefix(l, "kube_pod_") {
			got1Filtered = append(got1Filtered, l)
		}
	}

	got2Split := strings.Split(strings.TrimSpace(got2), "\n")
	sort.Strings(got2Split)

	got2Filtered := []string{}
	for _, l := range got2Split {
		if strings.HasPrefix(l, "kube_pod_") {
			got2Filtered = append(got2Filtered, l)
		}
	}

	// total metrics should be equal
	if len(expectedFiltered) != (len(got1Filtered) + len(got2Filtered)) {
		t.Fatalf("expected different output length, expected total %d got 1) %d 2) %d", len(expectedFiltered), len(got1Filtered), len(got2Filtered))
	}
	// smoke test to test that each shard actually represents a subset
	if len(got1Filtered) == 0 {
		t.Fatal("shard 1 has 0 metrics when it shouldn't")
	}
	if len(got2Filtered) == 0 {
		t.Fatal("shard 2 has 0 metrics when it shouldn't")
	}

	got1Filtered = append(got1Filtered, got2Filtered...)
	sort.Strings(got1Filtered)

	for i := 0; i < len(expectedFiltered); i++ {
		expected := strings.TrimSpace(expectedFiltered[i])
		got := strings.TrimSpace(got1Filtered[i])
		if expected != got {
			t.Fatalf("\n\nexpected:\n\n%q\n\nbut got:\n\n%q\n\n", expected, got)
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
			UID:             types.UID("abc-" + i),
		},
	}
	_, err := client.CoreV1().ConfigMaps(metav1.NamespaceDefault).Create(context.TODO(), &configMap, metav1.CreateOptions{})
	return err
}

func service(client *fake.Clientset, index int) error {
	i := strconv.Itoa(index)

	service := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "service" + i,
			ResourceVersion: "123456",
			UID:             types.UID("abc-" + i),
		},
	}
	_, err := client.CoreV1().Services(metav1.NamespaceDefault).Create(context.TODO(), &service, metav1.CreateOptions{})
	return err
}

func pod(client *fake.Clientset, index int) error {
	i := strconv.Itoa(index)

	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "pod" + i,
			CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
			Namespace:         "default",
			UID:               types.UID("abc-" + i),
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyAlways,
			NodeName:      "node1",
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

	_, err := client.CoreV1().Pods(metav1.NamespaceDefault).Create(context.TODO(), &pod, metav1.CreateOptions{})
	return err
}
