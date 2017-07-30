/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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

package collectors

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

type mockServiceStore struct {
	list func() ([]v1.Service, error)
}

func (ss mockServiceStore) List() ([]v1.Service, error) {
	return ss.list()
}

func TestServiceCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_service_info Information about service.
		# TYPE kube_service_info gauge
		# HELP kube_service_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_service_labels gauge
	`
	cases := []struct {
		services []v1.Service
		metrics  []string // which metrics should be checked
		want     string
	}{
		{
			services: []v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service",
						Namespace: "default",
						Labels: map[string]string{
							"app": "example",
						},
					},
				},
			},
			want: metadata + `
				kube_service_info{namespace="default",service="test-service"} 1
				kube_service_labels{label_app="example",namespace="default",service="test-service"} 1
			`,
		},
	}
	for _, c := range cases {
		sc := &serviceCollector{
			store: &mockServiceStore{
				list: func() ([]v1.Service, error) {
					return c.services, nil
				},
			},
		}
		if err := gatherAndCompare(sc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
