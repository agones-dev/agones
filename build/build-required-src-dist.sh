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

set -o errexit
set -o nounset
set -o pipefail

SRC_ROOT=$(dirname "${BASH_SOURCE}")/..

TMP_DEPS_SRC=/tmp/dependencies-src.tgz

# Pack the source code of dependencies that are MPL
tar -zcf ${TMP_DEPS_SRC} -C ${SRC_ROOT}/vendor/ \
  github.com/hashicorp/golang-lru \
  github.com/hashicorp/hcl

for ddir in ${SRC_ROOT}/cmd/controller/bin/ ${SRC_ROOT}/cmd/extensions/bin/ ${SRC_ROOT}/cmd/ping/bin/ ${SRC_ROOT}/cmd/sdk-server/bin/ ${SRC_ROOT}/cmd/allocator/bin/ ${SRC_ROOT}/cmd/processor/bin/  ; do
  mkdir -p ${ddir}
  cp ${TMP_DEPS_SRC} ${ddir}
done

