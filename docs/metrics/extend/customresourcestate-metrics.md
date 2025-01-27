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
              kind: CustomResourceStateMetrics
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
```

It's also possible to configure kube-state-metrics to run in a `custom-resource-mode` only. In addition to specifying one of `--custom-resource-state-config*` flags, you could set `--custom-resource-state-only` to `true`.
With this configuration only the known custom resources configured in `--custom-resource-state-config*` will be taken into account by kube-state-metrics.

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
              kind: CustomResourceStateMetrics
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
          - --custom-resource-state-only=true
```

NOTE: The `customresource_group`, `customresource_version`, and `customresource_kind` common labels are reserved, and will be overwritten by the values from the `groupVersionKind` field.

### RBAC-enabled Clusters

Please be aware that kube-state-metrics needs list and watch permissions granted to `customresourcedefinitions.apiextensions.k8s.io` as well as to the resources you want to gather metrics from.

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
    refs:
        - my_other_foo
        - foo_2
        - foo_with_extensions
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
kube_customresource_uptime{customresource_group="myteam.io", customresource_kind="Foo", customresource_version="v1"} 43.21
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
            # a prefix before the asterisk will be used as a label prefix
            "lorem_*": [metadata, annotations]
            "**": [metadata, annotations]
            
            # or specific fields may be copied. these fields will always override values from *s
            name: [metadata, name]
            foo: [metadata, labels, foo]
```

Produces the following metrics:

```prometheus
kube_customresource_ready_count{customresource_group="myteam.io", customresource_kind="Foo", 
customresource_version="v1", active="1",custom_metric="yes",foo="bar",name="foo",bar="baz",qux="quxx",type="type-a",
lorem_bar="baz",lorem_qux="quxx",} 2
kube_customresource_ready_count{customresource_group="myteam.io", customresource_kind="Foo", 
customresource_version="v1", active="3",custom_metric="yes",foo="bar",name="foo",bar="baz",qux="quxx",type="type-b",
lorem_bar="baz",lorem_qux="quxx",} 4
```

#### Non-map Arrays

```yaml
kind: CustomResourceStateMetrics
spec:
  resources:
    - groupVersionKind:
        group: myteam.io
        kind: "Foo"
        version: "v1"
      labelsFromPath:
        name: [metadata, name]
      metrics:
        - name: "ref_info"
          help: "Reference to other Foo"
          each:
            type: Info
            info:
              # targeting an array will produce a metric for each element
              # labelsFromPath and value are relative to this path
              path: [spec, refs]

              # if path targets a list of values (e.g. strings or numbers, not objects or maps), individual values can
              # referenced by a label using this syntax
              labelsFromPath:
                ref: []
```

Produces the following metrics:

```prometheus
kube_customresource_ref_info{customresource_group="myteam.io", customresource_kind="Foo", customresource_version="v1", name="foo",ref="my_other_foo"} 1
kube_customresource_ref_info{customresource_group="myteam.io", customresource_kind="Foo", customresource_version="v1", name="foo",ref="foo_2"} 1
kube_customresource_ref_info{customresource_group="myteam.io", customresource_kind="Foo", customresource_version="v1", name="foo",ref="foo_with_extensions"} 1
```

#### Same Metrics with Different Labels

```yaml
  recommendation:
    containerRecommendations:
    - containerName: consumer
      lowerBound:
        cpu: 100m
        memory: 262144k
```

For example in VPA we have above attributes and we want to have a same metrics for both CPU and Memory, you can use below config:

```
kind: CustomResourceStateMetrics
spec:
  resources:
    - groupVersionKind:
        group: autoscaling.k8s.io
        kind: "VerticalPodAutoscaler"
        version: "v1"
      labelsFromPath:
        verticalpodautoscaler: [metadata, name]
        namespace: [metadata, namespace]
        target_api_version: [apiVersion]
        target_kind: [spec, targetRef, kind]
        target_name: [spec, targetRef, name]
      metrics:
        # for memory
        - name: "verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound"
          help: "Minimum memory resources the container can use before the VerticalPodAutoscaler updater evicts it."
          commonLabels:
            unit: "byte"
            resource: "memory"
          each:
            type: Gauge
            gauge:
              path: [status, recommendation, containerRecommendations]
              labelsFromPath:
                container: [containerName]
              valueFrom: [lowerBound, memory]
        # for CPU
        - name: "verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound"
          help: "Minimum cpu resources the container can use before the VerticalPodAutoscaler updater evicts it."
          commonLabels:
            unit: "core"
            resource: "cpu"
          each:
            type: Gauge
            gauge:
              path: [status, recommendation, containerRecommendations]
              labelsFromPath:
                container: [containerName]
              valueFrom: [lowerBound, cpu]
```

Produces the following metrics:

```prometheus
# HELP kube_customresource_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound Minimum memory resources the container can use before the VerticalPodAutoscaler updater evicts it.
# TYPE kube_customresource_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound gauge
kube_customresource_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound{container="consumer",customresource_group="autoscaling.k8s.io",customresource_kind="VerticalPodAutoscaler",customresource_version="v1",namespace="namespace-example",resource="memory",target_api_version="apps/v1",target_kind="Deployment",target_name="target-name-example",unit="byte",verticalpodautoscaler="vpa-example"} 123456
# HELP kube_customresource_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound Minimum cpu resources the container can use before the VerticalPodAutoscaler updater evicts it.
# TYPE kube_customresource_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound gauge
kube_customresource_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound{container="consumer",customresource_group="autoscaling.k8s.io",customresource_kind="VerticalPodAutoscaler",customresource_version="v1",namespace="namespace-example",resource="cpu",target_api_version="apps/v1",target_kind="Deployment",target_name="target-name-example",unit="core",verticalpodautoscaler="vpa-example"} 0.1
```

#### VerticalPodAutoscaler

In v2.9.0 the `vericalpodautoscalers` resource was removed from the list of default resources. In order to generate metrics for `verticalpodautoscalers`, you can use the following Custom Resource State config:

```yaml
# Using --resource=verticalpodautoscalers, we get the following output:
# HELP kube_verticalpodautoscaler_annotations Kubernetes annotations converted to Prometheus labels.
# TYPE kube_verticalpodautoscaler_annotations gauge
# kube_verticalpodautoscaler_annotations{namespace="default",verticalpodautoscaler="hamster-vpa",target_api_version="apps/v1",target_kind="Deployment",target_name="hamster"} 1
# A similar result can be achieved by specifying the following in --custom-resource-state-config:
kind: CustomResourceStateMetrics
spec:
  resources:
    - groupVersionKind:
        group: autoscaling.k8s.io
        kind: "VerticalPodAutoscaler"
        version: "v1"
      labelsFromPath:
        verticalpodautoscaler: [metadata, name]
        namespace: [metadata, namespace]
        target_api_version: [apiVersion]
        target_kind: [spec, targetRef, kind]
        target_name: [spec, targetRef, name]
      metrics:
        - name: "annotations"
          help: "Kubernetes annotations converted to Prometheus labels."
          each:
            type: Gauge
            gauge:
              path: [metadata, annotations]
# This will output the following metric:
# HELP kube_customresource_autoscaling_annotations Kubernetes annotations converted to Prometheus labels.
# TYPE kube_customresource_autoscaling_annotations gauge
# kube_customresource_autoscaling_annotations{customresource_group="autoscaling.k8s.io", customresource_kind="VerticalPodAutoscaler", customresource_version="v1", namespace="default",target_api_version="autoscaling.k8s.io/v1",target_kind="Deployment",target_name="hamster",verticalpodautoscaler="hamster-vpa"} 123
```

The above configuration was tested on [this](https://github.com/kubernetes/autoscaler/blob/master/vertical-pod-autoscaler/examples/hamster.yaml) VPA configuration, with an added annotation (`foo: 123`).

#### All VerticalPodAutoscaler Metrics

As an addition for the above configuration, here's the complete `CustomResourceStateMetrics` spec to re-enable all of the VPA metrics which are removed from the list of the default resources:

<details>

 <summary>VPA CustomResourceStateMetrics</summary>

```yaml
kind: CustomResourceStateMetrics
spec:
  resources:
    - groupVersionKind:
        group: autoscaling.k8s.io
        kind: "VerticalPodAutoscaler"
        version: "v1"
      labelsFromPath:
        namespace: [metadata, namespace]
        target_api_version: [spec, targetRef, apiVersion]
        target_kind: [spec, targetRef, kind]
        target_name: [spec, targetRef, name]
        verticalpodautoscaler: [metadata, name]
      metricNamePrefix: "kube"
      metrics:
        # kube_verticalpodautoscaler_annotations
        - name: "verticalpodautoscaler_annotations"
          help: "Kubernetes annotations converted to Prometheus labels."
          each:
            type: Info
            info:
              labelsFromPath:
                annotation_*: [metadata, annotations]
                name: [metadata, name]
        # kube_verticalpodautoscaler_labels
        - name: "verticalpodautoscaler_labels"
          help: "Kubernetes labels converted to Prometheus labels."
          each:
            type: Info
            info:
              labelsFromPath:
                label_*: [metadata, labels]
                name: [metadata, name]
        # kube_verticalpodautoscaler_spec_updatepolicy_updatemode
        - name: "verticalpodautoscaler_spec_updatepolicy_updatemode"
          help: "Update mode of the VerticalPodAutoscaler."
          each:
            type: StateSet
            stateSet:
              labelName: "update_mode"
              path: [spec, updatePolicy, updateMode]
              list: ["Auto", "Initial", "Off", "Recreate"]
        # Memory kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_minallowed_memory
        - name: "verticalpodautoscaler_spec_resourcepolicy_container_policies_minallowed_memory"
          help: "Minimum memory resources the VerticalPodAutoscaler can set for containers matching the name."
          commonLabels:
            unit: "byte"
            resource: "memory"
          each:
            type: Gauge
            gauge:
              path: [spec, resourcePolicy, containerPolicies]
              labelsFromPath:
                container: [containerName]
              valueFrom: [minAllowed, memory]
        # CPU kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_minallowed_cpu
        - name: "verticalpodautoscaler_spec_resourcepolicy_container_policies_minallowed_cpu"
          help: "Minimum cpu resources the VerticalPodAutoscaler can set for containers matching the name."
          commonLabels:
            unit: "core"
            resource: "cpu"
          each:
            type: Gauge
            gauge:
              path: [spec, resourcePolicy, containerPolicies]
              labelsFromPath:
                container: [containerName]
              valueFrom: [minAllowed, cpu]
        # Memory kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_maxallowed_memory
        - name: "verticalpodautoscaler_spec_resourcepolicy_container_policies_maxallowed_memory"
          help: "Maximum memory resources the VerticalPodAutoscaler can set for containers matching the name."
          commonLabels:
            unit: "byte"
            resource: "memory"
          each:
            type: Gauge
            gauge:
              path: [spec, resourcePolicy, containerPolicies]
              labelsFromPath:
                container: [containerName]
              valueFrom: [maxAllowed, memory]
        # CPU kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_maxallowed_cpu
        - name: "verticalpodautoscaler_spec_resourcepolicy_container_policies_maxallowed_cpu"
          help: "Maximum cpu resources the VerticalPodAutoscaler can set for containers matching the name."
          commonLabels:
            unit: "core"
            resource: "cpu"
          each:
            type: Gauge
            gauge:
              path: [spec, resourcePolicy, containerPolicies]
              labelsFromPath:
                container: [containerName]
              valueFrom: [maxAllowed, cpu]
        # Memory kube_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound_memory
        - name: "verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound_memory"
          help: "Minimum memory resources the container can use before the VerticalPodAutoscaler updater evicts it."
          commonLabels:
            unit: "byte"
            resource: "memory"
          each:
            type: Gauge
            gauge:
              path: [status, recommendation, containerRecommendations]
              labelsFromPath:
                container: [containerName]
              valueFrom: [lowerBound, memory]
        # CPU kube_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound_cpu
        - name: "verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound_cpu"
          help: "Minimum cpu resources the container can use before the VerticalPodAutoscaler updater evicts it."
          commonLabels:
            unit: "core"
            resource: "cpu"
          each:
            type: Gauge
            gauge:
              path: [status, recommendation, containerRecommendations]
              labelsFromPath:
                container: [containerName]
              valueFrom: [lowerBound, cpu]
        # Memory kube_verticalpodautoscaler_status_recommendation_containerrecommendations_upperbound_memory
        - name: "verticalpodautoscaler_status_recommendation_containerrecommendations_upperbound_memory"
          help: "Maximum memory resources the container can use before the VerticalPodAutoscaler updater evicts it."
          commonLabels:
            unit: "byte"
            resource: "memory"
          each:
            type: Gauge
            gauge:
              path: [status, recommendation, containerRecommendations]
              labelsFromPath:
                container: [containerName]
              valueFrom: [upperBound, memory]
        # CPU kube_verticalpodautoscaler_status_recommendation_containerrecommendations_upperbound_cpu
        - name: "verticalpodautoscaler_status_recommendation_containerrecommendations_upperbound_cpu"
          help: "Maximum cpu resources the container can use before the VerticalPodAutoscaler updater evicts it."
          commonLabels:
            unit: "core"
            resource: "cpu"
          each:
            type: Gauge
            gauge:
              path: [status, recommendation, containerRecommendations]
              labelsFromPath:
                container: [containerName]
              valueFrom: [upperBound, cpu]
        # Memory kube_verticalpodautoscaler_status_recommendation_containerrecommendations_target_memory
        - name: "verticalpodautoscaler_status_recommendation_containerrecommendations_target_memory"
          help: "Target memory resources the VerticalPodAutoscaler recommends for the container."
          commonLabels:
            unit: "byte"
            resource: "memory"
          each:
            type: Gauge
            gauge:
              path: [status, recommendation, containerRecommendations]
              labelsFromPath:
                container: [containerName]
              valueFrom: [target, memory]
        # CPU kube_verticalpodautoscaler_status_recommendation_containerrecommendations_target_cpu
        - name: "verticalpodautoscaler_status_recommendation_containerrecommendations_target_cpu"
          help: "Target cpu resources the VerticalPodAutoscaler recommends for the container."
          commonLabels:
            unit: "core"
            resource: "cpu"
          each:
            type: Gauge
            gauge:
              path: [status, recommendation, containerRecommendations]
              labelsFromPath:
                container: [containerName]
              valueFrom: [target, cpu]
        # Memory kube_verticalpodautoscaler_status_recommendation_containerrecommendations_uncappedtarget_memory
        - name: "verticalpodautoscaler_status_recommendation_containerrecommendations_uncappedtarget_memory"
          help: "Target memory resources the VerticalPodAutoscaler recommends for the container ignoring bounds."
          commonLabels:
            unit: "byte"
            resource: "memory"
          each:
            type: Gauge
            gauge:
              path: [status, recommendation, containerRecommendations]
              labelsFromPath:
                container: [containerName]
              valueFrom: [uncappedTarget, memory]
        # CPU kube_verticalpodautoscaler_status_recommendation_containerrecommendations_uncappedtarget_cpu
        - name: "verticalpodautoscaler_status_recommendation_containerrecommendations_uncappedtarget_cpu"
          help: "Target memory resources the VerticalPodAutoscaler recommends for the container ignoring bounds."
          commonLabels:
            unit: "core"
            resource: "cpu"
          each:
            type: Gauge
            gauge:
              path: [status, recommendation, containerRecommendations]
              labelsFromPath:
                container: [containerName]
              valueFrom: [uncappedTarget, cpu]
```

</details>

### Metric types

The configuration supports three kind of metrics from the [OpenMetrics specification](https://github.com/prometheus/OpenMetrics/blob/v1.0.0/specification/OpenMetrics.md).

The metric type is specified by the `type` field and its specific configuration at the types specific struct.

#### Gauge

> Gauges are current measurements, such as bytes of memory currently used or the number of items in a queue. For gauges the absolute value is what is of interest to a user. [[0]](https://github.com/prometheus/OpenMetrics/blob/v1.0.0/specification/OpenMetrics.md#gauge)

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
kube_customresource_uptime{customresource_group="myteam.io", customresource_kind="Foo", customresource_version="v1"} 43.21
```

##### Type conversion and special handling

Gauges produce values of type float64 but custom resources can be of all kinds of types.
Kube-state-metrics performs implicit type conversions for a lot of type.
Supported types are:

* (u)int32/64, int, float32 and byte are cast to float64
* `nil` is generally mapped to `0.0` if NilIsZero is `true`, otherwise it will throw an error
* for bool `true` is mapped to `1.0` and `false` is mapped to `0.0`
* for string the following logic applies
  * `"true"` and `"yes"` are mapped to `1.0`, `"false"`, `"no"` and `"unknown"` are mapped to `0.0` (all case-insensitive)
  * RFC3339 times are parsed to float timestamp  
  * Quantities like "250m" or "512Gi" are parsed to float using <https://github.com/kubernetes/apimachinery/blob/master/pkg/api/resource/quantity.go>
  * Percentages ending with a "%" are parsed to float
  * finally the string is parsed to float using <https://pkg.go.dev/strconv#ParseFloat> which should support all common number formats. If that fails an error is yielded

##### Example for status conditions on Kubernetes Controllers

```yaml
kind: CustomResourceStateMetrics
spec:
  resources:
  - groupVersionKind:
      group: myteam.io
      kind: "Foo"
      version: "v1"
    labelsFromPath:
      name:
      - metadata
      - name
      namespace:
      - metadata
      - namespace
    metrics:
    - name: "foo_status"
      help: "status condition "
      each:
        type: Gauge
        gauge:
          path: [status, conditions]
          labelsFromPath:
            type: ["type"]
          valueFrom: ["status"]
```

This will work for kubernetes controller CRs which expose status conditions according to the kubernetes api (<https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Condition>):

```yaml
status:
  conditions:
    - lastTransitionTime: "2019-10-22T16:29:31Z"
      status: "True"
      type: Ready
```

kube_customresource_foo_status{customresource_group="myteam.io", customresource_kind="Foo", customresource_version="v1", type="Ready"} 1.0

#### StateSet

> StateSets represent a series of related boolean values, also called a bitset. If ENUMs need to be encoded this MAY be done via StateSet. [[1]](https://github.com/prometheus/OpenMetrics/blob/v1.0.0/specification/OpenMetrics.md#stateset)

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
kube_customresource_status_phase{customresource_group="myteam.io", customresource_kind="Foo", customresource_version="v1", phase="Pending"} 1
kube_customresource_status_phase{customresource_group="myteam.io", customresource_kind="Foo", customresource_version="v1", phase="Bar"} 0
kube_customresource_status_phase{customresource_group="myteam.io", customresource_kind="Foo", customresource_version="v1", phase="Baz"} 0
```

#### Info

> Info metrics are used to expose textual information which SHOULD NOT change during process lifetime. Common examples are an application's version, revision control commit, and the version of a compiler. [[2]](https://github.com/prometheus/OpenMetrics/blob/v1.0.0/specification/OpenMetrics.md#info)

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
kube_customresource_version{customresource_group="myteam.io", customresource_kind="Foo", customresource_version="v1", version="v1.2.3"} 1
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
          # ...
```

Produces:

```prometheus
myteam_foos_uptime{customresource_group="myteam.io", customresource_kind="Foo", customresource_version="v1"} 43.21
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
          # ...
```

Produces:

```prometheus
uptime{customresource_group="myteam.io", customresource_kind="Foo", customresource_version="v1"} 43.21
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

# For generally matching against a field in an object schema, use the following syntax:
[metadata, "name=foo"] # if v, ok := metadata[name]; ok && v == "foo" { return v; } else { /* ignore */ }
```

### Wildcard matching of version and kind fields

The Custom Resource State (CRS hereon) configuration also allows you to monitor all versions and/or kinds that come under a group. It watches
the installed CRDs for this purpose. Taking the aforementioned `Foo` object as reference, the configuration below allows
you to monitor all objects under all versions *and* all kinds that come under the `myteam.io` group.

```yaml
kind: CustomResourceStateMetrics
spec:
  resources:
    - groupVersionKind:
        group: "myteam.io"
        version: "*" # Set to `v1 to monitor all kinds under `myteam.io/v1`. Wildcard matches all installed versions that come under this group.
        kind: "*" # Set to `Foo` to monitor all `Foo` objects under the `myteam.io` group (under all versions). Wildcard matches all installed kinds that come under this group (and version, if specified).
      metrics:
        - name: "myobject_info"
          help: "Foo Bar Baz"
          each:
            type: Info
            info:
              path: [metadata]
              labelsFromPath:
                object: [name]
                namespace: [namespace]
```

The configuration above produces these metrics.

```yaml
kube_customresource_myobject_info{customresource_group="myteam.io",customresource_kind="Foo",customresource_version="v1",namespace="ns",object="foo"} 1
kube_customresource_myobject_info{customresource_group="myteam.io",customresource_kind="Bar",customresource_version="v1",namespace="ns",object="bar"} 1
```

#### Note

* For cases where the GVKs defined in a CRD have multiple versions under a single group for the same kind, as expected, the wildcard value will resolve to *all* versions, but a query for any specific version will return all resources under all versions, in that versions' representation. This basically means that for two such versions `A` and `B`,  if a resource exists under `B`, it will reflect in the metrics generated for `A` as well, in addition to any resources of itself, and vice-versa. This logic is based on the [current `list`ing behavior](https://github.com/kubernetes/client-go/issues/1251#issuecomment-1544083071) of the client-go library.
* The introduction of this feature further discourages (and discontinues) the use of native objects in the CRS featureset, since these do not have an explicit CRD associated with them, and conflict with internal stores defined specifically for such native resources. Please consider opening an issue or raising a PR if you'd like to expand on the current metric labelsets for them. Also, any such configuration will be ignored, and no metrics will be generated for the same.
