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

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestRoleBindingStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)

	cases := []generateMetricsTestCase{
		{
			AllowAnnotationsList: []string{
				"app.k8s.io/owner",
			},
			AllowLabelsList: []string{
				"app",
			},
			Obj: &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "rolebinding1",
					Namespace:       "ns1",
					ResourceVersion: "BBBBB",
					Annotations: map[string]string{
						"app":              "mysql-server",
						"app.k8s.io/owner": "@foo",
					},
					Labels: map[string]string{
						"excluded": "me",
						"app":      "mysql-server",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "role",
				},
			},
			Want: `
				# HELP kube_rolebinding_annotations Kubernetes annotations converted to Prometheus labels.
				# HELP kube_rolebinding_labels Kubernetes labels converted to Prometheus labels.
				# HELP kube_rolebinding_info Information about rolebinding.
				# HELP kube_rolebinding_metadata_resource_version Resource version representing a specific version of the rolebinding.
				# TYPE kube_rolebinding_annotations gauge
				# TYPE kube_rolebinding_labels gauge
				# TYPE kube_rolebinding_info gauge
				# TYPE kube_rolebinding_metadata_resource_version gauge
				kube_rolebinding_annotations{annotation_app_k8s_io_owner="@foo",rolebinding="rolebinding1",namespace="ns1"} 1
				kube_rolebinding_labels{rolebinding="rolebinding1",label_app="mysql-server",namespace="ns1"} 1
				kube_rolebinding_info{rolebinding="rolebinding1",namespace="ns1",roleref_kind="Role",roleref_name="role"} 1
`,
			MetricNames: []string{
				"kube_rolebinding_annotations",
				"kube_rolebinding_labels",
				"kube_rolebinding_info",
				"kube_rolebinding_metadata_resource_version",
			},
		},
		{
			Obj: &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "rolebinding2",
					Namespace:         "ns2",
					CreationTimestamp: metav1StartTime,
					ResourceVersion:   "10596",
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "role",
				},
			},
			Want: `
				# HELP kube_rolebinding_created Unix creation timestamp
				# HELP kube_rolebinding_info Information about rolebinding.
				# HELP kube_rolebinding_metadata_resource_version Resource version representing a specific version of the rolebinding.
				# TYPE kube_rolebinding_created gauge
				# TYPE kube_rolebinding_info gauge
				# TYPE kube_rolebinding_metadata_resource_version gauge
				kube_rolebinding_info{rolebinding="rolebinding2",namespace="ns2",roleref_kind="Role",roleref_name="role"} 1
				kube_rolebinding_created{rolebinding="rolebinding2",namespace="ns2"} 1.501569018e+09
				kube_rolebinding_metadata_resource_version{rolebinding="rolebinding2",namespace="ns2"} 10596
				`,
			MetricNames: []string{"kube_rolebinding_info", "kube_rolebinding_created", "kube_rolebinding_metadata_resource_version"},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(roleBindingMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(roleBindingMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
