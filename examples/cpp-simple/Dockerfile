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

FROM debian:trixie as builder

RUN apt-get update && apt-get install -y \
  build-essential autoconf libtool git pkg-config \
  automake libtool curl make g++ unzip moreutils cmake \
  && apt-get clean

WORKDIR /project

# Install gRPC and dependencies
RUN git clone --recurse-submodules -b v1.80.0 --depth 1 --shallow-submodules https://github.com/grpc/grpc /var/local/git/grpc && \
    cd /var/local/git/grpc && \
    mkdir -p cmake/build && \
    cd cmake/build && \
    export CMAKE_BUILD_PARALLEL_LEVEL=$(nproc) && \
    cmake -DCMAKE_BUILD_TYPE=Release \
      -DgRPC_INSTALL=ON \
      -DgRPC_BUILD_TESTS=OFF ../.. && \
    make -j$(nproc) && make install

# Install Agones SDK
COPY ./sdks/cpp sdk
RUN cd sdk &&  \
    mkdir -p .build && \
    cd .build && \
    export CMAKE_BUILD_PARALLEL_LEVEL=$(nproc) && \
    cmake .. -DCMAKE_BUILD_TYPE=Release -DAGONES_SILENT_OUTPUT=OFF -DCMAKE_INSTALL_PREFIX=/project/sdk/.build \
      -G "Unix Makefiles" -Wno-dev && \
    cmake --build . --target install -j$(nproc)

# Build sample application
COPY ./examples/cpp-simple cpp-simple
RUN cd cpp-simple && mkdir -p .build && cd .build && \
    export CMAKE_BUILD_PARALLEL_LEVEL=$(nproc) && \
    cmake .. -G "Unix Makefiles" \
       -DCMAKE_BUILD_TYPE=Release \
       -DCMAKE_PREFIX_PATH=/project/sdk/.build \
       -Dagones_DIR=/project/sdk/.build/agones/cmake \
       -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=.bin && \
    cmake --build . --target install -j$(nproc)

FROM debian:trixie
RUN useradd -u 1000 -m server

COPY --from=builder --chown=server:server /project/cpp-simple/.build/.bin/cpp-simple /home/server/cpp-simple

USER 1000
ENTRYPOINT /home/server/cpp-simple