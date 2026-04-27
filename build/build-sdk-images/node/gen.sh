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
    cat ./build/boilerplate.go.txt $1 >> ./tmp.js && mv ./tmp.js $1
}

sdk=/go/src/agones.dev/agones/proto/sdk
googleapis=/go/src/agones.dev/agones/proto/googleapis
gatewaygrpc=/go/src/agones.dev/agones/proto/grpc-gateway

cd /go/src/agones.dev/agones

mkdir -p ./sdks/nodejs/lib/alpha
mkdir -p ./sdks/nodejs/lib/beta

grpc_tools_node_protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk} --js_out=import_style=commonjs,binary:./sdks/nodejs/lib google/api/annotations.proto google/api/client.proto google/api/field_behavior.proto google/api/http.proto google/api/launch_stage.proto google/api/resource.proto protoc-gen-openapiv2/options/annotations.proto protoc-gen-openapiv2/options/openapiv2.proto
grpc_tools_node_protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk}/alpha --js_out=import_style=commonjs,binary:./sdks/nodejs/lib/alpha google/api/annotations.proto google/api/http.proto protoc-gen-openapiv2/options/annotations.proto protoc-gen-openapiv2/options/openapiv2.proto
grpc_tools_node_protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk}/beta --js_out=import_style=commonjs,binary:./sdks/nodejs/lib/beta google/api/annotations.proto google/api/client.proto google/api/field_behavior.proto google/api/http.proto google/api/launch_stage.proto google/api/resource.proto protoc-gen-openapiv2/options/annotations.proto protoc-gen-openapiv2/options/openapiv2.proto

grpc_tools_node_protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk} --grpc_out=generate_package_definition:./sdks/nodejs/lib --js_out=import_style=commonjs,binary:./sdks/nodejs/lib sdk.proto
grpc_tools_node_protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk}/alpha --grpc_out=generate_package_definition:./sdks/nodejs/lib/alpha --js_out=import_style=commonjs,binary:./sdks/nodejs/lib/alpha alpha.proto
grpc_tools_node_protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk}/beta --grpc_out=generate_package_definition:./sdks/nodejs/lib/beta --js_out=import_style=commonjs,binary:./sdks/nodejs/lib/beta beta.proto

header ./sdks/nodejs/lib/sdk_pb.js
header ./sdks/nodejs/lib/sdk_grpc_pb.js
header ./sdks/nodejs/lib/google/api/annotations_pb.js
header ./sdks/nodejs/lib/google/api/http_pb.js
header ./sdks/nodejs/lib/protoc-gen-openapiv2/options/annotations_pb.js
header ./sdks/nodejs/lib/protoc-gen-openapiv2/options/openapiv2_pb.js

header ./sdks/nodejs/lib/alpha/alpha_pb.js
header ./sdks/nodejs/lib/alpha/alpha_grpc_pb.js
header ./sdks/nodejs/lib/alpha/google/api/annotations_pb.js
header ./sdks/nodejs/lib/alpha/google/api/http_pb.js
header ./sdks/nodejs/lib/alpha/protoc-gen-openapiv2/options/annotations_pb.js
header ./sdks/nodejs/lib/alpha/protoc-gen-openapiv2/options/openapiv2_pb.js

header ./sdks/nodejs/lib/beta/beta_pb.js
header ./sdks/nodejs/lib/beta/beta_grpc_pb.js
header ./sdks/nodejs/lib/beta/google/api/annotations_pb.js
header ./sdks/nodejs/lib/beta/google/api/http_pb.js
header ./sdks/nodejs/lib/beta/protoc-gen-openapiv2/options/annotations_pb.js
header ./sdks/nodejs/lib/beta/protoc-gen-openapiv2/options/openapiv2_pb.js
