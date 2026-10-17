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

func TestOptionsClone(t *testing.T) {
	opts := NewOptions()
	opts.AnnotationsAllowList = LabelsAllowList{"pods": {"team"}}
	opts.LabelsAllowList = LabelsAllowList{"pods": {"app"}}
	opts.MetricAllowlist = MetricSet{"kube_pod_info": {}}
	opts.MetricDenylist = MetricSet{"kube_pod_labels": {}}
	opts.MetricOptInList = MetricSet{"kube_pod_ips": {}}
	opts.Resources = ResourceSet{"pods": {}}
	opts.Namespaces = NamespaceList{"team-a"}
	opts.NamespacesDenylist = NamespaceList{"team-b"}
	opts.Port = 8080

	c := opts.Clone()
	c.AnnotationsAllowList["secrets"] = []string{"*"}
	c.AnnotationsAllowList["pods"][0] = "changed"
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
