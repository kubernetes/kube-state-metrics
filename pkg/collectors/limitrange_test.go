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

package collectors

import (
	"testing"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/pkg/collectors/testutils"
	"k8s.io/kube-state-metrics/pkg/options"
)

type mockLimitRangeStore struct {
	list func() (v1.LimitRangeList, error)
}

func (ns mockLimitRangeStore) List() (v1.LimitRangeList, error) {
	return ns.list()
}

func TestLimitRangeollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	testMemory := "2.1G"
	testMemoryQuantity := resource.MustParse(testMemory)
	const metadata = `
	# HELP kube_limitrange_created Unix creation timestamp
	# TYPE kube_limitrange_created gauge
	# HELP kube_limitrange Information about limit range.
	# TYPE kube_limitrange gauge
	`
	cases := []struct {
		ranges  []v1.LimitRange
		metrics []string // which metrics should be checked
		want    string
	}{
		{
			ranges: []v1.LimitRange{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "quotaTest",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
						Namespace:         "testNS",
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
			},
			want: metadata + `
		kube_limitrange_created{limitrange="quotaTest",namespace="testNS"} 1.5e+09
		kube_limitrange{limitrange="quotaTest",namespace="testNS",resource="memory",type="Pod",constraint="min"} 2.1e+09
		kube_limitrange{limitrange="quotaTest",namespace="testNS",resource="memory",type="Pod",constraint="max"} 2.1e+09
		kube_limitrange{limitrange="quotaTest",namespace="testNS",resource="memory",type="Pod",constraint="default"} 2.1e+09
		kube_limitrange{limitrange="quotaTest",namespace="testNS",resource="memory",type="Pod",constraint="defaultRequest"} 2.1e+09
		kube_limitrange{limitrange="quotaTest",namespace="testNS",resource="memory",type="Pod",constraint="maxLimitRequestRatio"} 2.1e+09
		`,
		},
	}
	for _, c := range cases {
		dc := &limitRangeCollector{
			store: &mockLimitRangeStore{
				list: func() (v1.LimitRangeList, error) {
					return v1.LimitRangeList{Items: c.ranges}, nil
				},
			},
			opts: &options.Options{},
		}
		if err := testutils.GatherAndCompare(dc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
