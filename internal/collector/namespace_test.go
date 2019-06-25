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

package collector

import (
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/pkg/metric"
)

func TestNamespaceCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_namespace_created Unix creation timestamp
		# TYPE kube_namespace_created gauge
		# HELP kube_namespace_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_namespace_labels gauge
		# HELP kube_namespace_annotations Kubernetes annotations converted to Prometheus labels.
		# TYPE kube_namespace_annotations gauge
		# HELP kube_namespace_status_phase kubernetes namespace status phase.
		# TYPE kube_namespace_status_phase gauge
	`

	cases := []generateMetricsTestCase{
		{
			Obj: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nsActiveTest",
				},
				Spec: v1.NamespaceSpec{
					Finalizers: []v1.FinalizerName{v1.FinalizerKubernetes},
				},
				Status: v1.NamespaceStatus{
					Phase: v1.NamespaceActive,
				},
			},
			Want: `
				kube_namespace_labels{namespace="nsActiveTest"} 1
				kube_namespace_annotations{namespace="nsActiveTest"} 1
				kube_namespace_status_phase{namespace="nsActiveTest",phase="Active"} 1
				kube_namespace_status_phase{namespace="nsActiveTest",phase="Terminating"} 0
`,
		},
		{
			Obj: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nsTerminateTest",
				},
				Spec: v1.NamespaceSpec{
					Finalizers: []v1.FinalizerName{v1.FinalizerKubernetes},
				},
				Status: v1.NamespaceStatus{
					Phase: v1.NamespaceTerminating,
				},
			},
			Want: `
				kube_namespace_labels{namespace="nsTerminateTest"} 1
				kube_namespace_annotations{namespace="nsTerminateTest"} 1
				kube_namespace_status_phase{namespace="nsTerminateTest",phase="Active"} 0
				kube_namespace_status_phase{namespace="nsTerminateTest",phase="Terminating"} 1
`,
		},
		{

			Obj: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "ns1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Labels: map[string]string{
						"app": "example1",
					},
					Annotations: map[string]string{
						"app": "example1",
					},
				},
				Spec: v1.NamespaceSpec{
					Finalizers: []v1.FinalizerName{v1.FinalizerKubernetes},
				},
				Status: v1.NamespaceStatus{
					Phase: v1.NamespaceActive,
				},
			},
			Want: `
				kube_namespace_created{namespace="ns1"} 1.5e+09
				kube_namespace_labels{label_app="example1",namespace="ns1"} 1
				kube_namespace_annotations{annotation_app="example1",namespace="ns1"} 1
				kube_namespace_status_phase{namespace="ns1",phase="Active"} 1
				kube_namespace_status_phase{namespace="ns1",phase="Terminating"} 0
`,
		},
		{
			Obj: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns2",
					Labels: map[string]string{
						"app": "example2",
						"l2":  "label2",
					},
					Annotations: map[string]string{
						"app": "example2",
						"l2":  "label2",
					},
				},
				Spec: v1.NamespaceSpec{
					Finalizers: []v1.FinalizerName{v1.FinalizerKubernetes},
				},
				Status: v1.NamespaceStatus{
					Phase: v1.NamespaceActive,
				},
			},
			Want: `
				kube_namespace_labels{label_app="example2",label_l2="label2",namespace="ns2"} 1
				kube_namespace_annotations{annotation_app="example2",annotation_l2="label2",namespace="ns2"} 1
				kube_namespace_status_phase{namespace="ns2",phase="Active"} 1
				kube_namespace_status_phase{namespace="ns2",phase="Terminating"} 0
`,
		},
	}

	for i, c := range cases {
		c.Func = metric.ComposeMetricGenFuncs(namespaceMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
