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

package app

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"k8s.io/kube-state-metrics/v2/pkg/optin"

	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	samplev1alpha1 "k8s.io/sample-controller/pkg/apis/samplecontroller/v1alpha1"
	samplefake "k8s.io/sample-controller/pkg/generated/clientset/versioned/fake"

	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/internal/store"
	"k8s.io/kube-state-metrics/v2/pkg/allowdenylist"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
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

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	klogFlags.Set("logtostderr", "false")
	defer cancel()
	reg := prometheus.NewRegistry()

	builder := store.NewBuilder()
	builder.WithMetrics(reg)
	err := builder.WithEnabledResources(options.DefaultResources.AsSlice())
	if err != nil {
		b.Fatal(err)
	}
	builder.WithKubeClient(kubeClient)
	builder.WithSharding(0, 1)
	builder.WithContext(ctx)
	builder.WithNamespaces(options.DefaultNamespaces)
	builder.WithGenerateStoresFunc(builder.DefaultGenerateStoresFunc())

	allowDenyListFilter, err := allowdenylist.New(map[string]struct{}{}, map[string]struct{}{})
	if err != nil {
		b.Fatal(err)
	}

	builder.WithFamilyGeneratorFilter(generator.NewCompositeFamilyGeneratorFilter(
		allowDenyListFilter,
	))

	builder.WithAllowAnnotations(map[string][]string{})
	builder.WithAllowLabels(map[string][]string{})

	// This test is not suitable to be compared in terms of time, as it includes
	// a one second wait. Use for memory allocation comparisons, profiling, ...
	handler := metricshandler.New(&options.Options{}, kubeClient, builder, false)
	b.Run("GenerateMetrics", func(_ *testing.B) {
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
			_, err := buf.ReadFrom(resp.Body)
			if err != nil {
				b.Fatal(err)
			}
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
	err = builder.WithEnabledResources(options.DefaultResources.AsSlice())
	if err != nil {
		t.Fatal(err)
	}
	builder.WithKubeClient(kubeClient)
	builder.WithNamespaces(options.DefaultNamespaces)
	builder.WithGenerateStoresFunc(builder.DefaultGenerateStoresFunc())

	l, err := allowdenylist.New(map[string]struct{}{}, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	optInMetrics := make(map[string]struct{})
	optInMetricFamilyFilter, err := optin.NewMetricFamilyFilter(optInMetrics)
	if err != nil {
		t.Fatal(err)
	}

	builder.WithFamilyGeneratorFilter(generator.NewCompositeFamilyGeneratorFilter(
		l,
		optInMetricFamilyFilter,
	))
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
# HELP kube_pod_completion_time [STABLE] Completion time in unix timestamp for a pod.
# HELP kube_pod_container_info [STABLE] Information about a container in a pod.
# HELP kube_pod_container_resource_limits The number of requested limit resource by a container. It is recommended to use the kube_pod_resource_limits metric exposed by kube-scheduler instead, as it is more precise.
# HELP kube_pod_container_resource_requests The number of requested request resource by a container. It is recommended to use the kube_pod_resource_requests metric exposed by kube-scheduler instead, as it is more precise.
# HELP kube_pod_container_state_started [STABLE] Start time in unix timestamp for a pod container.
# HELP kube_pod_container_status_last_terminated_exitcode Describes the exit code for the last container in terminated state.
# HELP kube_pod_container_status_last_terminated_reason Describes the last reason the container was in terminated state.
# HELP kube_pod_container_status_last_terminated_timestamp Last terminated time for a pod container in unix timestamp.
# HELP kube_pod_container_status_ready [STABLE] Describes whether the containers readiness check succeeded.
# HELP kube_pod_container_status_restarts_total [STABLE] The number of container restarts per container.
# HELP kube_pod_container_status_running [STABLE] Describes whether the container is currently in running state.
# HELP kube_pod_container_status_terminated [STABLE] Describes whether the container is currently in terminated state.
# HELP kube_pod_container_status_terminated_reason Describes the reason the container is currently in terminated state.
# HELP kube_pod_container_status_waiting [STABLE] Describes whether the container is currently in waiting state.
# HELP kube_pod_container_status_waiting_reason [STABLE] Describes the reason the container is currently in waiting state.
# HELP kube_pod_created [STABLE] Unix creation timestamp
# HELP kube_pod_deletion_timestamp Unix deletion timestamp
# HELP kube_pod_info [STABLE] Information about pod.
# HELP kube_pod_init_container_info [STABLE] Information about an init container in a pod.
# HELP kube_pod_init_container_resource_limits The number of requested limit resource by an init container.
# HELP kube_pod_init_container_resource_requests The number of requested request resource by an init container.
# HELP kube_pod_init_container_status_last_terminated_reason Describes the last reason the init container was in terminated state.
# HELP kube_pod_init_container_status_ready [STABLE] Describes whether the init containers readiness check succeeded.
# HELP kube_pod_init_container_status_restarts_total [STABLE] The number of restarts for the init container.
# HELP kube_pod_init_container_status_running [STABLE] Describes whether the init container is currently in running state.
# HELP kube_pod_init_container_status_terminated [STABLE] Describes whether the init container is currently in terminated state.
# HELP kube_pod_init_container_status_terminated_reason Describes the reason the init container is currently in terminated state.
# HELP kube_pod_init_container_status_waiting [STABLE] Describes whether the init container is currently in waiting state.
# HELP kube_pod_init_container_status_waiting_reason Describes the reason the init container is currently in waiting state.
# HELP kube_pod_ips Pod IP addresses
# HELP kube_pod_labels [STABLE] Kubernetes labels converted to Prometheus labels.
# HELP kube_pod_overhead_cpu_cores The pod overhead in regards to cpu cores associated with running a pod.
# HELP kube_pod_overhead_memory_bytes The pod overhead in regards to memory associated with running a pod.
# HELP kube_pod_runtimeclass_name_info The runtimeclass associated with the pod.
# HELP kube_pod_scheduler The scheduler for a pod.
# HELP kube_pod_service_account The service account for a pod.
# HELP kube_pod_owner [STABLE] Information about the Pod's owner.
# HELP kube_pod_restart_policy [STABLE] Describes the restart policy in use by this pod.
# HELP kube_pod_spec_volumes_persistentvolumeclaims_info [STABLE] Information about persistentvolumeclaim volumes in a pod.
# HELP kube_pod_spec_volumes_persistentvolumeclaims_readonly [STABLE] Describes whether a persistentvolumeclaim is mounted read only.
# HELP kube_pod_start_time [STABLE] Start time in unix timestamp for a pod.
# HELP kube_pod_status_container_ready_time Readiness achieved time in unix timestamp for a pod containers.
# HELP kube_pod_status_initialized_time Initialized time in unix timestamp for a pod.
# HELP kube_pod_status_qos_class The pods current qosClass.
# HELP kube_pod_status_phase [STABLE] The pods current phase.
# HELP kube_pod_status_ready_time Readiness achieved time in unix timestamp for a pod.
# HELP kube_pod_status_ready [STABLE] Describes whether the pod is ready to serve requests.
# HELP kube_pod_status_reason The pod status reasons
# HELP kube_pod_status_scheduled [STABLE] Describes the status of the scheduling process for the pod.
# HELP kube_pod_status_scheduled_time [STABLE] Unix timestamp when pod moved into scheduled status
# HELP kube_pod_status_unschedulable [STABLE] Describes the unschedulable status for the pod.
# HELP kube_pod_status_unscheduled_time Unix timestamp when pod moved into unscheduled status
# HELP kube_pod_tolerations Information about the pod tolerations
# TYPE kube_pod_annotations gauge
# TYPE kube_pod_completion_time gauge
# TYPE kube_pod_container_info gauge
# TYPE kube_pod_container_resource_limits gauge
# TYPE kube_pod_container_resource_requests gauge
# TYPE kube_pod_container_state_started gauge
# TYPE kube_pod_container_status_last_terminated_exitcode gauge
# TYPE kube_pod_container_status_last_terminated_reason gauge
# TYPE kube_pod_container_status_last_terminated_timestamp gauge
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
# TYPE kube_pod_init_container_resource_requests gauge
# TYPE kube_pod_init_container_status_last_terminated_reason gauge
# TYPE kube_pod_init_container_status_ready gauge
# TYPE kube_pod_init_container_status_restarts_total counter
# TYPE kube_pod_init_container_status_running gauge
# TYPE kube_pod_init_container_status_terminated gauge
# TYPE kube_pod_init_container_status_terminated_reason gauge
# TYPE kube_pod_init_container_status_waiting gauge
# TYPE kube_pod_init_container_status_waiting_reason gauge
# TYPE kube_pod_ips gauge
# TYPE kube_pod_labels gauge
# TYPE kube_pod_overhead_cpu_cores gauge
# TYPE kube_pod_overhead_memory_bytes gauge
# TYPE kube_pod_runtimeclass_name_info gauge
# TYPE kube_pod_scheduler gauge
# TYPE kube_pod_service_account gauge
# TYPE kube_pod_owner gauge
# TYPE kube_pod_restart_policy gauge
# TYPE kube_pod_spec_volumes_persistentvolumeclaims_info gauge
# TYPE kube_pod_spec_volumes_persistentvolumeclaims_readonly gauge
# TYPE kube_pod_start_time gauge
# TYPE kube_pod_status_container_ready_time gauge
# TYPE kube_pod_status_initialized_time gauge
# TYPE kube_pod_status_phase gauge
# TYPE kube_pod_status_qos_class gauge
# TYPE kube_pod_status_ready gauge
# TYPE kube_pod_status_ready_time gauge
# TYPE kube_pod_status_reason gauge
# TYPE kube_pod_status_scheduled gauge
# TYPE kube_pod_status_scheduled_time gauge
# TYPE kube_pod_status_unschedulable gauge
# TYPE kube_pod_status_unscheduled_time gauge
# TYPE kube_pod_tolerations gauge
kube_pod_container_info{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1",image_spec="k8s.gcr.io/hyperkube2_spec",image="k8s.gcr.io/hyperkube2",image_id="docker://sha256:bbb",container_id="docker://cd456"} 1
kube_pod_container_info{namespace="default",pod="pod0",uid="abc-0",container="pod1_con2",image_spec="k8s.gcr.io/hyperkube3_spec",image="k8s.gcr.io/hyperkube3",image_id="docker://sha256:ccc",container_id="docker://ef789"} 1
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
kube_pod_container_status_last_terminated_exitcode{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1"} 137
kube_pod_container_status_last_terminated_reason{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1",reason="OOMKilled"} 1
kube_pod_container_status_last_terminated_timestamp{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1"} 1.501779547e+09
kube_pod_container_status_ready{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1"} 0
kube_pod_container_status_ready{namespace="default",pod="pod0",uid="abc-0",container="pod1_con2"} 0
kube_pod_container_status_restarts_total{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1"} 0
kube_pod_container_status_restarts_total{namespace="default",pod="pod0",uid="abc-0",container="pod1_con2"} 0
kube_pod_container_status_running{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1"} 0
kube_pod_container_status_running{namespace="default",pod="pod0",uid="abc-0",container="pod1_con2"} 0
kube_pod_container_status_terminated{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1"} 0
kube_pod_container_status_terminated{namespace="default",pod="pod0",uid="abc-0",container="pod1_con2"} 0
kube_pod_container_status_waiting_reason{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1",reason="CrashLoopBackOff"} 1
kube_pod_container_status_waiting{namespace="default",pod="pod0",uid="abc-0",container="pod1_con1"} 1
kube_pod_container_status_waiting{namespace="default",pod="pod0",uid="abc-0",container="pod1_con2"} 0
kube_pod_created{namespace="default",pod="pod0",uid="abc-0"} 1.5e+09
kube_pod_info{namespace="default",pod="pod0",uid="abc-0",host_ip="1.1.1.1",pod_ip="1.2.3.4",node="node1",created_by_kind="",created_by_name="",priority_class="",host_network="false"} 1
kube_pod_owner{namespace="default",pod="pod0",uid="abc-0",owner_kind="",owner_name="",owner_is_controller=""} 1
kube_pod_restart_policy{namespace="default",pod="pod0",uid="abc-0",type="Always"} 1
kube_pod_scheduler{namespace="default",pod="pod0",uid="abc-0",name="scheduler1"} 1
kube_pod_service_account{namespace="default",pod="pod0",uid="abc-0",service_account=""} 1
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

	telemetryMux := buildTelemetryServer(reg, false, nil)

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
	err = unshardedBuilder.WithEnabledResources(options.DefaultResources.AsSlice())
	if err != nil {
		t.Fatal(err)
	}
	unshardedBuilder.WithKubeClient(kubeClient)
	unshardedBuilder.WithNamespaces(options.DefaultNamespaces)
	unshardedBuilder.WithFamilyGeneratorFilter(l)
	unshardedBuilder.WithAllowLabels(map[string][]string{})
	unshardedBuilder.WithGenerateStoresFunc(unshardedBuilder.DefaultGenerateStoresFunc())

	unshardedHandler := metricshandler.New(&options.Options{}, kubeClient, unshardedBuilder, false)
	unshardedHandler.ConfigureSharding(ctx, 0, 1)

	regShard1 := prometheus.NewRegistry()
	shardedBuilder1 := store.NewBuilder()
	shardedBuilder1.WithMetrics(regShard1)
	err = shardedBuilder1.WithEnabledResources(options.DefaultResources.AsSlice())
	if err != nil {
		t.Fatal(err)
	}
	shardedBuilder1.WithKubeClient(kubeClient)
	shardedBuilder1.WithNamespaces(options.DefaultNamespaces)
	shardedBuilder1.WithFamilyGeneratorFilter(l)
	shardedBuilder1.WithAllowLabels(map[string][]string{})
	shardedBuilder1.WithGenerateStoresFunc(shardedBuilder1.DefaultGenerateStoresFunc())

	shardedHandler1 := metricshandler.New(&options.Options{}, kubeClient, shardedBuilder1, false)
	shardedHandler1.ConfigureSharding(ctx, 0, 2)

	regShard2 := prometheus.NewRegistry()
	shardedBuilder2 := store.NewBuilder()
	shardedBuilder2.WithMetrics(regShard2)
	err = shardedBuilder2.WithEnabledResources(options.DefaultResources.AsSlice())
	if err != nil {
		t.Fatal(err)
	}
	shardedBuilder2.WithKubeClient(kubeClient)
	shardedBuilder2.WithNamespaces(options.DefaultNamespaces)
	shardedBuilder2.WithFamilyGeneratorFilter(l)
	shardedBuilder2.WithAllowLabels(map[string][]string{})
	shardedBuilder2.WithGenerateStoresFunc(shardedBuilder2.DefaultGenerateStoresFunc())

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

// TestCustomResourceExtension is a simple smoke test covering the custom resource metrics collection.
// We use custom resource object samplev1alpha1.Foo in kubernetes/sample-controller as an example.
func TestCustomResourceExtension(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	factories := []customresource.RegistryFactory{new(fooFactory)}
	resources := options.DefaultResources.AsSlice()
	customResourceClients := make(map[string]interface{}, len(factories))
	// enable custom resource
	for _, f := range factories {
		resources = append(resources, f.Name())
		customResourceClient, err := f.CreateClient(nil)
		if err != nil {
			t.Fatalf("Failed to create customResourceClient for foo: %v", err)
		}
		customResourceClients[f.Name()] = customResourceClient
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reg := prometheus.NewRegistry()
	builder := store.NewBuilder()
	builder.WithCustomResourceStoreFactories(factories...)
	builder.WithMetrics(reg)
	err := builder.WithEnabledResources(resources)
	if err != nil {
		t.Fatal(err)
	}

	builder.WithKubeClient(kubeClient)
	builder.WithCustomResourceClients(customResourceClients)
	builder.WithNamespaces(options.DefaultNamespaces)
	builder.WithGenerateStoresFunc(builder.DefaultGenerateStoresFunc())
	builder.WithGenerateCustomResourceStoresFunc(builder.DefaultGenerateCustomResourceStoresFunc())

	l, err := allowdenylist.New(map[string]struct{}{}, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	builder.WithFamilyGeneratorFilter(l)
	builder.WithAllowLabels(map[string][]string{
		"kube_foo_labels": {
			"namespace",
			"foo",
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

	expected := `# HELP kube_foo_spec_replicas Number of desired replicas for a foo.
# HELP kube_foo_status_replicas_available The number of available replicas per foo.
# TYPE kube_foo_spec_replicas gauge
# TYPE kube_foo_status_replicas_available gauge
kube_foo_spec_replicas{namespace="default",foo="foo0"} 0
kube_foo_spec_replicas{namespace="default",foo="foo1"} 1
kube_foo_spec_replicas{namespace="default",foo="foo2"} 2
kube_foo_spec_replicas{namespace="default",foo="foo3"} 3
kube_foo_spec_replicas{namespace="default",foo="foo4"} 4
kube_foo_spec_replicas{namespace="default",foo="foo5"} 5
kube_foo_spec_replicas{namespace="default",foo="foo6"} 6
kube_foo_spec_replicas{namespace="default",foo="foo7"} 7
kube_foo_spec_replicas{namespace="default",foo="foo8"} 8
kube_foo_spec_replicas{namespace="default",foo="foo9"} 9
kube_foo_status_replicas_available{namespace="default",foo="foo0"} 0
kube_foo_status_replicas_available{namespace="default",foo="foo1"} 1
kube_foo_status_replicas_available{namespace="default",foo="foo2"} 2
kube_foo_status_replicas_available{namespace="default",foo="foo3"} 3
kube_foo_status_replicas_available{namespace="default",foo="foo5"} 5
kube_foo_status_replicas_available{namespace="default",foo="foo6"} 6
kube_foo_status_replicas_available{namespace="default",foo="foo7"} 7
kube_foo_status_replicas_available{namespace="default",foo="foo8"} 8
kube_foo_status_replicas_available{namespace="default",foo="foo4"} 4
kube_foo_status_replicas_available{namespace="default",foo="foo9"} 9
`

	expectedSplit := strings.Split(strings.TrimSpace(expected), "\n")
	sort.Strings(expectedSplit)

	gotSplit := strings.Split(strings.TrimSpace(string(body)), "\n")

	gotFiltered := []string{}
	for _, l := range gotSplit {
		if strings.Contains(l, "kube_foo_") {
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
			SchedulerName: "scheduler1",
			RestartPolicy: v1.RestartPolicyAlways,
			NodeName:      "node1",
			Containers: []v1.Container{
				{
					Image: "k8s.gcr.io/hyperkube2_spec",
					Name:  "pod1_con1",
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
					Image: "k8s.gcr.io/hyperkube3_spec",
					Name:  "pod1_con2",
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
					Name:        "pod1_con1",
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
							FinishedAt: metav1.Time{
								Time: time.Unix(1501779547, 0),
							},
							Reason:   "OOMKilled",
							ExitCode: 137,
						},
					},
				},
				{
					Name:        "pod1_con2",
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

func foo(client *samplefake.Clientset, index int) error {
	i := strconv.Itoa(index)
	desiredReplicas := int32(index) //nolint:gosec

	foo := samplev1alpha1.Foo{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "foo" + i,
			CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
			UID:               types.UID("abc-" + i),
		},
		Spec: samplev1alpha1.FooSpec{
			DeploymentName: "foo" + i,
			Replicas:       &desiredReplicas,
		},
		Status: samplev1alpha1.FooStatus{
			AvailableReplicas: desiredReplicas,
		},
	}

	_, err := client.SamplecontrollerV1alpha1().Foos(metav1.NamespaceDefault).Create(context.TODO(), &foo, metav1.CreateOptions{})
	return err
}

var (
	descFooLabelsDefaultLabels = []string{"namespace", "foo"}
)

type fooFactory struct{}

func (f *fooFactory) Name() string {
	return "foos"
}

// CreateClient use fake client set to establish 10 foos.
func (f *fooFactory) CreateClient(_ *rest.Config) (interface{}, error) {
	fooClient := samplefake.NewSimpleClientset()
	for i := 0; i < 10; i++ {
		err := foo(fooClient, i)
		if err != nil {
			return nil, fmt.Errorf("failed to insert sample pod %v", err)
		}
	}
	return fooClient, nil
}

func (f *fooFactory) MetricFamilyGenerators() []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_foo_spec_replicas",
			"Number of desired replicas for a foo.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapFooFunc(func(f *samplev1alpha1.Foo) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(*f.Spec.Replicas),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_foo_status_replicas_available",
			"The number of available replicas per foo.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapFooFunc(func(f *samplev1alpha1.Foo) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(f.Status.AvailableReplicas),
						},
					},
				}
			}),
		),
	}
}

func wrapFooFunc(f func(*samplev1alpha1.Foo) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		foo := obj.(*samplev1alpha1.Foo)

		metricFamily := f(foo)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descFooLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{foo.Namespace, foo.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func (f *fooFactory) ExpectedType() interface{} {
	return &samplev1alpha1.Foo{}
}

func (f *fooFactory) ListWatch(customResourceClient interface{}, ns string, fieldSelector string) cache.ListerWatcher {
	client := customResourceClient.(*samplefake.Clientset)
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return client.SamplecontrollerV1alpha1().Foos(ns).List(context.Background(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return client.SamplecontrollerV1alpha1().Foos(ns).Watch(context.Background(), opts)
		},
	}
}
func TestConfigureResourcesAndMetrics(t *testing.T) {
	// Prepare a config file in YAML format
	configYAML := `
"resources":
  "pod": {}
  "service": {}
"metric_allowlist":
  "kube_pod_info": {}
"metric_denylist":
  "kube_pod_labels": {}
"metric_opt_in_list":
  "kube_pod_status_phase": {}
"labels_allow_list":
  "labelX": 
    - foo 
    - bar
"annotations_allow_list":
  "annotationY": 
     - baz
`
	opts := options.NewOptions()
	// Set some initial values to be overwritten
	opts.Resources = options.ResourceSet{"oldresource": {}}
	opts.MetricAllowlist = options.MetricSet{"oldallow": {}}
	opts.MetricDenylist = options.MetricSet{"olddeny": {}}
	opts.MetricOptInList = options.MetricSet{"oldoptin": {}}
	opts.LabelsAllowList = options.LabelsAllowList{"oldlabel": {"oldvalue"}}
	opts.AnnotationsAllowList = options.LabelsAllowList{"oldannotation": {"oldvalue"}}

	newOpts := configureResourcesAndMetrics(opts, []byte(configYAML))

	// Check resources
	expectedResources := []string{"pod", "service"}
	for _, r := range expectedResources {
		if _, ok := newOpts.Resources[r]; !ok {
			t.Errorf("expected resource %q in opts.Resources", r)
		}
	}
	if _, ok := newOpts.Resources["oldresource"]; ok {
		t.Errorf("expected oldresource to be overwritten")
	}

	// Check metric allowlist
	if _, ok := newOpts.MetricAllowlist["kube_pod_info"]; !ok {
		t.Errorf("expected kube_pod_info in MetricAllowlist")
	}
	if _, ok := newOpts.MetricAllowlist["oldallow"]; ok {
		t.Errorf("expected oldallow to be overwritten")
	}

	// Check metric denylist
	if _, ok := newOpts.MetricDenylist["kube_pod_labels"]; !ok {
		t.Errorf("expected kube_pod_labels in MetricDenylist")
	}
	if _, ok := newOpts.MetricDenylist["olddeny"]; ok {
		t.Errorf("expected olddeny to be overwritten")
	}

	// Check metric opt-in list
	if _, ok := newOpts.MetricOptInList["kube_pod_status_phase"]; !ok {
		t.Errorf("expected kube_pod_status_phase in MetricOptInList")
	}
	if _, ok := newOpts.MetricOptInList["oldoptin"]; ok {
		t.Errorf("expected oldoptin to be overwritten")
	}

	// Check labels allow list
	if vals, ok := newOpts.LabelsAllowList["labelX"]; !ok || len(vals) != 2 || vals[0] != "foo" || vals[1] != "bar" {
		t.Errorf("expected labelX with values [foo bar], got %v", vals)
	}
	if vals, ok := newOpts.LabelsAllowList["oldlabel"]; ok {
		t.Errorf("expected oldlabel to be overwritten, got %v", vals)
	}

	// Check annotations allow list
	if vals, ok := newOpts.AnnotationsAllowList["annotationY"]; !ok || len(vals) != 1 || vals[0] != "baz" {
		t.Errorf("expected annotationY with value [baz], got %v", vals)
	}
	if vals, ok := newOpts.AnnotationsAllowList["oldannotation"]; ok {
		t.Errorf("expected oldannotation to be overwritten, got %v", vals)
	}

}

func TestConfigureResourcesAndMetrics_InvalidYAML(t *testing.T) {
	opts := options.NewOptions()
	invalidYAML := []byte("invalid: [unclosed")
	// Should not panic or overwrite opts
	result := configureResourcesAndMetrics(opts, invalidYAML)
	if result != opts {
		t.Errorf("expected opts to be returned unchanged on invalid YAML")
	}
}
