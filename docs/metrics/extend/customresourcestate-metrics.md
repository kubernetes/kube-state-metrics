# Custom Resource State Metrics

This section describes how to add metrics based on the state of a custom resource without writing a custom resource 
registry and running your own build of KSM.

## Configuration

A YAML configuration file described below is required to define your custom resources and the fields to turn into metrics.

Two flags can be used:

* `--custom-resource-state-config "inline yaml (see example)"` or
* `--custom-resource-state-config-file /path/to/config.yaml`

If both flags are provided, the inline configuration will take precedence.
When multiple entries for the same resource exist, kube-state-metrics will exit with an error.
This includes configuration which refers to a different API version.

In addition to specifying one of `--custom-resource-state-config*` flags, you should also add the custom resource *Kind*s in plural form to the list of exposed resources in the `--resources` flag. If you don't specify `--resources`, then all known custom resources configured in `--custom-resource-state-config*` and all available default kubernetes objects will be taken into account by kube-state-metrics.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-state-metrics
  namespace: kube-system
spec:
  template:
    spec:
      containers:
      - name: kube-state-metrics
        args:
          - --custom-resource-state-config
          # in YAML files, | allows a multi-line string to be passed as a flag value
          # see https://yaml-multiline.info
          -  |
              spec:
                resources:
                  - groupVersionKind:
                      group: myteam.io
                      version: "v1"
                      kind: Foo
                    metrics:
                      - name: active_count
                        help: "Count of active Foo"
                        each:
                          type: Gauge
                          ...
          - --resources=certificatesigningrequests,configmaps,cronjobs,daemonsets,deployments,endpoints,foos,horizontalpodautoscalers,ingresses,jobs,limitranges,mutatingwebhookconfigurations,namespaces,networkpolicies,nodes,persistentvolumeclaims,persistentvolumes,poddisruptionbudgets,pods,replicasets,replicationcontrollers,resourcequotas,secrets,services,statefulsets,storageclasses,validatingwebhookconfigurations,volumeattachments,verticalpodautoscalers
```

NOTE: The `group`, `version`, and `kind` common labels are reserved, and will be overwritten by the values from the `groupVersionKind` field.

### Examples

The examples in this section will use the following custom resource:

```yaml
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

#### Single Values

The config:

```yaml
kind: CustomResourceStateMetrics
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

Produces the metric:

```prometheus
kube_crd_uptime{group="myteam.io", kind="Foo", version="v1"} 43.21
```

#### Multiple Metrics/Kitchen Sink

```yaml
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

Produces the following metrics:

```prometheus
kube_crd_ready_count{group="myteam.io", kind="Foo", version="v1", active="1",custom_metric="yes",foo="bar",name="foo",bar="baz",qux="quxx",type="type-a"} 2
kube_crd_ready_count{group="myteam.io", kind="Foo", version="v1", active="3",custom_metric="yes",foo="bar",name="foo",bar="baz",qux="quxx",type="type-b"} 4
```

### Metric types

The configuration supports three kind of metrics from the [OpenMetrics specification](https://github.com/OpenObservability/OpenMetrics/blob/main/specification/OpenMetrics.md).

The metric type is specified by the `type` field and its specific configuration at the types specific struct.

#### Gauge

> Gauges are current measurements, such as bytes of memory currently used or the number of items in a queue. For gauges the absolute value is what is of interest to a user. [[0]](https://github.com/OpenObservability/OpenMetrics/blob/main/specification/OpenMetrics.md#gauge)

Example:

```yaml
kind: CustomResourceStateMetrics
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

Produces the metric:

```prometheus
kube_crd_uptime{group="myteam.io", kind="Foo", version="v1"} 43.21
```

#### StateSet

> StateSets represent a series of related boolean values, also called a bitset. If ENUMs need to be encoded this MAY be done via StateSet. [[1]](https://github.com/OpenObservability/OpenMetrics/blob/main/specification/OpenMetrics.md#stateset)

```yaml
kind: CustomResourceStateMetrics
spec:
  resources:
    - groupVersionKind:
        group: myteam.io
        kind: "Foo"
        version: "v1"
      metrics:
        - name: "status_phase"
          help: "Foo status_phase"
          each:
            type: StateSet
            stateSet:
              labelName: phase
              path: [status, phase]
              list: [Pending, Bar, Baz]
```

Metrics of type `StateSet` will generate a metric for each value defined in `list` for each resource.
The value will be 1, if the value matches the one in list.

Produces the metric:

```prometheus
kube_crd_status_phase{group="myteam.io", kind="Foo", version="v1", phase="Pending"} 1
kube_crd_status_phase{group="myteam.io", kind="Foo", version="v1", phase="Bar"} 0
kube_crd_status_phase{group="myteam.io", kind="Foo", version="v1", phase="Baz"} 0
```

#### Info

> Info metrics are used to expose textual information which SHOULD NOT change during process lifetime. Common examples are an application's version, revision control commit, and the version of a compiler. [[2]](https://github.com/OpenObservability/OpenMetrics/blob/main/specification/OpenMetrics.md#info)

Metrics of type `Info` will always have a value of 1.

```yaml
kind: CustomResourceStateMetrics
spec:
  resources:
    - groupVersionKind:
        group: myteam.io
        kind: "Foo"
        version: "v1"
      metrics:
        - name: "version"
          help: "Foo version"
          each:
            type: Info
            info:
              labelsFromPath:
                version: [spec, version]
```

Produces the metric:

```prometheus
kube_crd_version{group="myteam.io", kind="Foo", version="v1", version="v1.2.3"} 1
```

### Naming

The default metric names are prefixed to avoid collisions with other metrics.
By default, a metric prefix of `kube_` concatenated with your custom resource's group+version+kind is used.
You can override this behavior with the `metricNamePrefix` field.

```yaml
kind: CustomResourceStateMetrics
spec:
  resources:
    - groupVersionKind: ...
      metricNamePrefix: myteam_foos
      metrics:
        - name: uptime
          ...
```

Produces:
```prometheus
myteam_foos_uptime{group="myteam.io", kind="Foo", version="v1"} 43.21
```

To omit namespace and/or subsystem altogether, set them to the empty string:

```yaml
kind: CustomResourceStateMetrics
spec:
  resources:
    - groupVersionKind: ...
      metricNamePrefix: ""
      metrics:
        - name: uptime
          ...
```

Produces:
```prometheus
uptime{group="myteam.io", kind="Foo", version="v1"} 43.21
```

### Logging

If a metric path is registered but not found on a custom resource, an error will be logged. For some resources,
this may produce a lot of noise. The error log [verbosity][vlog] for a metric or resource can be set with `errorLogV` on
the resource or metric:

```yaml
kind: CustomResourceStateMetrics
spec:
  resources:
    - groupVersionKind: ...
      errorLogV: 0  # 0 = default for errors
      metrics:
        - name: uptime
          errorLogV: 10  # only log at high verbosity
```

[vlog]: https://github.com/go-logr/logr#why-v-levels

### Path Syntax

Paths are specified as a list of strings. Each string is a path segment, resolved dynamically against the data of the custom resource.
If any part of a path is missing, the result is nil.

Examples:

```yaml
# simple path lookup
[spec, replicas]                         # spec.replicas == 1

# indexing an array
[spec, order, "0", value]                # spec.order[0].value = true

# finding an element in a list by key=value  
[status, conditions, "[name=a]", value]  # status.conditions[0].value = 45

# if the value to be matched is a number or boolean, the value is compared as a number or boolean  
[status, conditions, "[value=66]", name]  # status.conditions[1].name = "b"
```
