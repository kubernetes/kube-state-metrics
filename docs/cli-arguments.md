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

## Available options:

[embedmd]:# (../help.txt)
```txt
$ kube-state-metrics -h
Usage of ./kube-state-metrics:
      --alsologtostderr                             log to standard error as well as files
      --apiserver string                            The URL of the apiserver to use as a master
      --collectors string                           Comma-separated list of collectors to be enabled. Defaults to "certificatesigningrequests,configmaps,cronjobs,daemonsets,deployments,endpoints,horizontalpodautoscalers,ingresses,jobs,limitranges,namespaces,nodes,persistentvolumeclaims,persistentvolumes,poddisruptionbudgets,pods,replicasets,replicationcontrollers,resourcequotas,secrets,services,statefulsets,storageclasses"
      --disable-node-non-generic-resource-metrics   Disable node non generic resource request and limit metrics
      --disable-pod-non-generic-resource-metrics    Disable pod non generic resource request and limit metrics
      --enable-gzip-encoding                        Gzip responses when requested by clients via 'Accept-Encoding: gzip' header.
  -h, --help                                        Print Help text
      --host string                                 Host to expose metrics on. (default "0.0.0.0")
      --kubeconfig string                           Absolute path to the kubeconfig file
      --log_backtrace_at traceLocation              when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                              If non-empty, write log files in this directory
      --log_file string                             If non-empty, use this log file
      --log_file_max_size uint                      Defines the maximum size a log file can grow to. Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                                 log to standard error instead of files (default true)
      --metric-blacklist string                     Comma-separated list of metrics not to be enabled. This list comprises of exact metric names and/or regex patterns. The whitelist and blacklist are mutually exclusive.
      --metric-whitelist string                     Comma-separated list of metrics to be exposed. This list comprises of exact metric names and/or regex patterns. The whitelist and blacklist are mutually exclusive.
      --namespace string                            Comma-separated list of namespaces to be enabled. Defaults to ""
      --port int                                    Port to expose metrics on. (default 80)
      --skip_headers                                If true, avoid header prefixes in the log messages
      --skip_log_headers                            If true, avoid headers when opening log files
      --stderrthreshold severity                    logs at or above this threshold go to stderr (default 2)
      --telemetry-host string                       Host to expose kube-state-metrics self metrics on. (default "0.0.0.0")
      --telemetry-port int                          Port to expose kube-state-metrics self metrics on. (default 81)
  -v, --v Level                                     number for the log level verbosity
      --version                                     kube-state-metrics build version information
      --vmodule moduleSpec                          comma-separated list of pattern=N settings for file-filtered logging
```
