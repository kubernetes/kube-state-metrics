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

package store

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"k8s.io/kube-state-metrics/v2/pkg/allowdenylist"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func TestCustomResourceDefinitionStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)

	cases := []generateMetricsTestCase{
		{
			Obj: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "foos.example.com",
					CreationTimestamp: metav1StartTime,
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "example.com",
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Kind: "Foo",
					},
					Scope: apiextensionsv1.NamespaceScoped,
				},
			},
			Want: `
				# HELP kube_customresourcedefinition_created Unix creation timestamp
				# HELP kube_customresourcedefinition_info Information about a CustomResourceDefinition.
				# TYPE kube_customresourcedefinition_created gauge
				# TYPE kube_customresourcedefinition_info gauge
				kube_customresourcedefinition_created{customresourcedefinition="foos.example.com"} 1.501569018e+09
				kube_customresourcedefinition_info{customresourcedefinition="foos.example.com",group="example.com",kind="Foo",scope="Namespaced"} 1
			`,
			MetricNames: []string{
				"kube_customresourcedefinition_info",
				"kube_customresourcedefinition_created",
			},
		},
		{
			Obj: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bars.example.com",
					Annotations: map[string]string{
						"app.k8s.io/owner": "@foo",
					},
					Labels: map[string]string{
						"app": "mysql-server",
					},
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "example.com",
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Kind: "Bar",
					},
					Scope: apiextensionsv1.ClusterScoped,
				},
			},
			AllowAnnotationsList: []string{"app.k8s.io/owner"},
			AllowLabelsList:      []string{"app"},
			Want: `
				# HELP kube_customresourcedefinition_annotations Kubernetes annotations converted to Prometheus labels.
				# HELP kube_customresourcedefinition_labels Kubernetes labels converted to Prometheus labels.
				# TYPE kube_customresourcedefinition_annotations gauge
				# TYPE kube_customresourcedefinition_labels gauge
				kube_customresourcedefinition_annotations{annotation_app_k8s_io_owner="@foo",customresourcedefinition="bars.example.com"} 1
				kube_customresourcedefinition_labels{customresourcedefinition="bars.example.com",label_app="mysql-server"} 1
			`,
			MetricNames: []string{
				"kube_customresourcedefinition_annotations",
				"kube_customresourcedefinition_labels",
			},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(customResourceDefinitionMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(customResourceDefinitionMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}

// The builder's field selector carries namespace and node selectors that do not
// apply to CustomResourceDefinitions (the API server rejects spec.nodeName for
// them), so the cluster-scoped list-watch must drop it and the namespace, and
// build a single store regardless of how many namespaces are configured.
func TestCustomResourceDefinitionStoresAreClusterScoped(t *testing.T) {
	l, err := allowdenylist.New(map[string]struct{}{}, map[string]struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	client := apiextensionsfake.NewSimpleClientset()
	listed := make(chan clienttesting.ListAction, 1)
	client.PrependReactor("list", "customresourcedefinitions", func(action clienttesting.Action) (bool, runtime.Object, error) {
		select {
		case listed <- action.(clienttesting.ListAction):
		default:
		}
		return false, nil, nil
	})

	b := NewBuilder()
	b.WithMetrics(prometheus.NewRegistry())
	b.WithKubeClient(fake.NewSimpleClientset())
	b.WithSharding(0, 1)
	b.WithContext(t.Context())
	b.WithNamespaces(options.NamespaceList{"ns1", "ns2"})
	b.WithFieldSelectorFilter("metadata.namespace!=kube-system,spec.nodeName=node-1")
	b.WithFamilyGeneratorFilter(l)
	b.WithGenerateStoresFunc(b.DefaultGenerateStoresFunc())

	if got := len(b.buildCustomResourceDefinitionStores()); got != 0 {
		t.Fatalf("stores without an apiextensions client: got %d, want 0", got)
	}

	b.WithApiextensionsClient(client)
	if got := len(b.buildCustomResourceDefinitionStores()); got != 1 {
		t.Fatalf("stores across 2 namespaces: got %d, want 1", got)
	}

	select {
	case action := <-listed:
		if ns := action.GetNamespace(); ns != "" {
			t.Errorf("list namespace: got %q, want cluster-scoped", ns)
		}
		if fs := action.GetListRestrictions().Fields.String(); fs != "" {
			t.Errorf("list field selector: got %q, want none", fs)
		}
	case <-time.After(wait.ForeverTestTimeout):
		t.Fatal("customresourcedefinitions were never listed")
	}
}
