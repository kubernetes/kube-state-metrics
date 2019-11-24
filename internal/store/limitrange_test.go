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
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kube-state-metrics/v2/pkg/allow"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestLimitRangeStore(t *testing.T) {
	testMemory := "2.1G"
	testMemoryQuantity := resource.MustParse(testMemory)
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
	# HELP kube_limitrange_annotations Kubernetes annotations converted to Prometheus labels.
	# TYPE kube_limitrange_annotations gauge
	# HELP kube_limitrange_created Unix creation timestamp
	# TYPE kube_limitrange_created gauge
	# HELP kube_limitrange Information about limit range.
	# TYPE kube_limitrange gauge
	`
	cases := []generateMetricsTestCase{
		{
			Obj: &v1.LimitRange{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "quotaTest",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "testNS",
					Annotations: map[string]string{
						"whitelisted":     "true",
						"not-whitelisted": "false",
					},
				},
				Spec: v1.LimitRangeSpec{
					Limits: []v1.LimitRangeItem{
						{
							Type: v1.LimitTypePod,
							Max: map[v1.ResourceName]resource.Quantity{
								v1.ResourceMemory: testMemoryQuantity,
							},
							Min: map[v1.ResourceName]resource.Quantity{
								v1.ResourceMemory: testMemoryQuantity,
							},
							Default: map[v1.ResourceName]resource.Quantity{
								v1.ResourceMemory: testMemoryQuantity,
							},
							DefaultRequest: map[v1.ResourceName]resource.Quantity{
								v1.ResourceMemory: testMemoryQuantity,
							},
							MaxLimitRequestRatio: map[v1.ResourceName]resource.Quantity{
								v1.ResourceMemory: testMemoryQuantity,
							},
						},
					},
				},
			},
			Want: metadata + `
		kube_limitrange_annotations{annotation_whitelisted="true",limitrange="quotaTest",namespace="testNS"} 1
        kube_limitrange_created{limitrange="quotaTest",namespace="testNS"} 1.5e+09
        kube_limitrange{constraint="default",limitrange="quotaTest",namespace="testNS",resource="memory",type="Pod"} 2.1e+09
        kube_limitrange{constraint="defaultRequest",limitrange="quotaTest",namespace="testNS",resource="memory",type="Pod"} 2.1e+09
        kube_limitrange{constraint="max",limitrange="quotaTest",namespace="testNS",resource="memory",type="Pod"} 2.1e+09
        kube_limitrange{constraint="maxLimitRequestRatio",limitrange="quotaTest",namespace="testNS",resource="memory",type="Pod"} 2.1e+09
        kube_limitrange{constraint="min",limitrange="quotaTest",namespace="testNS",resource="memory",type="Pod"} 2.1e+09
		`,
			allowLabels: allow.Labels{"kube_limitrange_annotations": append([]string{"annotation_whitelisted"}, descLimitRangeLabelsDefaultLabels...)},
		},
	}
	for i, c := range cases {
		filteredWhitelistedAnnotationMetricFamilies := generator.FilterMetricFamiliesLabels(c.allowLabels, limitRangeMetricFamilies)
		c.Func = generator.ComposeMetricGenFuncs(filteredWhitelistedAnnotationMetricFamilies)
		c.Headers = generator.ExtractMetricFamilyHeaders(filteredWhitelistedAnnotationMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
