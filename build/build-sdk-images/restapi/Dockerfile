# Copyright 2019 Google LLC All Rights Reserved.
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
ARG BASE_IMAGE=agones-build-sdk-base:latest
FROM $BASE_IMAGE

RUN apt-get --allow-releaseinfo-change update && \
    apt-get install -y wget jq gnupg && \
    apt-get clean

# install go
WORKDIR /usr/local
ENV GO_VERSION=1.26.0
ENV GO111MODULE=on
ENV GOPATH /go
RUN wget -q https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -xzf go${GO_VERSION}.linux-amd64.tar.gz && rm go${GO_VERSION}.linux-amd64.tar.gz && mkdir -p ${GOPATH}

RUN echo '§' && apt-get -qy update

RUN apt-get install -qq -y openjdk-21-jre > /dev/null

ENV PATH /usr/local/go/bin:/go/bin:$PATH

# code generation scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh
