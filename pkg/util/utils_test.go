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

package util

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// writeKubeconfig writes a minimal kubeconfig pointing at server, and returns its
// path. Rewriting the same path stands in for a credential rotation, which is
// what the file watcher in internal/wrapper.go restarts KSM for.
func writeKubeconfig(t *testing.T, path, server, token string) {
	t.Helper()

	contents := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: %s
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
users:
- name: test
  user:
    token: %s
`, server, token)

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing kubeconfig: %v", err)
	}
}

// The kubeconfig path stays the same across a rotation while its contents
// change, so nothing here may be memoized on the arguments -- or at all.
func TestBuildConfigReflectsCurrentKubeconfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kubeconfig")

	writeKubeconfig(t, path, "https://first.example.com", "token-one")
	first, err := buildConfig("", path)
	if err != nil {
		t.Fatalf("building config: %v", err)
	}

	writeKubeconfig(t, path, "https://second.example.com", "token-two")
	second, err := buildConfig("", path)
	if err != nil {
		t.Fatalf("rebuilding config: %v", err)
	}

	if first.Host != "https://first.example.com" {
		t.Errorf("first host: got %q, want %q", first.Host, "https://first.example.com")
	}
	if second.Host != "https://second.example.com" {
		t.Errorf("second host: got %q, want %q", second.Host, "https://second.example.com")
	}
	if second.BearerToken != "token-two" {
		t.Errorf("second token: got %q, want %q", second.BearerToken, "token-two")
	}
}

// The kubeconfig watcher restarts KSM so it picks up new credentials, which only
// works if the client is rebuilt from the file as it stands at that moment.
func TestCreateKubeClientRebuildsAfterRotation(t *testing.T) {
	var firstHits, secondHits atomic.Int32

	version := func(hits *atomic.Int32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/version" {
				hits.Add(1)
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"major":"1","minor":"33","gitVersion":"v1.33.0"}`)
		}
	}

	first := httptest.NewServer(version(&firstHits))
	defer first.Close()
	second := httptest.NewServer(version(&secondHits))
	defer second.Close()

	path := filepath.Join(t.TempDir(), "kubeconfig")

	writeKubeconfig(t, path, first.URL, "token-one")
	if _, err := CreateKubeClient("", path); err != nil {
		t.Fatalf("creating the first client: %v", err)
	}

	// Stand in for a rotation: same path, new contents.
	writeKubeconfig(t, path, second.URL, "token-two")
	if _, err := CreateKubeClient("", path); err != nil {
		t.Fatalf("creating the client after rotation: %v", err)
	}

	if got := firstHits.Load(); got != 1 {
		t.Errorf("first server saw %d version requests, want 1", got)
	}
	if got := secondHits.Load(); got != 1 {
		t.Errorf("second server saw %d version requests, want 1 -- the client was not rebuilt from the rotated kubeconfig", got)
	}
}
