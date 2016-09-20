all: build

FLAGS =
ENVVAR = GOOS=linux GOARCH=amd64 CGO_ENABLED=0
REGISTRY = gcr.io/google_containers
TAG = v0.2.0

deps:
	go get github.com/tools/godep

build: clean deps
	$(ENVVAR) godep go build -o kube-state-metrics 

test-unit: clean deps build
	$(ENVVAR) godep go test --race . $(FLAGS)

container: build
	docker build -t ${REGISTRY}/kube-state-metrics:$(TAG) .

push: container
	gcloud docker push ${REGISTRY}/kube-state-metrics:$(TAG)

clean:
	rm -f kube-state-metrics

.PHONY: all deps build test-unit container push clean
