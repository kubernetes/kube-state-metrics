# Kube-State-Metrics - CustomResourceMonitor CRD Proposal


---

Authors: Catherine Fang (CatherineF-dev@), Christian Schlotter (chrischdi@)

Date: 26. Jun 2023

Target release: v

---

## Table of Contents
- [Glossary](#glossary)
- [Problem Statement](#problem-statement)
- [Goal](#goal)
- [Status Quo](#status-quo)
- [Proposal](#proposal)
   - [New flags](#new-flags)
   - [CustomResourceMonitor Definition](#customresourcemonitor-definition)
   - [Watch and Reconcile on CustomResourceMonitor CRs](#watch-and-reconcile-on-customresourcemonitor-crs)
- [CUJ](#cuj)



## Glossary

- kube-state-metrics: “Simple service that listens to the Kubernetes API server
  and generates metrics about the state of the objects”
- Custom Resource: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources
- [CustomResourceState](https://github.com/kubernetes/kube-state-metrics/blob/main/docs/customresourcestate-metrics.md) monitoring feature: existing feature which collects custom resource metrics


## Problem Statement

1. Using CustomResourceState monitoring feature is not user-friendly. Current ways on configuring CustomResourceState monitoring are:
* `--custom-resource-state-config "inline yaml (see example)"` or
* `--custom-resource-state-config-file /path/to/config.yaml`. Either mounted or configmap.

2. Current CustomResourceState monitoring feature doesn't support multiple configuration files.

For example, for a company with 10 teams, each team wants to collect Custom Resource metrics for their owned Custom Resources. 


## Goal

A better UX to collect custom resource metrics

## Proposal

Add a custom resource definition (CustomResourceMonitor) which contains customresourcestate.Metrics.

kube-state-metrics watched on CustomResourceMonitor CRs and concatenate these CRs into one config `customresourcestate.Metrics` which has the same content using `--custom-resource-state-config`. 

### New flags
Apart from existing two flags (`--custom-resource-state-config ` and `--custom-resource-state-config-file`), these three flags will be added: 
* `--custom_resource_monitor`: whether watch CustomResourceMonitor CRs or not.
* `--custom_resource_monitor_labels`: only watch CustomResourceMonitor with [labelSelectors](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#list-and-watch-filtering). For example, `environment=production,tier=frontend` means selecting CustomResourceMonitor CRs which have these two labels.  It's used to avoid double custom metrics collection when multiple kube-state-metrics are installed.
* `--custom_resource_monitor_namespaces`: only watch CustomResourceMonitor under namespaces=x.

If `--custom_resources_monitor_enabled` is set, `--custom-resource-state-config` and `--custom-resource-state-config-file` will be ignored.

### CustomResourceMonitor Definition

* GroupName: kubestatemetrics.io
   * Alternative kubestatemetrics.k8s.io: 1. *.k8s.io needs approval 2. ksm isn't inside k/k repo (need to double confirm)
   * Alternative ksm.io:  has been used by a company
* Version: v1alpha1
* Kind: CustomResourceMonitor

```go
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

```yaml
# Example CR
apiVersion: kubestatemetrics.io/v1alpha1
kind: CustomResourceMonitor
metadata:
  name: test-cr2
  namespace: kube-system
  generation: 1
spec:
  resources:
    - groupVersionKind:
        group: myteam.io
        kind: "Foo"
        version: "v1"
      metrics:
        - name: "uptime"
          help: "Foo uptime"
          each:
            type: Gauge
            gauge:
              path: [status, uptime]
```

### Watch and Reconcile on CustomResourceMonitor CRs

Kube-state-metrics listens on add, update and delete events of CustomResourceMonitor via Kubernetes
client-go reflectors. On these events kube-state-metrics lists all CustomResourceMonitor CRs and concatenate CRs into one config `customresourcestate.Metrics` which has the almost same content with `--custom-resource-state-config` config. 

This generated custom resource config updates CustomResourceStore by adding monitored custom resource stores and deleting unmonitored custom resource stores.


```yaml
# example cr
apiVersion: kubestatemetrics.io/v1alpha1
kind: CustomResourceMonitor
metadata:
  name: nodepool
spec:
  resources:
    - groupVersionKind:
        group: addons.k8s.io
        kind: "FakedNodePool"
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

Custom Resource store always [adds](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/builder.go#L186) new custom resource metrics. Deletion of custom resource metrics needs to be implemented.

### Alternatives
- Generate metrics configuration based on field annotations: https://github.com/kubernetes/kube-state-metrics/issues/1899
  - Limitation: need to have source code permission 

## Migrate from CustomResourceState
```
+ apiVersion: kubestatemetrics.io/v1alpha1
- kind: CustomResourceStateMetrics
+ kind: CustomResourceMonitor
+ metadata:
+  name: crm_nodepool
+  labels:
+    monitoring.backend.io: true
spec: # copy content from --custom-resource-state-config-file
```

## Critical User Journey (CUJ)
* cloud-provider: watch CustomResourceMonitor CRs with label `monitoring.(gke|aks|eks).io=true` under system namespaces
* application platform: watch CustomResourceMonitor CRs with label `monitoring.frontend.io=true` under non-system namespaces
* monitoring platform team: watch CustomResourceMonitor CRs with label `monitoring.platform.io=true` under non-system namespaces
