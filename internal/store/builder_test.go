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
	"reflect"
	"slices"
	"testing"

	"k8s.io/kube-state-metrics/v2/pkg/options"
)

type LabelsAllowList options.LabelsAllowList

type expectedError struct {
	expectedResourceError bool
	expectedLabelError    bool
	expectedNotEqual      bool
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
		if err != nil && !test.err.expectedResourceError {
			t.Log("Did not expect error while setting resources (--resources).")
			t.Errorf("Test error for Desc: %s. Got Error: %v", test.Desc, err)
		}

		// Evaluate.
		if !slices.Equal(b.enabledResources, test.Wanted) && err == nil {
			t.Log("Expected enabled resources to be equal.")
			t.Errorf("Test error for Desc: %s\n Want: \n%+v\n Got: \n%#+v", test.Desc, test.Wanted, b.enabledResources)
		}
	}
}
