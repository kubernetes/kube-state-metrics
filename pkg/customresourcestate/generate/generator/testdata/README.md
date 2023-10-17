# Testdata for generator tests

The files in this directory are used for testing the `kube-state-metrics generate` command and to provide an example.

## foo-config.yaml

This file is used in the test at [generate_integration_test.go](../generate_integration_test.go) to verify that the resulting configuration does not change during changes in the codebase.

If there are intended changes this file needs to get regenerated to make the test succeed again.
This could be done via:

```sh
go run generate \
  ./pkg/customresourcestate/generate/generator/testdata/ \
  > ./pkg/customresourcestate/generate/generator/testdata/foo-config.yaml
```

Or by using the go:generate marker inside [foo_types.go](foo_types.go):

```sh
go generate ./pkg/customresourcestate/generate/generator/testdata/
```

## Example files: foo-cr-example.yaml and foo-cr-example-metrics.txt

There is also an example CR ([foo-cr-example.yaml](foo-cr-example.yaml)) and resulting example metrics ([foo-cr-example-metrics.txt](foo-cr-example-metrics.txt)).

The example metrics file got created by:

1. Generating a CustomResourceDefinition yaml by using [controller-gen](https://github.com/kubernetes-sigs/kubebuilder/blob/master/docs/book/src/reference/controller-gen.md):

    ```sh
    controller-gen crd paths=./pkg/customresourcestate/generate/generator/testdata/ output:dir=./pkg/customresourcestate/generate/generator/testdata/
    ```

2. Creating a cluster using [kind](https://kind.sigs.k8s.io/)
3. Applying the CRD and example CR to the cluster:

    ```sh
    kubectl apply -f /pkg/customresourcestate/generate/generator/testdata/bar.example.com_foos.yaml
    kubectl apply -f /pkg/customresourcestate/generate/generator/testdata/foo-cr-example.yaml
    ```

4. Running kube-state-metrics with the provided configuration file:

    ```sh
    go run ./ --kubeconfig $HOME/.kube/config --custom-resource-state-only \
    --custom-resource-state-config-file pkg/customresourcestate/generate/generator/testdata/foo-config.yaml
    ```

5. Querying the metrics endpoint in a second terminal:

    ```sh
    curl localhost:8080/metrics > ./pkg/customresourcestate/generate/generator/testdata/foo-cr-example-metrics.txt
    ```
