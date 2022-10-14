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

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestPodDisruptionBudgetStore(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const labelsAndAnnotationsMetaData = `
	# HELP kube_poddisruptionbudget_annotations Kubernetes annotations converted to Prometheus labels.
	# TYPE kube_poddisruptionbudget_annotations gauge
	# HELP kube_poddisruptionbudget_labels Kubernetes labels converted to Prometheus labels.
	# TYPE kube_poddisruptionbudget_labels gauge
	`
	const metadata = labelsAndAnnotationsMetaData + `
	# HELP kube_poddisruptionbudget_created [STABLE] Unix creation timestamp
	# TYPE kube_poddisruptionbudget_created gauge
	# HELP kube_poddisruptionbudget_status_current_healthy [STABLE] Current number of healthy pods
	# TYPE kube_poddisruptionbudget_status_current_healthy gauge
	# HELP kube_poddisruptionbudget_status_desired_healthy [STABLE] Minimum desired number of healthy pods
	# TYPE kube_poddisruptionbudget_status_desired_healthy gauge
	# HELP kube_poddisruptionbudget_status_pod_disruptions_allowed [STABLE] Number of pod disruptions that are currently allowed
	# TYPE kube_poddisruptionbudget_status_pod_disruptions_allowed gauge
	# HELP kube_poddisruptionbudget_status_expected_pods [STABLE] Total number of pods counted by this disruption budget
	# TYPE kube_poddisruptionbudget_status_expected_pods gauge
	# HELP kube_poddisruptionbudget_status_observed_generation [STABLE] Most recent generation observed when updating this PDB status
	# TYPE kube_poddisruptionbudget_status_observed_generation gauge
	`
	cases := []generateMetricsTestCase{
		{
			Obj: &policyv1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pdb1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "ns1",
					Generation:        21,
				},
				Status: policyv1.PodDisruptionBudgetStatus{
					CurrentHealthy:     12,
					DesiredHealthy:     10,
					DisruptionsAllowed: 2,
					ExpectedPods:       15,
					ObservedGeneration: 111,
				},
			},
			Want: metadata + `
			kube_poddisruptionbudget_annotations{namespace="ns1",poddisruptionbudget="pdb1"} 1
			kube_poddisruptionbudget_labels{namespace="ns1",poddisruptionbudget="pdb1"} 1
			kube_poddisruptionbudget_created{namespace="ns1",poddisruptionbudget="pdb1"} 1.5e+09
			kube_poddisruptionbudget_status_current_healthy{namespace="ns1",poddisruptionbudget="pdb1"} 12
			kube_poddisruptionbudget_status_desired_healthy{namespace="ns1",poddisruptionbudget="pdb1"} 10
			kube_poddisruptionbudget_status_pod_disruptions_allowed{namespace="ns1",poddisruptionbudget="pdb1"} 2
			kube_poddisruptionbudget_status_expected_pods{namespace="ns1",poddisruptionbudget="pdb1"} 15
			kube_poddisruptionbudget_status_observed_generation{namespace="ns1",poddisruptionbudget="pdb1"} 111
			`,
		},
		{
			Obj: &policyv1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "pdb2",
					Namespace:  "ns2",
					Generation: 14,
				},
				Status: policyv1.PodDisruptionBudgetStatus{
					CurrentHealthy:     8,
					DesiredHealthy:     9,
					DisruptionsAllowed: 0,
					ExpectedPods:       10,
					ObservedGeneration: 1111,
				},
			},
			Want: metadata + `
				kube_poddisruptionbudget_annotations{namespace="ns2",poddisruptionbudget="pdb2"} 1
				kube_poddisruptionbudget_labels{namespace="ns2",poddisruptionbudget="pdb2"} 1
				kube_poddisruptionbudget_status_current_healthy{namespace="ns2",poddisruptionbudget="pdb2"} 8
				kube_poddisruptionbudget_status_desired_healthy{namespace="ns2",poddisruptionbudget="pdb2"} 9
				kube_poddisruptionbudget_status_pod_disruptions_allowed{namespace="ns2",poddisruptionbudget="pdb2"} 0
				kube_poddisruptionbudget_status_expected_pods{namespace="ns2",poddisruptionbudget="pdb2"} 10
				kube_poddisruptionbudget_status_observed_generation{namespace="ns2",poddisruptionbudget="pdb2"} 1111
			`,
		},
		{
			AllowAnnotationsList: []string{
				"app.k8s.io/owner",
			},
			AllowLabelsList: []string{
				"app",
			},
			Obj: &policyv1.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pdb_with_allowed_labels_and_annotations",
					Namespace: "ns",
					Annotations: map[string]string{
						"app.k8s.io/owner": "mysql-server",
						"foo":              "bar",
					},
					Labels: map[string]string{
						"app":   "mysql-server",
						"hello": "world",
					},
				},
			},
			Want: labelsAndAnnotationsMetaData + `
				kube_poddisruptionbudget_annotations{annotation_app_k8s_io_owner="mysql-server",namespace="ns",poddisruptionbudget="pdb_with_allowed_labels_and_annotations"} 1
				kube_poddisruptionbudget_labels{label_app="mysql-server",namespace="ns",poddisruptionbudget="pdb_with_allowed_labels_and_annotations"} 1
			`,
			MetricNames: []string{
				"kube_poddisruptionbudget_annotations",
				"kube_poddisruptionbudget_labels",
			},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(podDisruptionBudgetMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(podDisruptionBudgetMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
