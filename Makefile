FLAGS =
TESTENVVAR =
REGISTRY ?= gcr.io/k8s-staging-kube-state-metrics
TAG_PREFIX = v
VERSION = $(shell cat VERSION)
TAG ?= $(TAG_PREFIX)$(VERSION)
LATEST_RELEASE_BRANCH := release-$(shell grep -ohE "[0-9]+.[0-9]+" VERSION)
BRANCH = $(strip $(shell git rev-parse --abbrev-ref HEAD))
DOCKER_CLI ?= docker
PROMTOOL_CLI ?= promtool
PKGS = $(shell go list ./... | grep -v /vendor/ | grep -v /tests/e2e)
ARCH ?= $(shell go env GOARCH)
BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
OS ?= $(shell uname -s | tr A-Z a-z)
ALL_ARCH = amd64 arm arm64 ppc64le s390x
PKG = github.com/prometheus/common
PROMETHEUS_VERSION = 2.37.0
GO_VERSION = 1.19.2
IMAGE = $(REGISTRY)/kube-state-metrics
MULTI_ARCH_IMG = $(IMAGE)-$(ARCH)
USER ?= $(shell id -u -n)
HOST ?= $(shell hostname)

export DOCKER_CLI_EXPERIMENTAL=enabled

validate-modules:
	@echo "- Verifying that the dependencies have expected content..."
	go mod verify
	@echo "- Checking for any unused/missing packages in go.mod..."
	go mod tidy
	@git diff --exit-code -- go.sum go.mod

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
	golangci-lint run

doccheck: generate
	@echo "- Checking if the generated documentation is up to date..."
	@git diff --exit-code
	@echo "- Checking if the documentation is in sync with the code..."
	@grep -hoE -d skip '\| kube_[^ |]+' docs/* --exclude=README.md | sed -E 's/\| //g' | sort -u > documented_metrics
	@find internal/store -type f -not -name '*_test.go' -exec sed -nE 's/.*"(kube_[^"]+)".*/\1/p' {} \; | sort -u > code_metrics
	@diff -u0 code_metrics documented_metrics || (echo "ERROR: Metrics with - are present in code but missing in documentation, metrics with + are documented but not found in code."; exit 1)
	@echo OK
	@rm -f code_metrics documented_metrics
	@echo "- Checking for orphan documentation files"
	@cd docs; for doc in *.md; do if [ "$$doc" != "README.md" ] && ! grep -q "$$doc" *.md; then echo "ERROR: No link to documentation file $${doc} detected"; exit 1; fi; done
	@echo OK

build-local:
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 go build -ldflags "-s -w -X ${PKG}/version.Version=${TAG} -X ${PKG}/version.Revision=${GIT_COMMIT} -X ${PKG}/version.Branch=${BRANCH} -X ${PKG}/version.BuildUser=${USER}@${HOST} -X ${PKG}/version.BuildDate=${BUILD_DATE}" -o kube-state-metrics

build: kube-state-metrics

kube-state-metrics:
	${DOCKER_CLI} run --rm -v "${PWD}:/go/src/k8s.io/kube-state-metrics" -w /go/src/k8s.io/kube-state-metrics -e GOOS=$(OS) -e GOARCH=$(ARCH) golang:${GO_VERSION} make build-local

test-unit:
	GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(ARCH) $(TESTENVVAR) go test --race $(FLAGS) $(PKGS)

test-rules:
	${PROMTOOL_CLI} test rules tests/rules/alerts-test.yaml

shellcheck:
	${DOCKER_CLI} run -v "${PWD}:/mnt" koalaman/shellcheck:stable $(shell find . -type f -name "*.sh" -not -path "*vendor*")

# Runs benchmark tests on the current git ref and the last release and compares
# the two.
test-benchmark-compare:
	@git fetch
	./tests/compare_benchmarks.sh master
	./tests/compare_benchmarks.sh ${LATEST_RELEASE_BRANCH}

all: all-container

# Container build for multiple architectures as defined in ALL_ARCH

container: container-$(ARCH)

container-%:
	${DOCKER_CLI} build --pull -t $(IMAGE)-$*:$(TAG) --build-arg GOVERSION=$(GO_VERSION) --build-arg GOARCH=$* .

sub-container-%:
	$(MAKE) --no-print-directory ARCH=$* container

all-container: $(addprefix sub-container-,$(ALL_ARCH))

# Container push, push is the target to push for multiple architectures as defined in ALL_ARCH

push: $(addprefix sub-push-,$(ALL_ARCH)) push-multi-arch;

sub-push-%: container-% do-push-% ;

do-push-%:
	${DOCKER_CLI} push $(IMAGE)-$*:$(TAG)

push-multi-arch:
	${DOCKER_CLI} manifest create --amend $(IMAGE):$(TAG) $(shell echo $(ALL_ARCH) | sed -e "s~[^ ]*~$(IMAGE)\-&:$(TAG)~g")
	@for arch in $(ALL_ARCH); do ${DOCKER_CLI} manifest annotate --arch $${arch} $(IMAGE):$(TAG) $(IMAGE)-$${arch}:$(TAG); done
	${DOCKER_CLI} manifest push --purge $(IMAGE):$(TAG)

clean:
	rm -f kube-state-metrics
	git clean -Xfd .

e2e:
	./tests/e2e.sh

generate: build-local
	@echo ">> generating docs"
	@./scripts/generate-help-text.sh
	embedmd -w `find . -path ./vendor -prune -o -name "*.md" -print`

validate-manifests: examples
	@git diff --exit-code

mixin: examples/prometheus-alerting-rules/alerts.yaml

examples/prometheus-alerting-rules/alerts.yaml: jsonnet $(shell find jsonnet | grep ".libsonnet") scripts/mixin.jsonnet scripts/vendor
	mkdir -p examples/prometheus-alerting-rules
	jsonnet -J scripts/vendor scripts/mixin.jsonnet | gojsontoyaml > examples/prometheus-alerting-rules/alerts.yaml

examples: examples/standard examples/autosharding mixin

examples/standard: jsonnet $(shell find jsonnet | grep ".libsonnet") scripts/standard.jsonnet scripts/vendor VERSION
	mkdir -p examples/standard
	jsonnet -J scripts/vendor -m examples/standard --ext-str version="$(VERSION)" scripts/standard.jsonnet | xargs -I{} sh -c 'cat {} | gojsontoyaml > `echo {} | sed "s/\(.\)\([A-Z]\)/\1-\2/g" | tr "[:upper:]" "[:lower:]"`.yaml' -- {}
	find examples -type f ! -name '*.yaml' -delete

examples/autosharding: jsonnet $(shell find jsonnet | grep ".libsonnet") scripts/autosharding.jsonnet scripts/vendor VERSION
	mkdir -p examples/autosharding
	jsonnet -J scripts/vendor -m examples/autosharding --ext-str version="$(VERSION)" scripts/autosharding.jsonnet | xargs -I{} sh -c 'cat {} | gojsontoyaml > `echo {} | sed "s/\(.\)\([A-Z]\)/\1-\2/g" | tr "[:upper:]" "[:lower:]"`.yaml' -- {}
	find examples -type f ! -name '*.yaml' -delete

scripts/vendor: scripts/jsonnetfile.json scripts/jsonnetfile.lock.json
	cd scripts && jb install

install-tools:
	@echo Installing tools from tools.go
	@cat tools/tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %

install-promtool:
	@echo Installing promtool
	@wget -qO- "https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VERSION}/prometheus-${PROMETHEUS_VERSION}.${OS}-${ARCH}.tar.gz" |\
	tar xvz --strip-components=1 prometheus-${PROMETHEUS_VERSION}.${OS}-${ARCH}/promtool

.PHONY: all build build-local all-push all-container container container-* do-push-* sub-push-* push push-multi-arch test-unit test-rules test-benchmark-compare clean e2e validate-modules shellcheck licensecheck lint generate embedmd
