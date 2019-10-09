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
	"fmt"
	"os"
	"path/filepath"

	"github.com/jsonnet-bundler/jsonnet-bundler/spec"
	"github.com/pkg/errors"
)

type LocalPackage struct {
	Source *spec.LocalSource
}

func NewLocalPackage(source *spec.LocalSource) Interface {
	return &LocalPackage{
		Source: source,
	}
}

func (p *LocalPackage) Install(ctx context.Context, name, dir, version string) (lockVersion string, err error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %v", err)
	}

	destPath := filepath.Join(dir, name)

	err = os.RemoveAll(destPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to clean previous destination path")
	}

	err = os.Symlink(filepath.Join(wd, p.Source.Directory), filepath.Join(wd, destPath))
	if err != nil {
		return "", fmt.Errorf("failed to create symlink for local dependency: %v", err)
	}

	return "", nil
}
