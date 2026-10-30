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

	"github.com/fsnotify/fsnotify"

	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func TestReloadedOptions(t *testing.T) {
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
	base.Kubeconfig = "/etc/ksm/kubeconfig-from-flag"

	tests := []struct {
		name           string
		config         string
		wantCRC        string
		wantKubeconfig string
		wantOK         bool
	}{
		{
			name:           "config file overrides the flags",
			config:         "custom_resource_state_config_file: /etc/ksm/from-config.yaml\nkubeconfig: /etc/ksm/kubeconfig-from-config\n",
			wantCRC:        "/etc/ksm/from-config.yaml",
			wantKubeconfig: "/etc/ksm/kubeconfig-from-config",
			wantOK:         true,
		},
		{
			name:           "override removed falls back to the flags",
			config:         "port: 8080\n",
			wantCRC:        "/etc/ksm/from-flag.yaml",
			wantKubeconfig: "/etc/ksm/kubeconfig-from-flag",
			wantOK:         true,
		},
		{name: "unparsable config file", config: "port: [\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeConfig(tt.config)
			got, ok := reloadedOptions(base)
			if ok != tt.wantOK {
				t.Fatalf("got ok %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.CustomResourceConfigFile != tt.wantCRC || got.Kubeconfig != tt.wantKubeconfig {
				t.Errorf("got (%q, %q), want (%q, %q)", got.CustomResourceConfigFile, got.Kubeconfig, tt.wantCRC, tt.wantKubeconfig)
			}
		})
	}
	if base.CustomResourceConfigFile != "/etc/ksm/from-flag.yaml" || base.Kubeconfig != "/etc/ksm/kubeconfig-from-flag" {
		t.Errorf("the base options must not change, got (%q, %q)", base.CustomResourceConfigFile, base.Kubeconfig)
	}

	t.Run("removed key", func(t *testing.T) {
		writeConfig("custom_resource_config_file: /etc/ksm/removed.yaml\n")
		if _, ok := reloadedOptions(base); ok {
			t.Error("expected a config file with the removed key to be rejected")
		}
	})
}

func TestMovableWatch(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.yaml")
	second := filepath.Join(dir, "second.yaml")
	for _, f := range []string{first, second} {
		if err := os.WriteFile(f, []byte("a: 1\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	events := make(chan fsnotify.Event, 16)
	onChange := func(e fsnotify.Event) { events <- e }
	w := &movableWatch{name: "test file"}
	t.Cleanup(func() { w.watch("", onChange) })

	w.watch(first, onChange)
	w.watch(second, onChange)

	if err := os.WriteFile(first, []byte("a: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectNoEvent(t, events)

	if err := os.WriteFile(second, []byte("a: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectEvent(t, events)
}
