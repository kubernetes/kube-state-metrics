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
