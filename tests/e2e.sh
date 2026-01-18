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

set -x
set -e
set -o pipefail

case $(uname -m) in
	aarch64)	ARCH="arm64";;
	x86_64)		ARCH="amd64";;
	*)		ARCH="$(uname -m)";;
esac

NODE_IMAGE_NAME="docker.io/kindest/node"
KUBERNETES_VERSION=${KUBERNETES_VERSION:-"v1.34.3"}
KUBE_STATE_METRICS_LOG_DIR=./log
KUBE_STATE_METRICS_CURRENT_IMAGE_NAME="registry.k8s.io/kube-state-metrics/kube-state-metrics"
KUBE_STATE_METRICS_IMAGE_NAME="registry.k8s.io/kube-state-metrics/kube-state-metrics-${ARCH}"
E2E_SETUP_KIND=${E2E_SETUP_KIND:-}
E2E_SETUP_KUBECTL=${E2E_SETUP_KUBECTL:-}
KIND_VERSION=v0.30.0
SUDO=${SUDO:-}

OS=$(uname -s | awk '{print tolower($0)}')
OS=${OS:-linux}

function finish() {
    echo "calling cleanup function"
    # kill kubectl proxy in background
    kill %1 || true
    kubectl delete -f examples/standard/ || true
    kubectl delete -f tests/manifests/ || true
}

function setup_kind() {
    curl -sLo kind "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-${OS}-${ARCH}" \
        && chmod +x kind \
        && ${SUDO} mv kind /usr/local/bin/
}

function setup_kubectl() {
    curl -sLo kubectl https://dl.k8s.io/release/"$(curl -sL https://dl.k8s.io/release/stable.txt)"/bin/"${OS}"/"${ARCH}"/kubectl \
        && chmod +x kubectl \
        && ${SUDO} mv kubectl /usr/local/bin/
}

[[ -n "${E2E_SETUP_KIND}" ]] && setup_kind

kind version

[[ -n "${E2E_SETUP_KUBECTL}" ]] && setup_kubectl

mkdir "${HOME}"/.kube || true
touch "${HOME}"/.kube/config

export KUBECONFIG=$HOME/.kube/config

kind create cluster --image="${NODE_IMAGE_NAME}:${KUBERNETES_VERSION}"

kind export kubeconfig

set +e
kubectl proxy &

function kube_state_metrics_up() {
    serviceName="kube-state-metrics"
    if [[ -n "$1" ]]; then
        serviceName="$1"
    fi
    is_kube_state_metrics_running="false"
    # this for loop waits until kube-state-metrics is running by accessing the healthz endpoint
    for _ in {1..30}; do # timeout for 1 minutes
        KUBE_STATE_METRICS_STATUS=$(curl -s "http://localhost:8001/api/v1/namespaces/kube-system/services/${serviceName}:http-metrics/proxy/healthz")
        if [[ "${KUBE_STATE_METRICS_STATUS}" == "OK" ]]; then
            is_kube_state_metrics_running="true"
            break
        fi

        echo "waiting for kube-state-metrics to come up"
        sleep 2
    done

    if [[ ${is_kube_state_metrics_running} != "true" ]]; then
        kubectl --namespace=kube-system logs deployment/kube-state-metrics kube-state-metrics
        echo "kube-state-metrics does not start within 1 minute"
        exit 1
    fi
}
function kube_pod_up() {
    is_pod_running="false"

    for _ in {1..90}; do # timeout for 3 minutes
        kubectl get pods -A | grep "$1" 1>/dev/null 2>&1
        if [[ $? -ne 1 ]]; then
            is_pod_running="true"
            break
        fi

        echo "waiting for pod $1 to come up"
        sleep 2
    done

    if [[ ${is_pod_running} == "false" ]]; then
        echo "Pod does not show up within 3 minutes"
        exit 1
    fi
}

function test_daemonset() {
    sed -i "s|${KUBE_STATE_METRICS_CURRENT_IMAGE_NAME}:v.*|${KUBE_STATE_METRICS_IMAGE_NAME}:${KUBE_STATE_METRICS_IMAGE_TAG}|g" ./examples/daemonsetsharding/deployment.yaml
    sed -i "s|${KUBE_STATE_METRICS_CURRENT_IMAGE_NAME}:v.*|${KUBE_STATE_METRICS_IMAGE_NAME}:${KUBE_STATE_METRICS_IMAGE_TAG}|g" ./examples/daemonsetsharding/daemonset.yaml
    sed -i "s|${KUBE_STATE_METRICS_CURRENT_IMAGE_NAME}:v.*|${KUBE_STATE_METRICS_IMAGE_NAME}:${KUBE_STATE_METRICS_IMAGE_TAG}|g" ./examples/daemonsetsharding/deployment-unscheduled-pods-fetching.yaml

    kubectl get deployment -n kube-system
    kubectl create -f ./examples/daemonsetsharding
    kube_state_metrics_up kube-state-metrics-unscheduled-pods-fetching
    kube_state_metrics_up kube-state-metrics
    kube_state_metrics_up kube-state-metrics-shard
    kubectl apply -f ./tests/e2e/testdata/pods.yaml
    kube_pod_up runningpod1
    kube_pod_up pendingpod2

    kubectl get deployment -n default
    runningpod1="$(curl -s "http://localhost:8001/api/v1/namespaces/kube-system/services/kube-state-metrics-shard:http-metrics/proxy/metrics" | grep "runningpod1"  | grep -c "kube_pod_info" )"
    node1="$(curl -s "http://localhost:8001/api/v1/namespaces/kube-system/services/kube-state-metrics:http-metrics/proxy/metrics" | grep -c "# TYPE kube_node_info" )"
    expected_num_pod=1
    if [ "${runningpod1}" != "${expected_num_pod}" ]; then
        echo "metric kube_pod_info for runningpod1 doesn't show up only once, got ${runningpod1} times"
        exit 1
    fi

    if [ "${node1}" != "1" ]; then
        echo "metric kube_node_info doesn't show up only once, got ${node1} times"
        exit 1
    fi

    kubectl logs deployment/kube-state-metrics-unscheduled-pods-fetching -n kube-system
    sleep 2
    kubectl get pods -A --field-selector spec.nodeName=""
    sleep 2
    pendingpod2="$(curl -s "http://localhost:8001/api/v1/namespaces/kube-system/services/kube-state-metrics-unscheduled-pods-fetching:http-metrics/proxy/metrics"  | grep "pendingpod2" | grep -c "kube_pod_info" )"
    if [ "${pendingpod2}" != "${expected_num_pod}" ]; then
        echo "metric kube_pod_info for pendingpod2 doesn't show up only once, got ${runningpod1} times"
        exit 1
    fi

    kubectl delete -f ./tests/e2e/testdata/pods.yaml
    kubectl delete -f ./examples/daemonsetsharding
    sleep 20
}


is_kube_running="false"

# this for loop waits until kubectl can access the api server that kind has created
for _ in {1..90}; do # timeout for 3 minutes
   kubectl get po 1>/dev/null 2>&1
   if [[ $? -ne 1 ]]; then
      is_kube_running="true"
      break
   fi

   echo "waiting for Kubernetes cluster to come up"
   sleep 2
done

if [[ ${is_kube_running} == "false" ]]; then
   echo "Kubernetes does not start within 3 minutes"
   exit 1
fi

set -e

kubectl version

# query kube-state-metrics image tag
REGISTRY="registry.k8s.io/kube-state-metrics" make container
docker images -a
KUBE_STATE_METRICS_IMAGE_TAG=$(docker images -a|grep "${KUBE_STATE_METRICS_IMAGE_NAME}" |grep -v 'latest'|awk '{print $2}'|sort -u)
echo "local kube-state-metrics image tag: $KUBE_STATE_METRICS_IMAGE_TAG"

kind load docker-image "${KUBE_STATE_METRICS_IMAGE_NAME}:${KUBE_STATE_METRICS_IMAGE_TAG}"

# update kube-state-metrics image tag in deployment.yaml
sed -i.bak "s|${KUBE_STATE_METRICS_CURRENT_IMAGE_NAME}:v.*|${KUBE_STATE_METRICS_IMAGE_NAME}:${KUBE_STATE_METRICS_IMAGE_TAG}|g" ./examples/standard/deployment.yaml

mkdir -p ${KUBE_STATE_METRICS_LOG_DIR}

test_daemonset

cat ./examples/standard/deployment.yaml

trap finish EXIT

# set up kube-state-metrics manifests
kubectl create -f ./examples/standard/service-account.yaml

kubectl create -f ./examples/standard/cluster-role.yaml
kubectl create -f ./examples/standard/cluster-role-binding.yaml

kubectl create -f ./examples/standard/deployment.yaml

kubectl create -f ./examples/standard/service.yaml

kubectl create -f ./tests/manifests/

echo "make requests to kube-state-metrics"

set +e


kube_state_metrics_up
set -e

echo "kube-state-metrics is up and running"

echo "start e2e test for kube-state-metrics"
KSM_HTTP_METRICS_URL='http://localhost:8001/api/v1/namespaces/kube-system/services/kube-state-metrics:http-metrics/proxy'
KSM_TELEMETRY_URL='http://localhost:8001/api/v1/namespaces/kube-system/services/kube-state-metrics:telemetry/proxy'

go test -v ./tests/e2e/main_test.go --ksm-http-metrics-url=${KSM_HTTP_METRICS_URL} --ksm-telemetry-url=${KSM_TELEMETRY_URL}

# TODO: re-implement the following test cases in Go with the goal of removing this file.
echo "access kube-state-metrics metrics endpoint"
curl -s "http://localhost:8001/api/v1/namespaces/kube-system/services/kube-state-metrics:http-metrics/proxy/metrics" >${KUBE_STATE_METRICS_LOG_DIR}/metrics

KUBE_STATE_METRICS_STATUS=$(curl -s "http://localhost:8001/api/v1/namespaces/kube-system/services/kube-state-metrics:http-metrics/proxy/healthz")
if [[ "${KUBE_STATE_METRICS_STATUS}" == "OK" ]]; then
    echo "kube-state-metrics is still running after accessing metrics endpoint"
fi

# wait for klog to flush to log file
sleep 33
klog_err=E$(date +%m%d)
echo "check for errors in logs"

echo "running authfiler tests.."
go test -race -v ./tests/e2e/auth-filter_test.go

echo "running discovery tests..."
go test -race -v ./tests/e2e/discovery_test.go

echo "running object limits test..."
go test -race -v ./tests/e2e/object-limits_test.go

echo "running hot-reload tests..."
go test -race -v ./tests/e2e/hot-reload_test.go

echo "running hot-reload-kubeconfig tests..."
go test -race -v ./tests/e2e/hot-reload-kubeconfig_test.go

echo "running CRD UpdateFunc handler tests..."
go test -race -v ./tests/e2e/crd_informer_updatefunc_test.go

output_logs=$(kubectl --namespace=kube-system logs deployment/kube-state-metrics kube-state-metrics)
if echo "${output_logs}" | grep "^${klog_err}"; then
    echo ""
    echo "==========================================="
    echo "Found errors in the kube-state-metrics logs"
    echo "==========================================="
    echo ""
    echo "${output_logs}"
    exit 1
fi
