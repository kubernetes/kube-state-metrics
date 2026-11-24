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

package watch

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
)

// podList returns a PodList of n pods carrying a continue token, the way the
// API server answers a paginated list that has more to give.
func podList(n int, continueToken string) *v1.PodList {
	list := &v1.PodList{
		ResourceVersion: "1",
		Continue:        continueToken,
	}
	for i := range n {
		list.Items = append(list.Items, v1.Pod{
			Name: fmt.Sprintf("pod-%d", i), Namespace: "ns",
		})
	}
	return list
}

type fakeListerWatcher struct {
	list        runtime.Object
	gotOptions  metav1.ListOptions
	listCallCnt int
}

func (f *fakeListerWatcher) List(options metav1.ListOptions) (runtime.Object, error) {
	f.gotOptions = options
	f.listCallCnt++
	return f.list.DeepCopyObject(), nil
}

func (f *fakeListerWatcher) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	return watch.NewFake(), nil
}

func newTestListerWatcher(lw cache.ListerWatcher, limit int64) cache.ListerWatcher {
	return NewInstrumentedListerWatcher(lw, NewListWatchMetrics(prometheus.NewRegistry()), "pods", false, limit, nil)
}

func TestListObjectLimit(t *testing.T) {
	tests := []struct {
		Desc string
		// Items returned by the underlying lister.
		Items int
		// Continue token the underlying lister sets on the response.
		Continue string
		Limit    int64

		WantItems    int
		WantContinue string
		WantOptLimit int64
	}{
		{
			Desc:         "no limit leaves the response untouched",
			Items:        5,
			Continue:     "token",
			Limit:        0,
			WantItems:    5,
			WantContinue: "token",
			WantOptLimit: 0,
		},
		{
			Desc:         "a full page is truncated and the continue token dropped",
			Items:        5,
			Continue:     "token",
			Limit:        3,
			WantItems:    3,
			WantContinue: "",
			WantOptLimit: 3,
		},
		{
			// The API server may return a short page and still ask to be
			// continued. Honouring the token would blow past the limit.
			Desc:         "a short page still terminates the list",
			Items:        2,
			Continue:     "token",
			Limit:        3,
			WantItems:    2,
			WantContinue: "",
			WantOptLimit: 3,
		},
		{
			Desc:         "an exhausted list is unaffected",
			Items:        2,
			Continue:     "",
			Limit:        3,
			WantItems:    2,
			WantContinue: "",
			WantOptLimit: 3,
		},
	}

	for _, test := range tests {
		t.Run(test.Desc, func(t *testing.T) {
			fake := &fakeListerWatcher{list: podList(test.Items, test.Continue)}
			lw := newTestListerWatcher(fake, test.Limit)

			res, err := lw.List(metav1.ListOptions{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fake.gotOptions.Limit != test.WantOptLimit {
				t.Errorf("passed Limit: want %d, got %d", test.WantOptLimit, fake.gotOptions.Limit)
			}

			items, err := meta.ExtractList(res)
			if err != nil {
				t.Fatalf("extracting the list: %v", err)
			}
			if len(items) != test.WantItems {
				t.Errorf("items: want %d, got %d", test.WantItems, len(items))
			}

			listMeta, err := meta.ListAccessor(res)
			if err != nil {
				t.Fatalf("accessing the list metadata: %v", err)
			}
			if got := listMeta.GetContinue(); got != test.WantContinue {
				t.Errorf("continue token: want %q, got %q", test.WantContinue, got)
			}
		})
	}
}

// The reflector pages by re-calling List with the previous response's continue
// token. Clearing the token is what actually bounds the total.
func TestListObjectLimitStopsPaging(t *testing.T) {
	fake := &fakeListerWatcher{list: podList(500, "token")}
	lw := newTestListerWatcher(fake, 100)

	res, err := lw.List(metav1.ListOptions{Limit: 500})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	listMeta, err := meta.ListAccessor(res)
	if err != nil {
		t.Fatalf("accessing the list metadata: %v", err)
	}
	if listMeta.GetContinue() != "" {
		t.Fatal("continue token survived: the pager would fetch another page and the limit would not hold")
	}
	if listMeta.GetRemainingItemCount() != nil {
		t.Error("remainingItemCount survived a truncated list")
	}
}

func TestListUseAPIServerCache(t *testing.T) {
	fake := &fakeListerWatcher{list: podList(1, "")}
	lw := NewInstrumentedListerWatcher(fake, NewListWatchMetrics(prometheus.NewRegistry()), "pods", true, 0, nil)

	if _, err := lw.List(metav1.ListOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fake.gotOptions.ResourceVersion != "0" {
		t.Errorf("resourceVersion: want %q, got %q", "0", fake.gotOptions.ResourceVersion)
	}
}

// listThenWatchAdds lists listN pods, then its first watch delivers watchN
// Added events for pods created after the list.
type listThenWatchAdds struct {
	listN, watchN int
	watchCalls    atomic.Int32
}

func uidPod(i int) *v1.Pod {
	return &v1.Pod{
		Name: fmt.Sprintf("pod-%d", i), Namespace: "ns",
		UID: types.UID(fmt.Sprintf("uid-%d", i)), ResourceVersion: strconv.Itoa(i + 1),
	}
}

func (l *listThenWatchAdds) List(_ metav1.ListOptions) (runtime.Object, error) {
	list := &v1.PodList{ResourceVersion: "1"}
	for i := range l.listN {
		list.Items = append(list.Items, *uidPod(i))
	}
	return list, nil
}

func (l *listThenWatchAdds) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	if l.watchCalls.Add(1) > 1 {
		return watch.NewFake(), nil
	}
	fw := watch.NewFakeWithChanSize(l.watchN, false)
	for i := l.listN; i < l.listN+l.watchN; i++ {
		fw.Add(uidPod(i))
	}
	return fw, nil
}

func TestObjectLimitAppliesToWatch(t *testing.T) {
	const limit, listN, watchN = 5, 50, 200

	var dropped atomic.Int64
	store := metricsstore.NewMetricsStore(
		[]string{"# HELP kube_pod_test test\n# TYPE kube_pod_test gauge"},
		func(obj any) []metric.FamilyInterface {
			o, err := meta.Accessor(obj)
			if err != nil {
				t.Error(err)
				return nil
			}
			return []metric.FamilyInterface{&metric.Family{
				Name:    "kube_pod_test",
				Type:    metric.Gauge,
				Metrics: []*metric.Metric{{LabelKeys: []string{"uid"}, LabelValues: []string{string(o.GetUID())}, Value: 1}},
			}}
		},
		metricsstore.WithObjectLimit(limit, func() { dropped.Add(1) }),
	)

	lw := newTestListerWatcher(&listThenWatchAdds{listN: listN, watchN: watchN}, limit)
	reflector := cache.NewReflectorWithOptions(lw, &v1.Pod{}, store, cache.ReflectorOptions{})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	go reflector.Run(ctx.Done())

	for dropped.Load() < watchN {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for watch events, dropped %d of %d", dropped.Load(), watchN)
		case <-time.After(10 * time.Millisecond):
		}
	}

	var buf strings.Builder
	if err := metricsstore.NewMetricsWriter("pods", store).WriteAll(&buf); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(buf.String(), "kube_pod_test{"); got != limit {
		t.Fatalf("store holds %d objects, want %d", got, limit)
	}
}
