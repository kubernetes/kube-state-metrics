/*
Copyright 2022 The Kubernetes Authors All rights reserved.

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
	"k8s.io/utils/pointer"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestServiceAccountStore(t *testing.T) {
	cases := []generateMetricsTestCase{
		{
			Obj: &v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "serviceAccountName",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					DeletionTimestamp: &metav1.Time{Time: time.Unix(3000000000, 0)},
					Namespace:         "serviceAccountNS",
					UID:               "serviceAccountUID",
				},
				AutomountServiceAccountToken: pointer.Bool(true),
				Secrets: []v1.ObjectReference{
					{
						APIVersion: "v1",
						Kind:       "Secret",
						Name:       "secretName",
						Namespace:  "serviceAccountNS",
					},
				},
				ImagePullSecrets: []v1.LocalObjectReference{
					{
						Name: "imagePullSecretName",
					},
				},
			},
			Want: `
			# HELP kube_serviceaccount_info Information about a service account
			# HELP kube_serviceaccount_created Unix creation timestamp
			# HELP kube_serviceaccount_deleted Unix deletion timestamp
			# HELP kube_serviceaccount_secret Secret being referenced by a service account
			# HELP kube_serviceaccount_image_pull_secret Secret being referenced by a service account for the purpose of pulling images
			# TYPE kube_serviceaccount_info gauge
			# TYPE kube_serviceaccount_created gauge
			# TYPE kube_serviceaccount_deleted gauge
			# TYPE kube_serviceaccount_secret gauge
            # TYPE kube_serviceaccount_image_pull_secret gauge
			kube_serviceaccount_info{namespace="serviceAccountNS",serviceaccount="serviceAccountName",uid="serviceAccountUID",automount_token="true"} 1
			kube_serviceaccount_created{namespace="serviceAccountNS",serviceaccount="serviceAccountName",uid="serviceAccountUID"} 1.5e+09
			kube_serviceaccount_deleted{namespace="serviceAccountNS",serviceaccount="serviceAccountName",uid="serviceAccountUID"} 3e+09
			kube_serviceaccount_secret{namespace="serviceAccountNS",serviceaccount="serviceAccountName",uid="serviceAccountUID",name="secretName"} 1
			kube_serviceaccount_image_pull_secret{namespace="serviceAccountNS",serviceaccount="serviceAccountName",uid="serviceAccountUID",name="imagePullSecretName"} 1`,
			MetricNames: []string{
				"kube_serviceaccount_info",
				"kube_serviceaccount_created",
				"kube_serviceaccount_deleted",
				"kube_serviceaccount_secret",
				"kube_serviceaccount_image_pull_secret",
			},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(serviceAccountMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(serviceAccountMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
