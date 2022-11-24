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

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestIngressClassStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)

	cases := []generateMetricsTestCase{
		{
			Obj: &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test_ingressclass-info",
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "controller",
				},
			},
			Want: `
					# HELP kube_ingressclass_info Information about ingressclass.
					# TYPE kube_ingressclass_info gauge
					kube_ingressclass_info{ingressclass="test_ingressclass-info",controller="controller"} 1
				`,
			MetricNames: []string{
				"kube_ingressclass_info",
			},
		},
		{
			Obj: &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test_kube_ingressclass-created",
					CreationTimestamp: metav1StartTime,
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "controller",
				},
			},
			Want: `
					# HELP kube_ingressclass_created Unix creation timestamp
					# TYPE kube_ingressclass_created gauge
					kube_ingressclass_created{ingressclass="test_kube_ingressclass-created"} 1.501569018e+09
				`,
			MetricNames: []string{
				"kube_ingressclass_created",
			},
		},
		{
			AllowAnnotationsList: []string{
				"ingressclass.kubernetes.io/is-default-class",
			},
			Obj: &networkingv1.IngressClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test_ingressclass-labels",
					Annotations: map[string]string{
						"ingressclass.kubernetes.io/is-default-class": "true",
					},
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: networkingv1.IngressClassSpec{
					Controller: "controller",
				},
			},
			Want: `
					# HELP kube_ingressclass_annotations Kubernetes annotations converted to Prometheus labels.
					# HELP kube_ingressclass_labels Kubernetes labels converted to Prometheus labels.
					# TYPE kube_ingressclass_annotations gauge
					# TYPE kube_ingressclass_labels gauge
					kube_ingressclass_annotations{ingressclass="test_ingressclass-labels",annotation_ingressclass_kubernetes_io_is_default_class="true"} 1
					kube_ingressclass_labels{ingressclass="test_ingressclass-labels"} 1
				`,
			MetricNames: []string{
				"kube_ingressclass_annotations", "kube_ingressclass_labels",
			},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(ingressClassMetricFamilies(c.AllowAnnotationsList, nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(ingressClassMetricFamilies(c.AllowAnnotationsList, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
