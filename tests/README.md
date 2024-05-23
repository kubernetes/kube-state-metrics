# End2end testsuite

This folder contains simple e2e tests.
When launched, it spins up a kubernetes cluster using [kind](https://kind.sigs.k8s.io/), creates several kubernetes resources and launches a kube-state-metrics deployment.
Then, it runs verification tests: check metrics' presence, lint metrics, check service health, etc.

The test suite is run automatically using Github Actions.

## Running locally

To run the e2e tests locally run the following command:

```bash
./tests/e2e.sh
```
