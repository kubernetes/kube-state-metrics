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

package collectors

import (
	"testing"
	"time"

	v1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/pkg/collectors/testutils"
	"k8s.io/kube-state-metrics/pkg/options"
)

type mockCRDStore struct {
	list func() ([]v1beta1.CustomResourceDefinition, error)
}

func (crd mockCRDStore) List() ([]v1beta1.CustomResourceDefinition, error) {
	return crd.list()
}

func TestCRDCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_customresourcedefinition_created Unix creation timestamp.
		# TYPE kube_customresourcedefinition_created gauge
		# HELP kube_customresourcedefinition_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_customresourcedefinition_labels gauge
		# HELP kube_customresourcedefinition_spec_groupversion Information about the customresourcedefinition group and version.
		# TYPE kube_customresourcedefinition_spec_groupversion gauge
		# HELP kube_customresourcedefinition_spec_scope kubernetes customresourcedefinition spec scope.
		# TYPE kube_customresourcedefinition_spec_scope gauge
		# HELP kube_customresourcedefinition_status_condition The condition of a customresourcedefinition.
		# TYPE kube_customresourcedefinition_status_condition gauge
	`

	cases := []struct {
		crds    []v1beta1.CustomResourceDefinition
		metrics []string // which metrics should be checked
		want    string
	}{
		{
			crds: []v1beta1.CustomResourceDefinition{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "crdTestCreateTimeWithoutLabels",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "crdTestLablesWithoutCreateTime",
						Labels: map[string]string{
							"kubebuilder.k8s.io": "0.1.10",
						},
					},
				},
			},

			want: metadata + `
			kube_customresourcedefinition_created{customresourcedefinition="crdTestCreateTimeWithoutLabels"} 1.5e+09
			kube_customresourcedefinition_labels{customresourcedefinition="crdTestCreateTimeWithoutLabels"} 1
			kube_customresourcedefinition_labels{customresourcedefinition="crdTestLablesWithoutCreateTime",label_kubebuilder_k8s_io="0.1.10"} 1
			
			`,
			metrics: []string{"kube_customresourcedefinition_created", "kube_customresourcedefinition_labels"},
		},
		{
			crds: []v1beta1.CustomResourceDefinition{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "crdTestGroupVersionandNamespaceScope",
					},
					Spec: v1beta1.CustomResourceDefinitionSpec{
						Group:   "test",
						Version: "v1beta1",
						Scope:   v1beta1.NamespaceScoped,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "crdTestGroupVersionandClusterScope",
					},
					Spec: v1beta1.CustomResourceDefinitionSpec{
						Group:   "test",
						Version: "v1beta1",
						Scope:   v1beta1.ClusterScoped,
					},
				},
			},

			want: metadata + `
			kube_customresourcedefinition_spec_groupversion{group="test",node="crdTestGroupVersionandClusterScope",version="v1beta1"} 1
			kube_customresourcedefinition_spec_groupversion{group="test",node="crdTestGroupVersionandNamespaceScope",version="v1beta1"} 1
			kube_customresourcedefinition_spec_scope{Scope="Cluster",customresourcedefinition="crdTestGroupVersionandClusterScope"} 1
			kube_customresourcedefinition_spec_scope{Scope="Cluster",customresourcedefinition="crdTestGroupVersionandNamespaceScope"} 0
			kube_customresourcedefinition_spec_scope{Scope="Namespaced",customresourcedefinition="crdTestGroupVersionandClusterScope"} 0
			kube_customresourcedefinition_spec_scope{Scope="Namespaced",customresourcedefinition="crdTestGroupVersionandNamespaceScope"} 1
			`,
			metrics: []string{"kube_customresourcedefinition_spec_groupversion", "kube_customresourcedefinition_spec_scope"},
		},
		{
			crds: []v1beta1.CustomResourceDefinition{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "crdTestStatusTrue",
					},
					Status: v1beta1.CustomResourceDefinitionStatus{
						Conditions: []v1beta1.CustomResourceDefinitionCondition{
							v1beta1.CustomResourceDefinitionCondition{
								Type:   v1beta1.Established,
								Status: v1beta1.ConditionTrue,
							},
							{
								Type:   v1beta1.NamesAccepted,
								Status: v1beta1.ConditionTrue,
							},
							{
								Type:   v1beta1.Terminating,
								Status: v1beta1.ConditionTrue,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "crdTestStatusFalse",
					},
					Status: v1beta1.CustomResourceDefinitionStatus{
						Conditions: []v1beta1.CustomResourceDefinitionCondition{
							v1beta1.CustomResourceDefinitionCondition{
								Type:   v1beta1.Established,
								Status: v1beta1.ConditionFalse,
							},
							{
								Type:   v1beta1.NamesAccepted,
								Status: v1beta1.ConditionFalse,
							},
							{
								Type:   v1beta1.Terminating,
								Status: v1beta1.ConditionFalse,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "crdTestStatusUnknown",
					},
					Status: v1beta1.CustomResourceDefinitionStatus{
						Conditions: []v1beta1.CustomResourceDefinitionCondition{
							v1beta1.CustomResourceDefinitionCondition{
								Type:   v1beta1.Established,
								Status: v1beta1.ConditionUnknown,
							},
							{
								Type:   v1beta1.NamesAccepted,
								Status: v1beta1.ConditionUnknown,
							},
							{
								Type:   v1beta1.Terminating,
								Status: v1beta1.ConditionUnknown,
							},
						},
					},
				},
			},

			want: metadata + `
			kube_customresourcedefinition_status_condition{condition="Established",node="crdTestStatusFalse",status="false"} 1
			kube_customresourcedefinition_status_condition{condition="Established",node="crdTestStatusFalse",status="true"} 0
			kube_customresourcedefinition_status_condition{condition="Established",node="crdTestStatusFalse",status="unknown"} 0
			kube_customresourcedefinition_status_condition{condition="Established",node="crdTestStatusTrue",status="false"} 0
			kube_customresourcedefinition_status_condition{condition="Established",node="crdTestStatusTrue",status="true"} 1
			kube_customresourcedefinition_status_condition{condition="Established",node="crdTestStatusTrue",status="unknown"} 0
			kube_customresourcedefinition_status_condition{condition="Established",node="crdTestStatusUnknown",status="false"} 0
			kube_customresourcedefinition_status_condition{condition="Established",node="crdTestStatusUnknown",status="true"} 0
			kube_customresourcedefinition_status_condition{condition="Established",node="crdTestStatusUnknown",status="unknown"} 1
			kube_customresourcedefinition_status_condition{condition="NamesAccepted",node="crdTestStatusFalse",status="false"} 1
			kube_customresourcedefinition_status_condition{condition="NamesAccepted",node="crdTestStatusFalse",status="true"} 0
			kube_customresourcedefinition_status_condition{condition="NamesAccepted",node="crdTestStatusFalse",status="unknown"} 0
			kube_customresourcedefinition_status_condition{condition="NamesAccepted",node="crdTestStatusTrue",status="false"} 0
			kube_customresourcedefinition_status_condition{condition="NamesAccepted",node="crdTestStatusTrue",status="true"} 1
			kube_customresourcedefinition_status_condition{condition="NamesAccepted",node="crdTestStatusTrue",status="unknown"} 0
			kube_customresourcedefinition_status_condition{condition="NamesAccepted",node="crdTestStatusUnknown",status="false"} 0
			kube_customresourcedefinition_status_condition{condition="NamesAccepted",node="crdTestStatusUnknown",status="true"} 0
			kube_customresourcedefinition_status_condition{condition="NamesAccepted",node="crdTestStatusUnknown",status="unknown"} 1
			kube_customresourcedefinition_status_condition{condition="Terminating",node="crdTestStatusFalse",status="false"} 1
			kube_customresourcedefinition_status_condition{condition="Terminating",node="crdTestStatusFalse",status="true"} 0
			kube_customresourcedefinition_status_condition{condition="Terminating",node="crdTestStatusFalse",status="unknown"} 0
			kube_customresourcedefinition_status_condition{condition="Terminating",node="crdTestStatusTrue",status="false"} 0
			kube_customresourcedefinition_status_condition{condition="Terminating",node="crdTestStatusTrue",status="true"} 1
			kube_customresourcedefinition_status_condition{condition="Terminating",node="crdTestStatusTrue",status="unknown"} 0
			kube_customresourcedefinition_status_condition{condition="Terminating",node="crdTestStatusUnknown",status="false"} 0
			kube_customresourcedefinition_status_condition{condition="Terminating",node="crdTestStatusUnknown",status="true"} 0
			kube_customresourcedefinition_status_condition{condition="Terminating",node="crdTestStatusUnknown",status="unknown"} 1
			`,
			metrics: []string{"kube_customresourcedefinition_status_condition"},
		},
	}
	for _, c := range cases {
		crdc := &crdCollector{
			store: mockCRDStore{
				list: func() ([]v1beta1.CustomResourceDefinition, error) { return c.crds, nil },
			},
			opts: &options.Options{},
		}
		if err := testutils.GatherAndCompare(crdc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
