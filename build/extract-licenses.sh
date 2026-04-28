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

append_license() {
  lib=$1
  path=$2
  echo "================================================================================" >> ${TMP_LICENSES}
  echo "= ${lib} licensed under: =" >> ${TMP_LICENSES}
  echo >> ${TMP_LICENSES}
  cat "$path" >> ${TMP_LICENSES}
  echo >> ${TMP_LICENSES}
  echo "= ${path} MD5 $(cat "${path}" | md5sum | awk '{print $1}')" >> ${TMP_LICENSES}
  echo "================================================================================" >> ${TMP_LICENSES}
  echo >> ${TMP_LICENSES}

}

SRC_ROOT=$(dirname "${BASH_SOURCE}")/..
TMP_LICENSES=/tmp/LICENSES

cd ${SRC_ROOT}

# Clear file
echo > ${TMP_LICENSES}

append_license "Agones" "LICENSE"

while read -r entry; do
  LIBRARY=${entry#vendor\/}
  LIBRARY=$(expr match "$LIBRARY" '\(.*\)/LICENSE.*\?')
  append_license ${LIBRARY} ${entry}
done <<< "$(find vendor/ -regextype posix-extended -iregex '.*LICENSE(\.txt)?')"

for ddir in ${SRC_ROOT}/cmd/controller/bin/ ${SRC_ROOT}/cmd/extensions/bin/ ${SRC_ROOT}/cmd/ping/bin/ ${SRC_ROOT}/cmd/sdk-server/bin/ ${SRC_ROOT}/cmd/allocator/bin/ ${SRC_ROOT}/cmd/processor/bin/; do
  mkdir -p ${ddir}
  cp ${TMP_LICENSES} ${ddir}
done

