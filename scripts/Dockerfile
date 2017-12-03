FROM alpine:3.6

RUN apk add --no-cache curl

CMD curl -s -k -H "Authorization: Bearer `cat /var/run/secrets/kubernetes.io/serviceaccount/token`" http://kube-state-metrics.kube-system.svc:8080/metrics
