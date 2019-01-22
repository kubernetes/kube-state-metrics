# End2end testsuite

This folder contains simple e2e tests.
When launched it spins up a kubernetes cluster using minikube, creates several kubernetes resources and launches a kube-state-metrics deployment.
Then, it downloads kube-state-metrics' metrics and examines validity using `promtool` tool.

The testsuite is run automatically using Travis.

## Running locally

To run the e2e tests locally run the following command:

```bash
export MINIKUBE_DRIVER=virtualbox # choose minikube's driver of your choice
./tests/e2e.sh
```
