FROM gcr.io/cloud-builders/gcloud

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    wget \
    psmisc \
    make \
    gcc \
    jq \
    zip && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# install go
WORKDIR /usr/local
ENV GO_VERSION=1.26.0
ENV GOPATH /go
ENV GO111MODULE=on
RUN wget -q https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -xzf go${GO_VERSION}.linux-amd64.tar.gz && rm go${GO_VERSION}.linux-amd64.tar.gz && mkdir ${GOPATH}

ENV PATH /usr/local/go/bin:/go/bin:$PATH

# install kubectl without gcloud as we want the last version
ENV KUBECTL_VER 1.34.6
RUN curl -LO https://dl.k8s.io/release/v${KUBECTL_VER}/bin/linux/amd64/kubectl && \
    chmod go+rx ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl

# install Helm package manager
ENV HELM_VER 3.18.4
ENV HELM_URL https://get.helm.sh/helm-v${HELM_VER}-linux-amd64.tar.gz
RUN curl -L  ${HELM_URL} > /tmp/helm.tar.gz \
    && tar -zxvf /tmp/helm.tar.gz -C /tmp \
    && mv /tmp/linux-amd64/helm /usr/local/bin/helm \
    && chmod go+rx /usr/local/bin/helm \
    && rm /tmp/helm.tar.gz && rm -rf /tmp/linux-amd64

# install gotestsum test runner
RUN go install gotest.tools/gotestsum@latest

# make sure we keep the path to go
RUN echo "export PATH=/usr/local/go/bin:/go/bin/:\$PATH" >> /root/.bashrc
# scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh

WORKDIR /go

ENTRYPOINT [ "/root/entrypoint.sh" ]
