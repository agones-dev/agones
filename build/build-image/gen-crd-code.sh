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

#!/bin/bash

set -x
set -o errexit
set -o nounset
set -o pipefail

CODEGEN_SCRIPT="/go/src/k8s.io/code-generator/kube_codegen.sh"

source "${CODEGEN_SCRIPT}"

echo "Generating CRD client code..."
OUTPUT_DIR="/go/src/agones.dev/agones/pkg/client"
OUTPUT_PKG="agones.dev/agones/pkg/client"

kube::codegen::gen_client \
  --with-watch \
  --with-applyconfig \
  --output-dir "${OUTPUT_DIR}" \
  --output-pkg "${OUTPUT_PKG}" \
  --boilerplate /go/src/agones.dev/agones/build/boilerplate.go.txt \
  /go/src/agones.dev/agones/pkg/apis

echo "CRD client code generation complete."

echo "Generating CRD conversions, deepcopy, and defaults code..."

kube::codegen::gen_helpers \
  --boilerplate /go/src/agones.dev/agones/build/boilerplate.go.txt \
  /go/src/agones.dev/agones/pkg/apis

echo "CRD conversions, deepcopy, and defaults code generation complete."
