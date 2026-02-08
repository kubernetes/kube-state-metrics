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
