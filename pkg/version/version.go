/*
Copyright 2017 The Kubernetes Authors.
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

package version

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var (
	// Release returns the release version
	Release = "UNKNOWN"
	// Commit returns the short sha from git
	Commit = "UNKNOWN"
	// BuildDate is the build date
	BuildDate = ""
)

type Version struct {
	GitCommit string
	BuildDate string
	Release   string
	GoVersion string
	Compiler  string
	Platform  string
}

func (v Version) String() string {
	return fmt.Sprintf("%s/%s (%s/%s) kube-state-metrics/%s",
		filepath.Base(os.Args[0]), v.Release,
		runtime.GOOS, runtime.GOARCH, v.GitCommit)
}

// GetVersion returns kube-state-metrics version
func GetVersion() Version {
	return Version{
		GitCommit: Commit,
		BuildDate: BuildDate,
		Release:   Release,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
