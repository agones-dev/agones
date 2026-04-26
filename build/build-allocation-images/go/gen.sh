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

export GO111MODULE=on

mkdir -p /go/src/
cp -r /go/src/agones.dev/agones/vendor/* /go/src/

cd /go/src/agones.dev/agones
go install -mod=vendor github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
go install -mod=vendor github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2

outputpath=pkg/allocation/go
protopath=proto/allocation
googleapis=/go/src/agones.dev/agones/proto/googleapis
gatewaygrpc=/go/src/agones.dev/agones/proto/grpc-gateway
protofile=${protopath}/allocation.proto
processor_protofile=${protopath}/processor.proto

rm ./${outputpath}/allocation.pb.go || true
rm ./${outputpath}/allocation.gw.pb.go || true
rm ./${outputpath}/allocation_grpc.pb.go || true
rm ./${outputpath}/processor.pb.go || true
rm ./${outputpath}/processor_grpc.pb.go || true

# generate the go code
protoc -I ${googleapis} -I ${gatewaygrpc} -I . -I ./vendor ${protofile} --go_out=proto --go-grpc_opt=require_unimplemented_servers=false --go-grpc_out=proto
protoc -I ${googleapis} -I ${gatewaygrpc} -I . -I ./vendor ${processor_protofile} --go_out=proto --go-grpc_opt=require_unimplemented_servers=false --go-grpc_out=proto

# generate grpc gateway
protoc -I ${googleapis} -I ${gatewaygrpc} -I . -I ./vendor ${protofile} --go_out=proto --grpc-gateway_out=logtostderr=true:proto

# generate openapi v2
protoc -I ${googleapis} -I ${gatewaygrpc} -I . -I ./vendor ${protofile} --openapiv2_opt=logtostderr=true,simple_operation_ids=true,disable_default_errors=true --openapiv2_out=json_names_for_fields=false,logtostderr=true:.

# hard coding because protoc-gen-openapiv2 doesn't work well in description
cat ./${protopath}/allocation.swagger.json | jq '.definitions.allocationAllocationRequest.properties.multiClusterSetting |= .+{"description": "If specified, multi-cluster policies are applied. Otherwise, allocation will happen locally."}' | sponge ./${protopath}/allocation.swagger.json
cat ./${protopath}/allocation.swagger.json | jq '.definitions.allocationAllocationRequest.properties.requiredGameServerSelector |= .+{"description": "Deprecated: Please use gameServerSelectors instead. This field is ignored if the\ngameServerSelectors field is set\nThe required allocation. Defaults to all GameServers."}' | sponge ./${protopath}/allocation.swagger.json
cat ./${protopath}/allocation.swagger.json | jq '.definitions.allocationAllocationRequest.properties.scheduling |= .+{"description": "Scheduling strategy. Defaults to \"Packed\"."}' | sponge ./${protopath}/allocation.swagger.json
cat ./${protopath}/allocation.swagger.json | jq '.definitions.allocationAllocationRequest.properties.metaPatch |= .+{"title": "Deprecated: Please use metadata instead. This field is ignored if the\nmetadata field is set"}' | sponge ./${protopath}/allocation.swagger.json
cat ./${protopath}/allocation.swagger.json | jq '.definitions.allocationAllocationRequest.properties.metadata |= .+{"title": "Metadata is optional custom metadata that is added to the game server at\nallocation. You can use this to tell the server necessary session data"}' | sponge ./${protopath}/allocation.swagger.json
cat ./${protopath}/allocation.swagger.json | jq '.definitions.allocationMultiClusterSetting.properties.policySelector |= .+{"description": "Selects multi-cluster allocation policies to apply. If not specified, all multi-cluster allocation policies are to be applied."}' | sponge ./${protopath}/allocation.swagger.json

jq 'del(.schemes[] | select(. == "http"))' ./${protopath}/allocation.swagger.json > ./${outputpath}/allocation.swagger.json

rm ${protopath}/allocation.swagger.json

header ${protopath}/allocation.pb.go
header ${protopath}/allocation.pb.gw.go
header ${protopath}/allocation_grpc.pb.go
header ${protopath}/processor.pb.go
header ${protopath}/processor_grpc.pb.go

mv ${protopath}/allocation.pb.go ${outputpath}/allocation.pb.go
mv ${protopath}/allocation.pb.gw.go ${outputpath}/allocation.pb.gw.go
mv ${protopath}/allocation_grpc.pb.go ${outputpath}/allocation_grpc.pb.go

mv ${protopath}/processor.pb.go ${outputpath}/processor.pb.go
mv ${protopath}/processor_grpc.pb.go ${outputpath}/processor_grpc.pb.go

goimports -w ./${outputpath}