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
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// podList returns a PodList of n pods carrying a continue token, the way the
// API server answers a paginated list that has more to give.
func podList(n int, continueToken string) *v1.PodList {
	list := &v1.PodList{
		ListMeta: metav1.ListMeta{
			ResourceVersion: "1",
			Continue:        continueToken,
		},
	}
	for i := 0; i < n; i++ {
		list.Items = append(list.Items, v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("pod-%d", i), Namespace: "ns"},
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
