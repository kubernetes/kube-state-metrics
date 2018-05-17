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
	"k8s.io/kube-state-metrics/pkg/collectors/testutils"
	"k8s.io/kube-state-metrics/pkg/options"
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
			# HELP kube_persistentvolume_labels Kubernetes labels converted to Prometheus labels.
			# TYPE kube_persistentvolume_labels gauge
			# HELP kube_persistentvolume_info Information about persistentvolume.
			# TYPE kube_persistentvolume_info gauge
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
						Name: "test-pv-pending",
					},
					Status: v1.PersistentVolumeStatus{
						Phase: v1.VolumePending,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pv-available",
					},
					Status: v1.PersistentVolumeStatus{
						Phase: v1.VolumeAvailable,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pv-bound",
					},
					Status: v1.PersistentVolumeStatus{
						Phase: v1.VolumeBound,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pv-released",
					},
					Status: v1.PersistentVolumeStatus{
						Phase: v1.VolumeReleased,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pv-failed",
					},
					Status: v1.PersistentVolumeStatus{
						Phase: v1.VolumeFailed,
					},
				},
			},
			want: metadata + `
					kube_persistentvolume_status_phase{persistentvolume="test-pv-available",phase="Available"} 1
					kube_persistentvolume_status_phase{persistentvolume="test-pv-available",phase="Bound"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-available",phase="Failed"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-available",phase="Pending"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-available",phase="Released"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-bound",phase="Available"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-bound",phase="Bound"} 1
					kube_persistentvolume_status_phase{persistentvolume="test-pv-bound",phase="Failed"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-bound",phase="Pending"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-bound",phase="Released"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-failed",phase="Available"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-failed",phase="Bound"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-failed",phase="Failed"} 1
					kube_persistentvolume_status_phase{persistentvolume="test-pv-failed",phase="Pending"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-failed",phase="Released"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Available"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Bound"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Failed"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Pending"} 1
					kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Released"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-released",phase="Available"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-released",phase="Bound"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-released",phase="Failed"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-released",phase="Pending"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-released",phase="Released"} 1
				`,
			metrics: []string{"kube_persistentvolume_status_phase"},
		},
		{
			pvs: []v1.PersistentVolume{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pv-pending",
					},
					Status: v1.PersistentVolumeStatus{
						Phase: v1.VolumePending,
					},
					Spec: v1.PersistentVolumeSpec{
						StorageClassName: "test",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pv-available",
					},
					Status: v1.PersistentVolumeStatus{
						Phase: v1.VolumeAvailable,
					},
				},
			},
			want: metadata + `
					kube_persistentvolume_info{persistentvolume="test-pv-available",storageclass=""} 1
					kube_persistentvolume_info{persistentvolume="test-pv-pending",storageclass="test"} 1
				`,
			metrics: []string{"kube_persistentvolume_info"},
		},
		{
			pvs: []v1.PersistentVolume{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-labeled-pv",
						Labels: map[string]string{
							"app": "mysql-server",
						},
					},
					Status: v1.PersistentVolumeStatus{
						Phase: v1.VolumePending,
					},
					Spec: v1.PersistentVolumeSpec{
						StorageClassName: "test",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-unlabeled-pv",
					},
					Status: v1.PersistentVolumeStatus{
						Phase: v1.VolumeAvailable,
					},
				},
			},
			want: metadata + `
					kube_persistentvolume_labels{persistentvolume="test-unlabeled-pv"} 1
					kube_persistentvolume_labels{label_app="mysql-server",persistentvolume="test-labeled-pv"} 1
				`,
			metrics: []string{"kube_persistentvolume_labels"},
		},
	}
	for _, c := range cases {
		dc := &persistentVolumeCollector{
			store: &mockPersistentVolumeStore{
				list: func() (v1.PersistentVolumeList, error) {
					return v1.PersistentVolumeList{Items: c.pvs}, nil
				},
			},
			opts: &options.Options{},
		}
		if err := testutils.GatherAndCompare(dc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
