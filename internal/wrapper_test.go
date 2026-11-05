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
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"

	"k8s.io/kube-state-metrics/v2/pkg/app"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func writeMinimalKubeconfig(t *testing.T, path string) {
	t.Helper()
	content := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: c
contexts:
- context:
    cluster: c
    user: u
  name: ctx
current-context: ctx
users:
- name: u
  user:
    token: test
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeMinimalCRSConfig(t *testing.T, path string) {
	t.Helper()
	content := `kind: CustomResourceStateMetrics
apiVersion: k8s.io/kube-state-metrics/v2/customresourcestate/v1alpha1
spec:
  resources: []
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestReloadedOptions(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	flagKube := filepath.Join(dir, "flag-kube.yaml")
	flagCRC := filepath.Join(dir, "flag-crc.yaml")
	writeMinimalKubeconfig(t, flagKube)
	writeMinimalCRSConfig(t, flagCRC)

	writeConfig := func(content string) {
		t.Helper()
		if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	base := options.NewOptions()
	base.Config = configPath
	base.CustomResourceConfigFile = flagCRC
	base.Kubeconfig = flagKube

	tests := []struct {
		name           string
		config         string
		wantCRC        string
		wantKubeconfig string
		wantOK         bool
	}{
		{
			name:           "config file overrides the flags",
			config:         "custom_resource_state_config_file: " + filepath.Join(dir, "cfg-crc.yaml") + "\nkubeconfig: " + filepath.Join(dir, "cfg-kube.yaml") + "\n",
			wantCRC:        filepath.Join(dir, "cfg-crc.yaml"),
			wantKubeconfig: filepath.Join(dir, "cfg-kube.yaml"),
			wantOK:         true,
		},
		{
			name:           "override removed falls back to the flags",
			config:         "port: 8080\n",
			wantCRC:        flagCRC,
			wantKubeconfig: flagKube,
			wantOK:         true,
		},
		{name: "unparsable config file", config: "port: [\n"},
		{name: "negative store sync timeout", config: "store_sync_timeout: -1s\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantOK {
				if tt.wantCRC != flagCRC {
					writeMinimalCRSConfig(t, tt.wantCRC)
				}
				if tt.wantKubeconfig != flagKube {
					writeMinimalKubeconfig(t, tt.wantKubeconfig)
				}
			}
			writeConfig(tt.config)
			got, err := app.PrepareRunOptionsForReload(base)
			ok := err == nil
			if ok != tt.wantOK {
				t.Fatalf("got ok %v, want %v (err=%v)", ok, tt.wantOK, err)
			}
			if !ok {
				return
			}
			if got.CustomResourceConfigFile != tt.wantCRC || got.Kubeconfig != tt.wantKubeconfig {
				t.Errorf("got (%q, %q), want (%q, %q)", got.CustomResourceConfigFile, got.Kubeconfig, tt.wantCRC, tt.wantKubeconfig)
			}
		})
	}
	if base.CustomResourceConfigFile != flagCRC || base.Kubeconfig != flagKube {
		t.Errorf("the base options must not change, got (%q, %q)", base.CustomResourceConfigFile, base.Kubeconfig)
	}

	t.Run("deprecated key", func(t *testing.T) {
		noFlag := options.NewOptions()
		noFlag.Config = configPath
		deprecatedCRC := filepath.Join(dir, "deprecated.yaml")
		writeMinimalCRSConfig(t, deprecatedCRC)
		writeMinimalKubeconfig(t, filepath.Join(dir, "deprecated-kube.yaml"))
		writeConfig("custom_resource_config_file: " + deprecatedCRC + "\nkubeconfig: " + filepath.Join(dir, "deprecated-kube.yaml") + "\n")
		got, err := app.PrepareRunOptionsForReload(noFlag)
		if err != nil || got.CustomResourceConfigFile != deprecatedCRC {
			t.Errorf("expected the deprecated key's file, got %v err=%v", got, err)
		}
	})
}

func TestReloadPreflightRejectsInvalidCustomResourceConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	crcPath := filepath.Join(dir, "crc.yaml")
	kubePath := filepath.Join(dir, "kube.yaml")
	writeMinimalKubeconfig(t, kubePath)
	if err := os.WriteFile(configPath, []byte("custom_resource_state_config_file: "+crcPath+"\nkubeconfig: "+kubePath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(crcPath, []byte("spec: not-an-object\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	base := options.NewOptions()
	base.Config = configPath
	if _, ok := reloadedOptions(base); ok {
		t.Fatal("expected malformed custom resource config to fail reload preflight")
	}
}

func TestWrapper_StartupReloadSingleRunLaunch(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	kubePath := filepath.Join(dir, "kube.yaml")
	crcPath := filepath.Join(dir, "crc.yaml")
	writeMinimalKubeconfig(t, kubePath)
	writeMinimalCRSConfig(t, crcPath)
	config := "custom_resource_state_config_file: " + crcPath + "\n" +
		"kubeconfig: " + kubePath + "\n"
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	base := options.NewOptions()
	base.Config = configPath

	var launches atomic.Int32
	firstEntry := make(chan struct{})
	checkpoint := make(chan struct{}, 1)
	allowUnlock := make(chan struct{})
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() { close(allowUnlock) })
	}
	t.Cleanup(release)

	done := make(chan struct{})
	t.Cleanup(func() { close(done) })

	run := func(ctx context.Context, _ *options.Options) {
		launches.Add(1)
		close(firstEntry)
		<-ctx.Done()
	}

	go runWrapperLifecycle(base, run, &wrapperLifecycleHooks{
		holdingReloadMuBeforeFirstLaunch: func(restart func(fsnotify.Event), proceedWithFirstLaunch func()) {
			go restart(fsnotify.Event{Name: "during-startup"})
			proceedWithFirstLaunch()
		},
		afterFirstLaunchScheduled: func() {
			select {
			case <-firstEntry:
			case <-allowUnlock:
				return
			case <-time.After(2 * time.Second):
				checkpoint <- struct{}{}
				<-allowUnlock
				return
			}
			checkpoint <- struct{}{}
			<-allowUnlock
		},
		restartPortReleaseDelay: time.Millisecond,
		done:                    done,
	})

	select {
	case <-checkpoint:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for startup checkpoint")
	}
	if launches.Load() != 1 {
		t.Fatalf("expected exactly one launch before reload proceeds, got %d", launches.Load())
	}
	release()
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
