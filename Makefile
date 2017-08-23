all: build

FLAGS =
COMMONENVVAR = GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m)))
BUILDENVVAR = CGO_ENABLED=0
TESTENVVAR = 
REGISTRY = gcr.io/google_containers
TAG = $(shell git describe --abbrev=0)
PKGS = $(shell go list ./... | grep -v /vendor/)

gofmtcheck:
	@go fmt $(PKGS) | grep ".*\.go"; if [ "$$?" = "0" ]; then exit 1; fi     

doccheck:
	grep -hoE '(kube_[^ |]+)' Documentation/* | sort -u > documented_metrics
	sed -n 's/.*# TYPE \(kube_[^ ]\+\).*/\1/p' collectors/*_test.go | sort -u > tested_metrics
	diff -u0 tested_metrics documented_metrics
	rm -f tested_metrics documented_metrics

deps:
	go get github.com/tools/godep

build: clean deps
	$(COMMONENVVAR) $(BUILDENVVAR) godep go build -o kube-state-metrics 

test-unit: clean deps build
	$(COMMONENVVAR) $(TESTENVVAR) godep go test --race $(FLAGS) $(PKGS)

container: build
	docker build -t ${REGISTRY}/kube-state-metrics:$(TAG) .

push: container
	gcloud docker -- push ${REGISTRY}/kube-state-metrics:$(TAG)

clean:
	rm -f kube-state-metrics

.PHONY: all deps build test-unit container push clean
