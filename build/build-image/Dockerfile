# Copyright 2017 Google LLC All Rights Reserved.
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

# ForceUpdate 12 -- change here if you need to force a rebuild

FROM debian:trixie

RUN apt-get update && \
    apt-get install -y build-essential gnupg curl git wget psmisc rsync make python3 bash-completion ruby\
    zip nano jq graphviz gettext-base plantuml && \
    apt-get clean

RUN gem install fpm && \
    fpm -s empty -t deb -p python2.7-0.0.0-0-any.deb --name python2.7 --license MIT --version 0.0.0 --architecture all --description 'Dummy package to satisfy google-cloud-cli dependencies.' --maintainer 'Yourself' && \
    dpkg -i python2.7-0.0.0-0-any.deb && \
    rm python2.7-0.0.0-0-any.deb

# install go
WORKDIR /usr/local
ENV GO_VERSION=1.26.0
ENV GOPATH=/go
ENV GO111MODULE=on
RUN wget -q https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -xzf go${GO_VERSION}.linux-amd64.tar.gz && rm go${GO_VERSION}.linux-amd64.tar.gz && mkdir ${GOPATH}

# install gcloud + kubectl, because it's an easy way to test/dev against kubernetes.
WORKDIR /opt

# credits https://cloud.google.com/sdk/docs/install#deb
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg && \
    apt-get update -y && \
    apt-get install google-cloud-cli google-cloud-cli-gke-gcloud-auth-plugin google-cloud-cli-app-engine-go -y && \
    echo "source /usr/share/google-cloud-sdk/completion.bash.inc" >> /root/.bashrc

# update the path for go
ENV PATH=/usr/local/go/bin:/go/bin:$PATH

# install go tooling for development, building and testing
RUN go install golang.org/x/tools/cmd/goimports@latest && \
    go install golang.org/x/pkgsite/cmd/pkgsite@latest

# the kubernetes version for the file
ENV KUBERNETES_VER=1.34.6

# overwrite kubectl as we want a specific version
RUN curl -LO https://dl.k8s.io/release/v${KUBERNETES_VER}/bin/linux/amd64/kubectl && \
    chmod go+rx ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl
RUN echo "source <(kubectl completion bash)" >> /root/.bashrc

# install the release branch of the code generator tools
RUN mkdir -p /go/src/k8s.io && cd /go/src/k8s.io && \
 git clone -b kubernetes-${KUBERNETES_VER} --depth=3 https://github.com/kubernetes/code-generator.git

# install Helm package manager
ENV HELM_VER=3.18.4
ENV HELM_URL=https://get.helm.sh/helm-v${HELM_VER}-linux-amd64.tar.gz
RUN curl -L  ${HELM_URL} > /tmp/helm.tar.gz \
    && tar -zxvf /tmp/helm.tar.gz -C /tmp \
    && mv /tmp/linux-amd64/helm /usr/local/bin/helm \
    && chmod go+rx /usr/local/bin/helm \
    && rm /tmp/helm.tar.gz && rm -rf /tmp/linux-amd64
RUN echo "source <(helm completion bash)" >> /root/.bashrc

# install golang-ci linter
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8

# Install yamllint
RUN apt-get update && \
    apt-get install -y yamllint

# install gotestsum test runner
RUN go install gotest.tools/gotestsum@latest

# install extism
RUN curl -s https://get.extism.org/cli | sh -s -- -v v1.6.2 -y && \
    extism lib install


#
#  \ \      / /__| |__  ___(_) |_ ___
#   \ \ /\ / / _ \ '_ \/ __| | __/ _ \
#    \ V  V /  __/ |_) \__ \ | |_  __/
#     \_/\_/ \___|_.__/|___/_|\__\___|
#

ENV HUGO_VER=0.152.2
RUN mkdir /tmp/hugo && \
    wget -q -O /tmp/hugo/hugo.tar.gz https://github.com/gohugoio/hugo/releases/download/v${HUGO_VER}/hugo_extended_${HUGO_VER}_Linux-64bit.tar.gz && \
    tar -zxvf /tmp/hugo/hugo.tar.gz -C /tmp/hugo/ && \
    mv /tmp/hugo/hugo /usr/local/bin/ && \
    rm -r /tmp/hugo

# Import the Nodesource GPG key
RUN apt-get update && \
    apt-get install -y ca-certificates curl gnupg && \
    mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg

# Create deb repository for a specific Node major version
ARG NODE_MAJOR=24
RUN echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_$NODE_MAJOR.x nodistro main" > /etc/apt/sources.list.d/nodesource.list

# Update and install Node.js
RUN apt-get update && apt-get install -y nodejs

# install API reference docs generator
RUN mkdir -p /go/src/github.com/ahmetb && \
    cd /go/src/github.com/ahmetb && git clone -b v0.3.0 https://github.com/ahmetb/gen-crd-api-reference-docs && \
    cd ./gen-crd-api-reference-docs && go build

# html checker
RUN mkdir /tmp/htmltest && \
    wget -O /tmp/htmltest/htmltest.tar.gz https://github.com/wjdp/htmltest/releases/download/v0.17.0/htmltest_0.17.0_linux_amd64.tar.gz && \
    tar -zxvf /tmp/htmltest/htmltest.tar.gz -C /tmp/htmltest && \
    mv /tmp/htmltest/htmltest /usr/local/bin && \
    rm -r /tmp/htmltest

# make sure we keep the path to go
RUN echo "export PATH=/usr/local/go/bin:/go/bin/:\$PATH" >> /root/.bashrc
# make nano the editor
RUN echo "export EDITOR=nano" >> /root/.bashrc

# install terraform
RUN wget -nv -O terraform.zip https://releases.hashicorp.com/terraform/1.5.4/terraform_1.5.4_linux_amd64.zip && unzip ./terraform.zip && mv terraform /usr/local/bin/

# code generation scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh

WORKDIR /go
