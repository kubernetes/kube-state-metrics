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

package metricshandler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

// newBenchStore returns a store holding nObjects pod-like objects with
// nFamilies families each.
func newBenchStore(tb testing.TB, nObjects, nFamilies int) *metricsstore.MetricsStore {
	tb.Helper()

	headers := make([]string, nFamilies)
	for j := range nFamilies {
		headers[j] = fmt.Sprintf("# HELP kube_bench_metric_%d benchmark metric %d\n# TYPE kube_bench_metric_%d gauge", j, j, j)
	}

	genFunc := func(obj any) []metric.FamilyInterface {
		o, _ := meta.Accessor(obj)
		families := make([]metric.FamilyInterface, nFamilies)
		for j := range nFamilies {
			// A mix of families with one and with several metrics, as for
			// per-container or per-condition families.
			n := 1 + j%4
			ms := make([]*metric.Metric, n)
			for k := range ms {
				ms[k] = &metric.Metric{
					LabelKeys:   []string{"namespace", "pod", "uid", "container", "condition"},
					LabelValues: []string{o.GetNamespace(), o.GetName(), string(o.GetUID()), fmt.Sprintf("container-%d", k), "true"},
					Value:       float64(j*k) + 0.5,
				}
			}
			families[j] = &metric.Family{
				Name:    fmt.Sprintf("kube_bench_metric_%d", j),
				Metrics: ms,
			}
		}
		return families
	}

	store := metricsstore.NewMetricsStore(headers, genFunc)
	for k := range nObjects {
		pod := &v1.Pod{}
		pod.UID = types.UID(fmt.Sprintf("8d0c7c5a-6b1e-4f2a-9c3d-%012d", k))
		pod.Name = fmt.Sprintf("workload-%d-7f9c8d6b5-x2kqz", k)
		pod.Namespace = fmt.Sprintf("namespace-%d", k%50)
		if err := store.Add(pod); err != nil {
			tb.Fatal(err)
		}
	}

	return store
}

// BenchmarkServeHTTP measures a full scrape over a loopback connection of a
// store of pod-like objects, with and without gzip. Besides the mean it
// reports the p99 of the scrape duration and the response size. Allocations
// include the client's.
func BenchmarkServeHTTP(b *testing.B) {
	const (
		nObjects  = 1000
		nFamilies = 40
	)

	store := newBenchStore(b, nObjects, nFamilies)

	stub := &stubStoreBuilder{buildWriters: metricsstore.MetricsWriterList{metricsstore.NewMetricsWriter("pods", store)}}
	handler := New(&options.Options{}, fake.NewSimpleClientset(), stub, true)
	handler.ConfigureSharding(context.Background(), 0, 1)

	srv := httptest.NewServer(handler)
	defer srv.Close()
	// Keep the response as sent: the client would otherwise decompress it.
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	defer client.CloseIdleConnections()

	for _, acceptEncoding := range []string{"identity", "gzip"} {
		b.Run(acceptEncoding, func(b *testing.B) {
			b.ReportAllocs()

			var durations []time.Duration
			var size int64
			for b.Loop() {
				req, err := http.NewRequest(http.MethodGet, srv.URL+"/metrics", nil)
				if err != nil {
					b.Fatal(err)
				}
				req.Header.Set("Accept-Encoding", acceptEncoding)

				start := time.Now()
				res, err := client.Do(req)
				if err != nil {
					b.Fatal(err)
				}
				size, err = io.Copy(io.Discard, res.Body)
				res.Body.Close()
				if err != nil {
					b.Fatal(err)
				}
				durations = append(durations, time.Since(start))
			}

			slices.Sort(durations)
			p99 := durations[(len(durations)*99)/100]
			b.ReportMetric(float64(p99.Nanoseconds()), "p99-ns")
			b.ReportMetric(float64(size), "resp-bytes")
		})
	}
}
