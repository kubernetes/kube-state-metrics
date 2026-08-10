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

	"github.com/spf13/cobra"

	yaml "sigs.k8s.io/yaml/goyaml.v3"
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

func TestCustomResourceConfigFileDeprecatedAlias(t *testing.T) {
	t.Run("deprecated key populates alias field", func(t *testing.T) {
		opts := NewOptions()
		if err := yaml.Unmarshal([]byte("custom_resource_config_file: /etc/ksm/crs.yaml\n"), opts); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if opts.CustomResourceConfigFileDeprecated != "/etc/ksm/crs.yaml" {
			t.Fatalf("expected deprecated alias to be set, got %q", opts.CustomResourceConfigFileDeprecated)
		}
		if opts.CustomResourceConfigFile != "" {
			t.Fatalf("expected canonical field to remain empty pre-merge, got %q", opts.CustomResourceConfigFile)
		}
	})

	t.Run("canonical key takes precedence when both are set", func(t *testing.T) {
		opts := NewOptions()
		yamlIn := "custom_resource_config_file: /old.yaml\ncustom_resource_state_config_file: /new.yaml\n"
		if err := yaml.Unmarshal([]byte(yamlIn), opts); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if opts.CustomResourceConfigFile != "/new.yaml" {
			t.Fatalf("expected canonical field to win, got %q", opts.CustomResourceConfigFile)
		}
		if opts.CustomResourceConfigFileDeprecated != "/old.yaml" {
			t.Fatalf("expected deprecated field to retain its value, got %q", opts.CustomResourceConfigFileDeprecated)
		}
	})
}

// The node check used to return early, which left every validation below it
// unreachable for the ordinary cluster-wide deployment.
func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mutate  func(*Options)
		wantErr bool
	}{
		{
			name:   "defaults are valid",
			mutate: func(_ *Options) {},
		},
		{
			name:    "gomemlimit ratio above 1 without node",
			mutate:  func(o *Options) { o.AutoGoMemlimitRatio = 5.0 },
			wantErr: true,
		},
		{
			name:    "gomemlimit ratio of 0 without node",
			mutate:  func(o *Options) { o.AutoGoMemlimitRatio = 0 },
			wantErr: true,
		},
		{
			name:    "negative object limit without node",
			mutate:  func(o *Options) { o.ObjectLimit = -1 },
			wantErr: true,
		},
		{
			name:    "gomemlimit ratio above 1 with node",
			mutate:  func(o *Options) { o.Node = "node-1"; o.AutoGoMemlimitRatio = 5.0 },
			wantErr: true,
		},
		{
			name:    "node scoped run with an unshardable resource",
			mutate:  func(o *Options) { o.Node = "node-1"; o.Resources = ResourceSet{"deployments": struct{}{}} },
			wantErr: true,
		},
		{
			name:   "node scoped run with pods",
			mutate: func(o *Options) { o.Node = "node-1"; o.Resources = ResourceSet{"pods": struct{}{}} },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := NewOptions()
			// A fresh command per case: AddFlags panics if the same command has
			// the flags registered twice, and it is what applies the defaults.
			opts.AddFlags(&cobra.Command{})
			tc.mutate(opts)

			err := opts.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected Validate() to fail, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected Validate() to pass, got %v", err)
			}
		})
	}
}
