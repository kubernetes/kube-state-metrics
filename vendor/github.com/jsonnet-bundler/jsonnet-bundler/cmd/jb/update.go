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
	"net/url"
	"os"

	"github.com/jsonnet-bundler/jsonnet-bundler/pkg"
	"github.com/jsonnet-bundler/jsonnet-bundler/pkg/jsonnetfile"
	"gopkg.in/alecthomas/kingpin.v2"
)

func updateCommand(jsonnetHome string, urls ...*url.URL) int {
	m, err := pkg.LoadJsonnetfile(jsonnetfile.File)
	if err != nil {
		kingpin.Fatalf("failed to load jsonnetfile: %v", err)
		return 1
	}

	err = os.MkdirAll(jsonnetHome, os.ModePerm)
	if err != nil {
		kingpin.Fatalf("failed to create jsonnet home path: %v", err)
		return 3
	}

	// When updating, the lockfile is explicitly ignored.
	isLock := false
	lock, err := pkg.Install(context.TODO(), isLock, jsonnetfile.File, m, jsonnetHome)
	if err != nil {
		kingpin.Fatalf("failed to install: %v", err)
		return 3
	}

	b, err := json.MarshalIndent(lock, "", "    ")
	if err != nil {
		kingpin.Fatalf("failed to encode jsonnet file: %v", err)
		return 3
	}
	b = append(b, []byte("\n")...)

	err = ioutil.WriteFile(jsonnetfile.LockFile, b, 0644)
	if err != nil {
		kingpin.Fatalf("failed to write lock file: %v", err)
		return 3
	}

	return 0
}
