# Documentation

This documentation is intended to be a complete reflection of the current state of the exposed metrics of kube-state-metrics.

Any contribution to improving this documentation or adding sample usages will be appreciated.

## Table of Contents

- [Metrics Stages](#metrics-stages)
- [Exposed Metrics](#exposed-metrics)
- [Join Metrics](#join-metrics)
- [CLI arguments](#cli-arguments)

## Metrics Stages

Stages about metrics are grouped into three categories：

| Stage        | Description                                                                                                                |
| ------------ | -------------------------------------------------------------------------------------------------------------------------- |
| EXPERIMENTAL | Metrics which normally correspond to the Kubernetes API object alpha status or spec fields and can be changed at any time. |
| STABLE       | Metrics which should have very few backwards-incompatible changes outside of major version updates.                        |
| DEPRECATED   | Metrics which will be removed once the deprecation timeline is met.                                                        |

## Opt-in Metrics

As of v2.3.0, kube-state-metrics supports additional opt-in metrics via the CLI flag `--metric-opt-in-list`. See the metric documentation to identify which metrics need to be specified.

## Exposed Metrics

Per group of metrics there is one file for each metrics.
See each file for specific documentation about the exposed metrics:

### Default Resources

- [CertificateSigningRequest Metrics](certificatesigningrequest-metrics.md)
- [ConfigMap Metrics](configmap-metrics.md)
- [CronJob Metrics](cronjob-metrics.md)
- [DaemonSet Metrics](daemonset-metrics.md)
- [Deployment Metrics](deployment-metrics.md)
- [Endpoint Metrics](endpoint-metrics.md)
- [Horizontal Pod Autoscaler Metrics](horizontalpodautoscaler-metrics.md)
- [Ingress Metrics](ingress-metrics.md)
- [Job Metrics](job-metrics.md)
- [Lease Metrics](lease-metrics.md)
- [LimitRange Metrics](limitrange-metrics.md)
- [MutatingWebhookConfiguration Metrics](mutatingwebhookconfiguration-metrics.md)
- [Namespace Metrics](namespace-metrics.md)
- [NetworkPolicy Metrics](networkpolicy-metrics.md)
- [Node Metrics](node-metrics.md)
- [PersistentVolume Metrics](persistentvolume-metrics.md)
- [PersistentVolumeClaim Metrics](persistentvolumeclaim-metrics.md)
- [Pod Disruption Budget Metrics](poddisruptionbudget-metrics.md)
- [Pod Metrics](pod-metrics.md)
- [ReplicaSet Metrics](replicaset-metrics.md)
- [ReplicationController Metrics](replicationcontroller-metrics.md)
- [ResourceQuota Metrics](resourcequota-metrics.md)
- [Secret Metrics](secret-metrics.md)
- [Service Metrics](service-metrics.md)
- [StatefulSet Metrics](statefulset-metrics.md)
- [StorageClass Metrics](storageclass-metrics.md)
- [ValidatingWebhookConfiguration Metrics](validatingwebhookconfiguration-metrics.md)
- [VolumeAttachment Metrics](volumeattachment-metrics.md)

### Optional Resources

- [ClusterRole Metrics](clusterrole-metrics.md)
- [ClusterRoleBinding Metrics](clusterrolebinding-metrics.md)
- [EndpointSlice Metrics](endpointslice-metrics.md)
- [IngressClass Metrics](ingressclass-metrics.md)
- [Role Metrics](role-metrics.md)
- [RoleBinding Metrics](rolebinding-metrics.md)
- [ServiceAccount Metrics](serviceaccount-metrics.md)
- [VerticalPodAutoscaler Metrics](verticalpodautoscaler-metrics.md)

## Join Metrics

When an additional, not provided by default label is needed, a [Prometheus matching operator](https://prometheus.io/docs/prometheus/latest/querying/operators/#vector-matching)
can be used to extend single metrics output.

This example adds `label_release` to the set of default labels of the `kube_pod_status_ready` metric
and allows you select or group the metrics by Helm release label:

```
kube_pod_status_ready * on (namespace, pod) group_left(label_release) kube_pod_labels
```

Another useful example would be to query the memory usage of pods by its `phase`, such as `Running`:

```
sum(kube_pod_container_resource_requests{resource="memory"}) by (namespace, pod, node)
  * on (namespace, pod) group_left() (sum(kube_pod_status_phase{phase="Running"}) by (pod, namespace) == 1)
```

## Metrics from Custom Resources

See [Custom Resource State Metrics](customresourcestate-metrics.md) for experimental support for custom resources.

## CLI Arguments

Additionally, options for `kube-state-metrics` can be passed when executing as a CLI, or in a kubernetes / openshift environment. More information can be found here: [CLI Arguments](cli-arguments.md)
