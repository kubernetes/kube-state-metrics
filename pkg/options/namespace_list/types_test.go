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

package namespace_list

import (
	"reflect"
	"testing"
)

func TestString(t *testing.T) {
	tests := []struct{
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
				"region": "eu-west-2",
			},
		}, "*=[environment=production,region=eu-west-2]"},
	}

	for _, test := range tests {
		actual := test.input.String()
		if actual != test.expected {
			t.Errorf("expected: %v, got: %v", test.expected, actual)
		}
	}
}

func TestSet(t *testing.T) {
	tests := []struct{
		input    string
		expected NamespaceList
	}{
		{"project-1,project-2,project-3", NamespaceList{
			"project-1": {},
			"project-2": {},
			"project-3": {},
		}},
		{"*=[environment=production,region=eu-west-2]", NamespaceList{
			"*": {
				"environment": "production",
				"region": "eu-west-2",
			},
		}},
	}

	for _, test := range tests {
		actual := NamespaceList{}
		err := actual.Set(test.input)
		if err != nil {
			t.Errorf("an unexpected error happened: %v", err)
		}
		if !reflect.DeepEqual(actual, test.expected) {
			t.Errorf("expected: %v, got: %v", test.expected, actual)
		}
	}
}