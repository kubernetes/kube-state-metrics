ARG GOVERSION=1.17
ARG GOARCH
FROM golang:${GOVERSION} as builder
USER 65534
ENV GOARCH=${GOARCH}
ENV GOCACHE=/tmp/.cache
WORKDIR /go/src/k8s.io/kube-state-metrics/
COPY --chown=65534:65534 . /go/src/k8s.io/kube-state-metrics/


RUN make build-local

FROM gcr.io/distroless/static:latest-${GOARCH}
USER 65534

COPY --from=builder /go/src/k8s.io/kube-state-metrics/output/kube-state-metrics /

ENTRYPOINT ["/kube-state-metrics", "--port=8080", "--telemetry-port=8081"]

EXPOSE 8080 8081
