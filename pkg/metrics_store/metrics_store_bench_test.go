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

package metricsstore

import (
	"fmt"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
)

// benchSyncStore returns a store with a generate function similar in cost to
// a resource with many families, and nObjects services to sync into it.
func benchSyncStore(nObjects int) (*MetricsStore, []any) {
	const nFamilies = 50

	headers := make([]string, nFamilies)
	genFunc := func(obj any) []metric.FamilyInterface {
		o, _ := meta.Accessor(obj)
		families := make([]metric.FamilyInterface, nFamilies)
		for j := range nFamilies {
			families[j] = &metric.Family{
				Name: fmt.Sprintf("kube_bench_metric_%d", j),
				Metrics: []*metric.Metric{
					{
						LabelKeys:   []string{"namespace", "service", "uid"},
						LabelValues: []string{o.GetNamespace(), o.GetName(), string(o.GetUID())},
						Value:       float64(j),
					},
				},
			}
		}
		return families
	}

	list := make([]any, nObjects)
	for i := range list {
		list[i] = &v1.Service{
			UID:       types.UID(fmt.Sprintf("uid-%d", i)),
			Name:      fmt.Sprintf("svc-%d", i),
			Namespace: "default",
		}
	}
	return NewMetricsStore(headers, genFunc), list
}

// BenchmarkReplace measures the initial sync of a List: Replace with full
// objects into an empty store.
func BenchmarkReplace(b *testing.B) {
	b.ReportAllocs()
	_, list := benchSyncStore(10000)
	for b.Loop() {
		b.StopTimer()
		store, _ := benchSyncStore(0)
		b.StartTimer()
		if err := store.Replace(list, "1"); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWatchListSync measures the initial sync of a WatchList: every
// object passes Transformer as it arrives, then the list is handed to Replace.
func BenchmarkWatchListSync(b *testing.B) {
	b.ReportAllocs()
	_, list := benchSyncStore(10000)
	transformed := make([]any, len(list))
	for b.Loop() {
		b.StopTimer()
		clear(transformed)
		store, _ := benchSyncStore(0)
		transform := store.Transformer()
		b.StartTimer()
		for i, o := range list {
			r, err := transform(o)
			if err != nil {
				b.Fatal(err)
			}
			transformed[i] = r
		}
		if err := store.Replace(transformed, "1"); err != nil {
			b.Fatal(err)
		}
	}
}
