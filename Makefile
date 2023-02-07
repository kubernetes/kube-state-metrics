.PHONY: build build-alpine build-mod-alpine clean test help default

BIN_NAME:=$(notdir $(shell pwd))

 VERSION := v2.0.2
# 使用分支名作为version
# VERSION := $(shell git branch | grep \* | cut -d ' ' -f2)
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_DIRTY=$(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
IMAGE_NAME := "ecf-edge/coreos/${BIN_NAME}"
REMOTE_DOCKER_URI := harbor-dev.eecos.cn:1443/ecf-edge/coreos/${BIN_NAME}

ifndef $(GOPATH)
    GOPATH=$(shell go env GOPATH)
endif

ifeq ($(findstring $(GOPATH), $(pwd))), $(GOPATH))
    # Found
    IN_GOPATH=true
else
    # Not found
    IN_GOPATH=false
endif

ifeq ($(VERSION), develop)
    ENABLE_RACE=-race
endif

ifneq ($(findstring release, $(VERSION)),)
    ENABLE_RACE=-race
endif

default: build

help:
	@echo 'Management commands:'
	@echo
	@echo 'Usage:'
	@echo '    make build           Compile the project.'
	@echo '    make build-alpine    Compile optimized for alpine linux.'
	@echo '    make package         Build final docker image with just the go binary inside'
	@echo '    make tag             Tag image created by package with latest, git commit and version'
	@echo '    make test            Run tests on a compiled project.'
	@echo '    make push            Push tagged images to registry'
	@echo '    make clean           Clean the directory tree.'
	@echo '    make update-cookiecutter           Update the base config. e.g. gitlab-ci.yml, Dockerfile, Makefile...'
	@echo

build:
	@echo "building ${BIN_NAME} ${VERSION}"
	go build ${ENABLE_RACE} -ldflags "-X main.GitCommit=${GIT_COMMIT}${GIT_DIRTY} -X main.VersionPrerelease=DEV" -o bin/${BIN_NAME}
build-alpine:
	@echo "building ${BIN_NAME} ${VERSION}"
	go build ${ENABLE_RACE} -ldflags "-X main.GitCommit=${GIT_COMMIT}${GIT_DIRTY} -X main.VersionPrerelease=VersionPrerelease=RC" -o bin/${BIN_NAME}

build-mod-alpine:
	@echo "building ${BIN_NAME} ${VERSION}"
	go build ${ENABLE_RACE} -mod vendor -ldflags "-X main.GitCommit=${GIT_COMMIT}${GIT_DIRTY} -X main.VersionPrerelease=VersionPrerelease=RC" -o bin/${BIN_NAME}

package:
	@echo "building image ${BIN_NAME} ${VERSION} $(GIT_COMMIT)"
	go mod vendor
	# 加快编译
	#docker build ${DOCKER_BUILD_ARGS} --build-arg APP_NAME=${BIN_NAME} --build-arg VERSION=${VERSION} --build-arg GIT_COMMIT=${GIT_COMMIT} -t ${IMAGE_NAME}:latest .
	docker buildx build ${DOCKER_BUILD_ARGS} --platform=linux/amd64,linux/arm64 -o type=docker --build-arg APP_NAME=${BIN_NAME} --build-arg VERSION=${VERSION} --build-arg GIT_COMMIT=${GIT_COMMIT} -t ${IMAGE_NAME}:latest .
	rm -rf vendor

clean Dockerfile: clean Dockerfile
	docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
	docker buildx create --use
	docker buildx build ${DOCKER_BUILD_ARGS} --platform=linux/amd64,linux/arm64 -o type=docker --build-arg APP_NAME=${BIN_NAME} --build-arg VERSION=${VERSION} --build-arg GIT_COMMIT=${GIT_COMMIT} -t ${IMAGE_NAME}:latest .

tag: 
	@echo "Tagging: latest ${VERSION} $(GIT_COMMIT)"
	docker tag $(IMAGE_NAME):latest $(REMOTE_DOCKER_URI):${VERSION}
	docker tag $(IMAGE_NAME):latest $(REMOTE_DOCKER_URI):latest

push: tag
	docker push $(REMOTE_DOCKER_URI):${VERSION}

local-run:
	docker run --rm -p 8082:8082 -v `pwd`/config.yml:/etc/${BIN_NAME}/config.yml $(IMAGE_NAME):latest


clean:
	@test ! -e bin/${BIN_NAME} || rm bin/${BIN_NAME}
	@test ! -e /tmp/${BIN_NAME} || rm -rf /tmp/${BIN_NAME}

update-cookiecutter:
	rm -rf /tmp/cookiecutter
	mkdir -p /tmp/cookiecutter
	cd /tmp/cookiecutter && cookiecutter https://gitlab.ctyuncdn.cn/ecf/service-template.git --no-input app_name=$(BIN_NAME)
	cp /tmp/cookiecutter/$(BIN_NAME)/Makefile . | true
	cp /tmp/cookiecutter/$(BIN_NAME)/Dockerfile . | true
	cp /tmp/cookiecutter/$(BIN_NAME)/.gitlab-ci.yml . | true
	rm -rf /tmp/cookiecutter

test:
	go test -v `go list ./... | grep -v vendor`