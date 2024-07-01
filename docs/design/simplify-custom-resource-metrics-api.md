# Kube-State-Metrics - Simplify Custom Resource State Metrics API Proposal


---

Author: Catherine Fang (CatherineF-dev@), Han Kang (logicalhan@)

Date: 7. May 2023

Target release: v

---


## Glossary
- CR: custom resource, similar to an instance of a class
- CRD: custom resource definition, similar to a class

## Problem Statement

### Background
Current [Custom Resource State Metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/docs/customresourcestate-metrics.md#multiple-metricskitchen-sink) supports 8+ operations to extract metric value and labels from custom resource.
- each
- path
- labelFromKey
- labelsFromPath
- valueFrom
- commonLabels
- labelsFromPath
- *.
- ...

### Problem 
1. Custom resource metrics API isn't scalable and it's a little hard to maintain. 
  1.1 The maintaining work is O(8) and there are several bugs around these 8 operations. For example, Crash on nonexistent metric paths in custom resources (#1992).
  1.2 More additional operations might be added to satisfy other needs.
2. Custom resource metrics API with existing 8 operations is not complete, which means some cases aren't covered. For example, it doesn't support querying number of CRs under one CRD.

## Goal

- Simplify 8 operations into one operation to reduce maintaining work.
- A complete API, so that can support more cases. For example, querying number of CRs under one CRD.

## Proposal

Use common expression language ([cel](https://kubernetes.io/docs/reference/using-api/cel/)) to extract fields from custom resource as metric labels or metric value.


```
kind: CustomResourceStateMetricsV2
spec:
  resources:
    - groupVersionKind:
        group: myteam.io
        kind: "Foo"
        version: "v1"
      mode: for_loop # or merged
      metrics:
        - name: "ready_count"
          help: "Number Foo Bars ready"
          values:  x.cel_selection_1 // [2, 4]
          labels:
          - x.cel_selection_2 // [{"cr_name": "bar"}], it will be copied into 2 same items
          - x.cel_selection_3 // [{active": 1}, {"active": 3}]
          - x.cel_selection_4 // [{"name": "type-a"}, {"name": "type-b"}]
```

Mode has two options:
- for_loop: it assigns x to each CR.
- merged: it assigns x to the merged CR of all CRs under one CRD. x := {"cr_name_foo": cr1, "cr_name_bar": cr2, ...}. It can count number of CRs under one CRD.

In this example (mode: for_loop), x is one CR under CRD (myteam.io/v1 Foo).
Assume it has N CRs under this CRD, it will generate these metrics:
- ready_count{cr_name=cr_1, active=1, name=type-a} = 2
- ready_count{cr_name=cr_1, active=3, name=type-b} = 4
- ...
- ready_count{cr_name=cr_n, active=2, name=type-c} = 5
- ready_count{cr_name=cr_n, active=3, name=type-d} = 6

### Mapping between existing operations and CEL

| Existing operation | CEL |
| :--- | :--- |
| path: [status, sub] \n labelFromKey: type | x.status.sub.map(y, {"name": y}) |
| path: [status, sub] \n valueFrom: [ready] |  x.status.sub.map(y, x.status.sub[y].ready) |
| commonLabels: \n custom_metric: "yes" | [{ "custom_metric":"yes" }] |
| labelsFromPath: "*": [metadata, labels] | [x.metadata.labels] |
| labelsFromPath \n foo: [metadata, labels, foo] | [{'name': x.metadata.name}] |

## Example
### CR
```
kind: Foo
apiVersion: myteam.io/vl
metadata:
    annotations:
        bar: baz
        qux: quxx
    labels:
        foo: bar
    name: foo
spec:
    version: v1.2.3
    order:
        - id: 1
          value: true
        - id: 3
          value: false
    replicas: 1
status:
    phase: Pending
    active:
        type-a: 1
        type-b: 3
    conditions:
        - name: a
          value: 45
        - name: b
          value: 66
    sub:
        type-a:
            active: 1
            ready: 2
        type-b:
            active: 3
            ready: 4
    uptime: 43.21
```

### Existing API - CustomResourceStateMetrics
```
kind: CustomResourceStateMetrics
spec:
  resources:
    - groupVersionKind:
        group: myteam.io
        kind: "Foo"
        version: "v1"
      # labels can be added to all metrics from a resource
      commonLabels:
        crd_type: "foo"
      labelsFromPath:
        name: [metadata, name]
      metrics:
        - name: "ready_count"
          help: "Number Foo Bars ready"
          each:
            type: Gauge
            gauge:
              # targeting an object or array will produce a metric for each element
              # labelsFromPath and value are relative to this path
              path: [status, sub]

              # if path targets an object, the object key will be used as label value
              # This is not supported for StateSet type as all values will be truthy, which is redundant.
              labelFromKey: type
              # label values can be resolved specific to this path
              labelsFromPath:
                active: [active]
              # The actual field to use as metric value. Should be a number, boolean or RFC3339 timestamp string.
              valueFrom: [ready]
          commonLabels:
            custom_metric: "yes"
          labelsFromPath:
            # whole objects may be copied into labels by prefixing with "*"
            # *anything will be copied into labels, with the highest sorted * strings first
            "*": [metadata, labels]
            "**": [metadata, annotations]

            # or specific fields may be copied. these fields will always override values from *s
            name: [metadata, name]
            foo: [metadata, labels, foo]
```

### Proposed API - CustomResourceStateMetricsV2
```
kind: CustomResourceStateMetricsV2
spec:
  resources:
    - groupVersionKind:
        group: myteam.io
        kind: "Foo"
        version: "v1"
      mode: for_loop # or merged
      metrics:
        - name: "ready_count"
          help: "Number Foo Bars ready"
          values: x.status.sub.map(y, x.status.sub[y].ready) # a cel query. jq '[.status.sub[].ready]', valueFrom: [ready] // [2,4]
          labels:
          - x.status.sub.map(y, {"name": y}) # a cel query. jq '[ .status.sub | keys | .[] | {name: .}]', labelFromKey: type // [{"name": "type-a"}, {"name": "type-b"}]
          - [{ "custom_metric":"yes" }] # a cel query. jq '[{ custom_metric:"yes" }]', custom_metric: "yes" // [{custom_metric="yes"}]
          - [x.metadata.labels] # a cel query. jq '[.metadata.labels]', "*": [metadata, labels] // [{"foo": "bar"}]
          - [x.metadata.annotations] # a cel query. jq '[.metadata.annotations]', "**": [metadata, annotations] // [{"bar": "baz","qux": "quxx"}]
          - [{'name': x.metadata.name}] # a cel query. jq '[{ name: .metadata.name }]', name: [metadata, name] // [{"name": "foo"}]
          - [{'foo': x.metadata.labels.foo}] # a cel query. jq '[{ foo: .metadata.labels.foo }]' # foo: [metadata, labels, foo] // [{foo": "bar"}]
          - [x.status.sub.map(y, {"active": x.status.sub[y].active})] # a cel query. jq '[.status.sub[].active | {active: .}]',labelsFromPath:  active: [active] // [{active": 1}, {"active": 3}]
```

