FLAGS =
TESTENVVAR =
REGISTRY = quay.io/coreos
TAG_PREFIX = v
VERSION = $(shell cat VERSION)
TAG = $(TAG_PREFIX)$(VERSION)
LATEST_RELEASE_BRANCH := release-$(shell grep -ohE "[0-9]+.[0-9]+" VERSION)
PKGS = $(shell go list ./... | grep -v /vendor/ | grep -v /tests/e2e)
ARCH ?= $(shell go env GOARCH)
BuildDate = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
Commit = $(shell git rev-parse --short HEAD)
ALL_ARCH = amd64 arm arm64 ppc64le s390x
PKG = k8s.io/kube-state-metrics/pkg
GO_VERSION = 1.13
FIRST_GOPATH := $(firstword $(subst :, ,$(shell go env GOPATH)))
BENCHCMP_BINARY := $(FIRST_GOPATH)/bin/benchcmp
GOLANGCI_VERSION := v1.19.1
HAS_GOLANGCI := $(shell which golangci-lint)

IMAGE = $(REGISTRY)/kube-state-metrics
MULTI_ARCH_IMG = $(IMAGE)-$(ARCH)

validate-modules:
	@echo "- Verifying that the dependencies have expected content..."
	go mod verify
	@echo "- Checking for any unused/missing packages in go.mod..."
	go mod tidy
	@echo "- Checking for unused packages in vendor..."
	go mod vendor
	@git diff --exit-code -- go.sum go.mod vendor/

licensecheck:
	@echo ">> checking license header"
	@licRes=$$(for file in $$(find . -type f -iname '*.go' ! -path './vendor/*') ; do \
               awk 'NR<=5' $$file | grep -Eq "(Copyright|generated|GENERATED)" || echo $$file; \
       done); \
       if [ -n "$${licRes}" ]; then \
               echo "license header checking failed:"; echo "$${licRes}"; \
               exit 1; \
       fi

lint: shellcheck licensecheck
ifndef HAS_GOLANGCI
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(GOPATH)/bin ${GOLANGCI_VERSION}
endif
	golangci-lint run

doccheck: generate
	@echo "- Checking if the generated documentation is up to date..."
	@git diff --exit-code
	@echo "- Checking if the documentation is in sync with the code..."
	@grep -hoE '(kube_[^ |]+)' docs/* --exclude=README.md| sort -u > documented_metrics
	@find internal/store -type f -not -name '*_test.go' -exec sed -nE 's/.*"(kube_[^"]+)"/\1/p' {} \; | sed -E 's/,//g' | sort -u > code_metrics
	@diff -u0 code_metrics documented_metrics || (echo "ERROR: Metrics with - are present in code but missing in documentation, metrics with + are documented but not found in code."; exit 1)
	@echo OK
	@rm -f code_metrics documented_metrics
	@echo "- Checking for orphan documentation files"
	@cd docs; for doc in *.md; do if [ "$$doc" != "README.md" ] && ! grep -q "$$doc" *.md; then echo "ERROR: No link to documentation file $${doc} detected"; exit 1; fi; done
	@echo OK

build-local: clean
	GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(ARCH) CGO_ENABLED=0 go build -ldflags "-s -w -X ${PKG}/version.Release=${TAG} -X ${PKG}/version.Commit=${Commit} -X ${PKG}/version.BuildDate=${BuildDate}" -o kube-state-metrics

build: clean
	docker run --rm -v "${PWD}:/go/src/k8s.io/kube-state-metrics" -w /go/src/k8s.io/kube-state-metrics golang:${GO_VERSION} make build-local

test-unit: clean build
	GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(ARCH) $(TESTENVVAR) go test --race $(FLAGS) $(PKGS)

shellcheck:
	docker run -v "${PWD}:/mnt" koalaman/shellcheck:stable $(shell find . -type f -name "*.sh" -not -path "*vendor*")

# Runs benchmark tests on the current git ref and the last release and compares
# the two.
test-benchmark-compare: $(BENCHCMP_BINARY)
	./tests/compare_benchmarks.sh master
	./tests/compare_benchmarks.sh ${LATEST_RELEASE_BRANCH}

TEMP_DIR := $(shell mktemp -d)

all: all-container

sub-container-%:
	$(MAKE) --no-print-directory ARCH=$* container

sub-push-%:
	$(MAKE) --no-print-directory ARCH=$* push

all-container: $(addprefix sub-container-,$(ALL_ARCH))

all-push: $(addprefix sub-push-,$(ALL_ARCH))

container: .container-$(ARCH)
.container-$(ARCH):
	docker run --rm -v "${PWD}:/go/src/k8s.io/kube-state-metrics" -w /go/src/k8s.io/kube-state-metrics -e GOOS=linux -e GOARCH=$(ARCH) -e CGO_ENABLED=0 golang:${GO_VERSION} go build -ldflags "-s -w -X ${PKG}/version.Release=${TAG} -X ${PKG}/version.Commit=${Commit} -X ${PKG}/version.BuildDate=${BuildDate}" -o kube-state-metrics
	cp -r * "${TEMP_DIR}"
	docker build -t $(MULTI_ARCH_IMG):$(TAG) "${TEMP_DIR}"
	docker tag $(MULTI_ARCH_IMG):$(TAG) $(MULTI_ARCH_IMG):latest
	rm -rf "${TEMP_DIR}"

ifeq ($(ARCH), amd64)
	# Adding check for amd64
	docker tag $(MULTI_ARCH_IMG):$(TAG) $(IMAGE):$(TAG)
	docker tag $(MULTI_ARCH_IMG):$(TAG) $(IMAGE):latest
endif

quay-push: .quay-push-$(ARCH)
.quay-push-$(ARCH): .container-$(ARCH)
	docker push $(MULTI_ARCH_IMG):$(TAG)
	docker push $(MULTI_ARCH_IMG):latest
ifeq ($(ARCH), amd64)
	docker push $(IMAGE):$(TAG)
	docker push $(IMAGE):latest
endif

push: .push-$(ARCH)
.push-$(ARCH): .container-$(ARCH)
	gcloud docker -- push $(MULTI_ARCH_IMG):$(TAG)
	gcloud docker -- push $(MULTI_ARCH_IMG):latest
ifeq ($(ARCH), amd64)
	gcloud docker -- push $(IMAGE):$(TAG)
	gcloud docker -- push $(IMAGE):latest
endif

clean:
	rm -f kube-state-metrics

e2e:
	./tests/e2e.sh

generate: build-local embedmd
	@echo ">> generating docs"
	@./scripts/generate-help-text.sh
	@$(GOPATH)/bin/embedmd -w `find . -path ./vendor -prune -o -name "*.md" -print`

validate-manifests: examples
	@git diff --exit-code

examples: examples/standard examples/autosharding

examples/standard: jsonnet $(shell find jsonnet | grep ".libsonnet") scripts/standard.jsonnet scripts/vendor VERSION
	mkdir -p examples/standard
	jsonnet -J scripts/vendor -m examples/standard --ext-str version="$(VERSION)" scripts/standard.jsonnet | xargs -I{} sh -c 'cat {} | gojsontoyaml > `echo {} | sed "s/\(.\)\([A-Z]\)/\1-\2/g" | tr "[:upper:]" "[:lower:]"`.yaml' -- {}
	find examples -type f ! -name '*.yaml' -delete

examples/autosharding: jsonnet $(shell find jsonnet | grep ".libsonnet") scripts/autosharding.jsonnet scripts/vendor VERSION
	mkdir -p examples/autosharding
	jsonnet -J scripts/vendor -m examples/autosharding --ext-str version="$(VERSION)" scripts/autosharding.jsonnet | xargs -I{} sh -c 'cat {} | gojsontoyaml > `echo {} | sed "s/\(.\)\([A-Z]\)/\1-\2/g" | tr "[:upper:]" "[:lower:]"`.yaml' -- {}
	find examples -type f ! -name '*.yaml' -delete

scripts/vendor: jb scripts/jsonnetfile.json scripts/jsonnetfile.lock.json
	cd scripts && jb install

jsonnet:
	 go install github.com/google/go-jsonnet/cmd/jsonnet

jb:
	 go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb

embedmd:
	 go install github.com/campoy/embedmd

$(BENCHCMP_BINARY):
	 go install golang.org/x/tools/cmd/benchcmp

.PHONY: all build build-local all-push all-container test-unit test-benchmark-compare container push quay-push clean e2e validate-modules shellcheck licensecheck lint generate embedmd
