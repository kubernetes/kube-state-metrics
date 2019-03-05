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

package collector

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/pkg/metric"
)

func TestConfigMapCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.

	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)

	const metadata = `
        # HELP kube_configmap_info Information about configmap.
		# TYPE kube_configmap_info gauge
		# HELP kube_configmap_created Unix creation timestamp
		# TYPE kube_configmap_created gauge
		# HELP kube_configmap_metadata_resource_version Resource version representing a specific version of the configmap.
		# TYPE kube_configmap_metadata_resource_version gauge
	`
	cases := []generateMetricsTestCase{
		{
			Obj: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "configmap1",
					Namespace:       "ns1",
					ResourceVersion: "123456",
				},
			},
			Want: `
				kube_configmap_info{configmap="configmap1",namespace="ns1"} 1
				kube_configmap_metadata_resource_version{configmap="configmap1",namespace="ns1",resource_version="123456"} 1
`,
			MetricNames: []string{"kube_configmap_info", "kube_configmap_metadata_resource_version"},
		},
		{
			Obj: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "configmap2",
					Namespace:         "ns2",
					CreationTimestamp: metav1StartTime,
					ResourceVersion:   "abcdef",
				},
			},
			Want: `
				kube_configmap_info{configmap="configmap2",namespace="ns2"} 1
				kube_configmap_created{configmap="configmap2",namespace="ns2"} 1.501569018e+09
				kube_configmap_metadata_resource_version{configmap="configmap2",namespace="ns2",resource_version="abcdef"} 1
				`,
			MetricNames: []string{"kube_configmap_info", "kube_configmap_created", "kube_configmap_metadata_resource_version"},
		},
	}
	for i, c := range cases {
		c.Func = metric.ComposeMetricGenFuncs(configMapMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
