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
	"os"
	"testing"
)

func TestOptionsParse(t *testing.T) {
	tests := []struct {
		Desc         string
		Args         []string
		ExpectsError bool
	}{
		{
			Desc:         "resources command line argument",
			Args:         []string{"./kube-state-metrics", "--resources=configmaps,pods"},
			ExpectsError: false,
		},
		{
			Desc:         "namespaces command line argument",
			Args:         []string{"./kube-state-metrics", "--namespaces=default,kube-system"},
			ExpectsError: false,
		},
		{
			Desc:         "foo command line argument",
			Args:         []string{"./kube-state-metrics", "--foo=bar,baz"},
			ExpectsError: true,
		},
	}

	opts := NewOptions()
	opts.AddFlags(InitCommand)

	for _, test := range tests {
		t.Run(test.Desc, func(t *testing.T) {
			os.Args = test.Args

			err := opts.Parse()

			if !test.ExpectsError && err != nil {
				t.Errorf("Error for test with description: %s: %v", test.Desc, err.Error())
			}

			if test.ExpectsError && err == nil {
				t.Errorf("Expected error for test with description: %s", test.Desc)
			}
		})
	}
}

func TestOptionsValidateLabelSelectors(t *testing.T) {
	tests := []struct {
		Desc         string
		Configure    func(*Options)
		ExpectsError bool
	}{
		{
			Desc: "known but disabled resource selector is allowed",
			Configure: func(opts *Options) {
				opts.Resources = ResourceSet{"nodes": {}}
				opts.LabelSelectors = LabelSelectorSet{"pods": "app=frontend"}
			},
			ExpectsError: false,
		},
		{
			Desc: "invalid label selector syntax",
			Configure: func(opts *Options) {
				opts.LabelSelectors = LabelSelectorSet{"pods": "app in (frontend"}
			},
			ExpectsError: true,
		},
		{
			Desc: "unknown label selector resource",
			Configure: func(opts *Options) {
				opts.LabelSelectors = LabelSelectorSet{"foos": "app=frontend"}
			},
			ExpectsError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Desc, func(t *testing.T) {
			opts := NewOptions()
			opts.AutoGoMemlimitRatio = 0.9
			test.Configure(opts)

			err := opts.Validate()
			if !test.ExpectsError && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
			if test.ExpectsError && err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
