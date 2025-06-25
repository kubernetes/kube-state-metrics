# Documentation

This documentation is intended to be a complete reflection of the current state of the exposed metrics of kube-state-metrics.

Any contribution to improving this documentation or adding sample usages will be appreciated.

## Table of Contents

* [Metrics Stages](#metrics-stages)
* [Exposed Metrics](#exposed-metrics)
* [Join Metrics](#join-metrics)
* [CLI arguments](#cli-arguments)

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

* [CertificateSigningRequest Metrics](metrics/auth/certificatesigningrequest-metrics.md)
* [ConfigMap Metrics](metrics/storage/configmap-metrics.md)
* [CronJob Metrics](metrics/workload/cronjob-metrics.md)
* [DaemonSet Metrics](metrics/workload/daemonset-metrics.md)
* [Deployment Metrics](metrics/workload/deployment-metrics.md)
* [Endpoint Metrics](metrics/service/endpoint-metrics.md)
* [Horizontal Pod Autoscaler Metrics](metrics/workload/horizontalpodautoscaler-metrics.md)
* [Ingress Metrics](metrics/service/ingress-metrics.md)
* [Job Metrics](metrics/workload/job-metrics.md)
* [Lease Metrics](metrics/cluster/lease-metrics.md)
* [LimitRange Metrics](metrics/policy/limitrange-metrics.md)
* [MutatingWebhookConfiguration Metrics](metrics/extend/mutatingwebhookconfiguration-metrics.md)
* [Namespace Metrics](metrics/cluster/namespace-metrics.md)
* [NetworkPolicy Metrics](metrics/policy/networkpolicy-metrics.md)
* [Node Metrics](metrics/cluster/node-metrics.md)
* [PersistentVolume Metrics](metrics/storage/persistentvolume-metrics.md)
* [PersistentVolumeClaim Metrics](metrics/storage/persistentvolumeclaim-metrics.md)
* [Pod Disruption Budget Metrics](metrics/policy/poddisruptionbudget-metrics.md)
* [Pod Metrics](metrics/workload/pod-metrics.md)
* [ReplicaSet Metrics](metrics/workload/replicaset-metrics.md)
* [ReplicationController Metrics](metrics/workload/replicationcontroller-metrics.md)
* [ResourceQuota Metrics](metrics/policy/resourcequota-metrics.md)
* [Secret Metrics](metrics/storage/secret-metrics.md)
* [Service Metrics](metrics/service/service-metrics.md)
* [StatefulSet Metrics](metrics/workload/statefulset-metrics.md)
* [StorageClass Metrics](metrics/storage/storageclass-metrics.md)
* [ValidatingWebhookConfiguration Metrics](metrics/extend/validatingwebhookconfiguration-metrics.md)
* [VolumeAttachment Metrics](metrics/storage/volumeattachment-metrics.md)

### Optional Resources

* [ClusterRole Metrics](metrics/cluster/clusterrole-metrics.md)
* [ClusterRoleBinding Metrics](metrics/cluster/clusterrolebinding-metrics.md)
* [EndpointSlice Metrics](metrics/service/endpointslice-metrics.md)
* [IngressClass Metrics](metrics/service/ingressclass-metrics.md)
* [Role Metrics](metrics/auth/role-metrics.md)
* [RoleBinding Metrics](metrics/auth/rolebinding-metrics.md)
* [ServiceAccount Metrics](metrics/auth/serviceaccount-metrics.md)

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

See [Custom Resource State Metrics](metrics/extend/customresourcestate-metrics.md) for experimental support for custom resources.

## CLI Arguments

Additionally, options for `kube-state-metrics` can be passed when executing as a CLI, or in a kubernetes / openshift environment. More information can be found here: [CLI Arguments](developer/cli-arguments.md)

## Protecting /metrics endpoints

Kube-State-Metrics' metrics can contain sensitive information about the state of the cluster, which you as an operator might want to additionally protect from unauthorized access.
In order to achieve this, you need to enable the `--auth-filter` flag on kube-state-metrics.
With this, kube-state-metrics will only accept authenticated and authorized requests to the /metrics endpoints.
Kube-state-metrics uses Kubernetes' RBAC mechanisms for this, so this means that every scrape will trigger a request against the API Server for TokenReview and SubjectAccessReview.
The clients scraping the endpoint, need to use a token which can be provided by a ServiceAccount that can be set up the following way:

A ClusterRole providing access like this:

```
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: metrics-reader
rules:
- nonResourceURLs:
  - "/metrics"
  verbs:
  - get
```

and a matching ClusterRoleBinding

```
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: metrics-reader-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: metrics-reader
subjects:
- kind: ServiceAccount
  name: YOUR_SERVICE_ACCOUNT
  namespace: NAMESPACE_OF_THE_SERVICE_ACCOUNT
```

Your client can then use either this ServiceAccount to gather metrics or you can create a token, that can be used to fetch data like this:

```
TOKEN=$(kubectl create token YOUR_SERVICE_ACCOUNT -n NAMESPACE_OF_THE_SERVICE_ACCOUNT)
curl -H "Authorization: Bearer $TOKEN" KUBE_STATE_METRICS_URL:8080/metrics
```
