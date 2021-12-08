/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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

package namespacelist

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
)

func TestString(t *testing.T) {
	tests := []struct {
		input    NamespaceList
		expected string
	}{
		{NamespaceList{
			"project-1": {},
			"project-2": {},
			"project-3": {},
		}, "project-1,project-2,project-3"},
		{NamespaceList{
			"*": {
				"environment": "production",
				"region":      "eu-west-2",
			},
		}, "*=[environment=production,region=eu-west-2]"},
	}

	for _, test := range tests {
		actual := test.input.String()
		if actual != test.expected {
			t.Errorf("expectedList: %v, got: %v", test.expected, actual)
		}
	}
}

func TestSet(t *testing.T) {
	tests := []struct {
		desc          string
		value         string
		expectedList  NamespaceList
		expectedError error
	}{
		{
			"namespace list with 3 namespaces and no labels",
			"project-1,project-2,project-3",
			NamespaceList{
				"project-1": {},
				"project-2": {},
				"project-3": {},
			},
			nil},
		{
			"namespace list with a wildcard namespace and 2 labels",
			"*=[environment=production,region=eu-west-2]",
			NamespaceList{
				"*": {
					"environment": "production",
					"region":      "eu-west-2",
				},
			},
			nil},
		{
			"unexpected eof syntax error",
			"foo-bar=",
			nil,
			UnexpectedSyntaxError{"EOF", 8},
		},
		{
			"unexpected eof syntax error",
			"foo-bar=[",
			nil,
			UnexpectedSyntaxError{"EOF", 9},
		},
		{
			"unexpected eof syntax error",
			"foo-bar=[a=b,c=d",
			nil,
			UnexpectedSyntaxError{"EOF", 16},
		},
		{
			"namespace with 2 labels, one with and one without a value",
			"foo-bar=[foo=,bar=test]", NamespaceList{
				"foo-bar": {
					"foo": "",
					"bar": "test",
				},
			},
			nil,
		},
	}

	for _, test := range tests {
		actual := NamespaceList{}
		err := actual.Set(test.value)

		if test.expectedError != nil {
			if errors.Is(err, test.expectedError) {
				continue
			}
			t.Errorf("%v: expected: \"%v\", got: \"%v\"", test.desc, test.expectedError, err)
			continue
		}
		if err != nil {
			t.Errorf("%v: there was an unexpected expectedError: %v", test.desc, err)
			continue
		}

		if !reflect.DeepEqual(actual, test.expectedList) {
			t.Errorf("%v: expected: %v, got: %v", test.desc, test.expectedList, actual)
		}
	}
}
