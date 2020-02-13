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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/pkg/metric_generator"
)

var (
	rc1Replicas int32 = 5
	rc2Replicas int32
	rc3Replicas int32
)

func TestReplicationControllerStore(t *testing.T) {
	var trueValue = true

	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_replicationcontroller_created Unix creation timestamp
		# TYPE kube_replicationcontroller_created gauge
		# HELP kube_replicationcontroller_metadata_generation Sequence number representing a specific generation of the desired state.
		# TYPE kube_replicationcontroller_metadata_generation gauge
		# HELP kube_replicationcontroller_owner Information about the ReplicationController's owner.
		# TYPE kube_replicationcontroller_owner gauge
		# HELP kube_replicationcontroller_status_replicas The number of replicas per ReplicationController.
		# TYPE kube_replicationcontroller_status_replicas gauge
		# HELP kube_replicationcontroller_status_fully_labeled_replicas The number of fully labeled replicas per ReplicationController.
		# TYPE kube_replicationcontroller_status_fully_labeled_replicas gauge
		# HELP kube_replicationcontroller_status_available_replicas The number of available replicas per ReplicationController.
		# TYPE kube_replicationcontroller_status_available_replicas gauge
		# HELP kube_replicationcontroller_status_ready_replicas The number of ready replicas per ReplicationController.
		# TYPE kube_replicationcontroller_status_ready_replicas gauge
		# HELP kube_replicationcontroller_status_observed_generation The generation observed by the ReplicationController controller.
		# TYPE kube_replicationcontroller_status_observed_generation gauge
		# HELP kube_replicationcontroller_spec_replicas Number of desired pods for a ReplicationController.
		# TYPE kube_replicationcontroller_spec_replicas gauge
	`
	cases := []generateMetricsTestCase{
		{
			Obj: &v1.ReplicationController{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "rc1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "ns1",
					Generation:        21,
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "DeploymentConfig",
							Name:       "dc-name",
							Controller: &trueValue,
						},
					},
				},
				Status: v1.ReplicationControllerStatus{
					Replicas:             5,
					FullyLabeledReplicas: 10,
					ReadyReplicas:        5,
					AvailableReplicas:    3,
					ObservedGeneration:   1,
				},
				Spec: v1.ReplicationControllerSpec{
					Replicas: &rc1Replicas,
				},
			},
			Want: metadata + `
				kube_replicationcontroller_created{namespace="ns1",replicationcontroller="rc1"} 1.5e+09
				kube_replicationcontroller_metadata_generation{namespace="ns1",replicationcontroller="rc1"} 21
				kube_replicationcontroller_owner{namespace="ns1",owner_is_controller="true",owner_kind="DeploymentConfig",owner_name="dc-name",replicationcontroller="rc1"} 1
				kube_replicationcontroller_status_replicas{namespace="ns1",replicationcontroller="rc1"} 5
				kube_replicationcontroller_status_observed_generation{namespace="ns1",replicationcontroller="rc1"} 1
				kube_replicationcontroller_status_fully_labeled_replicas{namespace="ns1",replicationcontroller="rc1"} 10
				kube_replicationcontroller_status_ready_replicas{namespace="ns1",replicationcontroller="rc1"} 5
				kube_replicationcontroller_status_available_replicas{namespace="ns1",replicationcontroller="rc1"} 3
				kube_replicationcontroller_spec_replicas{namespace="ns1",replicationcontroller="rc1"} 5
`,
		},
		{
			Obj: &v1.ReplicationController{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "rc2",
					Namespace:  "ns2",
					Generation: 14,
				},
				Status: v1.ReplicationControllerStatus{
					Replicas:             0,
					FullyLabeledReplicas: 5,
					ReadyReplicas:        0,
					AvailableReplicas:    0,
					ObservedGeneration:   5,
				},
				Spec: v1.ReplicationControllerSpec{
					Replicas: &rc2Replicas,
				},
			},
			Want: metadata + `
				kube_replicationcontroller_metadata_generation{namespace="ns2",replicationcontroller="rc2"} 14
				kube_replicationcontroller_owner{namespace="ns2",owner_is_controller="<none>",owner_kind="<none>",owner_name="<none>",replicationcontroller="rc2"} 1
				kube_replicationcontroller_status_replicas{namespace="ns2",replicationcontroller="rc2"} 0
				kube_replicationcontroller_status_observed_generation{namespace="ns2",replicationcontroller="rc2"} 5
				kube_replicationcontroller_status_fully_labeled_replicas{namespace="ns2",replicationcontroller="rc2"} 5
				kube_replicationcontroller_status_ready_replicas{namespace="ns2",replicationcontroller="rc2"} 0
				kube_replicationcontroller_status_available_replicas{namespace="ns2",replicationcontroller="rc2"} 0
				kube_replicationcontroller_spec_replicas{namespace="ns2",replicationcontroller="rc2"} 0
`,
		},
		{
			Obj: &v1.ReplicationController{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "rc3",
					Namespace:  "ns3",
					Generation: 5,
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "DeploymentConfig",
							Name:       "dc-test",
							Controller: nil,
						},
					},
				},
				Status: v1.ReplicationControllerStatus{
					Replicas:             1,
					FullyLabeledReplicas: 5,
					ReadyReplicas:        2,
					AvailableReplicas:    1,
					ObservedGeneration:   1,
				},
				Spec: v1.ReplicationControllerSpec{
					Replicas: &rc3Replicas,
				},
			},
			Want: metadata + `
				kube_replicationcontroller_metadata_generation{namespace="ns3",replicationcontroller="rc3"} 5
				kube_replicationcontroller_owner{namespace="ns3",owner_is_controller="false",owner_kind="DeploymentConfig",owner_name="dc-test",replicationcontroller="rc3"} 1
				kube_replicationcontroller_status_replicas{namespace="ns3",replicationcontroller="rc3"} 1
				kube_replicationcontroller_status_observed_generation{namespace="ns3",replicationcontroller="rc3"} 1
				kube_replicationcontroller_status_fully_labeled_replicas{namespace="ns3",replicationcontroller="rc3"} 5
				kube_replicationcontroller_status_ready_replicas{namespace="ns3",replicationcontroller="rc3"} 2
				kube_replicationcontroller_status_available_replicas{namespace="ns3",replicationcontroller="rc3"} 1
				kube_replicationcontroller_spec_replicas{namespace="ns3",replicationcontroller="rc3"} 0
`,
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(replicationControllerMetricFamilies)
		c.Headers = generator.ExtractMetricFamilyHeaders(replicationControllerMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
