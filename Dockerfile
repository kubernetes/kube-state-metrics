FROM alpine:3.9

COPY kube-state-metrics /

ENTRYPOINT ["/kube-state-metrics", "--port=8080", "--telemetry-port=8081"]

RUN adduser -D kube-state-metrics

USER kube-state-metrics

EXPOSE 8080 8081
