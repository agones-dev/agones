//go:build tools
// +build tools

// Copyright Contributors to Agones a Series of LF Projects, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package agones

/*
	github.com/ahmetb/gen-crd-api-reference-docs                        site/gen-api-docs.sh
	github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway   build-sdk-images/go/gen.sh
	github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2      build-sdk-images/go/gen.sh
	golang.org/x/tools/cmd/goimports                                    build-image/Dockerfile
	google.golang.org/protobuf/cmd/protoc-gen-go                        build-sdk-images/go/Dockerfile
*/

import (
	_ "github.com/ahmetb/gen-crd-api-reference-docs"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	_ "golang.org/x/tools/cmd/goimports"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
