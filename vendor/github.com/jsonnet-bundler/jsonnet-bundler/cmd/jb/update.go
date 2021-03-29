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
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/jsonnet-bundler/jsonnet-bundler/pkg"
	"github.com/jsonnet-bundler/jsonnet-bundler/pkg/jsonnetfile"
	v1 "github.com/jsonnet-bundler/jsonnet-bundler/spec/v1"
	"github.com/jsonnet-bundler/jsonnet-bundler/spec/v1/deps"
)

func updateCommand(dir, jsonnetHome string, uris []string) int {
	if dir == "" {
		dir = "."
	}

	// load jsonnetfiles
	jsonnetFile, err := jsonnetfile.Load(filepath.Join(dir, jsonnetfile.File))
	kingpin.FatalIfError(err, "failed to load jsonnetfile")

	lockFile, err := jsonnetfile.Load(filepath.Join(dir, jsonnetfile.LockFile))
	kingpin.FatalIfError(err, "failed to load lockfile")

	kingpin.FatalIfError(
		os.MkdirAll(filepath.Join(dir, jsonnetHome, ".tmp"), os.ModePerm),
		"creating vendor folder")

	locks := lockFile.Dependencies

	for _, u := range uris {
		d := deps.Parse(dir, u)
		if d == nil {
			kingpin.Fatalf("Unable to parse package URI `%s`", u)
		}

		delete(locks, d.Name())
	}

	// no uris: update all
	if len(uris) == 0 {
		locks = make(map[string]deps.Dependency)
	}

	newLocks, err := pkg.Ensure(jsonnetFile, filepath.Join(dir, jsonnetHome), locks)
	kingpin.FatalIfError(err, "updating")

	kingpin.FatalIfError(
		writeJSONFile(filepath.Join(dir, jsonnetfile.LockFile), v1.JsonnetFile{Dependencies: newLocks}),
		"updating jsonnetfile.lock.json")

	return 0
}
