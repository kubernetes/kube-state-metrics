/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/pkg/metric_generator"
)

func TestVolumeAttachmentStore(t *testing.T) {
	const metadata = `
		# HELP kube_volumeattachment_created Unix creation timestamp
        # HELP kube_volumeattachment_info Information about volumeattachment.
        # HELP kube_volumeattachment_labels Kubernetes labels converted to Prometheus labels.
        # HELP kube_volumeattachment_spec_source_persistentvolume PersistentVolume source reference.
        # HELP kube_volumeattachment_status_attached Information about volumeattachment.
        # HELP kube_volumeattachment_status_attachment_metadata volumeattachment metadata.
        # TYPE kube_volumeattachment_created gauge
        # TYPE kube_volumeattachment_info gauge
        # TYPE kube_volumeattachment_labels gauge
        # TYPE kube_volumeattachment_spec_source_persistentvolume gauge
        # TYPE kube_volumeattachment_status_attached gauge
        # TYPE kube_volumeattachment_status_attachment_metadata gauge
	`

	var (
		volumename = "pvc-44f6ff3f-ba9b-49c4-9b95-8b01c4bd4bab"
		cases      = []generateMetricsTestCase{
			{
				Obj: &storagev1.VolumeAttachment{
					ObjectMeta: metav1.ObjectMeta{
						Generation: 2,
						Name:       "csi-5ff16a1ad085261021e21c6cb3a6defb979a8794f25a4f90f6285664cff37224",
						Labels: map[string]string{
							"app": "foobar",
						},
					},
					Spec: storagev1.VolumeAttachmentSpec{
						Attacher: "cinder.csi.openstack.org",
						NodeName: "node1",
						Source: storagev1.VolumeAttachmentSource{
							PersistentVolumeName: &volumename,
							InlineVolumeSpec:     nil,
						},
					},
					Status: storagev1.VolumeAttachmentStatus{
						Attached: true,
						AttachmentMetadata: map[string]string{
							"DevicePath": "/dev/sdd",
						},
					},
				},
				Want: metadata + `
		        kube_volumeattachment_info{attacher="cinder.csi.openstack.org",node="node1",volumeattachment="csi-5ff16a1ad085261021e21c6cb3a6defb979a8794f25a4f90f6285664cff37224"} 1
        		kube_volumeattachment_labels{label_app="foobar",volumeattachment="csi-5ff16a1ad085261021e21c6cb3a6defb979a8794f25a4f90f6285664cff37224"} 1
		        kube_volumeattachment_spec_source_persistentvolume{volumeattachment="csi-5ff16a1ad085261021e21c6cb3a6defb979a8794f25a4f90f6285664cff37224",volumename="pvc-44f6ff3f-ba9b-49c4-9b95-8b01c4bd4bab"} 1
		        kube_volumeattachment_status_attached{volumeattachment="csi-5ff16a1ad085261021e21c6cb3a6defb979a8794f25a4f90f6285664cff37224"} 1
		        kube_volumeattachment_status_attachment_metadata{metadata_DevicePath="/dev/sdd",volumeattachment="csi-5ff16a1ad085261021e21c6cb3a6defb979a8794f25a4f90f6285664cff37224"} 1
			`,
				MetricNames: []string{
					"kube_volumeattachment_labels",
					"kube_volumeattachment_info",
					"kube_volumeattachment_created",
					"kube_volumeattachment_spec_source_persistentvolume",
					"kube_volumeattachment_status_attached",
					"kube_volumeattachment_status_attachment_metadata",
				},
			},
		}
	)
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(volumeAttachmentMetricFamilies)
		c.Headers = generator.ExtractMetricFamilyHeaders(volumeAttachmentMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
