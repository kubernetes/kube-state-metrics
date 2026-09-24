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

func TestValidateLabelPrefix(t *testing.T) {
	tests := []struct {
		prefix    string
		wantError bool
	}{
		// valid
		{prefix: "", wantError: false},
		{prefix: "label", wantError: false},
		{prefix: "annotation", wantError: false},
		{prefix: "my_prefix", wantError: false},
		{prefix: "L", wantError: false},
		// single underscore followed by letters is fine
		{prefix: "_a", wantError: false},
		// invalid: produces __<key> labels (reserved by Prometheus)
		{prefix: "_", wantError: true},
		{prefix: "__", wantError: true},
		{prefix: "__foo", wantError: true},
		// invalid characters
		{prefix: "foo.bar", wantError: true},
		{prefix: "foo-bar", wantError: true},
		{prefix: "foo bar", wantError: true},
		// starts with digit
		{prefix: "123bad", wantError: true},
	}

	for _, tc := range tests {
		t.Run("prefix="+tc.prefix, func(t *testing.T) {
			err := validateLabelPrefix(tc.prefix, "metric-labels-prefix")
			if tc.wantError && err == nil {
				t.Errorf("expected error for prefix %q, got nil", tc.prefix)
			}
			if !tc.wantError && err != nil {
				t.Errorf("unexpected error for prefix %q: %v", tc.prefix, err)
			}
		})
	}
}

func TestValidatePrefixViaOptions(t *testing.T) {
	tests := []struct {
		desc              string
		labelsPrefix      string
		annotationsPrefix string
		wantError         bool
	}{
		{
			desc:              "defaults are valid",
			labelsPrefix:      "label",
			annotationsPrefix: "annotation",
			wantError:         false,
		},
		{
			desc:              "empty prefixes are valid",
			labelsPrefix:      "",
			annotationsPrefix: "",
			wantError:         false,
		},
		{
			desc:              "invalid labels prefix rejected",
			labelsPrefix:      "foo.bar",
			annotationsPrefix: "annotation",
			wantError:         true,
		},
		{
			desc:              "invalid annotations prefix rejected",
			labelsPrefix:      "label",
			annotationsPrefix: "__reserved",
			wantError:         true,
		},
		{
			desc:              "underscore-only labels prefix rejected",
			labelsPrefix:      "_",
			annotationsPrefix: "annotation",
			wantError:         true,
		},
		{
			desc:              "underscore-only annotations prefix rejected",
			labelsPrefix:      "label",
			annotationsPrefix: "_",
			wantError:         true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			opts := NewOptions()
			opts.LabelsPrefix = tc.labelsPrefix
			opts.AnnotationsPrefix = tc.annotationsPrefix
			err := opts.Validate()
			if tc.wantError && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tc.wantError && err != nil {
				t.Errorf("unexpected validation error: %v", err)
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
