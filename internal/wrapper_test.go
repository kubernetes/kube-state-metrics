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

func TestLoadConfigFile(t *testing.T) {
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

	t.Run("missing file is left to the run", func(t *testing.T) {
		configFile, got, err := loadConfigFile(base)
		if err != nil || configFile != nil {
			t.Fatalf("got (%q, %v), want no contents and no error", configFile, err)
		}
		if got.CustomResourceConfigFile != "/etc/ksm/from-flag.yaml" {
			t.Errorf("expected the flags, got %q", got.CustomResourceConfigFile)
		}
	})

	tests := []struct {
		name           string
		config         string
		wantCRC        string
		wantKubeconfig string
		wantErr        bool
	}{
		{
			name:           "config file overrides the flags",
			config:         "custom_resource_state_config_file: /etc/ksm/from-config.yaml\nkubeconfig: /etc/ksm/kubeconfig-from-config\n",
			wantCRC:        "/etc/ksm/from-config.yaml",
			wantKubeconfig: "/etc/ksm/kubeconfig-from-config",
		},
		{
			name:           "override removed falls back to the flags",
			config:         "port: 8080\n",
			wantCRC:        "/etc/ksm/from-flag.yaml",
			wantKubeconfig: "/etc/ksm/kubeconfig-from-flag",
		},
		{
			name:           "empty file",
			config:         "",
			wantCRC:        "/etc/ksm/from-flag.yaml",
			wantKubeconfig: "/etc/ksm/kubeconfig-from-flag",
		},
		{name: "unparsable config file", config: "port: [\n", wantErr: true},
		{name: "unknown key", config: "prot: 9090\n", wantErr: true},
		{name: "removed key", config: "custom_resource_config_file: /etc/ksm/removed.yaml\n", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeConfig(tt.config)
			configFile, got, err := loadConfigFile(base)
			if (err != nil) != tt.wantErr {
				t.Fatalf("got error %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			// The run is given these contents instead of reading the file
			// again, and nil would tell it that there is no config file.
			if configFile == nil || string(configFile) != tt.config {
				t.Errorf("got contents %q, want %q", configFile, tt.config)
			}
			if got.CustomResourceConfigFile != tt.wantCRC || got.Kubeconfig != tt.wantKubeconfig {
				t.Errorf("got (%q, %q), want (%q, %q)", got.CustomResourceConfigFile, got.Kubeconfig, tt.wantCRC, tt.wantKubeconfig)
			}
		})
	}
	if base.CustomResourceConfigFile != "/etc/ksm/from-flag.yaml" || base.Kubeconfig != "/etc/ksm/kubeconfig-from-flag" {
		t.Errorf("the base options must not change, got (%q, %q)", base.CustomResourceConfigFile, base.Kubeconfig)
	}

	t.Run("no config file", func(t *testing.T) {
		if configFile, _, err := loadConfigFile(options.NewOptions()); err != nil || configFile != nil {
			t.Errorf("got (%q, %v), want no contents and no error", configFile, err)
		}
	})

	// A directory exists but cannot be read as a file, like a file without
	// read permission, and the run would exit on it.
	t.Run("unreadable file", func(t *testing.T) {
		unreadable := options.NewOptions()
		unreadable.Config = dir
		if _, _, err := loadConfigFile(unreadable); err == nil {
			t.Error("expected an error")
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
