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
```txt
{{ file.Read "help.txt" }}
```
<!-- markdownlint-enable link-image-reference-definitions -->
<!-- markdownlint-enable blanks-around-fences -->

## Options config file

Options can also be set in the YAML file passed with `--config`. Values in the file override the command line flags. The file must contain a single YAML document, and a key that is not listed below is rejected. The `--config` flag and the logging flags can only be set on the command line.

```yaml
port: 9090
namespaces: [default, kube-system]
resources:
  deployments: {}
  pods: {}
labels_allow_list:
  pods: [app, tier]
server_read_timeout: 30s
```

| Key | Flag | Value |
| --- | --- | --- |
| `annotations_allow_list` | `--metric-annotations-allowlist` | map of resource to list of annotation keys |
| `apiserver` | `--apiserver` | string |
| `auth_filter` | `--auth-filter` | boolean |
| `auto-gomemlimit` | `--auto-gomemlimit` | boolean |
| `auto-gomemlimit-ratio` | `--auto-gomemlimit-ratio` | float |
| `continue_without_config` | `--continue-without-config` | boolean |
| `continue_without_custom_resource_state_config_file` | `--continue-without-custom-resource-state-config-file` | boolean |
| `custom_resource_config` | `--custom-resource-state-config` | string |
| `custom_resource_state_config_file` | `--custom-resource-state-config-file` | string |
| `custom_resources_only` | `--custom-resource-state-only` | boolean |
| `enable_gzip_encoding` | `--enable-gzip-encoding` | boolean |
| `help` | `--help` | boolean |
| `host` | `--host` | string |
| `kubeconfig` | `--kubeconfig` | string |
| `labels_allow_list` | `--metric-labels-allowlist` | map of resource to list of label keys |
| `metric_allowlist` | `--metric-allowlist` | map of metric name to `{}` |
| `metric_denylist` | `--metric-denylist` | map of metric name to `{}` |
| `metric_opt_in_list` | `--metric-opt-in-list` | map of metric name to `{}` |
| `namespace` | `--pod-namespace` | string |
| `namespaces` | `--namespaces` | list of namespaces |
| `namespaces_denylist` | `--namespaces-denylist` | list of namespaces |
| `node` | `--node` | string |
| `object_limit` | `--object-limit` | integer |
| `pod` | `--pod` | string |
| `port` | `--port` | integer |
| `resources` | `--resources` | map of resource to `{}` |
| `server_idle_timeout` | `--server-idle-timeout` | duration |
| `server_read_header_timeout` | `--server-read-header-timeout` | duration |
| `server_read_timeout` | `--server-read-timeout` | duration |
| `server_write_timeout` | `--server-write-timeout` | duration |
| `shard` | `--shard` | integer |
| `telemetry_host` | `--telemetry-host` | string |
| `telemetry_port` | `--telemetry-port` | integer |
| `tls_config` | `--tls-config` | string |
| `total_shards` | `--total-shards` | integer |
| `track_unscheduled_pods` | `--track-unscheduled-pods` | boolean |
| `use_api_server_cache` | `--use-apiserver-cache` | boolean |
