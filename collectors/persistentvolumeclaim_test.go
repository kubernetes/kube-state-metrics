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

package collectors

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

type mockPersistentVolumeClaimStore struct {
	list func() (v1.PersistentVolumeClaimList, error)
}

func (ns mockPersistentVolumeClaimStore) List() (v1.PersistentVolumeClaimList, error) {
	return ns.list()
}

func TestPersistentVolumeClaimCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_persistentvolumeclaim_status_phase The phase the claim is currently in.
		# TYPE kube_persistentvolumeclaim_status_phase gauge
	`
	cases := []struct {
		pvcs    []v1.PersistentVolumeClaim
		metrics []string // which metrics should be checked
		want    string
	}{
		// Verify phase enumerations.
		{
			pvcs: []v1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mysql-data",
						Namespace: "default",
					},
					Status: v1.PersistentVolumeClaimStatus{
						Phase: v1.ClaimBound,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "prometheus-data",
						Namespace: "default",
					},
					Status: v1.PersistentVolumeClaimStatus{
						Phase: v1.ClaimPending,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mongo-data",
					},
					Status: v1.PersistentVolumeClaimStatus{
						Phase: v1.ClaimLost,
					},
				},
			},
			want: metadata + `
				kube_persistentvolumeclaim_status_phase{namespace="",persistentvolumeclaim="mongo-data",phase="Bound"} 0
				kube_persistentvolumeclaim_status_phase{namespace="",persistentvolumeclaim="mongo-data",phase="Lost"} 1
				kube_persistentvolumeclaim_status_phase{namespace="",persistentvolumeclaim="mongo-data",phase="Pending"} 0
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="mysql-data",phase="Bound"} 1
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="mysql-data",phase="Lost"} 0
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="mysql-data",phase="Pending"} 0
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="prometheus-data",phase="Bound"} 0
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="prometheus-data",phase="Lost"} 0
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="prometheus-data",phase="Pending"} 1
			`,
			metrics: []string{"kube_persistentvolumeclaim_status_phase"},
		},
	}
	for _, c := range cases {
		dc := &persistentVolumeClaimCollector{
			store: &mockPersistentVolumeClaimStore{
				list: func() (v1.PersistentVolumeClaimList, error) {
					return v1.PersistentVolumeClaimList{Items: c.pvcs}, nil
				},
			},
		}
		if err := gatherAndCompare(dc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
