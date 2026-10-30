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
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestReadYAMLFile(t *testing.T) {
	dir := t.TempDir()

	valid := filepath.Join(dir, "valid.yaml")
	if err := os.WriteFile(valid, []byte("kind: Config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := readYAMLFile(valid); err != nil {
		t.Errorf("valid file: unexpected error: %v", err)
	}

	invalid := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(invalid, []byte("kind: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := readYAMLFile(invalid); err == nil {
		t.Error("invalid file: expected an error")
	}

	if err := readYAMLFile(filepath.Join(dir, "missing.yaml")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: expected fs.ErrNotExist, got %v", err)
	}
}

func startWatch(t *testing.T, path string) <-chan fsnotify.Event {
	t.Helper()
	events := make(chan fsnotify.Event, 16)
	stop, err := watchFile(path, func(e fsnotify.Event) { events <- e })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stop)
	return events
}

func expectEvent(t *testing.T, events <-chan fsnotify.Event) {
	t.Helper()
	select {
	case <-events:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for change notification")
	}
}

func expectNoEvent(t *testing.T, events <-chan fsnotify.Event) {
	t.Helper()
	select {
	case e := <-events:
		t.Fatalf("unexpected change notification: %v", e)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestWatchFileWrite(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(file, []byte("a: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	events := startWatch(t, file)

	if err := os.WriteFile(filepath.Join(dir, "other.yaml"), []byte("b: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectNoEvent(t, events)

	if err := os.WriteFile(file, []byte("a: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectEvent(t, events)
}

// TestWatchFileConfigMapUpdate mimics how the kubelet updates a mounted
// ConfigMap: config.yaml -> ..data/config.yaml, and ..data is a symlink that
// is atomically swapped to a new timestamped directory.
func TestWatchFileConfigMapUpdate(t *testing.T) {
	dir := t.TempDir()
	writeVersion := func(name, content string) {
		t.Helper()
		if err := os.Mkdir(filepath.Join(dir, name), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name, "config.yaml"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeVersion("..v1", "a: 1\n")
	if err := os.Symlink("..v1", filepath.Join(dir, "..data")); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "config.yaml")
	if err := os.Symlink(filepath.Join("..data", "config.yaml"), file); err != nil {
		t.Fatal(err)
	}
	events := startWatch(t, file)

	writeVersion("..v2", "a: 2\n")
	if err := os.Symlink("..v2", filepath.Join(dir, "..data_tmp")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, "..data_tmp"), filepath.Join(dir, "..data")); err != nil {
		t.Fatal(err)
	}
	expectEvent(t, events)
}

func TestWatchFileStop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("a: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	events := make(chan fsnotify.Event, 16)
	stop, err := watchFile(path, func(e fsnotify.Event) { events <- e })
	if err != nil {
		t.Fatal(err)
	}
	stop()
	// Calling stop again, as a later reload might, must be harmless.
	stop()

	if err := os.WriteFile(path, []byte("a: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectNoEvent(t, events)
}
