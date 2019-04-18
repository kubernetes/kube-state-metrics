#!/bin/bash

# Copyright 2017 The Kubernetes Authors All rights reserved.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e
set -o pipefail

KUBERNETES_VERSION=v1.14.1
KUBE_STATE_METRICS_LOG_DIR=./log
KUBE_STATE_METRICS_IMAGE_NAME='quay.io/coreos/kube-state-metrics'
PROMETHEUS_VERSION=2.9.1
E2E_SETUP_MINIKUBE=${E2E_SETUP_MINIKUBE:-}
E2E_SETUP_KUBECTL=${E2E_SETUP_KUBECTL:-}
E2E_SETUP_PROMTOOL=${E2E_SETUP_PROMTOOL:-}
MINIKUBE_VERSION=v1.0.0
MINIKUBE_DRIVER=${MINIKUBE_DRIVER:-virtualbox}
SUDO=${SUDO:-}

mkdir -p ${KUBE_STATE_METRICS_LOG_DIR}

function finish() {
    echo "calling cleanup function"
    # kill kubectl proxy in background
    kill %1 || true
    kubectl delete -f kubernetes/ || true
    kubectl delete -f tests/manifests/ || true
}

function setup_minikube() {
    curl -sLo minikube https://storage.googleapis.com/minikube/releases/${MINIKUBE_VERSION}/minikube-linux-amd64 \
        && chmod +x minikube \
        && ${SUDO} mv minikube /usr/local/bin/
}

function setup_kubectl() {
    curl -sLo kubectl https://storage.googleapis.com/kubernetes-release/release/"$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)"/bin/linux/amd64/kubectl \
        && chmod +x kubectl \
        && ${SUDO} mv kubectl /usr/local/bin/
}

function setup_promtool() {
    wget -q -O /tmp/prometheus.tar.gz https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VERSION}/prometheus-${PROMETHEUS_VERSION}.linux-amd64.tar.gz
    tar zxfv /tmp/prometheus.tar.gz -C /tmp/ prometheus-${PROMETHEUS_VERSION}.linux-amd64/promtool
    ${SUDO} mv /tmp/prometheus-${PROMETHEUS_VERSION}.linux-amd64/promtool /usr/local/bin/
    rmdir /tmp/prometheus-${PROMETHEUS_VERSION}.linux-amd64
    rm /tmp/prometheus.tar.gz
}

[[ -n "$E2E_SETUP_MINIKUBE" ]] && setup_minikube

minikube version

[[ -n "$E2E_SETUP_KUBECTL" ]] && setup_kubectl

export MINIKUBE_WANTUPDATENOTIFICATION=false
export MINIKUBE_WANTREPORTERRORPROMPT=false
export MINIKUBE_HOME=$HOME
export CHANGE_MINIKUBE_NONE_USER=true
mkdir "$HOME"/.kube || true
touch "$HOME"/.kube/config

export KUBECONFIG=$HOME/.kube/config
${SUDO} minikube start --vm-driver="${MINIKUBE_DRIVER}" --kubernetes-version=${KUBERNETES_VERSION} --logtostderr

minikube update-context

set +e

is_kube_running="false"

# this for loop waits until kubectl can access the api server that Minikube has created
for _ in {1..90}; do # timeout for 3 minutes
   kubectl get po 1>/dev/null 2>&1
   if [[ $? -ne 1 ]]; then
      is_kube_running="true"
      break
   fi

   echo "waiting for Kubernetes cluster up"
   sleep 2
done

if [[ ${is_kube_running} == "false" ]]; then
   minikube logs
   echo "Kubernetes does not start within 3 minutes"
   exit 1
fi

set -e

kubectl version

# ensure that we build docker image in minikube
[[ "$MINIKUBE_DRIVER" != "none" ]] && eval "$(minikube docker-env)"

# query kube-state-metrics image tag
make container
docker images -a
KUBE_STATE_METRICS_IMAGE_TAG=$(docker images -a|grep 'quay.io/coreos/kube-state-metrics'|grep -v 'latest'|awk '{print $2}'|sort -u)
echo "local kube-state-metrics image tag: $KUBE_STATE_METRICS_IMAGE_TAG"

# update kube-state-metrics image tag in kube-state-metrics-deployment.yaml
sed -i.bak "s|$KUBE_STATE_METRICS_IMAGE_NAME:v.*|$KUBE_STATE_METRICS_IMAGE_NAME:$KUBE_STATE_METRICS_IMAGE_TAG|g" ./kubernetes/kube-state-metrics-deployment.yaml
cat ./kubernetes/kube-state-metrics-deployment.yaml

trap finish EXIT

# set up kube-state-metrics manifests
kubectl create -f ./kubernetes/kube-state-metrics-service-account.yaml

kubectl create -f ./kubernetes/kube-state-metrics-cluster-role.yaml
kubectl create -f ./kubernetes/kube-state-metrics-cluster-role-binding.yaml

kubectl create -f ./kubernetes/kube-state-metrics-role-binding.yaml
kubectl create -f ./kubernetes/kube-state-metrics-role.yaml

kubectl create -f ./kubernetes/kube-state-metrics-deployment.yaml

kubectl create -f ./kubernetes/kube-state-metrics-service.yaml

kubectl create -f ./tests/manifests/

echo "make requests to kube-state-metrics"

set +e

is_kube_state_metrics_running="false"

kubectl proxy &

# this for loop waits until kube-state-metrics is running by accessing the healthz endpoint
for _ in {1..30}; do # timeout for 1 minutes
    KUBE_STATE_METRICS_STATUS=$(curl -s "http://localhost:8001/api/v1/namespaces/kube-system/services/kube-state-metrics:http-metrics/proxy/healthz")
    if [[ "$KUBE_STATE_METRICS_STATUS" == "ok" ]]; then
        is_kube_state_metrics_running="true"
        break
    fi

    echo "waiting for Kube-state-metrics up"
    sleep 2
done

if [[ ${is_kube_state_metrics_running} != "true" ]]; then
    kubectl --namespace=kube-system logs deployment/kube-state-metrics kube-state-metrics
    echo "kube-state-metrics does not start within 1 minute"
    exit 1
fi

set -e

echo "kube-state-metrics is up and running"

echo "access kube-state-metrics metrics endpoint"
curl -s "http://localhost:8001/api/v1/namespaces/kube-system/services/kube-state-metrics:http-metrics/proxy/metrics" >${KUBE_STATE_METRICS_LOG_DIR}/metrics

echo "check metrics format with promtool"
[[ -n "$E2E_SETUP_PROMTOOL" ]] && setup_promtool
< ${KUBE_STATE_METRICS_LOG_DIR}/metrics promtool check metrics

collectors=$(find internal/collector/ -maxdepth 1 -name "*.go" -not -name "*_test.go" -not -name "builder.go" -not -name "testutils.go" -not -name "utils.go" -print0 | xargs -0 -n1 basename | awk -F. '{print $1}')
echo "available collectors: $collectors"
for collector in ${collectors}; do
    echo "checking that kube_${collector}* metrics exists"
    grep "^kube_${collector}_" ${KUBE_STATE_METRICS_LOG_DIR}/metrics
done

KUBE_STATE_METRICS_STATUS=$(curl -s "http://localhost:8001/api/v1/namespaces/kube-system/services/kube-state-metrics:http-metrics/proxy/healthz")
if [[ "$KUBE_STATE_METRICS_STATUS" == "ok" ]]; then
    echo "kube-state-metrics is still running after accessing metrics endpoint"
    exit 0
fi

# wait for klog to flush to log file
sleep 33
kubectl --namespace=kube-system logs deployment/kube-state-metrics kube-state-metrics
exit 1
