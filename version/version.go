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
	"runtime"
)

var (
	// RELEASE returns the release version
	RELEASE = "UNKNOWN"
	// COMMIT returns the short sha from git
	COMMIT = "UNKNOWN"
	// GitTreeState is the state of the git tree
	GitTreeState = ""
	// BuildDate is the build date
	BuildDate = ""
)

type Version struct {
	GitCommit string
	BuildDate string
	RELEASE   string
	GoVersion string
	Compiler  string
	Platform  string
}

// GetVersion returns representing the version
func GetVersion() Version {
	return Version{
		GitCommit: COMMIT,
		BuildDate: BuildDate,
		RELEASE:   RELEASE,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
