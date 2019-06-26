# Documentation

This documentation is intended to be a complete reflection of the current state of the exposed metrics of kube-state-metrics.

Any contribution to improving this documentation or adding sample usages will be appreciated.

## Table of Contents

- [Metrics Stages](#metrics-stages)
- [Metrics Deprecation](#metrics-deprecation)
- [Exposed Metrics](#exposed-metrics)
- [Join Metrics](#join-metrics)

## Metrics Stages

Stages about metrics are grouped into three categories：

| Stage        | Description                                                                                                                |
| ------------ | -------------------------------------------------------------------------------------------------------------------------- |
| EXPERIMENTAL | Metrics which normally correspond to the Kubernetes API object alpha status or spec fields and can be changed at any time. |
| STABLE       | Metrics which should have very few backwards-incompatible changes outside of major version updates.                        |
| DEPRECATED   | Metrics which will be removed once the deprecation timeline is met.                                                        |

## Metrics Deprecation

- **The following non-generic resource metrics for pods are marked deprecated. They will be removed in kube-state-metrics v2.0.0.**
  `kube_pod_container_resource_requests` and `kube_pod_container_resource_limits` are the replacements with `resource` labels
  representing the resource name and `unit` labels representing the resource unit.
  - kube_pod_container_resource_requests_cpu_cores
  - kube_pod_container_resource_limits_cpu_cores
  - kube_pod_container_resource_requests_memory_bytes
  - kube_pod_container_resource_limits_memory_bytes
- **The following non-generic resource metrics for nodes are marked deprecated. They will be removed in kube-state-metrics v2.0.0.**
  `kube_node_status_capacity` and `kube_node_status_allocatable` are the replacements with `resource` labels
  representing the resource name and `unit` labels representing the resource unit.
  - kube_node_status_capacity_pods
  - kube_node_status_capacity_cpu_cores
  - kube_node_status_capacity_memory_bytes
  - kube_node_status_allocatable_pods
  - kube_node_status_allocatable_cpu_cores
  - kube_node_status_allocatable_memory_bytes

## Exposed Metrics

Per group of metrics there is one file for each metrics. See each file for specific documentation about the exposed metrics:

- [CronJob Metrics](cronjob-metrics.md)
- [DaemonSet Metrics](daemonset-metrics.md)
- [Deployment Metrics](deployment-metrics.md)
- [Job Metrics](job-metrics.md)
- [LimitRange Metrics](limitrange-metrics.md)
- [Node Metrics](node-metrics.md)
- [PersistentVolume Metrics](persistentvolume-metrics.md)
- [PersistentVolumeClaim Metrics](persistentvolumeclaim-metrics.md)
- [Pod Metrics](pod-metrics.md)
- [Pod Disruption Budget Metrics](poddisruptionbudget-metrics.md)
- [ReplicaSet Metrics](replicaset-metrics.md)
- [ReplicationController Metrics](replicationcontroller-metrics.md)
- [ResourceQuota Metrics](resourcequota-metrics.md)
- [Service Metrics](service-metrics.md)
- [StatefulSet Metrics](statefulset-metrics.md)
- [Namespace Metrics](namespace-metrics.md)
- [Horizontal Pod Autoscaler Metrics](horizontalpodautoscaler-metrics.md)
- [Endpoint Metrics](endpoint-metrics.md)
- [Secret Metrics](secret-metrics.md)
- [ConfigMap Metrics](configmap-metrics.md)
- [Ingress Metrics](ingress-metrics.md)
- [CertificateSigningRequest Metrics](certificatessigningrequest-metrics.md)

## Join Metrics

When an additional, not provided by default label is needed, a [Prometheus matching operator](https://prometheus.io/docs/prometheus/latest/querying/operators/#vector-matching)
can be used to extend single metrics output.

This example adds `label_release` to the set of default labels of the `kube_pod_status_ready` metric
and allows you select or group the metrics by helm release label:

```
kube_pod_status_ready * on (namespace, pod) group_left(label_release)  kube_pod_labels
```

Another useful example would be to query the memory usage of pods by its `phase`, such as `Running`:

```
sum(kube_pod_container_resource_requests_memory_bytes) by (namespace, pod, node)
  * on (pod) group_left()  (sum(kube_pod_status_phase{phase="Running"}) by (pod, namespace) == 1)
```
