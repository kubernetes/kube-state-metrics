# go-jsonnet

[![GoDoc Widget]][GoDoc] [![Travis Widget]][Travis] [![Coverage Status Widget]][Coverage Status]

[GoDoc]: https://godoc.org/github.com/google/go-jsonnet
[GoDoc Widget]: https://godoc.org/github.com/google/go-jsonnet?status.png
[Travis]: https://travis-ci.org/google/go-jsonnet
[Travis Widget]: https://travis-ci.org/google/go-jsonnet.svg?branch=master
[Coverage Status Widget]: https://coveralls.io/repos/github/google/go-jsonnet/badge.svg?branch=master
[Coverage Status]: https://coveralls.io/github/google/go-jsonnet?branch=master

This an implementation of [Jsonnet](http://jsonnet.org/) in pure Go. It is a feature complete, production-ready implementation. It is compatible with the original [Jsonnet C++ implementation](https://github.com/google/jsonnet). Bindings to C and Python are available (but not battle-tested yet).

This code is known to work on Go 1.11 and above. We recommend always using the newest stable release of Go.

## Installation instructions

```
go get github.com/google/go-jsonnet/cmd/jsonnet
```

It's also available on Homebrew:

```
brew install go-jsonnet
```

## Build instructions (go 1.11+)

```bash
git clone git@github.com:google/go-jsonnet.git
cd go-jsonnet
go build ./cmd/jsonnet
go build ./cmd/jsonnetfmt
go build ./cmd/jsonnet-deps
```
To build with [Bazel](https://bazel.build/) instead:
```bash
git clone git@github.com:google/go-jsonnet.git
cd go-jsonnet
git submodule init
git submodule update
bazel build //cmd/jsonnet
bazel build //cmd/jsonnetfmt
bazel build //cmd/jsonnet-deps
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

## Running tests

```bash
./tests.sh  # Also runs `go test ./...`
```

## Running Benchmarks

### Method 1

```bash
go get golang.org/x/tools/cmd/benchcmp
```

1. Make sure you build a jsonnet binary _prior_ to making changes.

```bash
go build -o jsonnet-old ./cmd/jsonnet
```

2. Make changes (iterate as needed), and rebuild new binary

```bash
go build ./cmd/jsonnet
```

3. Run benchmark:

```bash
# e.g. ./benchmark.sh Builtin
./benchmark.sh <TestNameFilter>
```

### Method 2

1. get `benchcmp`

```bash
go get golang.org/x/tools/cmd/benchcmp
```

2. Make sure you build a jsonnet binary _prior_ to making changes.

```bash
make build-old
```

3. iterate with (which will also automatically rebuild the new binary `./jsonnet`)

_replace the FILTER with the name of the test you are working on_

```bash
FILTER=Builtin_manifestJsonEx make benchmark
```

## Implementation Notes

We are generating some helper classes on types by using http://clipperhouse.github.io/gen/.  Do the following to regenerate these if necessary:

```bash
go get github.com/clipperhouse/gen
go get github.com/clipperhouse/set
export PATH=$PATH:$GOPATH/bin  # If you haven't already
go generate
```

## Update cpp-jsonnet sub-repo

This repo depends on [the original Jsonnet repo](https://github.com/google/jsonnet). Shared parts include the standard library, headers files for C API and some tests.

You can update the submodule and regenerate dependent files with one command:
```
./update_cpp_jsonnet.sh
```

Note: It needs to be run from repo root.

## Updating and modifying the standard library

Standard library source code is kept in `cpp-jsonnet` submodule, because it is shared with [Jsonnet C++
implementation](https://github.com/google/jsonnet).

For performance reasons we perform preprocessing on the standard library, so for the changes to be visible, regeneration is necessary:

```bash
go run cmd/dumpstdlibast/dumpstdlibast.go cpp-jsonnet/stdlib/std.jsonnet > astgen/stdast.go
```

**The

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
