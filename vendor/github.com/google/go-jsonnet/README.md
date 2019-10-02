# go-jsonnet

[![GoDoc Widget]][GoDoc] [![Travis Widget]][Travis] [![Coverage Status Widget]][Coverage Status]

[GoDoc]: https://godoc.org/github.com/google/go-jsonnet
[GoDoc Widget]: https://godoc.org/github.com/google/go-jsonnet?status.png
[Travis]: https://travis-ci.org/google/go-jsonnet
[Travis Widget]: https://travis-ci.org/google/go-jsonnet.svg?branch=master
[Coverage Status Widget]: https://coveralls.io/repos/github/google/go-jsonnet/badge.svg?branch=master
[Coverage Status]: https://coveralls.io/github/google/go-jsonnet?branch=master

This an implementation of [Jsonnet](http://jsonnet.org/) in pure Go. It is feature complete but is not as heavily exercised as the [Jsonnet C++ implementation](https://github.com/google/jsonnet).  Please try it out and give feedback.

This code is known to work on Go 1.8 and above. We recommend always using the newest stable release of Go.

## Installation instructions

```
go get github.com/google/go-jsonnet/cmd/jsonnet
```

## Build instructions (go 1.11+)

```bash
git clone github.com/google/go-jsonnet
cd go-jsonnet
go build ./cmd/jsonnet
```
To build with [Bazel](https://bazel.build/) instead:
```bash
git clone github.com/google/go-jsonnet
cd go-jsonnet
git submodule init
git submodule update
bazel build //cmd/jsonnet
```
The resulting _jsonnet_ program will then be available at a platform-specific path, such as _bazel-bin/cmd/jsonnet/darwin_amd64_stripped/jsonnet_ for macOS.

Bazel also accommodates cross-compiling the program. To build the _jsonnet_ program for various popular platforms, run the following commands:

Target platform | Build command
--------------- | -------------------------------------------------------------------------------------
Current host    | _bazel build //cmd/jsonnet_
Linux           | _bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 //cmd/jsonnet_
macOS           | _bazel build --platforms=@io_bazel_rules_go//go/toolchain:darwin_amd64 //cmd/jsonnet_
Windows         | _bazel build --platforms=@io_bazel_rules_go//go/toolchain:windows_amd64 //cmd/jsonnet_

For additional target platform names, see the per-Go release definitions [here](https://github.com/bazelbuild/rules_go/blob/master/go/private/sdk_list.bzl#L21-L31) in the _rules_go_ Bazel package.

Additionally if any files were moved around, see the section [Keeping the Bazel files up to date](#keeping-the-bazel-files-up-to-date).

## Build instructions (go 1.8 - 1.10)

```bash
go get -u github.com/google/go-jsonnet
cd $GOPATH/src/github.com/google/go-jsonnet
go get -u .
go build ./cmd/jsonnet
```

## Running tests

```bash
./tests.sh  # Also runs `go test ./...`
```

## Implementation Notes

We are generating some helper classes on types by using http://clipperhouse.github.io/gen/.  Do the following to regenerate these if necessary:

```bash
go get github.com/clipperhouse/gen
go get github.com/clipperhouse/set
export PATH=$PATH:$GOPATH/bin  # If you haven't already
go generate
```

## Updating and modifying the standard library

Standard library source code is kept in `cpp-jsonnet` submodule, because it is shared with [Jsonnet C++
implementation](https://github.com/google/jsonnet).

For performance reasons we perform preprocessing on the standard library, so for the changes to be visible, regeneration is necessary:

```bash
git submodule init
git submodule update
go run cmd/dumpstdlibast/dumpstdlibast.go cpp-jsonnet/stdlib/std.jsonnet > astgen/stdast.go
```

The above command creates the _astgen/stdast.go_ file which puts the desugared standard library into the right data structures, which lets us avoid the parsing overhead during execution. Note that this step is not necessary to perform manually when building with Bazel; the Bazel target regenerates the _astgen/stdast.go_ (writing it into Bazel's build sandbox directory tree) file when necessary.

## Keeping the Bazel files up to date
Note that we maintain the Go-related Bazel targets with [the Gazelle tool](https://github.com/bazelbuild/bazel-gazelle). The Go module (_go.mod_ in the root directory) remains the primary source of truth. Gazelle analyzes both that file and the rest of the Go files in the repository to create and adjust appropriate Bazel targets for building Go packages and executable programs.

After changing any dependencies within the files covered by this Go module, it is helpful to run _go mod tidy_ to ensure that the module declarations match the state of the Go source code. In order to synchronize the Bazel rules with material changes to the Go module, run the following command to invoke [Gazelle's `update-repos` command](https://github.com/bazelbuild/bazel-gazelle#update-repos):
```bash
bazel run //:gazelle -- update-repos -from_file=go.mod -to_macro=bazel/deps.bzl%jsonnet_go_dependencies
```

Similarly, after adding or removing Go source files, it may be necessary to synchronize the Bazel rules by running the following command:
```bash
bazel run //:gazelle
```
