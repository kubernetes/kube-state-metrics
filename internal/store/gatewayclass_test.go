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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestGatewayClassStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)

	cases := []generateMetricsTestCase{
		{
			Obj: &gatewayapiv1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test_gatewayclass-info",
				},
				Spec: gatewayapiv1.GatewayClassSpec{
					ControllerName: "controller",
				},
			},
			Want: `
					# HELP kube_gatewayclass_info Information about gatewayclass.
					# TYPE kube_gatewayclass_info gauge
					kube_gatewayclass_info{gatewayclass="test_gatewayclass-info",controller="controller"} 1
				`,
			MetricNames: []string{
				"kube_gatewayclass_info",
			},
		},
		{
			Obj: &gatewayapiv1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test_kube_gatewayclass-created",
					CreationTimestamp: metav1StartTime,
				},
				Spec: gatewayapiv1.GatewayClassSpec{
					ControllerName: "controller",
				},
			},
			Want: `
					# HELP kube_gatewayclass_created Unix creation timestamp
					# TYPE kube_gatewayclass_created gauge
					kube_gatewayclass_created{gatewayclass="test_kube_gatewayclass-created"} 1.501569018e+09
				`,
			MetricNames: []string{
				"kube_gatewayclass_created",
			},
		},
		{
			AllowAnnotationsList: []string{
				"gatewayclass.kubernetes.io/is-default-class",
			},
			Obj: &gatewayapiv1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test_gatewayclass-labels",
					Annotations: map[string]string{
						"gatewayclass.kubernetes.io/is-default-class": "true",
					},
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: gatewayapiv1.GatewayClassSpec{
					ControllerName: "controller",
				},
			},
			Want: `
					# HELP kube_gatewayclass_annotations Kubernetes annotations converted to Prometheus labels.
					# HELP kube_gatewayclass_labels Kubernetes labels converted to Prometheus labels.
					# TYPE kube_gatewayclass_annotations gauge
					# TYPE kube_gatewayclass_labels gauge
					kube_gatewayclass_annotations{gatewayclass="test_gatewayclass-labels",annotation_gatewayclass_kubernetes_io_is_default_class="true"} 1
				`,
			MetricNames: []string{
				"kube_gatewayclass_annotations", "kube_gatewayclass_labels",
			},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(gatewayClassMetricFamilies(c.AllowAnnotationsList, nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(gatewayClassMetricFamilies(c.AllowAnnotationsList, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
