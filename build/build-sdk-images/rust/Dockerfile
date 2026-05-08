# Copyright Contributors to Agones a Series of LF Projects, LLC.
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
    apt-get install -y wget && \
    apt-get clean

# install rust
ENV RUSTUP_HOME=/usr/local/rustup \
    CARGO_HOME=/usr/local/cargo \
    PATH=/usr/local/cargo/bin:$PATH \
    RUST_VERSION=1.90.0
RUN ARCH=$(dpkg --print-architecture) && \
    case "${ARCH}" in \
        amd64) RUSTUP_ARCH=x86_64-unknown-linux-gnu ;; \
        arm64) RUSTUP_ARCH=aarch64-unknown-linux-gnu ;; \
        *) echo "Unsupported architecture: ${ARCH}" && exit 1 ;; \
    esac && \
    wget -q https://static.rust-lang.org/rustup/dist/${RUSTUP_ARCH}/rustup-init && \
    chmod +x rustup-init && \
    ./rustup-init -y --no-modify-path --default-toolchain $RUST_VERSION && \
    rm rustup-init && \
    rustup --version; \
    cargo --version; \
    rustc --version;

# code generation scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh
