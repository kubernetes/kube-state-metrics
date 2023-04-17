# Kube-State-Metrics - Custom Resource Metrics CRD Proposal


---

Authors: Catherine Fang (CatherineF-dev@), Christian Schlotter (chrischdi@)

Date: 17. Apr 2023

Target release: v

---


## Glossary

- kube-state-metrics: “Simple service that listens to the Kubernetes API server
  and generates metrics about the state of the objects”

## Problem Statement

Coupled custom resources metrics targets and kube-state-metrics deployment, which causes managing custom resource configuration not easy.

Currently, one of these flags is added into kube-state-metrics deployment.
* `--custom-resource-state-config "inline yaml (see example)"` or
* `--custom-resource-state-config-file /path/to/config.yaml`

For example, for a company with two teams (monitoring platform team and application team), application team needs to change kube-state-metrics deployment managed by monitoring platform team.


## Goal

1. Add new custom resource metrics without changing kube-state-metrics deployment
2. Isolate custom resource metrics when mulitple kube-state-metrics deployments are deployed. For a managed kubernetes solution (GKE/AKS/EKS), a kube-state-metrics might be deployed by cloud provider while another kube-state-metrics is deployed by customers. Cloud provider wants to monitor system CRs while customers want to monitor application CRs.

## Status Quo

Custom Resource store always [adds](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/builder.go#L186) new custom resource metrics. Deletion of custom resource metrics needs to be implemented.

## Proposal

Add a custom resource definition (CustomResourceMonitor) for custom resource metrics.
So that kube-state-metrics watched on these CRs and change custom metrics store dynamically.

Apart from existing two flags (`--custom-resource-state-config ` and `--custom-resource-state-config-file`), `--custom_resources_ksm_cr_watched` is added to watch all CustomResourceMonitor CRs.
If `--custom_resources_ksm_cr_watched` is set, `--custom-resource-state-config` and `--custom-resource-state-config-file` will be ignored.

All new flags are:
* `--custom_resources_ksm_cr_watched`: watch CustomResourceMonitor
* `--custom_resources_ksm_cr_watched_labels`: only watch CustomResourceMonitor with label=x
* `--custom_resources_ksm_cr_watched_namespaces`: only watch CustomResourceMonitor under namespaces=x


```
package v1alpha1

import (
	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
)

type CustomResourceMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	customresourcestate.Metrics `json:",inline"`
}
```

Kube-state-metrics listens on add, update and delete events of CustomResourceMonitor via Kubernetes
client-go reflectors. On these events kube-state-metrics lists all CustomResourceMonitor CRs and concatenate CRs into one config `customresourcestate.Metrics` which has the same format with configs of `--custom-resource-state-config`. This generated custom resource config updates CustomResourceStore by adding monitored custom resource stores and deleting unmonitored custom resource stores.


```yaml
# example cr
apiVersion: customresource.ksm.io/v1alpha1
kind: CustomResourceMonitor
metadata:
  name: nodepool
spec:
  resources:
    - groupVersionKind:
        group: addons.k8s.io
        kind: "FakedNodePools"
        version: "v1alpha1"
      metrics:
        - name: "nodepool_generation"
          help: "Nodepool generation"
          each:
            type: Gauge
            gauge:
              path: [metadata, generation]
```


```
               +---------------+ +---------------------+       +-----------------------+
               | CRM_informer  | | nodepool_reflector  |       | custom_resource_store |
               +---------------+ +---------------------+       +-----------------------+
---------------------\ |                    |                              |
| add/update/delete  |-|                    |                              |
| CustomResource     | |                    |                              |
| Monitor CR         | |                    |                              |
| (monitor-nodepool) | |                    |                              |
|--------------------| |                    |                              |
                       |                    |                              |
                       | ListAndAddCustomResourceMonitors()                |
                       |-------------------------------------------------->|
                       |                    |                              |
                       | DeleteOldCustomResourceMonitors()                 |
                       |-------------------------------------------------->|
                       |                    |                              |
                       |                    | Update(nodepool)             |
                       |                    |----------------------------->|
                       |                    |                              | ----------\
                       |                    |                              |-| Build() |
                       |                    |                              | |---------|
                       |                    |                              | ----------------------------\
                       |                    |                              |-| generateMetrics(nodepool) |
                       |                    |                              | |---------------------------|
                       |                    |                              |
```

<details>
 <summary>Code to reproduce diagram</summary>

Build via [text-diagram](http://weidagang.github.io/text-diagram/)

```
object CRM_informer nodepool_reflector custom_resource_store

note left of CRM_informer: add/update/delete \n CustomResource \n Monitor CR \n(monitor-nodepool)
CRM_informer -> custom_resource_store: ListAndAddCustomResourceMonitors()
CRM_informer -> custom_resource_store: DeleteOldCustomResourceMonitors()


nodepool_reflector -> custom_resource_store: Update(nodepool)

note right of custom_resource_store: Build()

note right of custom_resource_store: generateMetrics(nodepool)
```


</details>


## CUJ
* cloud-provider: watch CustomResourceMonitor CRs with label `monitoring.(gke|aks|eks).io=true` under system namespaces
* application platform: watch CustomResourceMonitor CRs with label `monitoring.app.io=true` under non-system namespaces
* monitoring platform team: watch CustomResourceMonitor CRs with label `monitoring.platform.io=true` under non-system namespaces