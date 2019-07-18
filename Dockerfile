FROM alpine:3.9

RUN adduser -D kube-state-metrics

FROM gcr.io/distroless/static

COPY kube-state-metrics /

COPY --from=0 /etc/passwd /etc/passwd

USER kube-state-metrics

ENTRYPOINT ["/kube-state-metrics", "--port=8080", "--telemetry-port=8081"]

EXPOSE 8080 8081
