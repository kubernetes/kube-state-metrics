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
	"context"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
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

func TestCreateCustomResourceDefinitionListWatchFieldSelector(t *testing.T) {
	client := apiextensionsfake.NewSimpleClientset()
	var gotFieldSelector string
	client.PrependReactor("list", "customresourcedefinitions", func(action clienttesting.Action) (bool, runtime.Object, error) {
		gotFieldSelector = action.(clienttesting.ListAction).GetListRestrictions().Fields.String()
		return false, nil, nil
	})

	lw := createCustomResourceDefinitionListWatch(client, "metadata.name!=foos.example.com").(*cache.ListWatch)
	_, err := lw.ListWithContext(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotFieldSelector != "metadata.name!=foos.example.com" {
		t.Errorf("expected field selector to be propagated, got %q", gotFieldSelector)
	}
}
