/*
Copyright 2018 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package options

import (
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	kcollectors "k8s.io/kube-state-metrics/pkg/collectors"
)

var (
	DefaultNamespaces = NamespaceList{metav1.NamespaceAll}
	DefaultCollectors = CollectorSet{
		"daemonsets":               struct{}{},
		"deployments":              struct{}{},
		"limitranges":              struct{}{},
		"nodes":                    struct{}{},
		"pods":                     struct{}{},
		"replicasets":              struct{}{},
		"replicationcontrollers":   struct{}{},
		"resourcequotas":           struct{}{},
		"services":                 struct{}{},
		"jobs":                     struct{}{},
		"cronjobs":                 struct{}{},
		"statefulsets":             struct{}{},
		"persistentvolumes":        struct{}{},
		"persistentvolumeclaims":   struct{}{},
		"Namespaces":               struct{}{},
		"horizontalpodautoscalers": struct{}{},
		"endpoints":                struct{}{},
		"secrets":                  struct{}{},
		"configmaps":               struct{}{},
	}
	AvailableCollectors = map[string]func(registry prometheus.Registerer, kubeClient clientset.Interface, namespaces []string){
		"cronjobs":                 kcollectors.RegisterCronJobCollector,
		"daemonsets":               kcollectors.RegisterDaemonSetCollector,
		"deployments":              kcollectors.RegisterDeploymentCollector,
		"jobs":                     kcollectors.RegisterJobCollector,
		"limitranges":              kcollectors.RegisterLimitRangeCollector,
		"nodes":                    kcollectors.RegisterNodeCollector,
		"pods":                     kcollectors.RegisterPodCollector,
		"replicasets":              kcollectors.RegisterReplicaSetCollector,
		"replicationcontrollers":   kcollectors.RegisterReplicationControllerCollector,
		"resourcequotas":           kcollectors.RegisterResourceQuotaCollector,
		"services":                 kcollectors.RegisterServiceCollector,
		"statefulsets":             kcollectors.RegisterStatefulSetCollector,
		"persistentvolumes":        kcollectors.RegisterPersistentVolumeCollector,
		"persistentvolumeclaims":   kcollectors.RegisterPersistentVolumeClaimCollector,
		"Namespaces":               kcollectors.RegisterNamespaceCollector,
		"horizontalpodautoscalers": kcollectors.RegisterHorizontalPodAutoScalerCollector,
		"endpoints":                kcollectors.RegisterEndpointCollector,
		"secrets":                  kcollectors.RegisterSecretCollector,
		"configmaps":               kcollectors.RegisterConfigMapCollector,
	}
)
