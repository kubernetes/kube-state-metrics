# End2end testsuite

This folder contains simple e2e tests.
When launched it spins up a kubernetes cluster using minikube, creates several kubernetes resources and launches a kube-state-metrics deployment.
Then, it runs verification tests: check metrics' presence, lint metrics, check service health, etc.

The test suite is run automatically using Travis.

## Running locally

To run the e2e tests locally run the following command:

```bash
export MINIKUBE_DRIVER=virtualbox # choose minikube's driver of your choice
./tests/e2e.sh
```
