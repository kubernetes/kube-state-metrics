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

func TestClusterRoleBindingStore(t *testing.T) {
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
			Obj: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "clusterrolebinding1",
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
				# HELP kube_clusterrolebinding_annotations Kubernetes annotations converted to Prometheus labels.
				# HELP kube_clusterrolebinding_labels Kubernetes labels converted to Prometheus labels.
				# HELP kube_clusterrolebinding_info Information about clusterrolebinding.
				# HELP kube_clusterrolebinding_metadata_resource_version Resource version representing a specific version of the clusterrolebinding.
				# TYPE kube_clusterrolebinding_annotations gauge
				# TYPE kube_clusterrolebinding_labels gauge
				# TYPE kube_clusterrolebinding_info gauge
				# TYPE kube_clusterrolebinding_metadata_resource_version gauge
				kube_clusterrolebinding_annotations{annotation_app_k8s_io_owner="@foo",clusterrolebinding="clusterrolebinding1"} 1
				kube_clusterrolebinding_labels{clusterrolebinding="clusterrolebinding1",label_app="mysql-server"} 1
				kube_clusterrolebinding_info{clusterrolebinding="clusterrolebinding1",roleref_kind="Role",roleref_name="role"} 1
`,
			MetricNames: []string{
				"kube_clusterrolebinding_annotations",
				"kube_clusterrolebinding_labels",
				"kube_clusterrolebinding_info",
				"kube_clusterrolebinding_metadata_resource_version",
			},
		},
		{
			Obj: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "clusterrolebinding2",
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
				# HELP kube_clusterrolebinding_created Unix creation timestamp
				# HELP kube_clusterrolebinding_info Information about clusterrolebinding.
				# HELP kube_clusterrolebinding_metadata_resource_version Resource version representing a specific version of the clusterrolebinding.
				# TYPE kube_clusterrolebinding_created gauge
				# TYPE kube_clusterrolebinding_info gauge
				# TYPE kube_clusterrolebinding_metadata_resource_version gauge
				kube_clusterrolebinding_info{clusterrolebinding="clusterrolebinding2",roleref_kind="Role",roleref_name="role"} 1
				kube_clusterrolebinding_created{clusterrolebinding="clusterrolebinding2"} 1.501569018e+09
				kube_clusterrolebinding_metadata_resource_version{clusterrolebinding="clusterrolebinding2"} 10596
				`,
			MetricNames: []string{"kube_clusterrolebinding_info", "kube_clusterrolebinding_created", "kube_clusterrolebinding_metadata_resource_version"},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(clusterRoleBindingMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(clusterRoleBindingMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
