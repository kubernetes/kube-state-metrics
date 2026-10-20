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
	"io"
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

// Validate used to be called by main after Parse returned, which never happens:
// Parse runs cmd.Execute, and cmd.Run only returns once the process is shutting
// down. It is wired into the command's PreRunE instead, so a rejected option
// aborts before anything is started.
func TestValidateRunsBeforeRun(t *testing.T) {
	tests := []struct {
		Desc         string
		Args         []string
		ExpectsError bool
	}{
		{
			Desc:         "node sharding rejects a resource without a spec.nodeName field selector",
			Args:         []string{"--node=node-1", "--resources=deployments"},
			ExpectsError: true,
		},
		{
			Desc:         "node sharding accepts pods",
			Args:         []string{"--node=node-1", "--resources=pods"},
			ExpectsError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Desc, func(t *testing.T) {
			ran := false
			cmd := &cobra.Command{
				Use:          "kube-state-metrics",
				Args:         cobra.NoArgs,
				SilenceUsage: true,
				Run:          func(_ *cobra.Command, _ []string) { ran = true },
			}

			opts := NewOptions()
			opts.AddFlags(cmd)
			cmd.SetArgs(test.Args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			err := opts.Parse()

			if test.ExpectsError {
				if err == nil {
					t.Fatal("expected Parse to surface the validation error, got nil")
				}
				if ran {
					t.Error("command ran despite failing validation")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !ran {
				t.Error("command did not run")
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
