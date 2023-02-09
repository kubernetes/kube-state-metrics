# Copyright 2016 The Kubernetes Authors All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
ARG BASEIMAGE

FROM golang:1.19 as builder

ENV GOPATH /gopath/
ENV PATH $GOPATH/bin:$PATH

RUN apt-get update --fix-missing && apt-get --yes install libsystemd-dev gcc-aarch64-linux-gnu
RUN go version
RUN  go env -w GOPROXY=https://goproxy.io,direct
RUN  go env -w GO111MODULE=on

COPY . /go/src/k8s.io/kube-state-metrics/
WORKDIR /go/src/k8s.io/kube-state-metrics/
RUN make build

ARG BASEIMAGE
FROM ${BASEIMAGE}
RUN clean-install util-linux libsystemd0 bash systemd

# Avoid symlink of /etc/localtime.
RUN test -h /etc/localtime && rm -f /etc/localtime && cp /usr/share/zoneinfo/UTC /etc/localtime || true

COPY --from=builder /go/src/k8s.io/kube-state-metrics/kube-state-metrics /


ENTRYPOINT ["/kube-state-metrics", "--port=8080", "--telemetry-port=8081"]

EXPOSE 8080 8081
