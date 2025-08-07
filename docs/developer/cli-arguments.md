# Command line arguments

kube-state-metrics can be configured through command line arguments.

Those arguments can be passed during startup when running locally:

`kube-state-metrics --telemetry-port=8081 --kubeconfig=<KUBE-CONFIG> --apiserver=<APISERVER> ...`

Or configured in the `args` section of your deployment configuration in a Kubernetes / Openshift context:

```yaml
spec:
  template:
    spec:
      containers:
        - args:
          - '--telemetry-port=8081'
          - '--kubeconfig=<KUBE-CONFIG>'
          - '--apiserver=<APISERVER>'
```

## Available options

<!-- markdownlint-disable blanks-around-fences -->
<!-- markdownlint-disable link-image-reference-definitions -->
[embedmd]:# (../../help.txt)
```txt
$ kube-state-metrics -h
kube-state-metrics is a simple service that listens to the Kubernetes API server and generates metrics about the state of the objects.

Usage:
  kube-state-metrics [flags]
  kube-state-metrics [command]

Available Commands:
  completion  Generate completion script for kube-state-metrics.
  help        Help about any command
  version     Print version information.

Flags:
      --add_dir_header                             If true, adds the file directory to the header of the log messages
      --alsologtostderr                            log to standard error as well as files (no effect when -logtostderr=true)
      --apiserver string                           The URL of the apiserver to use as a master
      --auth-filter                                If true, requires authentication and authorization through Kubernetes API to access metrics endpoints
      --auto-gomemlimit                            Automatically set GOMEMLIMIT to match container or system memory limit. (experimental)
      --auto-gomemlimit-ratio float                The ratio of reserved GOMEMLIMIT memory to the detected maximum container or system memory. (experimental) (default 0.9)
      --config string                              Path to the kube-state-metrics options config YAML file. If this flag is set, the flags defined in the file override the command line flags.
      --custom-resource-state-config string        Inline Custom Resource State Metrics config YAML (experimental)
      --custom-resource-state-config-file string   Path to a Custom Resource State Metrics config file (experimental)
      --custom-resource-state-only                 Only provide Custom Resource State metrics (experimental)
      --enable-gzip-encoding                       Gzip responses when requested by clients via 'Accept-Encoding: gzip' header.
  -h, --help                                       Print Help text
      --host string                                Host to expose metrics on. (default "::")
      --kubeconfig string                          Absolute path to the kubeconfig file
      --log_backtrace_at traceLocation             when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                             If non-empty, write log files in this directory (no effect when -logtostderr=true)
      --log_file string                            If non-empty, use this log file (no effect when -logtostderr=true)
      --log_file_max_size uint                     Defines the maximum size a log file can grow to (no effect when -logtostderr=true). Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                                log to standard error instead of files (default true)
      --metric-allowlist string                    Comma-separated list of metrics to be exposed. This list comprises of exact metric names and/or *ECMAScript-based* regex patterns. The allowlist and denylist are mutually exclusive.
      --metric-annotations-allowlist string        Comma-separated list of Kubernetes annotations keys that will be used in the resource' labels metric. By default the annotations metrics are not exposed. To include them, provide a list of resource names in their plural form and Kubernetes annotation keys you would like to allow for them (Example: '=namespaces=[kubernetes.io/team,...],pods=[kubernetes.io/team],...)'. A single '*' can be provided per resource instead to allow any annotations, but that has severe performance implications (Example: '=pods=[*]').
      --metric-denylist string                     Comma-separated list of metrics not to be enabled. This list comprises of exact metric names and/or *ECMAScript-based* regex patterns. The allowlist and denylist are mutually exclusive.
      --metric-labels-allowlist string             Comma-separated list of additional Kubernetes label keys that will be used in the resource' labels metric. By default the labels metrics are not exposed. To include them, provide a list of resource names in their plural form and Kubernetes label keys you would like to allow for them (Example: '=namespaces=[k8s-label-1,k8s-label-n,...],pods=[app],...)'. A single '*' can be provided per resource instead to allow any labels, but that has severe performance implications (Example: '=pods=[*]'). Additionally, an asterisk (*) can be provided as a key, which will resolve to all resources, i.e., assuming '--resources=deployments,pods', '=*=[*]' will resolve to '=deployments=[*],pods=[*]'.
      --metric-opt-in-list string                  Comma-separated list of metrics which are opt-in and not enabled by default. This is in addition to the metric allow- and denylists
      --namespaces string                          Comma-separated list of namespaces to be enabled. Defaults to ""
      --namespaces-denylist string                 Comma-separated list of namespaces not to be enabled. If namespaces and namespaces-denylist are both set, only namespaces that are excluded in namespaces-denylist will be used.
      --node string                                Name of the node that contains the kube-state-metrics pod. Most likely it should be passed via the downward API. This is used for daemonset sharding. Only available for resources (pod metrics) that support spec.nodeName fieldSelector. This is experimental.
      --object-limit int                           The total number of objects to list per resource from the API Server. (experimental)
      --one_output                                 If true, only write logs to their native severity level (vs also writing to each lower severity level; no effect when -logtostderr=true)
      --pod string                                 Name of the pod that contains the kube-state-metrics container. When set, it is expected that --pod and --pod-namespace are both set. Most likely this should be passed via the downward API. This is used for auto-detecting sharding. If set, this has preference over statically configured sharding. This is experimental, it may be removed without notice.
      --pod-namespace string                       Name of the namespace of the pod specified by --pod. When set, it is expected that --pod and --pod-namespace are both set. Most likely this should be passed via the downward API. This is used for auto-detecting sharding. If set, this has preference over statically configured sharding. This is experimental, it may be removed without notice.
      --port int                                   Port to expose metrics on. (default 8080)
      --resources string                           Comma-separated list of Resources to be enabled. Defaults to "certificatesigningrequests,configmaps,cronjobs,daemonsets,deployments,endpoints,horizontalpodautoscalers,ingresses,jobs,leases,limitranges,mutatingwebhookconfigurations,namespaces,networkpolicies,nodes,persistentvolumeclaims,persistentvolumes,poddisruptionbudgets,pods,replicasets,replicationcontrollers,resourcequotas,secrets,services,statefulsets,storageclasses,validatingwebhookconfigurations,volumeattachments"
      --server-idle-timeout duration               The maximum amount of time to wait for the next request when keep-alives are enabled. Align with the idletimeout of your scrape clients. (default 5m0s)
      --server-read-header-timeout duration        The maximum duration for reading the header of requests. (default 5s)
      --server-read-timeout duration               The maximum duration for reading the entire request, including the body. Align with the scrape interval or timeout of scraping clients.  (default 1m0s)
      --server-write-timeout duration              The maximum duration before timing out writes of the response. Align with the scrape interval or timeout of scraping clients.. (default 1m0s)
      --shard int32                                The instances shard nominal (zero indexed) within the total number of shards. (default 0)
      --skip_headers                               If true, avoid header prefixes in the log messages
      --skip_log_headers                           If true, avoid headers when opening log files (no effect when -logtostderr=true)
      --stderrthreshold severity                   logs at or above this threshold go to stderr when writing to files and stderr (no effect when -logtostderr=true or -alsologtostderr=true) (default 2)
      --telemetry-host string                      Host to expose kube-state-metrics self metrics on. (default "::")
      --telemetry-port int                         Port to expose kube-state-metrics self metrics on. (default 8081)
      --tls-config string                          Path to the TLS configuration file
      --total-shards int                           The total number of shards. Sharding is disabled when total shards is set to 1. (default 1)
      --track-unscheduled-pods                     This configuration is used in conjunction with node configuration. When this configuration is true, node configuration is empty and the metric of unscheduled pods is fetched from the Kubernetes API Server. This is experimental.
      --use-apiserver-cache                        Sets resourceVersion=0 for ListWatch requests, using cached resources from the apiserver instead of an etcd quorum read.
  -v, --v Level                                    number for the log level verbosity
      --vmodule moduleSpec                         comma-separated list of pattern=N settings for file-filtered logging

Use "kube-state-metrics [command] --help" for more information about a command.
```
<!-- markdownlint-enable link-image-reference-definitions -->
<!-- markdownlint-enable blanks-around-fences -->
