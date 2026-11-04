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

package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func TestRunKubeStateMetricsRejectsNegativeStoreSyncTimeoutFlag(t *testing.T) {
	opts := options.NewOptions()
	cmd := &cobra.Command{
		Use:          "kube-state-metrics",
		SilenceUsage: true,
		RunE: func(*cobra.Command, []string) error {
			return RunKubeStateMetrics(context.Background(), opts)
		},
	}
	opts.AddFlags(cmd)
	cmd.SetArgs([]string{"--store-sync-timeout=-1s"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "store-sync-timeout") {
		t.Fatalf("expected a negative flag to fail before the handler starts, got %v", err)
	}
}

func TestRunKubeStateMetricsNegativeFileTimeoutFailsAtStartup(t *testing.T) {
	opts := options.NewOptions()
	opts.StoreSyncTimeout = time.Second
	opts.Kubeconfig = writeStoreSyncKubeconfig(t)
	opts.Config = writeStoreSyncConfig(t, "store_sync_timeout: -1s\n")

	err := RunKubeStateMetrics(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "store-sync-timeout") {
		t.Fatalf("expected a negative file timeout to fail before servers start, got %v", err)
	}
	if opts.Prepared != nil {
		t.Fatal("expected Prepared to be cleared after a failed run")
	}
}

func TestRunKubeStateMetricsNegativeFileThenRemovedKeyRestoresCLI(t *testing.T) {
	opts := options.NewOptions()
	opts.StoreSyncTimeout = time.Second
	opts.Kubeconfig = writeStoreSyncKubeconfig(t)
	path := writeStoreSyncConfig(t, "store_sync_timeout: -5s\n")
	opts.Config = path

	err := RunKubeStateMetrics(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "store-sync-timeout") {
		t.Fatalf("expected a negative file timeout to fail at startup, got %v", err)
	}
	if err := os.WriteFile(path, []byte("port: 8080\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err = RunKubeStateMetrics(t.Context(), opts)
	if err == nil || strings.Contains(err.Error(), "store-sync-timeout") {
		t.Fatalf("expected the restored CLI timeout to pass validation, got %v", err)
	}
	if opts.StoreSyncTimeout != time.Second {
		t.Fatalf("expected removing the file key to restore the CLI timeout, got %s", opts.StoreSyncTimeout)
	}
}

func TestRunKubeStateMetricsNegativeCLIWithOmittedFileKeyFails(t *testing.T) {
	opts := options.NewOptions()
	opts.StoreSyncTimeout = -time.Second
	opts.Config = writeStoreSyncConfig(t, "port: 8080\n")

	err := RunKubeStateMetrics(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "store-sync-timeout") {
		t.Fatalf("expected an omitted file key to keep the negative CLI timeout, got %v", err)
	}
}

func TestRunKubeStateMetricsYAMLOverridesStoreSyncTimeout(t *testing.T) {
	opts := options.NewOptions()
	opts.StoreSyncTimeout = time.Second
	opts.Kubeconfig = writeStoreSyncKubeconfig(t)
	opts.Config = writeStoreSyncConfig(t, "store_sync_timeout: 5s\n")

	err := RunKubeStateMetrics(t.Context(), opts)
	if err == nil || strings.Contains(err.Error(), "store-sync-timeout") {
		t.Fatalf("expected a valid YAML timeout to pass validation, got %v", err)
	}
	if opts.StoreSyncTimeout != 5*time.Second {
		t.Fatalf("expected YAML to override the CLI timeout, got %s", opts.StoreSyncTimeout)
	}
}

func TestRunKubeStateMetricsKeepsValidStoreSyncTimeoutWithoutConfig(t *testing.T) {
	opts := options.NewOptions()
	opts.StoreSyncTimeout = time.Second
	opts.Kubeconfig = writeStoreSyncKubeconfig(t)

	err := RunKubeStateMetrics(t.Context(), opts)
	if err == nil || strings.Contains(err.Error(), "store-sync-timeout") {
		t.Fatalf("expected a valid CLI timeout to pass validation, got %v", err)
	}
	if opts.StoreSyncTimeout != time.Second {
		t.Fatalf("expected the CLI timeout to be kept, got %s", opts.StoreSyncTimeout)
	}
}

func writeStoreSyncConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeStoreSyncKubeconfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kubeconfig")
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
	return path
}
