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

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog/v2"
	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

// readYAMLFile reads path and checks that it holds a YAML document.
func readYAMLFile(path string) error {
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, &map[string]any{})
}

// watchFile calls onChange whenever the file at path changes, until the file
// is removed. It watches the parent directory rather than the file itself, so
// it also sees atomic saves and Kubernetes ConfigMap updates, which swap the
// ..data symlink instead of writing to the file.
func watchFile(path string, onChange func(fsnotify.Event)) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	file := filepath.Clean(path)
	if err := watcher.Add(filepath.Dir(file)); err != nil {
		watcher.Close()
		return err
	}
	realFile, _ := filepath.EvalSymlinks(file)

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				isFile := filepath.Clean(event.Name) == file
				currentFile, _ := filepath.EvalSymlinks(file)
				if (isFile && (event.Has(fsnotify.Write) || event.Has(fsnotify.Create))) ||
					(currentFile != "" && currentFile != realFile) {
					realFile = currentFile
					onChange(event)
				} else if isFile && event.Has(fsnotify.Remove) {
					return
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				klog.ErrorS(err, "Error watching file", "file", file)
			}
		}
	}()
	return nil
}
