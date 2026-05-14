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

FROM rust:1.92.0 as builder
RUN useradd -m build
RUN apt update && apt install -y protobuf-compiler

# SDK
COPY sdk /home/builder/agones/sdks/rust

# Example
COPY src /home/builder/agones/examples/rust-simple/src
COPY Cargo.toml /home/builder/agones/examples/rust-simple/
COPY Makefile /home/builder/agones/examples/rust-simple/

WORKDIR /home/builder/agones/examples/rust-simple
RUN make build-server

FROM debian:trixie
RUN useradd -u 1000 -m server

COPY --from=builder --chown=server:server /home/builder/agones/examples/rust-simple/target/release/rust-simple /home/server/rust-simple

USER 1000
ENTRYPOINT /home/server/rust-simple
