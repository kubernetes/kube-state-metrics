/*
Copyright 2012 The Kubernetes Authors All rights reserved.

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

func TestClusterRoleStore(t *testing.T) {
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
			Obj: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "role1",
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
			},
			Want: `
				# HELP kube_clusterrole_annotations Kubernetes annotations converted to Prometheus labels.
				# HELP kube_clusterrole_labels Kubernetes labels converted to Prometheus labels.
				# HELP kube_clusterrole_info Information about cluster role.
				# HELP kube_clusterrole_metadata_resource_version Resource version representing a specific version of the cluster role.
				# TYPE kube_clusterrole_annotations gauge
				# TYPE kube_clusterrole_labels gauge
				# TYPE kube_clusterrole_info gauge
				# TYPE kube_clusterrole_metadata_resource_version gauge
				kube_clusterrole_annotations{annotation_app_k8s_io_owner="@foo",clusterrole="role1"} 1
				kube_clusterrole_labels{clusterrole="role1",label_app="mysql-server"} 1
				kube_clusterrole_info{clusterrole="role1"} 1
`,
			MetricNames: []string{
				"kube_clusterrole_annotations",
				"kube_clusterrole_labels",
				"kube_clusterrole_info",
				"kube_clusterrole_metadata_resource_version",
			},
		},
		{
			Obj: &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "role2",
					CreationTimestamp: metav1StartTime,
					ResourceVersion:   "10596",
				},
			},
			Want: `
				# HELP kube_clusterrole_created Unix creation timestamp
				# HELP kube_clusterrole_info Information about cluster role.
				# HELP kube_clusterrole_metadata_resource_version Resource version representing a specific version of the cluster role.
				# TYPE kube_clusterrole_created gauge
				# TYPE kube_clusterrole_info gauge
				# TYPE kube_clusterrole_metadata_resource_version gauge
				kube_clusterrole_info{clusterrole="role2"} 1
				kube_clusterrole_created{clusterrole="role2"} 1.501569018e+09
				kube_clusterrole_metadata_resource_version{clusterrole="role2"} 10596
				`,
			MetricNames: []string{"kube_clusterrole_info", "kube_clusterrole_created", "kube_clusterrole_metadata_resource_version"},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(clusterRoleMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(clusterRoleMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
