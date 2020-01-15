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
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jsonnet-bundler/jsonnet-bundler/spec"
	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	installActionName = "install"
	updateActionName  = "update"
	initActionName    = "init"
)

var (
	gitSSHRegex                   = regexp.MustCompile("git\\+ssh://git@([^:]+):([^/]+)/([^/]+).git")
	gitSSHWithVersionRegex        = regexp.MustCompile("git\\+ssh://git@([^:]+):([^/]+)/([^/]+).git@(.*)")
	gitSSHWithPathRegex           = regexp.MustCompile("git\\+ssh://git@([^:]+):([^/]+)/([^/]+).git/(.*)")
	gitSSHWithPathAndVersionRegex = regexp.MustCompile("git\\+ssh://git@([^:]+):([^/]+)/([^/]+).git/(.*)@(.*)")

	githubSlugRegex                   = regexp.MustCompile("github.com/([-_a-zA-Z0-9]+)/([-_a-zA-Z0-9]+)")
	githubSlugWithVersionRegex        = regexp.MustCompile("github.com/([-_a-zA-Z0-9]+)/([-_a-zA-Z0-9]+)@(.*)")
	githubSlugWithPathRegex           = regexp.MustCompile("github.com/([-_a-zA-Z0-9]+)/([-_a-zA-Z0-9]+)/(.*)")
	githubSlugWithPathAndVersionRegex = regexp.MustCompile("github.com/([-_a-zA-Z0-9]+)/([-_a-zA-Z0-9]+)/(.*)@(.*)")
)

func main() {
	os.Exit(Main())
}

func Main() int {
	cfg := struct {
		JsonnetHome string
	}{}

	a := kingpin.New(filepath.Base(os.Args[0]), "A jsonnet package manager")
	a.HelpFlag.Short('h')

	a.Flag("jsonnetpkg-home", "The directory used to cache packages in.").
		Default("vendor").StringVar(&cfg.JsonnetHome)

	initCmd := a.Command(initActionName, "Initialize a new empty jsonnetfile")

	installCmd := a.Command(installActionName, "Install all dependencies or install specific ones")
	installCmdURIs := installCmd.Arg("uris", "URIs to packages to install, URLs or file paths").Strings()

	updateCmd := a.Command(updateActionName, "Update all dependencies.")

	command, err := a.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		a.Usage(os.Args[1:])
		return 2
	}

	workdir, err := os.Getwd()
	if err != nil {
		return 1
	}

	switch command {
	case initCmd.FullCommand():
		return initCommand(workdir)
	case installCmd.FullCommand():
		return installCommand(workdir, cfg.JsonnetHome, *installCmdURIs...)
	case updateCmd.FullCommand():
		return updateCommand(cfg.JsonnetHome)
	default:
		installCommand(workdir, cfg.JsonnetHome)
	}

	return 0
}

func parseDependency(dir, uri string) *spec.Dependency {
	if d := parseGitSSHDependency(uri); d != nil {
		return d
	}

	if d := parseGithubDependency(uri); d != nil {
		return d
	}

	if d := parseLocalDependency(dir, uri); d != nil {
		return d
	}

	return nil
}

func parseGitSSHDependency(p string) *spec.Dependency {
	if !gitSSHRegex.MatchString(p) {
		return nil
	}

	subdir := ""
	host := ""
	org := ""
	repo := ""
	version := "master"

	if gitSSHWithPathAndVersionRegex.MatchString(p) {
		matches := gitSSHWithPathAndVersionRegex.FindStringSubmatch(p)
		host = matches[1]
		org = matches[2]
		repo = matches[3]
		subdir = matches[4]
		version = matches[5]
	} else if gitSSHWithPathRegex.MatchString(p) {
		matches := gitSSHWithPathRegex.FindStringSubmatch(p)
		host = matches[1]
		org = matches[2]
		repo = matches[3]
		subdir = matches[4]
	} else if gitSSHWithVersionRegex.MatchString(p) {
		matches := gitSSHWithVersionRegex.FindStringSubmatch(p)
		host = matches[1]
		org = matches[2]
		repo = matches[3]
		version = matches[4]
	} else {
		matches := gitSSHRegex.FindStringSubmatch(p)
		host = matches[1]
		org = matches[2]
		repo = matches[3]
	}

	return &spec.Dependency{
		Name: repo,
		Source: spec.Source{
			GitSource: &spec.GitSource{
				Remote: fmt.Sprintf("git@%s:%s/%s", host, org, repo),
				Subdir: subdir,
			},
		},
		Version: version,
	}
}

func parseGithubDependency(p string) *spec.Dependency {
	if !githubSlugRegex.MatchString(p) {
		return nil
	}

	name := ""
	user := ""
	repo := ""
	subdir := ""
	version := "master"

	if githubSlugWithPathRegex.MatchString(p) {
		if githubSlugWithPathAndVersionRegex.MatchString(p) {
			matches := githubSlugWithPathAndVersionRegex.FindStringSubmatch(p)
			user = matches[1]
			repo = matches[2]
			subdir = matches[3]
			version = matches[4]
			name = path.Base(subdir)
		} else {
			matches := githubSlugWithPathRegex.FindStringSubmatch(p)
			user = matches[1]
			repo = matches[2]
			subdir = matches[3]
			name = path.Base(subdir)
		}
	} else {
		if githubSlugWithVersionRegex.MatchString(p) {
			matches := githubSlugWithVersionRegex.FindStringSubmatch(p)
			user = matches[1]
			repo = matches[2]
			name = repo
			version = matches[3]
		} else {
			matches := githubSlugRegex.FindStringSubmatch(p)
			user = matches[1]
			repo = matches[2]
			name = repo
		}
	}

	return &spec.Dependency{
		Name: name,
		Source: spec.Source{
			GitSource: &spec.GitSource{
				Remote: fmt.Sprintf("https://github.com/%s/%s", user, repo),
				Subdir: subdir,
			},
		},
		Version: version,
	}
}

func parseLocalDependency(dir, p string) *spec.Dependency {
	if p == "" {
		return nil
	}
	if strings.HasPrefix(p, "github.com") {
		return nil
	}
	if strings.HasPrefix(p, "git+ssh") {
		return nil
	}

	clean := filepath.Clean(p)
	abs := filepath.Join(dir, clean)

	info, err := os.Stat(abs)
	if err != nil {
		return nil
	}

	if !info.IsDir() {
		return nil
	}

	return &spec.Dependency{
		Name: info.Name(),
		Source: spec.Source{
			LocalSource: &spec.LocalSource{
				Directory: clean,
			},
		},
		Version: "",
	}
}
