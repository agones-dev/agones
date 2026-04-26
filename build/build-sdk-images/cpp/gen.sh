#!/usr/bin/env bash

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

set -ex

header() {
    cat /go/src/agones.dev/agones/build/boilerplate.go.txt ./$1 >> $2/$1
}

sdk=/go/src/agones.dev/agones/proto/sdk
googleapis=/go/src/agones.dev/agones/proto/googleapis
gatewaygrpc=/go/src/agones.dev/agones/proto/grpc-gateway
protoc_intermediate=/go/src/agones.dev/agones/sdks/cpp/.generated
protoc_destination=/go/src/agones.dev/agones/sdks/cpp

mkdir -p ${protoc_intermediate}
mkdir -p ${protoc_destination}/src/agones
mkdir -p ${protoc_destination}/src/google
mkdir -p ${protoc_destination}/src/protoc-gen-openapiv2
mkdir -p ${protoc_destination}/include/agones
mkdir -p ${protoc_destination}/include/google/api
mkdir -p ${protoc_destination}/include/protoc-gen-openapiv2/options

cd /go/src/agones.dev/agones/sdks/cpp
find -name '*.pb.*' -delete
cd /go/src/agones.dev/agones
protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk} --plugin=protoc-gen-grpc=`which grpc_cpp_plugin` --grpc_out=${protoc_intermediate} sdk.proto
protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk} --cpp_out=dllexport_decl=AGONES_EXPORT:${protoc_intermediate} sdk.proto ${googleapis}/google/api/annotations.proto ${googleapis}/google/api/http.proto ${gatewaygrpc}/protoc-gen-openapiv2/options/annotations.proto ${gatewaygrpc}/protoc-gen-openapiv2/options/openapiv2.proto

cd ${protoc_intermediate}
header sdk.grpc.pb.cc ${protoc_destination}/src/agones
header sdk.pb.cc ${protoc_destination}/src/agones
header sdk.grpc.pb.h ${protoc_destination}/include/agones
header sdk.pb.h ${protoc_destination}/include/agones

cd ${protoc_intermediate}/google/api
header annotations.pb.cc ${protoc_destination}/src/google
header http.pb.cc ${protoc_destination}/src/google
header annotations.pb.h ${protoc_destination}/include/google/api
header http.pb.h ${protoc_destination}/include/google/api

cd ${protoc_intermediate}/protoc-gen-openapiv2/options
header annotations.pb.cc ${protoc_destination}/src/protoc-gen-openapiv2
header openapiv2.pb.cc ${protoc_destination}/src/protoc-gen-openapiv2
header annotations.pb.h ${protoc_destination}/include/protoc-gen-openapiv2/options
header openapiv2.pb.h ${protoc_destination}/include/protoc-gen-openapiv2/options

rm -r ${protoc_intermediate}
