/*
Copyright 2022 The Kubernetes Authors All rights reserved.

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
	"reflect"
	"slices"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/options"
)

type LabelsAllowList options.LabelsAllowList

type expectedError struct {
	expectedResourceError bool
	expectedLabelError    bool
	expectedNotEqual      bool
}

type fakeListerWatcher struct {
	listOptions  []metav1.ListOptions
	watchOptions []metav1.ListOptions
}

func (r *fakeListerWatcher) ListWithContext(_ context.Context, options metav1.ListOptions) (runtime.Object, error) {
	r.listOptions = append(r.listOptions, options)
	return &v1.PodList{}, nil
}

func (r *fakeListerWatcher) WatchWithContext(_ context.Context, options metav1.ListOptions) (watch.Interface, error) {
	r.watchOptions = append(r.watchOptions, options)
	return watch.NewEmptyWatch(), nil
}

func TestWithAllowLabels(t *testing.T) {
	tests := []struct {
		Desc             string
		LabelsAllowlist  map[string][]string
		EnabledResources []string
		Wanted           LabelsAllowList
		err              expectedError
	}{
		{
			Desc:             "wildcard key-value as the only element",
			LabelsAllowlist:  map[string][]string{"*": {"*"}},
			EnabledResources: []string{"cronjobs", "pods", "deployments"},
			Wanted: LabelsAllowList(map[string][]string{
				"deployments": {"*"},
				"pods":        {"*"},
				"cronjobs":    {"*"},
			}),
		},
		{
			Desc:             "wildcard key-value as not the only element",
			LabelsAllowlist:  map[string][]string{"*": {"*"}, "pods": {"*"}, "cronjobs": {"*"}},
			EnabledResources: []string{"cronjobs", "pods", "deployments"},
			Wanted: LabelsAllowList(map[string][]string{
				"deployments": {"*"},
				"pods":        {"*"},
				"cronjobs":    {"*"},
			}),
		},
		{
			Desc:             "wildcard key-value as not the only element, with resource mismatch",
			LabelsAllowlist:  map[string][]string{"*": {"*"}, "pods": {"*"}, "cronjobs": {"*"}, "configmaps": {"*"}},
			EnabledResources: []string{"cronjobs", "pods", "deployments"},
			Wanted:           LabelsAllowList{},
			err: expectedError{
				expectedNotEqual: true,
			},
		},
		{
			Desc:             "wildcard key-value as not the only element, with other mutually-exclusive keys",
			LabelsAllowlist:  map[string][]string{"*": {"*"}, "foo": {"*"}, "bar": {"*"}, "cronjobs": {"*"}},
			EnabledResources: []string{"cronjobs", "pods", "deployments"},
			Wanted:           LabelsAllowList(nil),
			err: expectedError{
				expectedLabelError: true,
			},
		},
		{
			Desc:             "wildcard key-value as not the only element, with other resources that do not exist",
			LabelsAllowlist:  map[string][]string{"*": {"*"}, "cronjobs": {"*"}},
			EnabledResources: []string{"cronjobs", "pods", "deployments", "foo", "bar"},
			Wanted:           LabelsAllowList{},
			err: expectedError{
				expectedResourceError: true,
			},
		},
	}

	for _, test := range tests {
		b := NewBuilder()

		// Set the enabled resources.
		err := b.WithEnabledResources(test.EnabledResources)
		if err != nil && !test.err.expectedResourceError {
			t.Log("Did not expect error while setting resources (--resources).")
			t.Errorf("Test error for Desc: %s. Got Error: %v", test.Desc, err)
		}

		// Resolve the allow list.
		err = b.WithAllowLabels(test.LabelsAllowlist)
		if err != nil && !test.err.expectedLabelError {
			t.Log("Did not expect error while parsing allow list labels (--metric-labels-allowlist).")
			t.Errorf("Test error for Desc: %s. Got Error: %v", test.Desc, err)
		}
		resolvedAllowLabels := LabelsAllowList(b.allowLabelsList)

		// Evaluate.
		if !reflect.DeepEqual(resolvedAllowLabels, test.Wanted) && !test.err.expectedNotEqual {
			t.Log("Expected maps to be equal.")
			t.Errorf("Test error for Desc: %s\n Want: \n%+v\n Got: \n%#+v", test.Desc, test.Wanted, resolvedAllowLabels)
		}
	}
}

func TestWithAllowAnnotations(t *testing.T) {
	tests := []struct {
		Desc                 string
		AnnotationsAllowlist map[string][]string
		EnabledResources     []string
		Wanted               LabelsAllowList
		err                  expectedError
	}{
		{
			Desc:                 "wildcard key-value as the only element",
			AnnotationsAllowlist: map[string][]string{"*": {"*"}},
			EnabledResources:     []string{"cronjobs", "pods", "deployments"},
			Wanted: LabelsAllowList(map[string][]string{
				"deployments": {"*"},
				"pods":        {"*"},
				"cronjobs":    {"*"},
			}),
		},
		{
			Desc:                 "wildcard key-value as not the only element",
			AnnotationsAllowlist: map[string][]string{"*": {"*"}, "pods": {"*"}, "cronjobs": {"*"}},
			EnabledResources:     []string{"cronjobs", "pods", "deployments"},
			Wanted: LabelsAllowList(map[string][]string{
				"deployments": {"*"},
				"pods":        {"*"},
				"cronjobs":    {"*"},
			}),
		},
		{
			Desc:                 "wildcard key-value as not the only element, with resource mismatch",
			AnnotationsAllowlist: map[string][]string{"*": {"*"}, "pods": {"*"}, "cronjobs": {"*"}, "configmaps": {"*"}},
			EnabledResources:     []string{"cronjobs", "pods", "deployments"},
			Wanted:               LabelsAllowList{},
			err: expectedError{
				expectedNotEqual: true,
			},
		},
		{
			Desc:                 "wildcard key-value as not the only element, with other mutually-exclusive keys",
			AnnotationsAllowlist: map[string][]string{"*": {"*"}, "foo": {"*"}, "bar": {"*"}, "cronjobs": {"*"}},
			EnabledResources:     []string{"cronjobs", "pods", "deployments"},
			Wanted:               LabelsAllowList(nil),
			err: expectedError{
				expectedLabelError: true,
			},
		},
		{
			Desc:                 "wildcard key-value as not the only element, with other resources that do not exist",
			AnnotationsAllowlist: map[string][]string{"*": {"*"}, "cronjobs": {"*"}},
			EnabledResources:     []string{"cronjobs", "pods", "deployments", "foo", "bar"},
			Wanted:               LabelsAllowList{},
			err: expectedError{
				expectedResourceError: true,
			},
		},
	}

	for _, test := range tests {
		b := NewBuilder()

		// Set the enabled resources.
		err := b.WithEnabledResources(test.EnabledResources)
		if err != nil && !test.err.expectedResourceError {
			t.Log("Did not expect error while setting resources (--resources).")
			t.Errorf("Test error for Desc: %s. Got Error: %v", test.Desc, err)
		}

		// Resolve the allow list.
		err = b.WithAllowAnnotations(test.AnnotationsAllowlist)
		if err != nil && !test.err.expectedLabelError {
			t.Log("Did not expect error while parsing allow list annotations (--metric-annotations-allowlist).")
			t.Errorf("Test error for Desc: %s. Got Error: %v", test.Desc, err)
		}
		resolvedAllowAnnotations := LabelsAllowList(b.allowAnnotationsList)

		// Evaluate.
		if !reflect.DeepEqual(resolvedAllowAnnotations, test.Wanted) && !test.err.expectedNotEqual {
			t.Log("Expected maps to be equal.")
			t.Errorf("Test error for Desc: %s\n Want: \n%+v\n Got: \n%#+v", test.Desc, test.Wanted, resolvedAllowAnnotations)
		}
	}
}

func TestWithEnabledResources(t *testing.T) {

	tests := []struct {
		Desc             string
		EnabledResources []string
		Wanted           []string
		err              expectedError
	}{
		{
			Desc:             "sorts enabled resources",
			EnabledResources: []string{"pods", "cronjobs", "deployments"},
			Wanted: []string{
				"cronjobs",
				"deployments",
				"pods",
			},
		},
		{
			Desc:             "de-duplicates enabled resources",
			EnabledResources: []string{"pods", "cronjobs", "deployments", "pods"},
			Wanted: []string{
				"cronjobs",
				"deployments",
				"pods",
			},
		},
		{
			Desc:             "error if not exist",
			EnabledResources: []string{"pods", "cronjobs", "deployments", "foo"},
			Wanted:           []string{},
			err: expectedError{
				expectedResourceError: true,
			},
		},
	}
	for _, test := range tests {
		b := NewBuilder()

		// Set the enabled resources.
		err := b.WithEnabledResources(test.EnabledResources)
		if test.err.expectedResourceError {
			if err == nil {
				t.Log("Did not expect error while setting resources (--resources).")
				t.Fatalf("Test error for Desc: %s. Got Error: %v", test.Desc, err)
			} else {
				return
			}
		}
		if err != nil {
			t.Log("...")
			t.Fatal("...", test.Desc, err)
		}

		// Evaluate.
		if !slices.Equal(b.enabledResources, test.Wanted) {
			t.Log("Expected enabled resources to be equal.")
			t.Errorf("Test error for Desc: %s\n Want: \n%+v\n Got: \n%#+v", test.Desc, test.Wanted, b.enabledResources)
		}
	}
}

func TestWithLabelSelectorFilters(t *testing.T) {
	tests := []struct {
		Desc          string
		LabelSelector map[string]string
		Want          map[string]string
		WantError     bool
	}{
		{
			Desc:          "builtin resource selector",
			LabelSelector: map[string]string{"pods": "app=frontend", "nodes": "tenant=team-a"},
			Want:          map[string]string{"pods": "app=frontend", "nodes": "tenant=team-a"},
		},
		{
			Desc:          "unknown resource selector",
			LabelSelector: map[string]string{"foos": "app=frontend"},
			WantError:     true,
		},
	}

	for _, test := range tests {
		b := NewBuilder()
		err := b.WithLabelSelectorFilters(test.LabelSelector)

		if (err != nil) != test.WantError {
			t.Fatalf("Test error for Desc: %s. Wanted Error: %v, Got Error: %v", test.Desc, test.WantError, err)
		}
		if err == nil && !reflect.DeepEqual(b.labelSelectorFilters, test.Want) {
			t.Errorf("Test error for Desc: %s\n Want: \n%+v\n Got: \n%#+v", test.Desc, test.Want, b.labelSelectorFilters)
		}
	}
}

func TestWithLabelSelector(t *testing.T) {
	fakeLW := &fakeListerWatcher{}
	baseLW := &cache.ListWatch{
		ListWithContextFunc:  fakeLW.ListWithContext,
		WatchFuncWithContext: fakeLW.WatchWithContext,
	}
	labelSelectorLW := withLabelSelector(baseLW, "tenant in (team-a,team-b)")
	listerWatcherWithContext := cache.ToListerWatcherWithContext(labelSelectorLW)

	_, err := listerWatcherWithContext.ListWithContext(context.Background(), metav1.ListOptions{FieldSelector: "spec.nodeName=node-a", ResourceVersion: "10"})
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}

	_, err = listerWatcherWithContext.WatchWithContext(context.Background(), metav1.ListOptions{FieldSelector: "spec.nodeName=node-a"})
	if err != nil {
		t.Fatalf("unexpected watch error: %v", err)
	}

	if len(fakeLW.listOptions) != 1 {
		t.Fatalf("expected 1 list call, got %d", len(fakeLW.listOptions))
	}
	if len(fakeLW.watchOptions) != 1 {
		t.Fatalf("expected 1 watch call, got %d", len(fakeLW.watchOptions))
	}
	if got := fakeLW.listOptions[0].LabelSelector; got != "tenant in (team-a,team-b)" {
		t.Fatalf("expected list label selector to be propagated, got %q", got)
	}
	if got := fakeLW.watchOptions[0].LabelSelector; got != "tenant in (team-a,team-b)" {
		t.Fatalf("expected watch label selector to be propagated, got %q", got)
	}
	if got := fakeLW.listOptions[0].FieldSelector; got != "spec.nodeName=node-a" {
		t.Fatalf("expected list field selector to be preserved, got %q", got)
	}
	if got := fakeLW.listOptions[0].ResourceVersion; got != "10" {
		t.Fatalf("expected list resource version to be preserved, got %q", got)
	}
}
