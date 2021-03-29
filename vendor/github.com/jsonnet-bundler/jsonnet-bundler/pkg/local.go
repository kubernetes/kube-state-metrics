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

package pkg

import (
	"context"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/pkg/errors"

	"github.com/jsonnet-bundler/jsonnet-bundler/spec/v1/deps"
)

type LocalPackage struct {
	Source *deps.Local
}

func NewLocalPackage(source *deps.Local) Interface {
	return &LocalPackage{
		Source: source,
	}
}

func (p *LocalPackage) Install(ctx context.Context, name, dir, version string) (lockVersion string, err error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "failed to get current working directory: %w")
	}

	oldname := filepath.Join(wd, p.Source.Directory)
	newname := filepath.Join(dir, name)
	linkname, err := filepath.Rel(dir, oldname)

	if err != nil {
		linkname = oldname
	}

	err = os.RemoveAll(newname)
	if err != nil {
		return "", errors.Wrap(err, "failed to clean previous destination path: %w")
	}

	_, err = os.Stat(oldname)
	if os.IsNotExist(err) {
		return "", errors.Wrap(err, "symlink destination path does not exist: %w")
	}

	err = os.Symlink(linkname, newname)
	if err != nil {
		return "", errors.Wrap(err, "failed to create symlink for local dependency: %w")
	}

	color.Magenta("LOCAL %s -> %s", name, oldname)

	return "", nil
}
