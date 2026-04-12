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
	"math"
	"os"
	"reflect"
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

// A rejected option must abort Execute before Run starts anything.
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
				Use:  "kube-state-metrics",
				Args: cobra.NoArgs,
				Run:  func(_ *cobra.Command, _ []string) { ran = true },
			}

			opts := NewOptions()
			opts.AddFlags(cmd)
			cmd.SetArgs(test.Args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			err := opts.Parse()
			if (err != nil) != test.ExpectsError {
				t.Fatalf("Parse() error = %v, expectsError %v", err, test.ExpectsError)
			}
			if ran == test.ExpectsError {
				t.Errorf("Run called = %v, expectsError %v", ran, test.ExpectsError)
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
			name:    "gomemlimit ratio of NaN",
			mutate:  func(o *Options) { o.AutoGoMemlimitRatio = math.NaN() },
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
		{
			// The resource restriction exists because --node filters by
			// spec.nodeName, so it must stay scoped to node-mode runs.
			name:   "non-node run with an unshardable resource",
			mutate: func(o *Options) { o.Resources = ResourceSet{"deployments": struct{}{}} },
		},
		{
			name:    "zero total shards",
			mutate:  func(o *Options) { o.TotalShards = 0 },
			wantErr: true,
		},
		{
			name:    "negative total shards",
			mutate:  func(o *Options) { o.TotalShards = -1 },
			wantErr: true,
		},
		{
			name:    "shard index beyond total shards",
			mutate:  func(o *Options) { o.Shard = 3; o.TotalShards = 3 },
			wantErr: true,
		},
		{
			name:    "negative shard index",
			mutate:  func(o *Options) { o.Shard = -1 },
			wantErr: true,
		},
		{
			name:   "last shard of a sharded set",
			mutate: func(o *Options) { o.Shard = 2; o.TotalShards = 3 },
		},
		{
			name: "label selector for a known but disabled resource",
			mutate: func(o *Options) {
				o.Resources = ResourceSet{"nodes": {}}
				o.LabelSelectors = LabelSelectorSet{"pods": "app=frontend"}
			},
		},
		{
			name:    "invalid label selector syntax",
			mutate:  func(o *Options) { o.LabelSelectors = LabelSelectorSet{"pods": "app in (frontend"} },
			wantErr: true,
		},
		{
			// Unknown resources are rejected when the store builder is set up.
			name:   "label selector for an unknown resource",
			mutate: func(o *Options) { o.LabelSelectors = LabelSelectorSet{"foos": "app=frontend"} },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := NewOptions()
			// AddFlags applies the defaults; it panics if called twice on one command.
			opts.AddFlags(&cobra.Command{})
			tc.mutate(opts)

			if err := opts.Validate(); (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestOptionsClone(t *testing.T) {
	opts := NewOptions()
	opts.AnnotationsAllowList = LabelsAllowList{"pods": {"team"}}
	opts.LabelSelectors = LabelSelectorSet{"pods": "app=frontend"}
	opts.LabelsAllowList = LabelsAllowList{"pods": {"app"}}
	opts.MetricAllowlist = MetricSet{"kube_pod_info": {}}
	opts.MetricDenylist = MetricSet{"kube_pod_labels": {}}
	opts.MetricOptInList = MetricSet{"kube_pod_ips": {}}
	opts.Resources = ResourceSet{"pods": {}}
	opts.Namespaces = NamespaceList{"team-a"}
	opts.NamespacesDenylist = NamespaceList{"team-b"}
	opts.Port = 8080

	c := opts.Clone()
	// The clone is the baseline of each config reload, so it has to carry
	// every value over, not only be independent of the original.
	if !reflect.DeepEqual(c, opts) {
		t.Fatalf("clone differs from the original:\n got %+v\nwant %+v", c, opts)
	}

	c.AnnotationsAllowList["secrets"] = []string{"*"}
	c.AnnotationsAllowList["pods"][0] = "changed"
	c.LabelSelectors["nodes"] = "tenant=team-a"
	c.LabelsAllowList["secrets"] = []string{"*"}
	c.MetricAllowlist["changed"] = struct{}{}
	c.MetricDenylist["changed"] = struct{}{}
	c.MetricOptInList["changed"] = struct{}{}
	c.Resources["secrets"] = struct{}{}
	c.Namespaces[0] = "changed"
	c.NamespacesDenylist[0] = "changed"
	c.Port = 9090

	if len(opts.AnnotationsAllowList) != 1 || opts.AnnotationsAllowList["pods"][0] != "team" {
		t.Errorf("AnnotationsAllowList shared with clone: %v", opts.AnnotationsAllowList)
	}
	if len(opts.LabelSelectors) != 1 {
		t.Errorf("LabelSelectors shared with clone: %v", opts.LabelSelectors)
	}
	if len(opts.LabelsAllowList) != 1 {
		t.Errorf("LabelsAllowList shared with clone: %v", opts.LabelsAllowList)
	}
	if len(opts.MetricAllowlist) != 1 || len(opts.MetricDenylist) != 1 || len(opts.MetricOptInList) != 1 {
		t.Errorf("metric lists shared with clone: %v %v %v", opts.MetricAllowlist, opts.MetricDenylist, opts.MetricOptInList)
	}
	if len(opts.Resources) != 1 {
		t.Errorf("Resources shared with clone: %v", opts.Resources)
	}
	if opts.Namespaces[0] != "team-a" || opts.NamespacesDenylist[0] != "team-b" {
		t.Errorf("namespace lists shared with clone: %v %v", opts.Namespaces, opts.NamespacesDenylist)
	}
	if opts.Port != 8080 {
		t.Errorf("Port changed through clone: %d", opts.Port)
	}
}
