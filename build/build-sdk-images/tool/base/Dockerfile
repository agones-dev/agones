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


#
# Base images for SDKs Bump: 3
#

FROM debian:trixie

RUN apt-get update && apt-get install -y \
  build-essential autoconf libtool git pkg-config curl \
  automake libtool curl make g++ unzip moreutils cmake \
  && apt-get clean

ARG GRPC_RELEASE_TAG
# Install grpc locally for reuse across builds.
RUN git clone --recurse-submodules -b $GRPC_RELEASE_TAG --depth 1 --shallow-submodules https://github.com/grpc/grpc /var/local/git/grpc && \
    cd /var/local/git/grpc && \
    mkdir -p cmake/build && \
    cd cmake/build && \
    export CMAKE_BUILD_PARALLEL_LEVEL=$(nproc) && \
    cmake -DCMAKE_BUILD_TYPE=Release -DgRPC_INSTALL=ON -DgRPC_BUILD_TESTS=OFF ../.. && \
    make -j$(nproc) && make install

WORKDIR /go/src/agones.dev/agones

# code generation scripts
COPY entrypoint.sh /root/
RUN chmod +x /root/entrypoint.sh
ENTRYPOINT [ "/root/entrypoint.sh" ]
