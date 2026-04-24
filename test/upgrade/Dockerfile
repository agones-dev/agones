# Copyright 2024 Google LLC All Rights Reserved.
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

FROM golang:1.26.0-alpine AS builder

# install curl
RUN apk update && \
    apk upgrade && \
    apk --no-cache add curl

WORKDIR /usr/local

# install kubectl
ENV KUBECTL_VER=1.34.6
RUN curl -LO https://dl.k8s.io/release/v${KUBECTL_VER}/bin/linux/amd64/kubectl && \
    chmod go+rx ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl

# install Helm package manager
ENV HELM_VER=3.18.4
ENV HELM_URL=https://get.helm.sh/helm-v${HELM_VER}-linux-amd64.tar.gz
RUN curl -L  ${HELM_URL} > /tmp/helm.tar.gz \
    && tar -zxvf /tmp/helm.tar.gz -C /tmp \
    && mv /tmp/linux-amd64/helm /usr/local/bin/helm \
    && chmod go+rx /usr/local/bin/helm \
    && rm /tmp/helm.tar.gz && rm -rf /tmp/linux-amd64

# Copy and build the Go application
WORKDIR /agones.dev
COPY test/upgrade/main.go ./
COPY test/upgrade/go.mod ./
COPY test/upgrade/go.sum ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /upgrade-test

# Copy the dev build Agones Helm chart
WORKDIR /

# Use a non-root user for security best practices
RUN adduser -D -g '' adduser
USER adduser
COPY --chown=adduser install/helm/agones /install/helm

ENTRYPOINT ["/upgrade-test"]
