## Unreleased

* [CHANGE] `kube_job_status_start_time` and `kube_job_status_completion_time` metric types changed from counter to gauge.

## v1.3.0 / 2018-04-04

After a testing period of 12 days, there were no additional bugs found or features introduced.

## v1.3.0-rc.0 / 2018-03-23

* [FEATURE] Allow to specify multiple namespace.
* [FEATURE] Add `kube_pod_completion_time`, `kube_pod_spec_volumes_persistentvolumeclaims_info`, and `kube_pod_spec_volumes_persistentvolumeclaims_readonly` metrics to the Pod collector.
* [FEATURE] Add `kube_node_spec_taint` metric.
* [FEATURE] Add `kube_namespace_annotations` metric.
* [FEATURE] Add `kube_deployment_spec_strategy_rollingupdate_max_surge` metric.
* [FEATURE] Add `kube_persistentvolume_labels` metric.
* [FEATURE] Add `kube_persistentvolumeclaim_lables` metric.
* [FEATURE] Add `kube_daemonset_labels` metric.
* [FEATURE] Add Secret metrics.
* [FEATURE] Add ConfigMap metrics.
* [ENHANCEMENT] Add additional reasons to `kube_pod_container_status_waiting_reason` metric.
* [BUGFIX] Fix namespacing of HPA.
* [BUGFIX] Fix namespacing of PersistentVolumes.
* [BUGFIX] Fix CronJob tab parsing.

## v1.2.0 / 2018-01-15

After a testing period of 10 days, there were no additional bugs found or features introduced.

## v1.2.0-rc.0 / 2018-01-05

* [CHANGE] The CronJob collector now expects the version to be v1beta1.
* [FEATURE] Add `Endpoints` metrics collector.
* [FEATURE] Add `PersistentVolume` metrics collector.
* [FEATURE] Add `HorizontalPodAutoscaler` metrics collector.
* [FEATURE] Add `kube_pod_container_status_terminated_reason` metric.
* [FEATURE] Add `kube_job_labels` metric.
* [FEATURE] Add `kube_cronjob_labels` metric.
* [FEATURE] Add `kube_service_spec_type` metric.
* [FEATURE] Add `kube_statefulset_status_replicas_current` metric.
* [FEATURE] Add `kube_statefulset_status_replicas_ready` metric.
* [FEATURE] Add `kube_statefulset_status_replicas_updated` metric.
* [ENHANCEMENT] Allow specifying the host/IP kube-state-metrics binds to.
* [ENHANCEMENT] Add `volumename` label to `kube_persistentvolumeclaim_info` metric.
* [ENHANCEMENT] Add `cluster_ip` label to `kube_service_info` metric.
* [ENHANCEMENT] Print version on startup and useful debug information at runtime.
* [ENHANCEMENT] Add metrics for kube-state-metrics itself. For separation purposes this listens on a separate host/IP and port, both configurable respectively.

## v1.1.0 / 2017-10-19

After a testing period of one week, there were no additional bugs found or features introduced.

## v1.1.0-rc.0 / 2017-10-12

* [FEATURE] Add `kube_pod_container_status_waiting_reason` metric.
* [FEATURE] Add `kube_node_status_capacity_nvidia_gpu_cards` and `kube_node_status_allocatable_nvidia_gpu_cards` metrics.
* [FEATURE] Add `kube_persistentvolumeclaim_info`, `kube_persistentvolumeclaim_status_phase` and `kube_persistentvolumeclaim_resource_requests_storage_bytes` metrics.
* [FEATURE] Add `kube_cronjob_created` metric.
* [FEATURE] Add `kube_namespace_status_phase`, `kube_namespace_labels` and `kube_namespace_created` metrics.
* [FEATURE] Add `*_created` metrics for all available collectors and resources.
* [FEATURE] Add ability to specify glog flags.
* [FEATURE] Add ability to limit kube-state-metrics objects to single namespace.
* [ENHANCEMENT] Bump client-go to 5.0 release branch.
* [ENHANCEMENT] Add pprof endpoints for profiling.
* [ENHANCEMENT] Log resources and API versions used when collecting metrics from objects.
* [ENHANCEMENT] Log number of resources used to generate metrics off of.
* [ENHANCEMENT] Improve a usage message for collectors flag.
* [BUGFIX] Fix Job start time nil panic.

## v1.0.1 / 2017-08-24

* [BUGFIX] Fix nil pointer panic when pods have an owner without controllers.

## v1.0.0 / 2017-08-09

After a testing period of one week, there were no additional bugs found or features introduced.

## v1.0.0-rc.1 / 2017-08-02

* [CHANGE] Remove `kube_node_status_ready`, `kube_node_status_out_of_disk`, `kube_node_status_memory_pressure`, `kube_node_status_disk_pressure`, and `kube_node_status_network_unavailable` metrics in favor of one generic `kube_node_status_condition` metric.
* [CHANGE] Flatten created by label on `kube_pod_info` metric.
* [FEATURE] Add `kube_pod_start_time` metric.
* [FEATURE] Add PersistentVolumeClaim metrics.
* [FEATURE] Add StatefulSet metrics.
* [FEATURE] Add Job and CronJob metrics.
* [FEATURE] Add label metrics for deployments.
* [FEATURE] Add `kube_pod_owner` metrics.
* [ENHANCEMENT] Add `provider_id` label to `kube_node_info` metric.
* [BUGFIX] Fix various nil pointer panics.

## v0.5.0 / 2017-05-03

* [FEATURE] Add label metrics for Pods, Nodes and Services.
* [FEATURE] Expose number of ready Pods for DaemonSets.
* [FEATURE] Add LimitRange metrics.
* [FEATURE] Add ReplicationController metrics.
* [ENHANCEMENT] Add NodeMemoryPressure, NodeDiskPressure, NodeNetworkUnavailable condition metrics.
* [ENHANCEMENT] Add `created_by` label to `kube_pod_info` metric.

## v0.4.1 / 2017-02-10

* [BUGFIX] fix panic if max unavailable if rolling update is unset

## v0.4.0 / 2017-02-07

* [FEATURE] Add replicaset metrics
* [FEATURE] Add resourcequota metrics
* [FEATURE] Add daemonset metrics
* [FEATURE] Add resource limit and request metrics for pod containers
* [FEATURE] Add node name label to `kube_pod_info` metric
* [FEATURE] Add rolling update metrics for deployments
* [ENHANCEMENT] Allow disabling collectors
* [ENHANCEMENT] Improve in cluster vs non in cluster configuration

## v0.3.0 / 2016-10-18

* [FEATURE] Add pod metrics: `kube_pod_status_scheduled`, `kube_pod_container_requested_cpu_cores` and `kube_pod_container_requested_memory_bytes`
* [FEATURE] Add deployment metric `kube_deployment_metadata_generation`
* [FEATURE] Add node metric `kube_node_spec_unschedulable`
* [CHANGE] Rename `kube_node_status_allocateable_*` to `kube_node_status_allocatable_*`

## v0.2.0 / 2016-09-14

* [CHANGE] Prefix all metrics with `kube_`
* [CHANGE] Make metric collection synchronous
* [FEATURE] Add a number of node/pod/deployment metrics
