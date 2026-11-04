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
