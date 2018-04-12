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

KUBERNETES_VERSION=v1.8.0
KUBE_STATE_METRICS_LOG_DIR=./log
KUBE_STATE_METRICS_IMAGE_NAME='quay.io/coreos/kube-state-metrics'
KUBE_STATE_METRICS_IMAGE_NAME_PATTERN='quay.io\/coreos\/kube-state-metrics'
PROMETHEUS_VERSION=2.0.0

mkdir -p $KUBE_STATE_METRICS_LOG_DIR

# setup a Kubernetes cluster
curl -sLo minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 && chmod +x minikube && sudo mv minikube /usr/local/bin/

minikube version

curl -sLo kubectl https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/

export MINIKUBE_WANTUPDATENOTIFICATION=false
export MINIKUBE_WANTREPORTERRORPROMPT=false
export MINIKUBE_HOME=$HOME
export CHANGE_MINIKUBE_NONE_USER=true
mkdir $HOME/.kube || true
touch $HOME/.kube/config

export KUBECONFIG=$HOME/.kube/config
sudo minikube start --vm-driver=none --bootstrapper=localkube --kubernetes-version=$KUBERNETES_VERSION --logtostderr

minikube update-context

set +e

is_kube_running="false"

# this for loop waits until kubectl can access the api server that Minikube has created
for i in {1..90}; do # timeout for 3 minutes
   kubectl get po 1>/dev/null 2>&1
   if [ $? -ne 1 ]; then
      is_kube_running="true"
      break
   fi

   echo "waiting for Kubernetes cluster up"
   sleep 2
done

if [ $is_kube_running == "false" ]; then
   minikube logs
   echo "Kubernetes does not start within 3 minutes"
   exit 1
fi

set -e

kubectl version

# query kube-state-metrics image tag
make container
docker images -a
ksm_image_tag=`docker images -a|grep 'quay.io/coreos/kube-state-metrics'|awk '{print $2}'|sort -u`
echo "local kube-state-metrics image tag: $ksm_image_tag"

# update kube-state-metrics image tag in kube-state-metrics-deployment.yaml
sed -i.bak "s/$KUBE_STATE_METRICS_IMAGE_NAME_PATTERN:v.*/$KUBE_STATE_METRICS_IMAGE_NAME_PATTERN:$ksm_image_tag/g" ./kubernetes/kube-state-metrics-deployment.yaml
cat ./kubernetes/kube-state-metrics-deployment.yaml

# set up kube-state-metrics manifests
kubectl create -f ./kubernetes/kube-state-metrics-service-account.yaml

kubectl create -f ./kubernetes/kube-state-metrics-cluster-role.yaml
kubectl create -f ./kubernetes/kube-state-metrics-cluster-role-binding.yaml

kubectl create -f ./kubernetes/kube-state-metrics-role-binding.yaml
kubectl create -f ./kubernetes/kube-state-metrics-role.yaml

kubectl create -f ./kubernetes/kube-state-metrics-deployment.yaml

kubectl create -f ./kubernetes/kube-state-metrics-service.yaml

echo "make requests to kube-state-metrics"

set +e

is_kube_state_metrics_running="false"

kubectl proxy &

# this for loop waits until kube-state-metrics is running by accessing the healthz endpoint
for i in {1..30}; do # timeout for 1 minutes
    KUBE_STATE_METRICS_STATUS=$(curl -s "http://localhost:8001/api/v1/proxy/namespaces/kube-system/services/kube-state-metrics:8080/healthz")
    if [ "$KUBE_STATE_METRICS_STATUS" == "ok" ]; then
        is_kube_state_metrics_running="true"
        break
    fi

    echo "waiting for Kube-state-metrics up"
    sleep 2
done

if [ $is_kube_state_metrics_running != "true" ]; then
    kubectl --namespace=kube-system logs deployment/kube-state-metrics kube-state-metrics
    echo "kube-state-metrics does not start within 1 minute"
    exit 1
fi

set -e

echo "kube-state-metrics is up and running"

echo "access kube-state-metrics metrics endpoint"
curl -s "http://localhost:8001/api/v1/proxy/namespaces/kube-system/services/kube-state-metrics:8080/metrics" >$KUBE_STATE_METRICS_LOG_DIR/metrics

echo "check metrics format with promtool"
wget -q -O /tmp/prometheus.tar.gz https://github.com/prometheus/prometheus/releases/download/v$PROMETHEUS_VERSION/prometheus-$PROMETHEUS_VERSION.linux-amd64.tar.gz
tar zxfv /tmp/prometheus.tar.gz -C /tmp
cat $KUBE_STATE_METRICS_LOG_DIR/metrics | /tmp/prometheus-$PROMETHEUS_VERSION.linux-amd64/promtool check metrics

KUBE_STATE_METRICS_STATUS=$(curl -s "http://localhost:8001/api/v1/proxy/namespaces/kube-system/services/kube-state-metrics:8080/healthz")
if [ "$KUBE_STATE_METRICS_STATUS" == "ok" ]; then
    echo "kube-state-metrics is still running after accessing metrics endpoint"
    exit 0
fi

# wait for glog to flush to log file
sleep 33
kubectl --namespace=kube-system logs deployment/kube-state-metrics kube-state-metrics
exit 1
