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

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/pkg/metric"
)

func TestDaemonSetStore(t *testing.T) {
	cases := []generateMetricsTestCase{
		{
			Obj: &v1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ds1",
					Namespace: "ns1",
					Labels: map[string]string{
						"app": "example1",
					},
					Annotations: map[string]string{
						"app": "example1",
					},
					Generation: 21,
				},
				Status: v1.DaemonSetStatus{
					CurrentNumberScheduled: 15,
					NumberMisscheduled:     10,
					DesiredNumberScheduled: 5,
					NumberReady:            5,
				},
			},
			Want: `
				kube_daemonset_metadata_generation{daemonset="ds1",namespace="ns1"} 21
				kube_daemonset_status_current_number_scheduled{daemonset="ds1",namespace="ns1"} 15
				kube_daemonset_status_desired_number_scheduled{daemonset="ds1",namespace="ns1"} 5
				kube_daemonset_status_number_available{daemonset="ds1",namespace="ns1"} 0
				kube_daemonset_status_number_misscheduled{daemonset="ds1",namespace="ns1"} 10
				kube_daemonset_status_number_ready{daemonset="ds1",namespace="ns1"} 5
				kube_daemonset_status_number_unavailable{daemonset="ds1",namespace="ns1"} 0
				kube_daemonset_updated_number_scheduled{daemonset="ds1",namespace="ns1"} 0
				kube_daemonset_labels{daemonset="ds1",label_app="example1",namespace="ns1"} 1
				kube_daemonset_annotations{daemonset="ds1",namespace="ns1",annotation_app="example1"} 1
`,
			MetricNames: []string{
				"kube_daemonset_labels",
				"kube_daemonset_metadata_generation",
				"kube_daemonset_status_current_number_scheduled",
				"kube_daemonset_status_desired_number_scheduled",
				"kube_daemonset_status_number_available",
				"kube_daemonset_status_number_misscheduled",
				"kube_daemonset_status_number_ready",
				"kube_daemonset_status_number_unavailable",
				"kube_daemonset_updated_number_scheduled",
				"kube_daemonset_annotations",
			},
		},
		{
			Obj: &v1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "ds2",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "ns2",
					Labels: map[string]string{
						"app": "example2",
					},
					Annotations: map[string]string{
						"app": "example2",
					},
					Generation: 14,
				},
				Status: v1.DaemonSetStatus{
					CurrentNumberScheduled: 10,
					NumberMisscheduled:     5,
					DesiredNumberScheduled: 0,
					NumberReady:            0,
				},
			},
			Want: `
				kube_daemonset_created{daemonset="ds2",namespace="ns2"} 1.5e+09
				kube_daemonset_metadata_generation{daemonset="ds2",namespace="ns2"} 14
				kube_daemonset_status_current_number_scheduled{daemonset="ds2",namespace="ns2"} 10
				kube_daemonset_status_desired_number_scheduled{daemonset="ds2",namespace="ns2"} 0
				kube_daemonset_status_number_available{daemonset="ds2",namespace="ns2"} 0
				kube_daemonset_status_number_misscheduled{daemonset="ds2",namespace="ns2"} 5
				kube_daemonset_status_number_ready{daemonset="ds2",namespace="ns2"} 0
				kube_daemonset_status_number_unavailable{daemonset="ds2",namespace="ns2"} 0
				kube_daemonset_updated_number_scheduled{daemonset="ds2",namespace="ns2"} 0
				kube_daemonset_labels{daemonset="ds2",label_app="example2",namespace="ns2"} 1
				kube_daemonset_annotations{daemonset="ds2",namespace="ns2",annotation_app="example2"} 1
`,
			MetricNames: []string{
				"kube_daemonset_created",
				"kube_daemonset_labels",
				"kube_daemonset_metadata_generation",
				"kube_daemonset_status_current_number_scheduled",
				"kube_daemonset_status_desired_number_scheduled",
				"kube_daemonset_status_number_available",
				"kube_daemonset_status_number_misscheduled",
				"kube_daemonset_status_number_ready",
				"kube_daemonset_status_number_unavailable",
				"kube_daemonset_updated_number_scheduled",
				"kube_daemonset_annotations",
			},
		},
		{
			Obj: &v1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "ds3",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "ns3",
					Labels: map[string]string{
						"app": "example3",
					},
					Annotations: map[string]string{
						"app": "example3",
					},
					Generation: 15,
				},
				Status: v1.DaemonSetStatus{
					CurrentNumberScheduled: 10,
					NumberMisscheduled:     5,
					DesiredNumberScheduled: 15,
					NumberReady:            5,
					NumberAvailable:        5,
					NumberUnavailable:      5,
					UpdatedNumberScheduled: 5,
				},
			},
			Want: `
				kube_daemonset_created{daemonset="ds3",namespace="ns3"} 1.5e+09
				kube_daemonset_metadata_generation{daemonset="ds3",namespace="ns3"} 15
				kube_daemonset_status_current_number_scheduled{daemonset="ds3",namespace="ns3"} 10
				kube_daemonset_status_desired_number_scheduled{daemonset="ds3",namespace="ns3"} 15
				kube_daemonset_status_number_available{daemonset="ds3",namespace="ns3"} 5
				kube_daemonset_status_number_misscheduled{daemonset="ds3",namespace="ns3"} 5
				kube_daemonset_status_number_ready{daemonset="ds3",namespace="ns3"} 5
				kube_daemonset_status_number_unavailable{daemonset="ds3",namespace="ns3"} 5
				kube_daemonset_updated_number_scheduled{daemonset="ds3",namespace="ns3"} 5
				kube_daemonset_labels{daemonset="ds3",label_app="example3",namespace="ns3"} 1
				kube_daemonset_annotations{daemonset="ds3",namespace="ns3",annotation_app="example3"} 1
`,
			MetricNames: []string{
				"kube_daemonset_created",
				"kube_daemonset_labels",
				"kube_daemonset_metadata_generation",
				"kube_daemonset_status_current_number_scheduled",
				"kube_daemonset_status_desired_number_scheduled",
				"kube_daemonset_status_number_available",
				"kube_daemonset_status_number_misscheduled",
				"kube_daemonset_status_number_ready",
				"kube_daemonset_status_number_unavailable",
				"kube_daemonset_updated_number_scheduled",
				"kube_daemonset_annotations",
			},
		},
	}
	for i, c := range cases {
		c.Func = metric.ComposeMetricGenFuncs(daemonSetMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
