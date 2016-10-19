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

package main

import (
	"testing"
	"time"

	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/api/v1"
)

type mockEventStore struct {
	f func() ([]v1.Event, error)
}

func (ds mockEventStore) List() (events []v1.Event, err error) {
	return ds.f()
}

func TestEventCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_pod_healthcheck_num_of_failures Number of healthcheck failures for a given pod.
		# TYPE kube_pod_healthcheck_num_of_failures gauge
		# HELP kube_pod_healthcheck_seconds_since_last_failure Number of seconds since of last healthcheck failure.
		# TYPE kube_pod_healthcheck_seconds_since_last_failure gauge
	`

	var now = time.Now()
	var minus5 = now.Add(-5 * time.Minute)
	var minus10 = now.Add(-10 * time.Minute)

	cases := []struct {
		events  []v1.Event
		metrics []string
		want    string
	}{
		{
			events: []v1.Event{
				{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns1",
						Name:      "pod1",
					},
					InvolvedObject: v1.ObjectReference{
						Kind:      "Pod",
						Namespace: "ns1",
						Name:      "pod1",
					},
					Reason:        "Unhealthy",
					Count:         35,
					LastTimestamp: unversioned.NewTime(minus5),
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns2",
						Name:      "pod2",
					},
					InvolvedObject: v1.ObjectReference{
						Kind:      "Pod",
						Namespace: "ns2",
						Name:      "pod2",
					},
					Reason:        "Unhealthy",
					Count:         17,
					LastTimestamp: unversioned.NewTime(minus10),
				},
			},
			want: metadata + `
				kube_pod_healthcheck_num_of_failures{namespace="ns1",pod="pod1"} 35
				kube_pod_healthcheck_num_of_failures{namespace="ns2",pod="pod2"} 17
				kube_pod_healthcheck_seconds_since_last_failure{namespace="ns1",pod="pod1"} 300
				kube_pod_healthcheck_seconds_since_last_failure{namespace="ns2",pod="pod2"} 600
				`,
			metrics: []string{
				"kube_pod_healthcheck_num_of_failures",
				"kube_pod_healthcheck_seconds_since_last_failure",
			},
		},
	}

	for _, c := range cases {
		ec := &eventCollector{
			store: mockEventStore{
				f: func() ([]v1.Event, error) { return c.events, nil },
			},
		}
		if err := gatherAndCompare(ec, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
