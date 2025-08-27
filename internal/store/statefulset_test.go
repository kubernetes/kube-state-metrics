/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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

import (
	"testing"
	"time"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	statefulSet1Replicas int32 = 3
	statefulSet2Replicas int32 = 6
	statefulSet3Replicas int32 = 9
	statefulSet6Replicas int32 = 1

	statefulSet1ObservedGeneration int64 = 1
	statefulSet2ObservedGeneration int64 = 2
)

func TestStatefulSetStore(t *testing.T) {
	cases := []generateMetricsTestCase{
		{
			Obj: &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "statefulset1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "ns1",
					Labels: map[string]string{
						"app": "example1",
					},
					Generation: 3,
				},
				Spec: v1.StatefulSetSpec{
					Replicas:    &statefulSet1Replicas,
					ServiceName: "statefulset1service",
				},
				Status: v1.StatefulSetStatus{
					ObservedGeneration: statefulSet1ObservedGeneration,
					Replicas:           2,
					UpdateRevision:     "ur1",
					CurrentRevision:    "cr1",
				},
			},
			Want: `
				# HELP kube_statefulset_created [STABLE] Unix creation timestamp
				# HELP kube_statefulset_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_statefulset_metadata_generation [STABLE] Sequence number representing a specific generation of the desired state for the StatefulSet.
				# HELP kube_statefulset_persistentvolumeclaim_retention_policy Count of retention policy for StatefulSet template PVCs
				# HELP kube_statefulset_replicas [STABLE] Number of desired pods for a StatefulSet.
				# HELP kube_statefulset_ordinals_start [STABLE] Start ordinal of the StatefulSet.
				# HELP kube_statefulset_status_current_revision [STABLE] Indicates the version of the StatefulSet used to generate Pods in the sequence [0,currentReplicas).
				# HELP kube_statefulset_status_observed_generation [STABLE] The generation observed by the StatefulSet controller.
				# HELP kube_statefulset_status_replicas [STABLE] The number of replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_available The number of available replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_current [STABLE] The number of current replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_ready [STABLE] The number of ready replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_updated [STABLE] The number of updated replicas per StatefulSet.
				# HELP kube_statefulset_status_update_revision [STABLE] Indicates the version of the StatefulSet used to generate Pods in the sequence [replicas-updatedReplicas,replicas)
				# TYPE kube_statefulset_created gauge
				# TYPE kube_statefulset_labels gauge
				# TYPE kube_statefulset_metadata_generation gauge
				# TYPE kube_statefulset_persistentvolumeclaim_retention_policy gauge
				# TYPE kube_statefulset_replicas gauge
				# TYPE kube_statefulset_ordinals_start gauge
				# TYPE kube_statefulset_status_current_revision gauge
				# TYPE kube_statefulset_status_observed_generation gauge
				# TYPE kube_statefulset_status_replicas gauge
				# TYPE kube_statefulset_status_replicas_available gauge
				# TYPE kube_statefulset_status_replicas_current gauge
				# TYPE kube_statefulset_status_replicas_ready gauge
				# TYPE kube_statefulset_status_replicas_updated gauge
				# TYPE kube_statefulset_status_update_revision gauge
				kube_statefulset_status_update_revision{namespace="ns1",revision="ur1",statefulset="statefulset1"} 1
				kube_statefulset_created{namespace="ns1",statefulset="statefulset1"} 1.5e+09
				kube_statefulset_status_current_revision{namespace="ns1",revision="cr1",statefulset="statefulset1"} 1
 				kube_statefulset_status_replicas{namespace="ns1",statefulset="statefulset1"} 2
				kube_statefulset_status_replicas_available{namespace="ns1",statefulset="statefulset1"} 0
				kube_statefulset_status_replicas_current{namespace="ns1",statefulset="statefulset1"} 0
				kube_statefulset_status_replicas_ready{namespace="ns1",statefulset="statefulset1"} 0
				kube_statefulset_status_replicas_updated{namespace="ns1",statefulset="statefulset1"} 0
 				kube_statefulset_status_observed_generation{namespace="ns1",statefulset="statefulset1"} 1
 				kube_statefulset_replicas{namespace="ns1",statefulset="statefulset1"} 3
 				kube_statefulset_metadata_generation{namespace="ns1",statefulset="statefulset1"} 3
`,
			MetricNames: []string{
				"kube_statefulset_created",
				"kube_statefulset_labels",
				"kube_statefulset_metadata_generation",
				"kube_statefulset_replicas",
				"kube_statefulset_ordinals_start",
				"kube_statefulset_status_observed_generation",
				"kube_statefulset_status_replicas",
				"kube_statefulset_status_replicas_available",
				"kube_statefulset_status_replicas_current",
				"kube_statefulset_status_replicas_ready",
				"kube_statefulset_status_replicas_updated",
				"kube_statefulset_status_update_revision",
				"kube_statefulset_status_current_revision",
				"kube_statefulset_persistentvolumeclaim_retention_policy",
			},
		},
		{
			Obj: &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "statefulset2",
					Namespace: "ns2",
					Labels: map[string]string{
						"app": "example2",
					},
					Generation: 21,
				},
				Spec: v1.StatefulSetSpec{
					Replicas:    &statefulSet2Replicas,
					ServiceName: "statefulset2service",
				},
				Status: v1.StatefulSetStatus{
					CurrentReplicas:    2,
					ObservedGeneration: statefulSet2ObservedGeneration,
					ReadyReplicas:      5,
					Replicas:           5,
					AvailableReplicas:  4,
					UpdatedReplicas:    3,
					UpdateRevision:     "ur2",
					CurrentRevision:    "cr2",
				},
			},
			Want: `
				# HELP kube_statefulset_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_statefulset_metadata_generation [STABLE] Sequence number representing a specific generation of the desired state for the StatefulSet.
				# HELP kube_statefulset_persistentvolumeclaim_retention_policy Count of retention policy for StatefulSet template PVCs
				# HELP kube_statefulset_replicas [STABLE] Number of desired pods for a StatefulSet.
				# HELP kube_statefulset_status_current_revision [STABLE] Indicates the version of the StatefulSet used to generate Pods in the sequence [0,currentReplicas).
				# HELP kube_statefulset_status_observed_generation [STABLE] The generation observed by the StatefulSet controller.
				# HELP kube_statefulset_status_replicas [STABLE] The number of replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_available The number of available replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_current [STABLE] The number of current replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_ready [STABLE] The number of ready replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_updated [STABLE] The number of updated replicas per StatefulSet.
				# HELP kube_statefulset_status_update_revision [STABLE] Indicates the version of the StatefulSet used to generate Pods in the sequence [replicas-updatedReplicas,replicas)
				# TYPE kube_statefulset_labels gauge
				# TYPE kube_statefulset_metadata_generation gauge
				# TYPE kube_statefulset_persistentvolumeclaim_retention_policy gauge
				# TYPE kube_statefulset_replicas gauge
				# TYPE kube_statefulset_status_current_revision gauge
				# TYPE kube_statefulset_status_observed_generation gauge
				# TYPE kube_statefulset_status_replicas gauge
				# TYPE kube_statefulset_status_replicas_available gauge
				# TYPE kube_statefulset_status_replicas_current gauge
				# TYPE kube_statefulset_status_replicas_ready gauge
				# TYPE kube_statefulset_status_replicas_updated gauge
				# TYPE kube_statefulset_status_update_revision gauge
				kube_statefulset_status_update_revision{namespace="ns2",revision="ur2",statefulset="statefulset2"} 1
				kube_statefulset_status_replicas{namespace="ns2",statefulset="statefulset2"} 5
				kube_statefulset_status_replicas_available{namespace="ns2",statefulset="statefulset2"} 4
				kube_statefulset_status_replicas_current{namespace="ns2",statefulset="statefulset2"} 2
				kube_statefulset_status_replicas_ready{namespace="ns2",statefulset="statefulset2"} 5
				kube_statefulset_status_replicas_updated{namespace="ns2",statefulset="statefulset2"} 3
				kube_statefulset_status_observed_generation{namespace="ns2",statefulset="statefulset2"} 2
				kube_statefulset_replicas{namespace="ns2",statefulset="statefulset2"} 6
				kube_statefulset_metadata_generation{namespace="ns2",statefulset="statefulset2"} 21
				kube_statefulset_status_current_revision{namespace="ns2",revision="cr2",statefulset="statefulset2"} 1
`,
			MetricNames: []string{
				"kube_statefulset_labels",
				"kube_statefulset_metadata_generation",
				"kube_statefulset_replicas",
				"kube_statefulset_status_observed_generation",
				"kube_statefulset_status_replicas",
				"kube_statefulset_status_replicas_available",
				"kube_statefulset_status_replicas_current",
				"kube_statefulset_status_replicas_ready",
				"kube_statefulset_status_replicas_updated",
				"kube_statefulset_status_update_revision",
				"kube_statefulset_status_current_revision",
				"kube_statefulset_persistentvolumeclaim_retention_policy",
			},
		},
		{
			Obj: &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "statefulset3",
					Namespace: "ns3",
					Labels: map[string]string{
						"app": "example3",
					},
					Generation: 36,
				},
				Spec: v1.StatefulSetSpec{
					Replicas:    &statefulSet3Replicas,
					ServiceName: "statefulset2service",
				},
				Status: v1.StatefulSetStatus{
					ObservedGeneration: 0,
					Replicas:           7,
					UpdateRevision:     "ur3",
					CurrentRevision:    "cr3",
				},
			},
			Want: `
				# HELP kube_statefulset_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_statefulset_metadata_generation [STABLE] Sequence number representing a specific generation of the desired state for the StatefulSet.
				# HELP kube_statefulset_persistentvolumeclaim_retention_policy Count of retention policy for StatefulSet template PVCs
				# HELP kube_statefulset_replicas [STABLE] Number of desired pods for a StatefulSet.
				# HELP kube_statefulset_status_current_revision [STABLE] Indicates the version of the StatefulSet used to generate Pods in the sequence [0,currentReplicas).
				# HELP kube_statefulset_status_replicas [STABLE] The number of replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_available The number of available replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_current [STABLE] The number of current replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_ready [STABLE] The number of ready replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_updated [STABLE] The number of updated replicas per StatefulSet.
				# HELP kube_statefulset_status_update_revision [STABLE] Indicates the version of the StatefulSet used to generate Pods in the sequence [replicas-updatedReplicas,replicas)
				# TYPE kube_statefulset_labels gauge
				# TYPE kube_statefulset_metadata_generation gauge
				# TYPE kube_statefulset_persistentvolumeclaim_retention_policy gauge
				# TYPE kube_statefulset_replicas gauge
				# TYPE kube_statefulset_status_current_revision gauge
				# TYPE kube_statefulset_status_replicas gauge
				# TYPE kube_statefulset_status_replicas_available gauge
				# TYPE kube_statefulset_status_replicas_current gauge
				# TYPE kube_statefulset_status_replicas_ready gauge
				# TYPE kube_statefulset_status_replicas_updated gauge
				# TYPE kube_statefulset_status_update_revision gauge
				kube_statefulset_status_update_revision{namespace="ns3",revision="ur3",statefulset="statefulset3"} 1
				kube_statefulset_status_replicas{namespace="ns3",statefulset="statefulset3"} 7
				kube_statefulset_status_replicas_available{namespace="ns3",statefulset="statefulset3"} 0
				kube_statefulset_status_replicas_current{namespace="ns3",statefulset="statefulset3"} 0
				kube_statefulset_status_replicas_ready{namespace="ns3",statefulset="statefulset3"} 0
				kube_statefulset_status_replicas_updated{namespace="ns3",statefulset="statefulset3"} 0
				kube_statefulset_replicas{namespace="ns3",statefulset="statefulset3"} 9
				kube_statefulset_metadata_generation{namespace="ns3",statefulset="statefulset3"} 36
				kube_statefulset_status_current_revision{namespace="ns3",revision="cr3",statefulset="statefulset3"} 1
 			`,
			MetricNames: []string{
				"kube_statefulset_labels",
				"kube_statefulset_metadata_generation",
				"kube_statefulset_replicas",
				"kube_statefulset_status_replicas",
				"kube_statefulset_status_replicas_available",
				"kube_statefulset_status_replicas_current",
				"kube_statefulset_status_replicas_ready",
				"kube_statefulset_status_replicas_updated",
				"kube_statefulset_status_update_revision",
				"kube_statefulset_status_current_revision",
				"kube_statefulset_persistentvolumeclaim_retention_policy",
			},
		},
		{
			Obj: &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "statefulset4",
					Namespace: "ns4",
					Labels: map[string]string{
						"app": "example4",
					},
					Generation: 1,
				},
				Spec: v1.StatefulSetSpec{
					Replicas:    &statefulSet1Replicas,
					ServiceName: "statefulset4service",
					PersistentVolumeClaimRetentionPolicy: &v1.StatefulSetPersistentVolumeClaimRetentionPolicy{
						WhenDeleted: v1.RetainPersistentVolumeClaimRetentionPolicyType,
						WhenScaled:  v1.DeletePersistentVolumeClaimRetentionPolicyType,
					},
				},
				Status: v1.StatefulSetStatus{
					ObservedGeneration: 0,
					Replicas:           7,
					UpdateRevision:     "ur3",
					CurrentRevision:    "cr3",
				},
			},
			Want: `
				# HELP kube_statefulset_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_statefulset_metadata_generation [STABLE] Sequence number representing a specific generation of the desired state for the StatefulSet.
				# HELP kube_statefulset_persistentvolumeclaim_retention_policy Count of retention policy for StatefulSet template PVCs
				# HELP kube_statefulset_replicas [STABLE] Number of desired pods for a StatefulSet.
				# HELP kube_statefulset_status_current_revision [STABLE] Indicates the version of the StatefulSet used to generate Pods in the sequence [0,currentReplicas).
				# HELP kube_statefulset_status_replicas [STABLE] The number of replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_available The number of available replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_current [STABLE] The number of current replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_ready [STABLE] The number of ready replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_updated [STABLE] The number of updated replicas per StatefulSet.
				# HELP kube_statefulset_status_update_revision [STABLE] Indicates the version of the StatefulSet used to generate Pods in the sequence [replicas-updatedReplicas,replicas)
				# TYPE kube_statefulset_labels gauge
				# TYPE kube_statefulset_metadata_generation gauge
				# TYPE kube_statefulset_persistentvolumeclaim_retention_policy gauge
				# TYPE kube_statefulset_replicas gauge
				# TYPE kube_statefulset_status_current_revision gauge
				# TYPE kube_statefulset_status_replicas gauge
				# TYPE kube_statefulset_status_replicas_available gauge
				# TYPE kube_statefulset_status_replicas_current gauge
				# TYPE kube_statefulset_status_replicas_ready gauge
				# TYPE kube_statefulset_status_replicas_updated gauge
				# TYPE kube_statefulset_status_update_revision gauge
				kube_statefulset_status_update_revision{namespace="ns4",revision="ur3",statefulset="statefulset4"} 1
				kube_statefulset_status_replicas{namespace="ns4",statefulset="statefulset4"} 7
				kube_statefulset_status_replicas_available{namespace="ns4",statefulset="statefulset4"} 0
				kube_statefulset_status_replicas_current{namespace="ns4",statefulset="statefulset4"} 0
				kube_statefulset_status_replicas_ready{namespace="ns4",statefulset="statefulset4"} 0
				kube_statefulset_status_replicas_updated{namespace="ns4",statefulset="statefulset4"} 0
				kube_statefulset_replicas{namespace="ns4",statefulset="statefulset4"} 3
 				kube_statefulset_metadata_generation{namespace="ns4",statefulset="statefulset4"} 1
 				kube_statefulset_persistentvolumeclaim_retention_policy{namespace="ns4",statefulset="statefulset4",when_deleted="Retain",when_scaled="Delete"} 1
				kube_statefulset_status_current_revision{namespace="ns4",revision="cr3",statefulset="statefulset4"} 1
 			`,
			MetricNames: []string{
				"kube_statefulset_labels",
				"kube_statefulset_metadata_generation",
				"kube_statefulset_replicas",
				"kube_statefulset_status_replicas",
				"kube_statefulset_status_replicas_available",
				"kube_statefulset_status_replicas_current",
				"kube_statefulset_status_replicas_ready",
				"kube_statefulset_status_replicas_updated",
				"kube_statefulset_status_update_revision",
				"kube_statefulset_status_current_revision",
				"kube_statefulset_persistentvolumeclaim_retention_policy",
			},
		},
		{
			// Validate kube_statefulset_ordinals_start metric.
			Obj: &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "statefulset5",
					Namespace: "ns5",
					Labels: map[string]string{
						"app": "example5",
					},
					Generation: 1,
				},
				Spec: v1.StatefulSetSpec{
					Replicas:    &statefulSet1Replicas,
					ServiceName: "statefulset5service",
					Ordinals: &v1.StatefulSetOrdinals{
						Start: 2,
					},
				},
				Status: v1.StatefulSetStatus{
					ObservedGeneration: 0,
					Replicas:           3,
					UpdateRevision:     "ur5",
					CurrentRevision:    "cr5",
				},
			},
			Want: `
				# HELP kube_statefulset_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_statefulset_metadata_generation [STABLE] Sequence number representing a specific generation of the desired state for the StatefulSet.
				# HELP kube_statefulset_persistentvolumeclaim_retention_policy Count of retention policy for StatefulSet template PVCs
				# HELP kube_statefulset_replicas [STABLE] Number of desired pods for a StatefulSet.
				# HELP kube_statefulset_ordinals_start [STABLE] Start ordinal of the StatefulSet.
				# HELP kube_statefulset_status_current_revision [STABLE] Indicates the version of the StatefulSet used to generate Pods in the sequence [0,currentReplicas).
				# HELP kube_statefulset_status_replicas [STABLE] The number of replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_available The number of available replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_current [STABLE] The number of current replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_ready [STABLE] The number of ready replicas per StatefulSet.
				# HELP kube_statefulset_status_replicas_updated [STABLE] The number of updated replicas per StatefulSet.
				# HELP kube_statefulset_status_update_revision [STABLE] Indicates the version of the StatefulSet used to generate Pods in the sequence [replicas-updatedReplicas,replicas)
				# TYPE kube_statefulset_labels gauge
				# TYPE kube_statefulset_metadata_generation gauge
				# TYPE kube_statefulset_persistentvolumeclaim_retention_policy gauge
				# TYPE kube_statefulset_replicas gauge
				# TYPE kube_statefulset_ordinals_start gauge
				# TYPE kube_statefulset_status_current_revision gauge
				# TYPE kube_statefulset_status_replicas gauge
				# TYPE kube_statefulset_status_replicas_available gauge
				# TYPE kube_statefulset_status_replicas_current gauge
				# TYPE kube_statefulset_status_replicas_ready gauge
				# TYPE kube_statefulset_status_replicas_updated gauge
				# TYPE kube_statefulset_status_update_revision gauge
				kube_statefulset_status_update_revision{namespace="ns5",revision="ur5",statefulset="statefulset5"} 1
				kube_statefulset_status_replicas{namespace="ns5",statefulset="statefulset5"} 3
				kube_statefulset_status_replicas_available{namespace="ns5",statefulset="statefulset5"} 0
				kube_statefulset_status_replicas_current{namespace="ns5",statefulset="statefulset5"} 0
				kube_statefulset_status_replicas_ready{namespace="ns5",statefulset="statefulset5"} 0
				kube_statefulset_status_replicas_updated{namespace="ns5",statefulset="statefulset5"} 0
				kube_statefulset_replicas{namespace="ns5",statefulset="statefulset5"} 3
				kube_statefulset_ordinals_start{namespace="ns5",statefulset="statefulset5"} 2
 				kube_statefulset_metadata_generation{namespace="ns5",statefulset="statefulset5"} 1
				kube_statefulset_status_current_revision{namespace="ns5",revision="cr5",statefulset="statefulset5"} 1
 			`,
			MetricNames: []string{
				"kube_statefulset_labels",
				"kube_statefulset_metadata_generation",
				"kube_statefulset_replicas",
				"kube_statefulset_ordinals_start",
				"kube_statefulset_status_replicas",
				"kube_statefulset_status_replicas_available",
				"kube_statefulset_status_replicas_current",
				"kube_statefulset_status_replicas_ready",
				"kube_statefulset_status_replicas_updated",
				"kube_statefulset_status_update_revision",
				"kube_statefulset_status_current_revision",
				"kube_statefulset_persistentvolumeclaim_retention_policy",
			},
		},
		{
			Obj: &v1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "statefulset6",
					Namespace:         "ns6",
					DeletionTimestamp: &metav1.Time{Time: time.Unix(1800000000, 0)},
					Labels: map[string]string{
						"app": "example6",
					},
					Generation: 1,
				},
				Spec: v1.StatefulSetSpec{
					Replicas:    &statefulSet6Replicas,
					ServiceName: "statefulset6service",
				},
				Status: v1.StatefulSetStatus{
					ObservedGeneration: 0,
					Replicas:           1,
				},
			},
			Want: `
				# HELP kube_statefulset_deletion_timestamp Unix deletion timestamp
				# TYPE kube_statefulset_deletion_timestamp gauge
				kube_statefulset_deletion_timestamp{statefulset="statefulset6",namespace="ns6"} 1.8e+09
 			`,
			MetricNames: []string{
				"kube_statefulset_deletion_timestamp",
			},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(statefulSetMetricFamilies(nil, nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(statefulSetMetricFamilies(nil, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result for statefulset%d run:\n%s", i+1, err)
		}
	}
}
