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

# setup a Kubernetes cluster
curl -sLo minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 && chmod +x minikube && sudo mv minikube /usr/local/bin/

echo "minikube version is `minikube version`"

curl -sLo kubectl https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/

export MINIKUBE_WANTUPDATENOTIFICATION=false
export MINIKUBE_WANTREPORTERRORPROMPT=false
export MINIKUBE_HOME=$HOME
export CHANGE_MINIKUBE_NONE_USER=true
mkdir $HOME/.kube || true
touch $HOME/.kube/config

export KUBECONFIG=$HOME/.kube/config
sudo minikube start --vm-driver=none --kubernetes-version=v1.8.0

# this for loop waits until kubectl can access the api server that Minikube has created
for i in {1..150}; do # timeout for 5 minutes
   echo "--------" 
   cat $HOME/.kube/config
   minikube logs
   cat /Users/andy/.minikube/machines/minikube/config.json
   minikube ip
   minikube status
   minikube ssh docker ps 
   kubectl version
   kubectl get po 1>/dev/null 2>&1
   echo "======"
   if [ $? -ne 1 ]; then
      break
   fi

   echo "waiting for Kubernetes cluster up"
   sleep 2
done

echo "kubectl version is `kubectl version`"

# setting up kube-state-metrics
#kubectl create -f ./kubernetes/
