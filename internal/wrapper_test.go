/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

package internal

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func TestCustomResourceConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	writeConfig := func(content string) {
		t.Helper()
		if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	base := options.NewOptions()
	base.Config = configPath
	base.CustomResourceConfigFile = "/etc/ksm/from-flag.yaml"

	tests := []struct {
		name   string
		config string
		want   string
		wantOK bool
	}{
		{"config file overrides the flag", "custom_resource_state_config_file: /etc/ksm/from-config.yaml\n", "/etc/ksm/from-config.yaml", true},
		{"override removed falls back to the flag", "port: 8080\n", "/etc/ksm/from-flag.yaml", true},
		{"unparsable config file", "port: [\n", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeConfig(tt.config)
			got, ok := customResourceConfigFile(base)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("got (%q, %v), want (%q, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
	if base.CustomResourceConfigFile != "/etc/ksm/from-flag.yaml" {
		t.Errorf("the base options must not change, got %q", base.CustomResourceConfigFile)
	}

	t.Run("deprecated key", func(t *testing.T) {
		noFlag := options.NewOptions()
		noFlag.Config = configPath
		writeConfig("custom_resource_config_file: /etc/ksm/deprecated.yaml\n")
		if got, ok := customResourceConfigFile(noFlag); got != "/etc/ksm/deprecated.yaml" || !ok {
			t.Errorf("got (%q, %v), want the deprecated key's file", got, ok)
		}
	})
}
