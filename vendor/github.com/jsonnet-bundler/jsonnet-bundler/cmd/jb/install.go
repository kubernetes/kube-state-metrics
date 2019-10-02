// Copyright 2018 jsonnet-bundler authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jsonnet-bundler/jsonnet-bundler/pkg"
	"github.com/jsonnet-bundler/jsonnet-bundler/pkg/jsonnetfile"
	"github.com/jsonnet-bundler/jsonnet-bundler/spec"
	"gopkg.in/alecthomas/kingpin.v2"
)

func installCommand(dir, jsonnetHome string, uris ...string) int {
	if dir == "" {
		dir = "."
	}

	filename, isLock, err := jsonnetfile.Choose(dir)
	if err != nil {
		kingpin.Fatalf("failed to choose jsonnetfile: %v", err)
		return 1
	}

	jsonnetFile, err := jsonnetfile.Load(filename)
	if err != nil {
		kingpin.Fatalf("failed to load jsonnetfile: %v", err)
		return 1
	}

	if len(uris) > 0 {
		for _, uri := range uris {
			newDep := parseDependency(dir, uri)
			if newDep == nil {
				kingpin.Errorf("ignoring unrecognized uri: %s", uri)
				continue
			}

			oldDeps := jsonnetFile.Dependencies
			newDeps := []spec.Dependency{}
			oldDepReplaced := false
			for _, d := range oldDeps {
				if d.Name == newDep.Name {
					newDeps = append(newDeps, *newDep)
					oldDepReplaced = true
				} else {
					newDeps = append(newDeps, d)
				}
			}

			if !oldDepReplaced {
				newDeps = append(newDeps, *newDep)
			}

			jsonnetFile.Dependencies = newDeps
		}
	}

	srcPath := filepath.Join(jsonnetHome)
	err = os.MkdirAll(srcPath, os.ModePerm)
	if err != nil {
		kingpin.Fatalf("failed to create jsonnet home path: %v", err)
		return 3
	}

	lock, err := pkg.Install(context.TODO(), isLock, filename, jsonnetFile, jsonnetHome)
	if err != nil {
		kingpin.Fatalf("failed to install: %v", err)
		return 3
	}

	// If installing from lock file there is no need to write any files back.
	if !isLock {
		b, err := json.MarshalIndent(jsonnetFile, "", "    ")
		if err != nil {
			kingpin.Fatalf("failed to encode jsonnet file: %v", err)
			return 3
		}
		b = append(b, []byte("\n")...)

		err = ioutil.WriteFile(filepath.Join(dir, jsonnetfile.File), b, 0644)
		if err != nil {
			kingpin.Fatalf("failed to write jsonnet file: %v", err)
			return 3
		}

		b, err = json.MarshalIndent(lock, "", "    ")
		if err != nil {
			kingpin.Fatalf("failed to encode jsonnet file: %v", err)
			return 3
		}
		b = append(b, []byte("\n")...)

		err = ioutil.WriteFile(filepath.Join(dir, jsonnetfile.LockFile), b, 0644)
		if err != nil {
			kingpin.Fatalf("failed to write lock file: %v", err)
			return 3
		}
	}

	return 0
}
