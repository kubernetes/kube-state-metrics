# Kube-State-Metrics - Timeseries best practices

---

Author: Manuel Rüger (<manuel@rueg.eu>)

Date: October 17th 2024

---

## Introduction

Kube-State-Metrics' goal is to provide insights into the state of Kubernetes objects by exposing them as metrics.
This document provides guidelines with the goal to create a good user experience when using these metrics.

Please be aware that this document is introduced in a later stage of the project and there might be metrics that do not follow these best practices.
Feel encouraged to report these metrics and provide a pull request to improve them.

## General best practices

We follow [Prometheus](https://prometheus.io/docs/practices/naming/) best practices in terms of naming and labeling.

## Best practices for kube-state-metrics

### Avoid pre-computation

kube-state-metrics should expose metrics on an individual object level and avoid any sort of pre-computation unless it is required due to for example high cardinality on objects.
We prefer not to add metrics that can be derived from existing raw metrics. For example, we would not want to expose a metric called `kube_pod_total` as it can be computed with `count(kube_pod_info)`.
This way kube-state-metrics allows the user to have full control on how they want to use the metrics and gives them flexibility to do specific computation.

### Static object properties

An object usually has a stable set of properties that do not change during its lifecycle in Kubernetes.
This includes properties like name, namespace, uid etc. that have a 1:1 relationship with the object.
It is a good practice to group those together into an `_info` metric.
If there is a 1:n relationship (e.g. a list of ports), it should be in a separate metric to avoid generating too many metrics.

### Dynamic object properties

An object can also have a dynamic set of properties, which are usually part of the status field.
These change during the lifecycle of the object.
For example a pod can be in different states like "Pending", "Running" etc.
These should be part of a "State Set" that includes labels that identify the object as well as the dynamic property.

### Linked properties

If an object contains a substructure that links multiple properties together (e.g. endpoint address and port), those should be reported in the same metric.

### Optional properties

Some Kubernetes objects have optional fields. In case there is an optional value, the label should still be exposed, ideally as an empty string.

### Timestamps

Timestamps like creation time or modification time should be exposed as a value. The metric should end with `_timestamp_seconds`. The date value is represented in [UNIX epoch seconds](https://en.wikipedia.org/wiki/Unix_time).

### Cardinality

Some object properties can cause cardinality issues if they can contain a lot of different values or are linked together with multiple properties that also can change a lot.
In this case it is better to limit the number of values that can be exposed within kube-state-metrics by allowing only a few of them and have a default for others.
If for example the Kubernetes object contains a status field that contains an error message that can change a lot, it would be better to have a boolean `error="true"` label in case there is an error.
If there are some error messages that are worth exposing, those could be exposed and for any other message, a default value could be provided.

## Stability

We follow the stability framework derived from Kubernetes, in which we expose experimental and stable metrics.
Experimental metrics are recently introduced or expose alpha/beta resources in the Kubernetes API.
They can change anytime and should be used with caution.
They can be promoted to a stable metric once the object stabilized in the Kubernetes API or they were part of two consecutive releases and haven't observed any changes in them.

Stable metrics are considered frozen with the exception of new labels being added.
A stable metric or a label on a stable metric can be deprecated in release Major.Minor and the earliest point it will be removed is the release Major.Minor+2.
