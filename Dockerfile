FROM registry.svc.ci.openshift.org/openshift/release:golang-1.10 AS builder
WORKDIR /go/src/k8s.io/kube-state-metrics
COPY . .
RUN make build

FROM  registry.svc.ci.openshift.org/openshift/origin-v4.0:base
LABEL io.k8s.display-name="kube-state-metrics" \
      io.k8s.description="This is a component that exposes metrics about Kubernetes objects." \
      io.openshift.tags="kubernetes" \
      maintainer="Frederic Branczyk <fbranczy@redhat.com>"

ARG FROM_DIRECTORY=/go/src/k8s.io/kube-state-metrics
COPY --from=builder ${FROM_DIRECTORY}/kube-state-metrics  /usr/bin/kube-state-metrics

USER nobody
ENTRYPOINT ["/usr/bin/kube-state-metrics"]
