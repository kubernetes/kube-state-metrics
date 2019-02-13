FROM alpine:3.9

COPY kube-state-metrics /

ENTRYPOINT ["/kube-state-metrics", "--port=8080", "--telemetry-port=8081"]

EXPOSE 8080 8081
