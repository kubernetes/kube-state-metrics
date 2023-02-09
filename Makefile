CGO_ENABLED:=0
DOCKER_PLATFORMS=linux/amd64,linux/arm64
REGISTRY?=cloudx2021
TAG?=v0.2.2
IMAGE:=$(REGISTRY)/node-problem-detector:$(TAG)
BASEIMAGE:=k8s.gcr.io/debian-base:v2.0.0
ifeq ($(ENABLE_JOURNALD), 1)
	CGO_ENABLED:=1
	LOGCOUNTER=./bin/log-counter
endif


package:
	go mod tidy
	docker buildx create --use
	docker buildx build --push --platform $(DOCKER_PLATFORMS) -t $(IMAGE) --build-arg BASEIMAGE=$(BASEIMAGE) .

build: $(PKG_SOURCES)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GO111MODULE=on go build  -o kube-state-metrics