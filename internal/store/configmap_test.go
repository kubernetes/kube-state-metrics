/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kube-state-metrics/v2/pkg/allow"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestConfigMapStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)

	cases := []generateMetricsTestCase{
		{
			Obj: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "configmap1",
					Namespace:       "ns1",
					ResourceVersion: "BBBBB",
				},
			},
			Want: `
				# HELP kube_configmap_info Information about configmap.
				# HELP kube_configmap_metadata_resource_version Resource version representing a specific version of the configmap.
				# TYPE kube_configmap_info gauge
				# TYPE kube_configmap_metadata_resource_version gauge
				kube_configmap_info{configmap="configmap1",namespace="ns1"} 1
`,
			MetricNames: []string{"kube_configmap_info", "kube_configmap_metadata_resource_version"},
		},
		{
			Obj: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "configmap2",
					Namespace:         "ns2",
					CreationTimestamp: metav1StartTime,
					Annotations: map[string]string{
						"whitelisted":     "true",
						"not-whitelisted": "false",
					},
					ResourceVersion: "10596",
				},
			},
			Want: `
				# HELP kube_configmap_info Information about configmap.
				# HELP kube_configmap_metadata_resource_version Resource version representing a specific version of the configmap.
				# HELP kube_configmap_annotations Kubernetes annotations converted to Prometheus labels.
				# HELP kube_configmap_created Unix creation timestamp
				# TYPE kube_configmap_created gauge
				# TYPE kube_configmap_info gauge
				# TYPE kube_configmap_metadata_resource_version gauge
				# TYPE kube_configmap_annotations gauge
				kube_configmap_info{configmap="configmap2",namespace="ns2"} 1
				kube_configmap_created{configmap="configmap2",namespace="ns2"} 1.501569018e+09
				kube_configmap_metadata_resource_version{configmap="configmap2",namespace="ns2"} 10596
				kube_configmap_annotations{annotation_whitelisted="true",configmap="configmap2",namespace="ns2"} 1
				`,
			MetricNames: []string{"kube_configmap_info", "kube_configmap_created", "kube_configmap_metadata_resource_version", "kube_configmap_annotations"},
			allowLabels: allow.Labels{"kube_configmap_annotations": append([]string{"annotation_whitelisted"}, descConfigMapLabelsDefaultLabels...)},
		},
	}
	for i, c := range cases {
		filteredWhitelistedAnnotationMetricFamilies := generator.FilterMetricFamiliesLabels(c.allowLabels, configMapMetricFamilies)
		c.Func = generator.ComposeMetricGenFuncs(filteredWhitelistedAnnotationMetricFamilies)
		c.Headers = generator.ExtractMetricFamilyHeaders(filteredWhitelistedAnnotationMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
