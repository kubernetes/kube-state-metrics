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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func TestPrepareRunOptionsForReload_ResourcesPrecedence(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	kubePath := filepath.Join(dir, "kube.yaml")
	writeMinimalKubeconfig(t, kubePath)
	if err := os.WriteFile(configPath, []byte("resources:\n  pods: {}\n"+"kubeconfig: "+kubePath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	base := options.NewOptions()
	base.Config = configPath
	base.Kubeconfig = kubePath
	base.Resources = options.ResourceSet{"deployments": {}}

	got, err := PrepareRunOptionsForReload(base)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Resources["deployments"]; ok {
		t.Fatal("expected file resources to replace flag resources")
	}
	if _, ok := got.Resources["pods"]; !ok {
		t.Fatal("expected pods from config file")
	}
}

func TestPrepareRunOptionsForReload_ContinueWithoutMissingFiles(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "missing-config.yaml")
	kubePath := filepath.Join(dir, "kube.yaml")
	writeMinimalKubeconfig(t, kubePath)

	base := options.NewOptions()
	base.Config = configPath
	base.Kubeconfig = kubePath
	base.ContinueWithoutConfig = true
	base.ContinueWithoutCustomResourceConfigFile = true
	base.CustomResourceConfigFile = filepath.Join(dir, "missing-crc.yaml")

	got, err := PrepareRunOptionsForReload(base)
	if err != nil {
		t.Fatalf("expected preflight success with continue flags: %v", err)
	}
	if got.Prepared == nil {
		t.Fatal("expected prepared attachment")
	}
}

func TestPrepareRunOptionsForReload_InlineCRSWinsOverFile(t *testing.T) {
	dir := t.TempDir()
	crcPath := filepath.Join(dir, "crc.yaml")
	kubePath := filepath.Join(dir, "kube.yaml")
	writeMinimalKubeconfig(t, kubePath)
	writeMinimalCRSConfig(t, crcPath)

	inline := `kind: CustomResourceStateMetrics
apiVersion: k8s.io/kube-state-metrics/v2/customresourcestate/v1alpha1
spec:
  resources: []
`
	base := options.NewOptions()
	base.CustomResourceConfig = inline
	base.CustomResourceConfigFile = crcPath
	base.Kubeconfig = kubePath

	got, err := PrepareRunOptionsForReload(base)
	if err != nil {
		t.Fatal(err)
	}
	if got.Prepared.CRSInline != strings.TrimSpace(inline) {
		t.Fatalf("expected inline CRS to be captured, got %q", got.Prepared.CRSInline)
	}
	if len(got.Prepared.CRSFileContent) > 0 {
		t.Fatal("file content must not override inline precedence")
	}
}

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

func TestPrepareRunOptionsForReload_StoreSyncTimeoutNoYAMLKeyPreservesCLI(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	kubePath := filepath.Join(dir, "kube.yaml")
	writeMinimalKubeconfig(t, kubePath)
	if err := os.WriteFile(configPath, []byte("kubeconfig: "+kubePath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	base := options.NewOptions()
	base.Config = configPath
	base.Kubeconfig = kubePath
	base.StoreSyncTimeout = 7 * time.Second

	got, err := PrepareRunOptionsForReload(base)
	if err != nil {
		t.Fatal(err)
	}
	if got.StoreSyncTimeout != 7*time.Second {
		t.Fatalf("expected CLI timeout when the key is omitted, got %s", got.StoreSyncTimeout)
	}
	if base.StoreSyncTimeout != 7*time.Second {
		t.Fatalf("base options must stay immutable, got %s", base.StoreSyncTimeout)
	}
}

func TestPrepareRunOptionsForReload_StoreSyncTimeoutYAMLOverridesCLI(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	kubePath := filepath.Join(dir, "kube.yaml")
	writeMinimalKubeconfig(t, kubePath)
	if err := os.WriteFile(configPath, []byte("store_sync_timeout: 4s\nkubeconfig: "+kubePath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	base := options.NewOptions()
	base.Config = configPath
	base.Kubeconfig = kubePath
	base.StoreSyncTimeout = 2 * time.Second

	got, err := PrepareRunOptionsForReload(base)
	if err != nil {
		t.Fatal(err)
	}
	if got.StoreSyncTimeout != 4*time.Second {
		t.Fatalf("expected YAML to override CLI, got %s", got.StoreSyncTimeout)
	}
}

func TestPrepareRunOptionsForReload_StoreSyncTimeoutExplicitZero(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	kubePath := filepath.Join(dir, "kube.yaml")
	writeMinimalKubeconfig(t, kubePath)
	if err := os.WriteFile(configPath, []byte("store_sync_timeout: 0s\nkubeconfig: "+kubePath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	base := options.NewOptions()
	base.Config = configPath
	base.Kubeconfig = kubePath
	base.StoreSyncTimeout = 9 * time.Second

	got, err := PrepareRunOptionsForReload(base)
	if err != nil {
		t.Fatal(err)
	}
	if got.StoreSyncTimeout != 0 {
		t.Fatalf("expected explicit 0 from YAML, got %s", got.StoreSyncTimeout)
	}
}

func TestPrepareRunOptionsForReload_RemovingYAMLKeyRestoresCLI(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	kubePath := filepath.Join(dir, "kube.yaml")
	writeMinimalKubeconfig(t, kubePath)

	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("store_sync_timeout: 12s\nkubeconfig: " + kubePath + "\n")

	base := options.NewOptions()
	base.Config = configPath
	base.Kubeconfig = kubePath
	base.StoreSyncTimeout = 5 * time.Second

	got, err := PrepareRunOptionsForReload(base)
	if err != nil {
		t.Fatal(err)
	}
	if got.StoreSyncTimeout != 12*time.Second {
		t.Fatalf("expected YAML override first, got %s", got.StoreSyncTimeout)
	}

	write("kubeconfig: " + kubePath + "\n")
	got2, err := PrepareRunOptionsForReload(base)
	if err != nil {
		t.Fatal(err)
	}
	if got2.StoreSyncTimeout != 5*time.Second {
		t.Fatalf("expected CLI after removing key, got %s", got2.StoreSyncTimeout)
	}
}

func TestPrepareRunOptionsForReload_DefaultBaseRemovingYAMLKeyRestores30s(t *testing.T) {
	base := options.NewOptions()
	if base.StoreSyncTimeout != options.DefaultStoreSyncTimeout {
		t.Fatalf("precondition: default store sync timeout, got %s", base.StoreSyncTimeout)
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	kubePath := filepath.Join(dir, "kube.yaml")
	writeMinimalKubeconfig(t, kubePath)
	base.Config = configPath
	base.Kubeconfig = kubePath

	if err := os.WriteFile(configPath, []byte("store_sync_timeout: 3s\nkubeconfig: "+kubePath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := PrepareRunOptionsForReload(base)
	if err != nil {
		t.Fatal(err)
	}
	if got.StoreSyncTimeout != 3*time.Second {
		t.Fatalf("first prepare: expected YAML override 3s, got %s", got.StoreSyncTimeout)
	}
	if base.StoreSyncTimeout != options.DefaultStoreSyncTimeout {
		t.Fatalf("base must stay at default %s, got %s", options.DefaultStoreSyncTimeout, base.StoreSyncTimeout)
	}

	if err := os.WriteFile(configPath, []byte("kubeconfig: "+kubePath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got2, err := PrepareRunOptionsForReload(base)
	if err != nil {
		t.Fatal(err)
	}
	if got2.StoreSyncTimeout != options.DefaultStoreSyncTimeout {
		t.Fatalf("second prepare: expected default %s after key removed, got %s", options.DefaultStoreSyncTimeout, got2.StoreSyncTimeout)
	}
}

func TestPrepareRunOptionsForReload_InvalidStoreSyncTimeoutSyntax(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	kubePath := filepath.Join(dir, "kube.yaml")
	writeMinimalKubeconfig(t, kubePath)
	if err := os.WriteFile(configPath, []byte("store_sync_timeout: not-a-duration\nkubeconfig: "+kubePath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	base := options.NewOptions()
	base.Config = configPath
	base.Kubeconfig = kubePath

	_, err := PrepareRunOptionsForReload(base)
	if err == nil {
		t.Fatal("expected invalid duration to fail preparation")
	}
}
