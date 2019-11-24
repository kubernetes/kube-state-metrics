/*
Copyright 2020 The Kubernetes Authors All rights reserved.

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

package store

import "k8s.io/kube-state-metrics/v2/pkg/allow"

func init() {
	var labels []string
	for _, lbls := range [][]string{
		descCSRLabelsDefaultLabels,
		descConfigMapLabelsDefaultLabels,
		descCronJobLabelsDefaultLabels,
		descDaemonSetLabelsDefaultLabels,
		descDeploymentLabelsDefaultLabels,
		descEndpointLabelsDefaultLabels,
		descHorizontalPodAutoscalerLabelsDefaultLabels,
		descIngressLabelsDefaultLabels,
		descJobLabelsDefaultLabels,
		descLeaseLabelsDefaultLabels,
		descLimitRangeLabelsDefaultLabels,
		descMutatingWebhookConfigurationDefaultLabels,
		descNamespaceLabelsDefaultLabels,
		descNetworkPolicyLabelsDefaultLabels,
		descNodeLabelsDefaultLabels,
		descPersistentVolumeLabelsDefaultLabels,
		descPersistentVolumeClaimLabelsDefaultLabels,
		descPodLabelsDefaultLabels,
		descPodDisruptionBudgetLabelsDefaultLabels,
		descReplicaSetLabelsDefaultLabels,
		descReplicationControllerLabelsDefaultLabels,
		descResourceQuotaLabelsDefaultLabels,
		descSecretLabelsDefaultLabels,
		descServiceLabelsDefaultLabels,
		descStatefulSetLabelsDefaultLabels,
		descStorageClassLabelsDefaultLabels,
		descValidatingWebhookConfigurationDefaultLabels,
		descVerticalPodAutoscalerLabelsDefaultLabels,
		descVolumeAttachmentLabelsDefaultLabels,
	} {
		labels = append(labels, lbls...)
	}

	for defaultMetric, defaultLabels := range allow.DefaultMetricLabels {
		allow.DefaultMetricLabels[defaultMetric] = append(defaultLabels, labels...)
	}
}
