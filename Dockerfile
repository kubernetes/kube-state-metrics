ARG GOVERSION=1.19
ARG GOARCH
FROM golang:${GOVERSION} as builder
ARG GOARCH
ENV GOARCH=${GOARCH}
WORKDIR /go/src/k8s.io/kube-state-metrics/
COPY . /go/src/k8s.io/kube-state-metrics/
RUN  go env -w GOPROXY=https://goproxy.io,direct
RUN  go env -w GO111MODULE=on
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o kube-state-metrics   main.go

FROM alpine
COPY --from=builder /go/src/k8s.io/kube-state-metrics/kube-state-metrics /

USER nobody

ENTRYPOINT ["/kube-state-metrics", "--port=8080", "--telemetry-port=8081"]

EXPOSE 8080 8081
