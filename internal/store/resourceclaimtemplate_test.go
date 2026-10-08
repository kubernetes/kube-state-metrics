/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

	resourcev1beta1 "k8s.io/api/resource/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestResourceClaimTemplateStore(t *testing.T) {
	const metadata = `
		# HELP kube_resourceclaimtemplate_created Unix creation timestamp
		# TYPE kube_resourceclaimtemplate_created gauge
		# HELP kube_resourceclaimtemplate_info Information about resource claim template.
		# TYPE kube_resourceclaimtemplate_info gauge
	`

	cases := []generateMetricsTestCase{
		{
			Obj: &resourcev1beta1.ResourceClaimTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "template-1",
					Namespace:         "ns-1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Spec: resourcev1beta1.ResourceClaimTemplateSpec{},
			},
			Want: metadata + `
				kube_resourceclaimtemplate_created{namespace="ns-1",resourceclaimtemplate="template-1"} 1.5e+09
				kube_resourceclaimtemplate_info{namespace="ns-1",resourceclaimtemplate="template-1"} 1
			`,
			MetricNames: []string{
				"kube_resourceclaimtemplate_created",
				"kube_resourceclaimtemplate_info",
			},
		},
	}

	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(resourceClaimTemplateMetricFamilies)
		c.Headers = generator.ExtractMetricFamilyHeaders(resourceClaimTemplateMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %dth run:\n%v", i, err)
		}
	}
}
