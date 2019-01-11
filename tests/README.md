# End2end testsuite

This folder contains simple e2e tests.
When launched it spins up a kubernetes cluster using minikube, creates several kubernetes resources and launches a kube-state-metrics deployment.
Then, it downloads kube-state-metrics' metrics and examines validity using `promtool` tool.

The testsuite is run automatically using Travis.

## Running locally

In case you need to run e2e test manually on your local machine, you can configure the `e2e.sh` script by few environment variables.

```bash
export E2E_SETUP_MINIKUBE=        # set to empty string if you have already your own minikube binary, prevents from downloading one
export E2E_SETUP_KUBECTL=         # set to empty string if you have already your own kubectl binary, prevents from downloading one
export MINIKUBE_DRIVER=virtualbox # choose minikube's driver of your choice
export SUDO=                      # if you don't need sudo, you can redefine the SUDO variable from default `sudo`
./tests/e2e.sh
```
