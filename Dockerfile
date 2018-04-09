FROM alpine:3.7

COPY kube-state-metrics /
VOLUME /tmp

ENTRYPOINT ["/kube-state-metrics", "--port=8080", "--telemetry-port=8081"]

EXPOSE 8080
EXPOSE 8081
