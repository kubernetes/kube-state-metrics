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

package options

import (
	"reflect"
	"testing"
)

func TestResourceSetSet(t *testing.T) {
	tests := []struct {
		Desc        string
		Value       string
		Wanted      ResourceSet
		WantedError bool
	}{
		{
			Desc:        "empty resources",
			Value:       "",
			Wanted:      ResourceSet{},
			WantedError: false,
		},
		{
			Desc:  "normal resources",
			Value: "configmaps,cronjobs,daemonsets,deployments",
			Wanted: ResourceSet(map[string]struct{}{
				"configmaps":  {},
				"cronjobs":    {},
				"daemonsets":  {},
				"deployments": {},
			}),
			WantedError: false,
		},
	}

	for _, test := range tests {
		cs := &ResourceSet{}
		gotError := cs.Set(test.Value)
		if !(((gotError == nil && !test.WantedError) || (gotError != nil && test.WantedError)) && reflect.DeepEqual(*cs, test.Wanted)) {
			t.Errorf("Test error for Desc: %s. Want: %+v. Got: %+v. Wanted Error: %v, Got Error: %v", test.Desc, test.Wanted, *cs, test.WantedError, gotError)
		}
	}
}

func TestNamespaceListSet(t *testing.T) {
	tests := []struct {
		Desc   string
		Value  string
		Wanted NamespaceList
	}{
		{
			Desc:   "empty namespacelist",
			Value:  "",
			Wanted: NamespaceList{},
		},
		{
			Desc:  "normal namespacelist",
			Value: "default, kube-system",
			Wanted: NamespaceList([]string{
				"default",
				"kube-system",
			}),
		},
	}

	for _, test := range tests {
		ns := &NamespaceList{}
		gotError := ns.Set(test.Value)
		if gotError != nil || !reflect.DeepEqual(*ns, test.Wanted) {
			t.Errorf("Test error for Desc: %s. Want: %+v. Got: %+v. Got Error: %v", test.Desc, test.Wanted, *ns, gotError)
		}
	}
}

func TestNamespaceList_GetNamespaces(t *testing.T) {
	tests := []struct {
		Desc       string
		Namespaces NamespaceList
		Wanted     NamespaceList
	}{
		{
			Desc:       "empty DeniedNamespaces",
			Namespaces: NamespaceList{},
			Wanted:     NamespaceList{""},
		},
		{
			Desc:       "all DeniedNamespaces",
			Namespaces: DefaultNamespaces,
			Wanted:     NamespaceList{""},
		},
		{
			Desc:       "general namespaceDenylist",
			Namespaces: NamespaceList{"default", "kube-system"},
			Wanted:     NamespaceList{"default", "kube-system"},
		},
	}

	for i, test := range tests {
		ns := &tests[i].Namespaces
		allowedNamespaces := ns.GetNamespaces()
		if !reflect.DeepEqual(allowedNamespaces, test.Wanted) {
			t.Errorf("Test error for Desc: %s. Want: %+v. Got: %+v.", test.Desc, test.Wanted, allowedNamespaces)
		}
	}
}

func TestNamespaceList_ExcludeNamespacesFieldSelector(t *testing.T) {
	tests := []struct {
		Desc             string
		Namespaces       NamespaceList
		DeniedNamespaces NamespaceList
		Wanted           string
	}{
		{
			Desc:             "empty DeniedNamespaces",
			Namespaces:       NamespaceList{"default", "kube-system"},
			DeniedNamespaces: NamespaceList{},
			Wanted:           "",
		},
		{
			Desc:             "all DeniedNamespaces",
			Namespaces:       DefaultNamespaces,
			DeniedNamespaces: NamespaceList{"some-system"},
			Wanted:           "metadata.namespace!=some-system",
		},
		{
			Desc:             "general case",
			Namespaces:       DefaultNamespaces,
			DeniedNamespaces: NamespaceList{"case1-system", "case2-system"},
			Wanted:           "metadata.namespace!=case1-system,metadata.namespace!=case2-system",
		},
	}

	for _, test := range tests {
		ns := test.Namespaces
		deniedNS := test.DeniedNamespaces
		actual := ns.GetExcludeNSFieldSelector(deniedNS)
		if !reflect.DeepEqual(actual, test.Wanted) {
			t.Errorf("Test error for Desc: %s. Want: %+v. Got: %+v.", test.Desc, test.Wanted, actual)
		}
	}
}

func TestNodeFieldSelector(t *testing.T) {
	tests := []struct {
		Desc   string
		Node   NodeType
		Wanted string
	}{
		{
			Desc:   "empty node name",
			Node:   "",
			Wanted: "",
		},
		{
			Desc:   "with node name",
			Node:   "k8s-node-1",
			Wanted: "spec.nodeName=k8s-node-1",
		},
	}

	for _, test := range tests {
		node := test.Node
		actual := node.GetNodeFieldSelector()
		if !reflect.DeepEqual(actual, test.Wanted) {
			t.Errorf("Test error for Desc: %s. Want: %+v. Got: %+v.", test.Desc, test.Wanted, actual)
		}
	}
}

func TestMergeFieldSelectors(t *testing.T) {
	tests := []struct {
		Desc             string
		Namespaces       NamespaceList
		DeniedNamespaces NamespaceList
		Node             NodeType
		Wanted           string
	}{
		{
			Desc:             "empty DeniedNamespaces",
			Namespaces:       NamespaceList{"default", "kube-system"},
			DeniedNamespaces: NamespaceList{},
			Node:             "",
			Wanted:           "",
		},
		{
			Desc:             "all DeniedNamespaces",
			Namespaces:       DefaultNamespaces,
			DeniedNamespaces: NamespaceList{"some-system"},
			Node:             "",
			Wanted:           "metadata.namespace!=some-system",
		},
		{
			Desc:             "general case",
			Namespaces:       DefaultNamespaces,
			DeniedNamespaces: NamespaceList{"case1-system", "case2-system"},
			Node:             "",
			Wanted:           "metadata.namespace!=case1-system,metadata.namespace!=case2-system",
		},
		{
			Desc:             "empty DeniedNamespaces",
			Namespaces:       NamespaceList{"default", "kube-system"},
			DeniedNamespaces: NamespaceList{},
			Node:             "k8s-node-1",
			Wanted:           "spec.nodeName=k8s-node-1",
		},
		{
			Desc:             "all DeniedNamespaces",
			Namespaces:       DefaultNamespaces,
			DeniedNamespaces: NamespaceList{"some-system"},
			Node:             "k8s-node-1",
			Wanted:           "metadata.namespace!=some-system,spec.nodeName=k8s-node-1",
		},
		{
			Desc:             "general case",
			Namespaces:       DefaultNamespaces,
			DeniedNamespaces: NamespaceList{"case1-system", "case2-system"},
			Node:             "k8s-node-1",
			Wanted:           "metadata.namespace!=case1-system,metadata.namespace!=case2-system,spec.nodeName=k8s-node-1",
		},
	}

	for _, test := range tests {
		ns := test.Namespaces
		deniedNS := test.DeniedNamespaces
		selector1 := ns.GetExcludeNSFieldSelector(deniedNS)
		selector2 := test.Node.GetNodeFieldSelector()
		actual, err := MergeFieldSelectors([]string{selector1, selector2})
		if err != nil {
			t.Errorf("Test error for Desc: %s. Can't merge field selector %v.", test.Desc, err)
		}
		if !reflect.DeepEqual(actual, test.Wanted) {
			t.Errorf("Test error for Desc: %s. Want: %+v. Got: %+v.", test.Desc, test.Wanted, actual)
		}
	}
}

func TestMetricSetSet(t *testing.T) {
	tests := []struct {
		Desc   string
		Value  string
		Wanted MetricSet
	}{
		{
			Desc:   "empty metrics",
			Value:  "",
			Wanted: MetricSet{},
		},
		{
			Desc:  "normal metrics",
			Value: "kube_cronjob_info, kube_cronjob_labels, kube_daemonset_labels",
			Wanted: MetricSet(map[string]struct{}{
				"kube_cronjob_info":     {},
				"kube_cronjob_labels":   {},
				"kube_daemonset_labels": {},
			}),
		},
		{
			Desc:  "newlines are ignored",
			Value: "\n^kube_.+_annotations$,\n   ^kube_secret_labels$\n",
			Wanted: MetricSet{
				"^kube_secret_labels$":  struct{}{},
				"^kube_.+_annotations$": struct{}{},
			},
		},
	}

	for _, test := range tests {
		ms := &MetricSet{}
		gotError := ms.Set(test.Value)
		if gotError != nil || !reflect.DeepEqual(*ms, test.Wanted) {
			t.Errorf("Test error for Desc: %s. Want: %+v. Got: %+v. Got Error: %v", test.Desc, test.Wanted, *ms, gotError)
		}
	}
}

func TestLabelsAllowListSet(t *testing.T) {
	tests := []struct {
		Desc   string
		Value  string
		Wanted LabelsAllowList
		err    bool
	}{
		{
			Desc:   "empty labels list",
			Value:  "",
			Wanted: LabelsAllowList{},
		},
		{
			Desc:   "[invalid] space delimited",
			Value:  "cronjobs=[somelabel,label2] cronjobs=[label3,label4]",
			Wanted: LabelsAllowList(map[string][]string{}),
			err:    true,
		},
		{
			Desc:   "[invalid] normal missing bracket",
			Value:  "cronjobs=[somelabel,label2],cronjobs=label3,label4]",
			Wanted: LabelsAllowList(map[string][]string{}),
			err:    true,
		},

		{
			Desc:   "[invalid] no comma between metrics",
			Value:  "cronjobs=[somelabel,label2]cronjobs=[label3,label4]",
			Wanted: LabelsAllowList(map[string][]string{}),
			err:    true,
		},
		{
			Desc:   "[invalid] no '=' between name and label list",
			Value:  "cronjobs[somelabel,label2]cronjobs=[label3,label4]",
			Wanted: LabelsAllowList(map[string][]string{}),
			err:    true,
		},
		{
			Desc:  "one resource",
			Value: "cronjobs=[somelabel.io,label2/blah]",
			Wanted: LabelsAllowList(map[string][]string{
				"cronjobs": {
					"somelabel.io",
					"label2/blah",
				}}),
		},
		{
			Desc:  "two resources",
			Value: "pods=[podsone,pods-two],nodes=[nodesone,nodestwo],namespaces=[nsone,nstwo]",
			Wanted: LabelsAllowList(map[string][]string{
				"pods": {
					"podsone",
					"pods-two"},
				"nodes": {
					"nodesone",
					"nodestwo"},
				"namespaces": {
					"nsone",
					"nstwo"}}),
		},
		{
			Desc:  "with empty allow labels",
			Value: "cronjobs=[somelabel,label2],pods=[]",
			Wanted: LabelsAllowList(map[string][]string{
				"cronjobs": {
					"somelabel",
					"label2",
				},
				"pods": {}}),
		},
		{
			Desc:  "with wildcard",
			Value: "cronjobs=[*],pods=[*,foo],namespaces=[bar,*]",
			Wanted: LabelsAllowList(map[string][]string{
				"cronjobs": {
					"*",
				},
				"pods": {
					"*",
					"foo",
				},
				"namespaces": {
					"bar",
					"*"}}),
		},
		{
			Desc:  "with key as wildcard",
			Value: "*=[*]",
			Wanted: LabelsAllowList(map[string][]string{
				"*": {
					"*",
				},
			}),
		},
	}

	for _, test := range tests {
		lal := &LabelsAllowList{}
		gotError := lal.Set(test.Value)
		if gotError != nil && !test.err || !reflect.DeepEqual(*lal, test.Wanted) {
			t.Errorf("Test error for Desc: %s\n Want: \n%+v\n Got: \n%#+v\n Got Error: %#v", test.Desc, test.Wanted, *lal, gotError)
		}
	}
}
