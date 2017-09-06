FLAGS =
BUILDENVVAR = CGO_ENABLED=0
TESTENVVAR = 
REGISTRY = gcr.io/google_containers
TAG = $(shell git describe --abbrev=0)
PKGS = $(shell go list ./... | grep -v /vendor/)
ARCH ?= $(shell go env GOARCH)
ALL_ARCH = amd64 arm arm64 ppc64le s390x


IMAGE = $(REGISTRY)/kube-state-metrics
MULTI_ARCH_IMG = $(IMAGE)-$(ARCH)

BASEIMAGE ?= busybox:latest

ifeq ($(ARCH),arm)
	BASEIMAGE=arm32v7/busybox:latest
endif
ifeq ($(ARCH),arm64)
	BASEIMAGE=arm64v8/busybox:latest
endif
ifeq ($(ARCH),ppc64le)
	BASEIMAGE=ppc64le/busybox:latest
endif
ifeq ($(ARCH),s390x)
	BASEIMAGE=s390x/busybox:latest
endif

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
	GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(ARCH) $(BUILDENVVAR) go build -o kube-state-metrics

test-unit: clean build
	GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(ARCH) $(TESTENVVAR) go test --race $(FLAGS) $(PKGS)

TEMP_DIR := $(shell mktemp -d)

all: all-container

sub-container-%:
	$(MAKE) ARCH=$* container

sub-push-%:
	$(MAKE) ARCH=$* push

all-container: $(addprefix sub-container-,$(ALL_ARCH))

all-push: $(addprefix sub-push-,$(ALL_ARCH))

container: .container-$(ARCH)
.container-$(ARCH):
	cp -r * $(TEMP_DIR)
	OOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(ARCH) $(BUILDENVVAR) go build -o $(TEMP_DIR)/kube-state-metrics
	cd $(TEMP_DIR) && sed -i 's|BASEIMAGE|$(BASEIMAGE)|g' Dockerfile
	docker build -t $(MULTI_ARCH_IMG):$(TAG) $(TEMP_DIR)

push: .push-$(ARCH)
.push-$(ARCH): .container-$(ARCH)
	gcloud docker -- push $(MULTI_ARCH_IMG):$(TAG)
ifeq ($(ARCH), amd64)
	gcloud docker -- push $(IMAGE):$(TAG)
endif

clean:
	rm -f kube-state-metrics

.PHONY: all build test-unit container push clean
