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
	"strconv"
	"strings"
	"testing"
	"time"

	kcollectors "k8s.io/kube-state-metrics/pkg/collectors"
	"k8s.io/kube-state-metrics/pkg/options"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/kube-state-metrics/pkg/whiteblacklist"
)

func BenchmarkKubeStateMetrics(b *testing.B) {
	var collectors []*kcollectors.Collector
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

	builder := kcollectors.NewBuilder(context.TODO())
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

	err := service(kubeClient, 0)
	if err != nil {
		t.Fatalf("failed to insert sample pod %v", err.Error())
	}

	builder := kcollectors.NewBuilder(context.TODO())
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

	expected := `# HELP kube_configmap_info Information about configmap.
# HELP kube_configmap_created Unix creation timestamp
# HELP kube_configmap_metadata_resource_version Resource version representing a specific version of the configmap.
# HELP kube_cronjob_labels Kubernetes labels converted to Prometheus labels.
# HELP kube_cronjob_info Info about cronjob.
# HELP kube_cronjob_created Unix creation timestamp
# HELP kube_cronjob_status_active Active holds pointers to currently running jobs.
# HELP kube_cronjob_status_last_schedule_time LastScheduleTime keeps information of when was the last time the job was successfully scheduled.
# HELP kube_cronjob_spec_suspend Suspend flag tells the controller to suspend subsequent executions.
# HELP kube_cronjob_spec_starting_deadline_seconds Deadline in seconds for starting the job if it misses scheduled time for any reason.
# HELP kube_cronjob_next_schedule_time Next time the cronjob should be scheduled. The time after lastScheduleTime, or after the cron job's creation time if it's never been scheduled. Use this to determine if the job is delayed.
# HELP kube_daemonset_created Unix creation timestamp
# HELP kube_daemonset_status_current_number_scheduled The number of nodes running at least one daemon pod and are supposed to.
# HELP kube_daemonset_status_desired_number_scheduled The number of nodes that should be running the daemon pod.
# HELP kube_daemonset_status_number_available The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and available
# HELP kube_daemonset_status_number_misscheduled The number of nodes running a daemon pod but are not supposed to.
# HELP kube_daemonset_status_number_ready The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and ready.
# HELP kube_daemonset_status_number_unavailable The number of nodes that should be running the daemon pod and have none of the daemon pod running and available
# HELP kube_daemonset_updated_number_scheduled The total number of nodes that are running updated daemon pod
# HELP kube_daemonset_metadata_generation Sequence number representing a specific generation of the desired state.
# HELP kube_daemonset_labels Kubernetes labels converted to Prometheus labels.
# HELP kube_deployment_created Unix creation timestamp
# HELP kube_deployment_status_replicas The number of replicas per deployment.
# HELP kube_deployment_status_replicas_available The number of available replicas per deployment.
# HELP kube_deployment_status_replicas_unavailable The number of unavailable replicas per deployment.
# HELP kube_deployment_status_replicas_updated The number of updated replicas per deployment.
# HELP kube_deployment_status_observed_generation The generation observed by the deployment controller.
# HELP kube_deployment_spec_replicas Number of desired pods for a deployment.
# HELP kube_deployment_spec_paused Whether the deployment is paused and will not be processed by the deployment controller.
# HELP kube_deployment_spec_strategy_rollingupdate_max_unavailable Maximum number of unavailable replicas during a rolling update of a deployment.
# HELP kube_deployment_spec_strategy_rollingupdate_max_surge Maximum number of replicas that can be scheduled above the desired number of replicas during a rolling update of a deployment.
# HELP kube_deployment_metadata_generation Sequence number representing a specific generation of the desired state.
# HELP kube_deployment_labels Kubernetes labels converted to Prometheus labels.
# HELP kube_job_labels Kubernetes labels converted to Prometheus labels.
# HELP kube_job_info Information about job.
# HELP kube_job_created Unix creation timestamp
# HELP kube_job_spec_parallelism The maximum desired number of pods the job should run at any given time.
# HELP kube_job_spec_completions The desired number of successfully finished pods the job should be run with.
# HELP kube_job_spec_active_deadline_seconds The duration in seconds relative to the startTime that the job may be active before the system tries to terminate it.
# HELP kube_job_status_succeeded The number of pods which reached Phase Succeeded.
# HELP kube_job_status_failed The number of pods which reached Phase Failed.
# HELP kube_job_status_active The number of actively running pods.
# HELP kube_job_complete The job has completed its execution.
# HELP kube_job_failed The job has failed its execution.
# HELP kube_job_status_start_time StartTime represents time when the job was acknowledged by the Job Manager.
# HELP kube_job_status_completion_time CompletionTime represents time when the job was completed.
# HELP kube_limitrange Information about limit range.
# HELP kube_limitrange_created Unix creation timestamp
# HELP kube_namespace_created Unix creation timestamp
# HELP kube_namespace_labels Kubernetes labels converted to Prometheus labels.
# HELP kube_namespace_annotations Kubernetes annotations converted to Prometheus labels.
# HELP kube_namespace_status_phase kubernetes namespace status phase.
# HELP kube_node_info Information about a cluster node.
# HELP kube_node_created Unix creation timestamp
# HELP kube_node_labels Kubernetes labels converted to Prometheus labels.
# HELP kube_node_spec_unschedulable Whether a node can schedule new pods.
# HELP kube_node_spec_taint The taint of a cluster node.
# HELP kube_node_status_condition The condition of a cluster node.
# HELP kube_node_status_phase The phase the node is currently in.
# HELP kube_node_status_capacity The capacity for different resources of a node.
# HELP kube_node_status_capacity_pods The total pod resources of the node.
# HELP kube_node_status_capacity_cpu_cores The total CPU resources of the node.
# HELP kube_node_status_capacity_memory_bytes The total memory resources of the node.
# HELP kube_node_status_allocatable The allocatable for different resources of a node that are available for scheduling.
# HELP kube_node_status_allocatable_pods The pod resources of a node that are available for scheduling.
# HELP kube_node_status_allocatable_cpu_cores The CPU resources of a node that are available for scheduling.
# HELP kube_node_status_allocatable_memory_bytes The memory resources of a node that are available for scheduling.
# HELP kube_persistentvolumeclaim_labels Kubernetes labels converted to Prometheus labels.
# HELP kube_persistentvolumeclaim_info Information about persistent volume claim.
# HELP kube_persistentvolumeclaim_status_phase The phase the persistent volume claim is currently in.
# HELP kube_persistentvolumeclaim_resource_requests_storage_bytes The capacity of storage requested by the persistent volume claim.
# HELP kube_persistentvolume_labels Kubernetes labels converted to Prometheus labels.
# HELP kube_persistentvolume_status_phase The phase indicates if a volume is available, bound to a claim, or released by a claim.
# HELP kube_persistentvolume_info Information about persistentvolume.
# HELP kube_poddisruptionbudget_created Unix creation timestamp
# HELP kube_poddisruptionbudget_status_current_healthy Current number of healthy pods
# HELP kube_poddisruptionbudget_status_desired_healthy Minimum desired number of healthy pods
# HELP kube_poddisruptionbudget_status_pod_disruptions_allowed Number of pod disruptions that are currently allowed
# HELP kube_poddisruptionbudget_status_expected_pods Total number of pods counted by this disruption budget
# HELP kube_poddisruptionbudget_status_observed_generation Most recent generation observed when updating this PDB status
# HELP kube_pod_info Information about pod.
# HELP kube_pod_start_time Start time in unix timestamp for a pod.
# HELP kube_pod_completion_time Completion time in unix timestamp for a pod.
# HELP kube_pod_owner Information about the Pod's owner.
# HELP kube_pod_labels Kubernetes labels converted to Prometheus labels.
# HELP kube_pod_created Unix creation timestamp
# HELP kube_pod_status_scheduled_time Unix timestamp when pod moved into scheduled status
# HELP kube_pod_status_phase The pods current phase.
# HELP kube_pod_status_ready Describes whether the pod is ready to serve requests.
# HELP kube_pod_status_scheduled Describes the status of the scheduling process for the pod.
# HELP kube_pod_container_info Information about a container in a pod.
# HELP kube_pod_container_status_waiting Describes whether the container is currently in waiting state.
# HELP kube_pod_container_status_waiting_reason Describes the reason the container is currently in waiting state.
# HELP kube_pod_container_status_running Describes whether the container is currently in running state.
# HELP kube_pod_container_status_terminated Describes whether the container is currently in terminated state.
# HELP kube_pod_container_status_terminated_reason Describes the reason the container is currently in terminated state.
# HELP kube_pod_container_status_last_terminated_reason Describes the last reason the container was in terminated state.
# HELP kube_pod_container_status_ready Describes whether the containers readiness check succeeded.
# HELP kube_pod_container_status_restarts_total The number of container restarts per container.
# HELP kube_pod_container_resource_requests The number of requested request resource by a container.
# HELP kube_pod_container_resource_limits The number of requested limit resource by a container.
# HELP kube_pod_container_resource_requests_cpu_cores The number of requested cpu cores by a container.
# HELP kube_pod_container_resource_requests_memory_bytes The number of requested memory bytes by a container.
# HELP kube_pod_container_resource_limits_cpu_cores The limit on cpu cores to be used by a container.
# HELP kube_pod_container_resource_limits_memory_bytes The limit on memory to be used by a container in bytes.
# HELP kube_pod_spec_volumes_persistentvolumeclaims_info Information about persistentvolumeclaim volumes in a pod.
# HELP kube_pod_spec_volumes_persistentvolumeclaims_readonly Describes whether a persistentvolumeclaim is mounted read only.
# HELP kube_replicaset_created Unix creation timestamp
# HELP kube_replicaset_status_replicas The number of replicas per ReplicaSet.
# HELP kube_replicaset_status_fully_labeled_replicas The number of fully labeled replicas per ReplicaSet.
# HELP kube_replicaset_status_ready_replicas The number of ready replicas per ReplicaSet.
# HELP kube_replicaset_status_observed_generation The generation observed by the ReplicaSet controller.
# HELP kube_replicaset_spec_replicas Number of desired pods for a ReplicaSet.
# HELP kube_replicaset_metadata_generation Sequence number representing a specific generation of the desired state.
# HELP kube_replicaset_owner Information about the ReplicaSet's owner.
# HELP kube_secret_info Information about secret.
# HELP kube_secret_type Type about secret.
# HELP kube_secret_labels Kubernetes labels converted to Prometheus labels.
# HELP kube_secret_created Unix creation timestamp
# HELP kube_secret_metadata_resource_version Resource version representing a specific version of secret.
# HELP kube_service_info Information about service.
kube_service_info{namespace="default",service="service0",cluster_ip="",external_name="",load_balancer_ip=""} 1
# HELP kube_service_created Unix creation timestamp
# HELP kube_service_spec_type Type about service.
kube_service_spec_type{namespace="default",service="service0",type=""} 1
# HELP kube_service_labels Kubernetes labels converted to Prometheus labels.
kube_service_labels{namespace="default",service="service0"} 1
# HELP kube_service_spec_external_ip Service external ips. One series for each ip
# HELP kube_service_status_load_balancer_ingress Service load balancer ingress status`

	got := strings.TrimSpace(string(body))

	if expected != got {
		t.Fatalf("expected:\n%v\nbut got:\n%v", expected, got)
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
				v1.Container{
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
				v1.Container{
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
				v1.ContainerStatus{
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
				v1.ContainerStatus{
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
