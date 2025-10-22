/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	depl1Replicas int32 = 200
	depl2Replicas int32 = 5
	depl3Replicas int32 = 1
	depl4Replicas int32 = 10

	depl1MaxUnavailable = intstr.FromInt(10)
	depl2MaxUnavailable = intstr.FromString("25%")

	depl1MaxSurge = intstr.FromInt(10)
	depl2MaxSurge = intstr.FromString("20%")
)

func TestDeploymentStore(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_deployment_owner Information about the Deployment's owner.
		# TYPE kube_deployment_owner gauge
		# HELP kube_deployment_annotations Kubernetes annotations converted to Prometheus labels.
		# TYPE kube_deployment_annotations gauge
		# HELP kube_deployment_created [STABLE] Unix creation timestamp
		# TYPE kube_deployment_created gauge
		# HELP kube_deployment_metadata_generation [STABLE] Sequence number representing a specific generation of the desired state.
		# TYPE kube_deployment_metadata_generation gauge
		# HELP kube_deployment_spec_paused [STABLE] Whether the deployment is paused and will not be processed by the deployment controller.
		# TYPE kube_deployment_spec_paused gauge
		# HELP kube_deployment_spec_replicas [STABLE] Number of desired pods for a deployment.
		# TYPE kube_deployment_spec_replicas gauge
		# HELP kube_deployment_status_replicas [STABLE] The number of replicas per deployment.
		# TYPE kube_deployment_status_replicas gauge
		# HELP kube_deployment_status_replicas_ready [STABLE] The number of ready replicas per deployment.
		# TYPE kube_deployment_status_replicas_ready gauge
        # HELP kube_deployment_status_terminating_replicas The number of terminating replicas per deployment.
        # TYPE kube_deployment_status_terminating_replicas gauge
		# HELP kube_deployment_status_replicas_available [STABLE] The number of available replicas per deployment.
		# TYPE kube_deployment_status_replicas_available gauge
		# HELP kube_deployment_status_replicas_unavailable [STABLE] The number of unavailable replicas per deployment.
		# TYPE kube_deployment_status_replicas_unavailable gauge
		# HELP kube_deployment_status_replicas_updated [STABLE] The number of updated replicas per deployment.
		# TYPE kube_deployment_status_replicas_updated gauge
		# HELP kube_deployment_status_observed_generation [STABLE] The generation observed by the deployment controller.
		# TYPE kube_deployment_status_observed_generation gauge
		# HELP kube_deployment_status_condition [STABLE] The current status conditions of a deployment.
		# TYPE kube_deployment_status_condition gauge
		# HELP kube_deployment_spec_strategy_rollingupdate_max_unavailable [STABLE] Maximum number of unavailable replicas during a rolling update of a deployment.
		# TYPE kube_deployment_spec_strategy_rollingupdate_max_unavailable gauge
		# HELP kube_deployment_spec_strategy_rollingupdate_max_surge [STABLE] Maximum number of replicas that can be scheduled above the desired number of replicas during a rolling update of a deployment.
		# TYPE kube_deployment_spec_strategy_rollingupdate_max_surge gauge
		# HELP kube_deployment_labels [STABLE] Kubernetes labels converted to Prometheus labels.
		# TYPE kube_deployment_labels gauge
		# HELP kube_deployment_deletion_timestamp Unix deletion timestamp
		# TYPE kube_deployment_deletion_timestamp gauge
	`
	cases := []generateMetricsTestCase{
		{
			AllowAnnotationsList: []string{"company.io/team"},
			Obj: &v1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "depl1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "ns1",
					Annotations: map[string]string{
						"company.io/team": "my-brilliant-team",
					},
					Labels: map[string]string{
						"app": "example1",
					},
					Generation: 21,
				},
				Status: v1.DeploymentStatus{
					Replicas:            15,
					ReadyReplicas:       10,
					AvailableReplicas:   10,
					UnavailableReplicas: 5,
					UpdatedReplicas:     2,
					TerminatingReplicas: ptr.To[int32](3),
					ObservedGeneration:  111,
					Conditions: []v1.DeploymentCondition{
						{Type: v1.DeploymentAvailable, Status: corev1.ConditionTrue, Reason: "MinimumReplicasAvailable"},
						{Type: v1.DeploymentProgressing, Status: corev1.ConditionTrue, Reason: "NewReplicaSetAvailable"},
					},
				},
				Spec: v1.DeploymentSpec{
					Replicas: &depl1Replicas,
					Strategy: v1.DeploymentStrategy{
						RollingUpdate: &v1.RollingUpdateDeployment{
							MaxUnavailable: &depl1MaxUnavailable,
							MaxSurge:       &depl1MaxSurge,
						},
					},
				},
			},
			Want: metadata + `
        kube_deployment_annotations{annotation_company_io_team="my-brilliant-team",deployment="depl1",namespace="ns1"} 1
        kube_deployment_created{deployment="depl1",namespace="ns1"} 1.5e+09
        kube_deployment_owner{deployment="depl1",namespace="ns1",owner_kind="",owner_name=""} 1
        kube_deployment_metadata_generation{deployment="depl1",namespace="ns1"} 21
        kube_deployment_spec_paused{deployment="depl1",namespace="ns1"} 0
        kube_deployment_spec_replicas{deployment="depl1",namespace="ns1"} 200
        kube_deployment_spec_strategy_rollingupdate_max_surge{deployment="depl1",namespace="ns1"} 10
        kube_deployment_spec_strategy_rollingupdate_max_unavailable{deployment="depl1",namespace="ns1"} 10
        kube_deployment_status_observed_generation{deployment="depl1",namespace="ns1"} 111
        kube_deployment_status_replicas_available{deployment="depl1",namespace="ns1"} 10
        kube_deployment_status_replicas_unavailable{deployment="depl1",namespace="ns1"} 5
        kube_deployment_status_replicas_updated{deployment="depl1",namespace="ns1"} 2
        kube_deployment_status_replicas{deployment="depl1",namespace="ns1"} 15
        kube_deployment_status_replicas_ready{deployment="depl1",namespace="ns1"} 10
        kube_deployment_status_terminating_replicas{deployment="depl1",namespace="ns1"} 3
        kube_deployment_status_condition{condition="Available",deployment="depl1",namespace="ns1",reason="MinimumReplicasAvailable",status="true"} 1
        kube_deployment_status_condition{condition="Available",deployment="depl1",namespace="ns1",reason="MinimumReplicasAvailable",status="false"} 0
        kube_deployment_status_condition{condition="Available",deployment="depl1",namespace="ns1",reason="MinimumReplicasAvailable",status="unknown"} 0
        kube_deployment_status_condition{condition="Progressing",deployment="depl1",namespace="ns1",reason="NewReplicaSetAvailable",status="true"} 1
        kube_deployment_status_condition{condition="Progressing",deployment="depl1",namespace="ns1",reason="NewReplicaSetAvailable",status="false"} 0
        kube_deployment_status_condition{condition="Progressing",deployment="depl1",namespace="ns1",reason="NewReplicaSetAvailable",status="unknown"} 0
`,
		},
		{
			Obj: &v1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "depl2",
					Namespace: "ns2",
					Labels: map[string]string{
						"app": "example2",
					},
					Generation: 14,
				},
				Status: v1.DeploymentStatus{
					Replicas:            10,
					ReadyReplicas:       5,
					AvailableReplicas:   5,
					UnavailableReplicas: 0,
					UpdatedReplicas:     1,
					TerminatingReplicas: nil,
					ObservedGeneration:  1111,
					Conditions: []v1.DeploymentCondition{
						{Type: v1.DeploymentAvailable, Status: corev1.ConditionFalse, Reason: "MinimumReplicasUnavailable"},
						{Type: v1.DeploymentProgressing, Status: corev1.ConditionFalse, Reason: "ProgressDeadlineExceeded"},
						{Type: v1.DeploymentReplicaFailure, Status: corev1.ConditionTrue, Reason: "ReplicaSetCreateError"},
					},
				},
				Spec: v1.DeploymentSpec{
					Paused:   true,
					Replicas: &depl2Replicas,
					Strategy: v1.DeploymentStrategy{
						RollingUpdate: &v1.RollingUpdateDeployment{
							MaxUnavailable: &depl2MaxUnavailable,
							MaxSurge:       &depl2MaxSurge,
						},
					},
				},
			},
			Want: metadata + `
        kube_deployment_metadata_generation{deployment="depl2",namespace="ns2"} 14
        kube_deployment_owner{deployment="depl2",namespace="ns2",owner_kind="",owner_name=""} 1
        kube_deployment_spec_paused{deployment="depl2",namespace="ns2"} 1
        kube_deployment_spec_replicas{deployment="depl2",namespace="ns2"} 5
        kube_deployment_spec_strategy_rollingupdate_max_surge{deployment="depl2",namespace="ns2"} 1
        kube_deployment_spec_strategy_rollingupdate_max_unavailable{deployment="depl2",namespace="ns2"} 1
        kube_deployment_status_observed_generation{deployment="depl2",namespace="ns2"} 1111
        kube_deployment_status_replicas_available{deployment="depl2",namespace="ns2"} 5
        kube_deployment_status_replicas_unavailable{deployment="depl2",namespace="ns2"} 0
        kube_deployment_status_replicas_updated{deployment="depl2",namespace="ns2"} 1
        kube_deployment_status_replicas{deployment="depl2",namespace="ns2"} 10
        kube_deployment_status_replicas_ready{deployment="depl2",namespace="ns2"} 5
        kube_deployment_status_condition{condition="Available",deployment="depl2",namespace="ns2",reason="MinimumReplicasUnavailable",status="true"} 0
        kube_deployment_status_condition{condition="Available",deployment="depl2",namespace="ns2",reason="MinimumReplicasUnavailable",status="false"} 1
        kube_deployment_status_condition{condition="Available",deployment="depl2",namespace="ns2",reason="MinimumReplicasUnavailable",status="unknown"} 0
        kube_deployment_status_condition{condition="Progressing",deployment="depl2",namespace="ns2",reason="ProgressDeadlineExceeded",status="true"} 0
        kube_deployment_status_condition{condition="Progressing",deployment="depl2",namespace="ns2",reason="ProgressDeadlineExceeded",status="false"} 1
        kube_deployment_status_condition{condition="Progressing",deployment="depl2",namespace="ns2",reason="ProgressDeadlineExceeded",status="unknown"} 0
        kube_deployment_status_condition{condition="ReplicaFailure",deployment="depl2",namespace="ns2",reason="ReplicaSetCreateError",status="true"} 1
        kube_deployment_status_condition{condition="ReplicaFailure",deployment="depl2",namespace="ns2",reason="ReplicaSetCreateError",status="false"} 0
        kube_deployment_status_condition{condition="ReplicaFailure",deployment="depl2",namespace="ns2",reason="ReplicaSetCreateError",status="unknown"} 0
`,
		},
		{
			Obj: &v1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "depl3",
					Namespace: "ns3",
				},
				Status: v1.DeploymentStatus{
					Conditions: []v1.DeploymentCondition{
						{Type: v1.DeploymentAvailable, Status: corev1.ConditionFalse, Reason: "ThisReasonIsNotAllowed"},
						{Type: v1.DeploymentProgressing, Status: corev1.ConditionTrue},
					},
				},
				Spec: v1.DeploymentSpec{
					Replicas: &depl3Replicas,
				},
			},
			Want: metadata + `
        kube_deployment_metadata_generation{deployment="depl3",namespace="ns3"} 0
        kube_deployment_owner{deployment="depl3",namespace="ns3",owner_kind="",owner_name=""} 1
        kube_deployment_spec_paused{deployment="depl3",namespace="ns3"} 0
        kube_deployment_spec_replicas{deployment="depl3",namespace="ns3"} 1
        kube_deployment_status_condition{condition="Available",deployment="depl3",namespace="ns3",reason="unknown",status="true"} 0
        kube_deployment_status_condition{condition="Available",deployment="depl3",namespace="ns3",reason="unknown",status="false"} 1
        kube_deployment_status_condition{condition="Available",deployment="depl3",namespace="ns3",reason="unknown",status="unknown"} 0
        kube_deployment_status_observed_generation{deployment="depl3",namespace="ns3"} 0
        kube_deployment_status_replicas{deployment="depl3",namespace="ns3"} 0
        kube_deployment_status_replicas_available{deployment="depl3",namespace="ns3"} 0
        kube_deployment_status_replicas_ready{deployment="depl3",namespace="ns3"} 0
        kube_deployment_status_replicas_unavailable{deployment="depl3",namespace="ns3"} 0
        kube_deployment_status_replicas_updated{deployment="depl3",namespace="ns3"} 0
	    kube_deployment_status_condition{condition="Progressing",deployment="depl3",namespace="ns3",reason="",status="false"} 0
        kube_deployment_status_condition{condition="Progressing",deployment="depl3",namespace="ns3",reason="",status="true"} 1
        kube_deployment_status_condition{condition="Progressing",deployment="depl3",namespace="ns3",reason="",status="unknown"} 0
`,
		},
		{
			Obj: &v1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "deployment-terminating",
					Namespace:         "ns4",
					CreationTimestamp: metav1.Time{Time: time.Unix(1600000000, 0)},
					DeletionTimestamp: &metav1.Time{Time: time.Unix(1800000000, 0)},
					Labels: map[string]string{
						"app": "example4",
					},
					Generation: 22,
				},
				Spec: v1.DeploymentSpec{
					Paused:   true,
					Replicas: &depl4Replicas,
				},
			},
			Want: `
			    # HELP kube_deployment_deletion_timestamp Unix deletion timestamp
			    # TYPE kube_deployment_deletion_timestamp gauge
					kube_deployment_deletion_timestamp{deployment="deployment-terminating",namespace="ns4"} 1.8e+09`,
			MetricNames: []string{"kube_deployment_deletion_timestamp"},
		},
		{
			Obj: &v1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deployment-with-owner",
					Namespace: "ns5",
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "Application",
							Name:       "my-app",
							Controller: &[]bool{true}[0],
						},
					},
				},
				Spec: v1.DeploymentSpec{
					Replicas: &depl1Replicas,
				},
			},
			Want: metadata + `
				kube_deployment_owner{deployment="deployment-with-owner",namespace="ns5",owner_kind="Application",owner_name="my-app"} 1
				kube_deployment_metadata_generation{deployment="deployment-with-owner",namespace="ns5"} 0
				kube_deployment_spec_paused{deployment="deployment-with-owner",namespace="ns5"} 0
				kube_deployment_spec_replicas{deployment="deployment-with-owner",namespace="ns5"} 200
				kube_deployment_status_observed_generation{deployment="deployment-with-owner",namespace="ns5"} 0
				kube_deployment_status_replicas{deployment="deployment-with-owner",namespace="ns5"} 0
				kube_deployment_status_replicas_available{deployment="deployment-with-owner",namespace="ns5"} 0
				kube_deployment_status_replicas_ready{deployment="deployment-with-owner",namespace="ns5"} 0
				kube_deployment_status_replicas_unavailable{deployment="deployment-with-owner",namespace="ns5"} 0
				kube_deployment_status_replicas_updated{deployment="deployment-with-owner",namespace="ns5"} 0
			`,
		},
		{
			Obj: &v1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deployment-without-owner",
					Namespace: "ns5",
				},
				Spec: v1.DeploymentSpec{
					Replicas: &depl1Replicas,
				},
			},
			Want: metadata + `
				kube_deployment_owner{deployment="deployment-without-owner",namespace="ns5",owner_kind="",owner_name=""} 1
				kube_deployment_metadata_generation{deployment="deployment-without-owner",namespace="ns5"} 0
				kube_deployment_spec_paused{deployment="deployment-without-owner",namespace="ns5"} 0
				kube_deployment_spec_replicas{deployment="deployment-without-owner",namespace="ns5"} 200
				kube_deployment_status_observed_generation{deployment="deployment-without-owner",namespace="ns5"} 0
				kube_deployment_status_replicas{deployment="deployment-without-owner",namespace="ns5"} 0
				kube_deployment_status_replicas_available{deployment="deployment-without-owner",namespace="ns5"} 0
				kube_deployment_status_replicas_ready{deployment="deployment-without-owner",namespace="ns5"} 0
				kube_deployment_status_replicas_unavailable{deployment="deployment-without-owner",namespace="ns5"} 0
				kube_deployment_status_replicas_updated{deployment="deployment-without-owner",namespace="ns5"} 0
			`,
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(deploymentMetricFamilies(c.AllowAnnotationsList, nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(deploymentMetricFamilies(c.AllowAnnotationsList, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
