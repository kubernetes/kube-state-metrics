FLAGS =
TESTENVVAR =
REGISTRY = quay.io/coreos
TAG = $(shell git describe --abbrev=0)
PKGS = $(shell go list ./... | grep -v /vendor/)
ARCH ?= $(shell go env GOARCH)
BuildDate = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
Commit = $(shell git rev-parse --short HEAD)
ALL_ARCH = amd64 arm arm64 ppc64le s390x
PKG=k8s.io/kube-state-metrics
GO_VERSION=1.10.1

IMAGE = $(REGISTRY)/kube-state-metrics
MULTI_ARCH_IMG = $(IMAGE)-$(ARCH)

gofmtcheck:
	@go fmt $(PKGS) | grep ".*\.go"; if [ "$$?" = "0" ]; then exit 1; fi

doccheck:
	@echo "- Checking if documentation is up to date..."
	@grep -hoE '(kube_[^ |]+)' Documentation/* | sort -u > documented_metrics
	@sed -n 's/.*# TYPE \(kube_[^ ]\+\).*/\1/p' collectors/*_test.go | sort -u > tested_metrics
	@diff -u0 tested_metrics documented_metrics || (echo "ERROR: Metrics with - are present in tests but missing in documentation, metrics with + are documented but not tested."; exit 1)
	@echo OK
	@rm -f tested_metrics documented_metrics
	@echo "- Checking for orphan documentation files"
	@cd Documentation; for doc in *.md; do if [ "$$doc" != "README.md" ] && ! grep -q "$$doc" *.md; then echo "ERROR: No link to documentation file $${doc} detected"; exit 1; fi; done
	@echo OK

build: clean
	docker run --rm -v "$$PWD":/go/src/k8s.io/kube-state-metrics -w /go/src/k8s.io/kube-state-metrics -e GOOS=$(shell uname -s | tr A-Z a-z) -e GOARCH=$(ARCH) -e CGO_ENABLED=0 golang:${GO_VERSION} go build -ldflags "-s -w -X ${PKG}/version.Release=${TAG} -X ${PKG}/version.Commit=${Commit} -X ${PKG}/version.BuildDate=${BuildDate}" -o kube-state-metrics

test-unit: clean build
	GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(ARCH) $(TESTENVVAR) go test --race $(FLAGS) $(PKGS)

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
	docker run --rm -v "$$PWD":/go/src/k8s.io/kube-state-metrics -w /go/src/k8s.io/kube-state-metrics -e GOOS=linux -e GOARCH=$(ARCH) -e CGO_ENABLED=0 golang:${GO_VERSION} go build -ldflags "-s -w -X ${PKG}/version.Release=${TAG} -X ${PKG}/version.Commit=${Commit} -X ${PKG}/version.BuildDate=${BuildDate}" -o kube-state-metrics
	cp -r * $(TEMP_DIR)
	docker build -t $(MULTI_ARCH_IMG):$(TAG) $(TEMP_DIR)

ifeq ($(ARCH), amd64)
	# Adding check for amd64
	docker tag $(MULTI_ARCH_IMG):$(TAG) $(IMAGE):$(TAG)
endif


push: .push-$(ARCH)
.push-$(ARCH): .container-$(ARCH)
	gcloud docker -- push $(MULTI_ARCH_IMG):$(TAG)
ifeq ($(ARCH), amd64)
	gcloud docker -- push $(IMAGE):$(TAG)
endif

clean:
	rm -f kube-state-metrics

e2e:
	./tests/e2e.sh

.PHONY: all build all-push all-container test-unit container push clean e2e
