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

package collectors

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

var (
	rs1Replicas int32 = 5
	rs2Replicas int32 = 0
)

type mockReplicaSetStore struct {
	f func() ([]v1beta1.ReplicaSet, error)
}

func (rs mockReplicaSetStore) List() (replicasets []v1beta1.ReplicaSet, err error) {
	return rs.f()
}

func TestReplicaSetCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
	  # HELP kube_replicaset_metadata_generation Sequence number representing a specific generation of the desired state.
		# TYPE kube_replicaset_metadata_generation gauge
		# HELP kube_replicaset_status_replicas The number of replicas per ReplicaSet.
		# TYPE kube_replicaset_status_replicas gauge
		# HELP kube_replicaset_status_fully_labeled_replicas The number of fully labeled replicas per ReplicaSet.
		# TYPE kube_replicaset_status_fully_labeled_replicas gauge
		# HELP kube_replicaset_status_ready_replicas The number of ready replicas per ReplicaSet.
		# TYPE kube_replicaset_status_ready_replicas gauge
		# HELP kube_replicaset_status_observed_generation The generation observed by the ReplicaSet controller.
		# TYPE kube_replicaset_status_observed_generation gauge
		# HELP kube_replicaset_spec_replicas Number of desired pods for a ReplicaSet.
		# TYPE kube_replicaset_spec_replicas gauge
	`
	cases := []struct {
		rss  []v1beta1.ReplicaSet
		want string
	}{
		{
			rss: []v1beta1.ReplicaSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "rs1",
						Namespace:  "ns1",
						Generation: 21,
					},
					Status: v1beta1.ReplicaSetStatus{
						Replicas:             5,
						FullyLabeledReplicas: 10,
						ReadyReplicas:        5,
						ObservedGeneration:   1,
					},
					Spec: v1beta1.ReplicaSetSpec{
						Replicas: &rs1Replicas,
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:       "rs2",
						Namespace:  "ns2",
						Generation: 14,
					},
					Status: v1beta1.ReplicaSetStatus{
						Replicas:             0,
						FullyLabeledReplicas: 5,
						ReadyReplicas:        0,
						ObservedGeneration:   5,
					},
					Spec: v1beta1.ReplicaSetSpec{
						Replicas: &rs2Replicas,
					},
				},
			},
			want: metadata + `
				kube_replicaset_metadata_generation{namespace="ns1",replicaset="rs1"} 21
				kube_replicaset_metadata_generation{namespace="ns2",replicaset="rs2"} 14
				kube_replicaset_status_replicas{namespace="ns1",replicaset="rs1"} 5
				kube_replicaset_status_replicas{namespace="ns2",replicaset="rs2"} 0
				kube_replicaset_status_observed_generation{namespace="ns1",replicaset="rs1"} 1
				kube_replicaset_status_observed_generation{namespace="ns2",replicaset="rs2"} 5
				kube_replicaset_status_fully_labeled_replicas{namespace="ns1",replicaset="rs1"} 10
				kube_replicaset_status_fully_labeled_replicas{namespace="ns2",replicaset="rs2"} 5
				kube_replicaset_status_ready_replicas{namespace="ns1",replicaset="rs1"} 5
				kube_replicaset_status_ready_replicas{namespace="ns2",replicaset="rs2"} 0
				kube_replicaset_spec_replicas{namespace="ns1",replicaset="rs1"} 5
				kube_replicaset_spec_replicas{namespace="ns2",replicaset="rs2"} 0
			`,
		},
	}
	for _, c := range cases {
		dc := &replicasetCollector{
			store: mockReplicaSetStore{
				f: func() ([]v1beta1.ReplicaSet, error) { return c.rss, nil },
			},
		}
		if err := gatherAndCompare(dc, c.want, nil); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
