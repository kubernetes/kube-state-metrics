# Custom Resource State Metrics

This section describes how to add metrics based on the state of a custom resource without writing a custom resource 
registry and running your own build of KSM.

## Configuration

A YAML configuration file described below is required to define your custom resources and the fields to turn into metrics.

Two flags can be used:

 * `--custom-resource-state-config "inline yaml (see example)"` or
 * `--custom-resource-state-config-file /path/to/config.yaml`

If both flags are provided, the inline configuration will take precedence.

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
                        ...
```

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
    order:
        - id: 1
          value: true
        - id: 3
          value: false
    replicas: 1
status:
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
            path: [status, uptime]
```

Produces the metric:

```prometheus
kube_myteam_io_v1_Foo_uptime 43.21
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
            # targeting an object or array will produce a metric for each element
            # labelsFromPath and value are relative to this path
            path: [status, sub]
            
            # if path targets an object, the object key will be used as label value
            labelFromKey: type
            # label values can be resolved specific to this path 
            labelsFromPath:
              active: [active]
            # The actual field to use as metric value. Should be a number.
            value: [ready]
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
kube_myteam_io_v1_Foo_active_count{active="1",custom_metric="yes",foo="bar",name="foo",bar="baz",qux="quxx",type="type-a"} 1
kube_myteam_io_v1_Foo_active_count{active="3",custom_metric="yes",foo="bar",name="foo",bar="baz",qux="quxx",type="type-b"} 3
```

### Naming

The default metric names are prefixed to avoid collisions with other metrics.
By default, a namespace of `kube` and a subsystem based on your custom resource's group+version+kind is used.
You can override these with the namespace and subsystem fields.

```yaml
kind: CustomResourceStateMetrics
spec:
  resources:
    - groupVersionKind: ...
      namespace: myteam
      subsystem: foos
      metrics:
        - name: uptime
          ...
```

Produces:
```prometheus
myteam_foos_uptime 43.21
```

To omit namespace and/or subsystem altogether, set them to `_`.

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
