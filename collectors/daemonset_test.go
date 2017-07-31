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

type mockDaemonSetStore struct {
	f func() ([]v1beta1.DaemonSet, error)
}

func (ds mockDaemonSetStore) List() (daemonsets []v1beta1.DaemonSet, err error) {
	return ds.f()
}

func TestDaemonSetCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
	  # HELP kube_daemonset_metadata_generation Sequence number representing a specific generation of the desired state.
		# TYPE kube_daemonset_metadata_generation gauge
		# HELP kube_daemonset_status_current_number_scheduled The number of nodes running at least one daemon pod and are supposed to.
		# TYPE kube_daemonset_status_current_number_scheduled gauge
		# HELP kube_daemonset_status_number_misscheduled The number of nodes running a daemon pod but are not supposed to.
		# TYPE kube_daemonset_status_number_misscheduled gauge
		# HELP kube_daemonset_status_desired_number_scheduled The number of nodes that should be running the daemon pod.
		# TYPE kube_daemonset_status_desired_number_scheduled gauge
		# HELP kube_daemonset_status_number_ready The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and ready.
		# TYPE kube_daemonset_status_number_ready gauge
	`
	cases := []struct {
		dss  []v1beta1.DaemonSet
		want string
	}{
		{
			dss: []v1beta1.DaemonSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "ds1",
						Namespace:  "ns1",
						Generation: 21,
					},
					Status: v1beta1.DaemonSetStatus{
						CurrentNumberScheduled: 15,
						NumberMisscheduled:     10,
						DesiredNumberScheduled: 5,
						NumberReady:            5,
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:       "ds2",
						Namespace:  "ns2",
						Generation: 14,
					},
					Status: v1beta1.DaemonSetStatus{
						CurrentNumberScheduled: 10,
						NumberMisscheduled:     5,
						DesiredNumberScheduled: 0,
						NumberReady:            0,
					},
				},
			},
			want: metadata + `
				kube_daemonset_metadata_generation{namespace="ns1",daemonset="ds1"} 21
				kube_daemonset_metadata_generation{namespace="ns2",daemonset="ds2"} 14
				kube_daemonset_status_current_number_scheduled{namespace="ns1",daemonset="ds1"} 15
				kube_daemonset_status_current_number_scheduled{namespace="ns2",daemonset="ds2"} 10
				kube_daemonset_status_number_misscheduled{namespace="ns1",daemonset="ds1"} 10
				kube_daemonset_status_number_misscheduled{namespace="ns2",daemonset="ds2"} 5
				kube_daemonset_status_desired_number_scheduled{namespace="ns1",daemonset="ds1"} 5
				kube_daemonset_status_desired_number_scheduled{namespace="ns2",daemonset="ds2"} 0
				kube_daemonset_status_number_ready{namespace="ns1",daemonset="ds1"} 5
				kube_daemonset_status_number_ready{namespace="ns2",daemonset="ds2"} 0
			`,
		},
	}
	for _, c := range cases {
		dc := &daemonsetCollector{
			store: mockDaemonSetStore{
				f: func() ([]v1beta1.DaemonSet, error) { return c.dss, nil },
			},
		}
		if err := gatherAndCompare(dc, c.want, nil); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
