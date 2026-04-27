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
    cat /go/src/agones.dev/agones/build/boilerplate.go.txt "$1" | sponge "$1"
}

sdk=/go/src/agones.dev/agones/proto/sdk
googleapis=/go/src/agones.dev/agones/proto/googleapis
gatewaygrpc=/go/src/agones.dev/agones/proto/grpc-gateway

export GO111MODULE=on

mkdir -p /go/src/
cp -r /go/src/agones.dev/agones/vendor/* /go/src/

cd /go/src/agones.dev/agones
go install -mod=vendor github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
go install -mod=vendor github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2

mkdir -p ./pkg/sdk/{alpha,beta} || true

rm ./pkg/sdk/beta/beta.pb.go || true
rm ./pkg/sdk/alpha/alpha.pb.go || true
rm ./pkg/sdk/beta/beta_grpc.pb.go || true
rm ./pkg/sdk/alpha/alpha_grpc.pb.go || true
rm ./pkg/sdk/beta/beta.pb.gw.go || true
rm ./pkg/sdk/alpha/alpha.pb.gw.go || true

# generate the go code for each feature stage
protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk} sdk.proto --go_out=pkg --go-grpc_opt=require_unimplemented_servers=false --go-grpc_out=pkg
protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk}/alpha alpha.proto --go_out=pkg/sdk --go-grpc_opt=require_unimplemented_servers=false --go-grpc_out=pkg/sdk
protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk}/beta beta.proto --go_out=pkg/sdk --go-grpc_opt=require_unimplemented_servers=false --go-grpc_out=pkg/sdk

# generate grpc gateway
protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk} sdk.proto --grpc-gateway_out=logtostderr=true:pkg
protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk}/alpha alpha.proto --grpc-gateway_out=logtostderr=true:pkg/sdk
protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk}/beta beta.proto --grpc-gateway_out=logtostderr=true:pkg/sdk

# generate openapi v2
protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk} sdk.proto --openapiv2_opt=logtostderr=true,simple_operation_ids=true,disable_default_errors=true --openapiv2_out=json_names_for_fields=false,logtostderr=true:sdks/swagger
protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk}/alpha alpha.proto --openapiv2_opt=logtostderr=true,simple_operation_ids=true,disable_default_errors=true --openapiv2_out=json_names_for_fields=false,logtostderr=true:sdks/swagger
protoc -I ${googleapis} -I ${gatewaygrpc} -I ${sdk}/beta beta.proto --openapiv2_opt=logtostderr=true,simple_operation_ids=true,disable_default_errors=true --openapiv2_out=json_names_for_fields=false,logtostderr=true:sdks/swagger

# hard coding because protoc-gen-openapiv2 doesn't work well in Stream and doesn't generate 'googlerpcStatus' and 'protobufAny' definitions
cat sdks/swagger/sdk.swagger.json | jq '.definitions |= .+{"googlerpcStatus": {"type": "object", "properties": { "code": { "type": "integer", "format": "int32"}, "message": { "type":"string"}, "details": { "type": "array", "items": { "$ref": "#/definitions/protobufAny"}}}}}' | sponge sdks/swagger/sdk.swagger.json
cat sdks/swagger/sdk.swagger.json | jq '.definitions |= .+{"protobufAny": { "type": "object", "properties": { "@type": { "type": "string" }}, "additionalProperties": {}},}' | sponge sdks/swagger/sdk.swagger.json

header ./pkg/sdk/sdk.pb.go
header ./pkg/sdk/alpha/alpha.pb.go
header ./pkg/sdk/beta/beta.pb.go
header ./pkg/sdk/sdk.pb.gw.go
header ./pkg/sdk/sdk_grpc.pb.go
header ./pkg/sdk/alpha/alpha_grpc.pb.go
header ./pkg/sdk/beta/beta_grpc.pb.go

# these files may not exist if there are no grpc services
if [ -f "./pkg/sdk/alpha/alpha.pb.gw.go" ]; then
    header ./pkg/sdk/alpha/alpha.pb.gw.go
fi
if [ -f "./pkg/sdk/beta/beta.pb.gw.go" ]; then
    header ./pkg/sdk/beta/beta.pb.gw.go
fi

goimports -w ./pkg/sdk/*
goimports -w ./pkg/sdk/alpha/*
goimports -w ./pkg/sdk/beta/*