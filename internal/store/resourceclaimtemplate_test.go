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

	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestResourceClaimTemplateStore(t *testing.T) {
	const metadata = `
		# HELP kube_resourceclaimtemplate_created Unix creation timestamp
		# TYPE kube_resourceclaimtemplate_created gauge
	`

	cases := []generateMetricsTestCase{
		{
			Obj: &resourcev1.ResourceClaimTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "template-1",
					Namespace:         "ns-1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Spec: resourcev1.ResourceClaimTemplateSpec{},
			},
			Want: metadata + `
				kube_resourceclaimtemplate_created{namespace="ns-1",resourceclaimtemplate="template-1"} 1.5e+09
			`,
			MetricNames: []string{
				"kube_resourceclaimtemplate_created",
			},
		},
		{
			AllowAnnotationsList: []string{"team"},
			AllowLabelsList:      []string{"app"},
			Obj: &resourcev1.ResourceClaimTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "resourceclaimtemplate-labels",
					Namespace:   "ns-1",
					Annotations: map[string]string{"team": "gpu", "ignored": "x"},
					Labels:      map[string]string{"app": "trainer", "ignored": "x"},
				},
			},
			Want: `
				# HELP kube_resourceclaimtemplate_annotations Kubernetes annotations converted to Prometheus labels.
				# TYPE kube_resourceclaimtemplate_annotations gauge
				# HELP kube_resourceclaimtemplate_labels Kubernetes labels converted to Prometheus labels.
				# TYPE kube_resourceclaimtemplate_labels gauge
				kube_resourceclaimtemplate_annotations{annotation_team="gpu",namespace="ns-1",resourceclaimtemplate="resourceclaimtemplate-labels"} 1
				kube_resourceclaimtemplate_labels{label_app="trainer",namespace="ns-1",resourceclaimtemplate="resourceclaimtemplate-labels"} 1
			`,
			MetricNames: []string{"kube_resourceclaimtemplate_annotations", "kube_resourceclaimtemplate_labels"},
		},
	}

	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(resourceClaimTemplateMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(resourceClaimTemplateMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %dth run:\n%v", i, err)
		}
	}
}
