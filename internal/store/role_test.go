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

func TestRoleStore(t *testing.T) {
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
			Obj: &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "role1",
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
			},
			Want: `
				# HELP kube_role_annotations Kubernetes annotations converted to Prometheus labels.
				# HELP kube_role_labels Kubernetes labels converted to Prometheus labels.
				# HELP kube_role_info Information about role.
				# HELP kube_role_metadata_resource_version Resource version representing a specific version of the role.
				# TYPE kube_role_annotations gauge
				# TYPE kube_role_labels gauge
				# TYPE kube_role_info gauge
				# TYPE kube_role_metadata_resource_version gauge
				kube_role_annotations{annotation_app_k8s_io_owner="@foo",role="role1",namespace="ns1"} 1
				kube_role_labels{role="role1",label_app="mysql-server",namespace="ns1"} 1
				kube_role_info{role="role1",namespace="ns1"} 1
`,
			MetricNames: []string{
				"kube_role_annotations",
				"kube_role_labels",
				"kube_role_info",
				"kube_role_metadata_resource_version",
			},
		},
		{
			Obj: &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "role2",
					Namespace:         "ns2",
					CreationTimestamp: metav1StartTime,
					ResourceVersion:   "10596",
				},
			},
			Want: `
				# HELP kube_role_created Unix creation timestamp
				# HELP kube_role_info Information about role.
				# HELP kube_role_metadata_resource_version Resource version representing a specific version of the role.
				# TYPE kube_role_created gauge
				# TYPE kube_role_info gauge
				# TYPE kube_role_metadata_resource_version gauge
				kube_role_info{role="role2",namespace="ns2"} 1
				kube_role_created{role="role2",namespace="ns2"} 1.501569018e+09
				kube_role_metadata_resource_version{role="role2",namespace="ns2"} 10596
				`,
			MetricNames: []string{"kube_role_info", "kube_role_created", "kube_role_metadata_resource_version"},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(roleMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(roleMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
