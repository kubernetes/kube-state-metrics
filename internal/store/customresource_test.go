/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestCustomResourceSetStore(t *testing.T) {
	cases := []generateMetricsTestCase{
		{
			AllowLabelsList: []string{
				"app.k8s.io/owner",
			},
			Obj: &v1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ds1",
					Namespace: "ns1",
					Labels: map[string]string{
						"app":              "example1",
						"app.k8s.io/owner": "@foo",
					},
				},
			},
			Want: `
				# HELP kube_daemonset_labels Kubernetes labels converted to Prometheus labels.
				# TYPE kube_daemonset_labels gauge
				kube_daemonset_labels{label_app_k8s_io_owner="@foo"} 1
`,
			MetricNames: []string{
				"kube_daemonset_labels",
			},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(customResourceMetricFamilies("daemonset", c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(customResourceMetricFamilies("daemonset", c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
