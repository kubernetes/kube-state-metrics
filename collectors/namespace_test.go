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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

type mockNamespaceStore struct {
	list func() ([]v1.Namespace, error)
}

func (ns mockNamespaceStore) List() ([]v1.Namespace, error) {
	return ns.list()
}

func TestNamespaceCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.

	const metadata = `
	# HELP kube_namespace_status_phase kubernetes namespace status phase.
	# TYPE kube_namespace_status_phase gauge
	`
	cases := []struct {
		ns      []v1.Namespace
		metrics []string // which metrics should be checked
		want    string
	}{
		{
			ns: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "nsActiveTest",
					},
					Spec: v1.NamespaceSpec{
						Finalizers: []v1.FinalizerName{v1.FinalizerKubernetes},
					},
					Status: v1.NamespaceStatus{
						Phase: v1.NamespaceActive,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "nsTerminateTest",
					},
					Spec: v1.NamespaceSpec{
						Finalizers: []v1.FinalizerName{v1.FinalizerKubernetes},
					},
					Status: v1.NamespaceStatus{
						Phase: v1.NamespaceTerminating,
					},
				},
			},

			want: metadata + `
		kube_namespace_status_phase{name="nsActiveTest",status="Active"} 1
		kube_namespace_status_phase{name="nsTerminateTest",status="Terminating"} 1
		`,
		},
	}

	for _, c := range cases {
		dc := &namespaceCollector{
			store: &mockNamespaceStore{
				list: func() ([]v1.Namespace, error) {
					return c.ns, nil
				},
			},
		}
		if err := gatherAndCompare(dc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
