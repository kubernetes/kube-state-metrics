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
