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

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockPersistentVolumeStore struct {
	list func() (v1.PersistentVolumeList, error)
}

func (ns mockPersistentVolumeStore) List() (v1.PersistentVolumeList, error) {
	return ns.list()
}

func TestPersistentVolumeCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_persistentvolume_status_phase The phase indicates if a volume is available, bound to a claim, or released by a claim.
		# TYPE kube_persistentvolume_status_phase gauge
	`
	cases := []struct {
		pvs     []v1.PersistentVolume
		metrics []string // which metrics should be checked
		want    string
	}{
		// Verify phase enumerations.
		{
			pvs: []v1.PersistentVolume{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "mongo-data",
						Namespace: "default",
					},
					Status: v1.PersistentVolumeStatus{
						Phase: v1.VolumePending,
					},
				},
			},
			want: metadata + `
				kube_persistentvolume_status_phase{namespace="default",persistentvolume="mongo-data",phase="Bound"} 0
				kube_persistentvolume_status_phase{namespace="default",persistentvolume="mongo-data",phase="Failed"} 0
				kube_persistentvolume_status_phase{namespace="default",persistentvolume="mongo-data",phase="Pending"} 1
				kube_persistentvolume_status_phase{namespace="default",persistentvolume="mongo-data",phase="Available"} 0
				kube_persistentvolume_status_phase{namespace="default",persistentvolume="mongo-data",phase="Released"} 0
				`,
			metrics: []string{"kube_persistentvolume_status_phase"},
		},
	}
	for _, c := range cases {
		dc := &persistentVolumeCollector{
			store: &mockPersistentVolumeStore{
				list: func() (v1.PersistentVolumeList, error) {
					return v1.PersistentVolumeList{Items: c.pvs}, nil
				},
			},
		}
		if err := gatherAndCompare(dc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
