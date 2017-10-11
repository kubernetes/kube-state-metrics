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

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockcomponentStatusStore struct {
	list func() ([]v1.ComponentStatus, error)
}

func (ns mockcomponentStatusStore) List() ([]v1.ComponentStatus, error) {
	return ns.list()
}

func TestComponentStatusCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_componentstatus kube component status.
		# TYPE kube_componentstatus gauge
	`
	cases := []struct {
		cms     []v1.ComponentStatus
		metrics []string // which metrics should be checked
		want    string
	}{
		// Verify phase enumerations.
		{
			cms: []v1.ComponentStatus{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "etcd1",
					},
					Conditions: []v1.ComponentCondition{
						{
							Type:   v1.ComponentHealthy,
							Status: v1.ConditionTrue,
						},
					},
				},
			},
			want: metadata + `
				kube_componentstatus{name="etcd1",status="True"} 1
				kube_componentstatus{name="etcd1",status="False"} 0
				kube_componentstatus{name="etcd1",status="Unknown"} 0
			`,
			metrics: []string{"kube_componentstatus"},
		},
	}
	for _, c := range cases {
		dc := &componentStatusCollector{
			store: &mockcomponentStatusStore{
				list: func() ([]v1.ComponentStatus, error) {
					return c.cms, nil
				},
			},
		}
		if err := gatherAndCompare(dc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
