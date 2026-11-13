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

# Builds and pushes the example images used by the e2e tests, tagged as <registry>/<image>:<tag>.
#
# Each image is first pushed under a tag derived from the git tree hashes of its build inputs, so
# if none of those inputs have changed since a previous run, the build is skipped and the existing
# image is just re-tagged.
#
# Usage: push-e2e-example-images.sh <registry> <tag>

set -euo pipefail

REGISTRY=$1
TAG=$2

root_path="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# example directory|image name|Makefile version variable|other build inputs, relative to the repository root.
# Examples built with the repository root as their Docker context include the root .dockerignore.
examples=(
  "simple-game-server|simple-game-server|VERSION|sdks/go pkg go.mod go.sum vendor .dockerignore"
  "supertuxkart|supertuxkart-example|version|"
  "xonotic|xonotic-example|version|"
  "rust-simple|rust-simple-server|version|sdks/rust"
  "cpp-simple|cpp-simple-server|version|sdks/cpp .dockerignore"
  "nodejs-simple|nodejs-simple-server|version|sdks/nodejs .dockerignore"
  "autoscaler-webhook|autoscaler-webhook|version|pkg go.mod go.sum .dockerignore"
)

push_example() {
  local dir=$1 image=$2 version_var=$3 inputs=$4
  local paths=("examples/${dir}") hash src

  read -r -a extra <<< "${inputs}"
  paths+=("${extra[@]}")

  hash=$(cd "${root_path}" && git rev-parse "${paths[@]/#/HEAD:}" | sha256sum | cut -c1-12)
  src="${REGISTRY}/${image}:src-${hash}"

  if docker buildx imagetools inspect "${src}" > /dev/null 2>&1; then
    echo "${src} exists, skipping build"
  else
    echo "${src} not found, building"
    # Only build and push linux amd64 images for the moment. We don't have the infrastructure to test arm64 or windows in CI at the moment.
    make -C "${root_path}/examples/${dir}" push REPOSITORY="${REGISTRY}" "${version_var}=src-${hash}" WITH_WINDOWS=0 WITH_ARM64=0
  fi

  docker buildx imagetools create --tag "${REGISTRY}/${image}:${TAG}" "${src}"
}

pids=()
for example in "${examples[@]}"; do
  IFS='|' read -r dir image version_var inputs <<< "${example}"
  { push_example "${dir}" "${image}" "${version_var}" "${inputs}" 2>&1 | sed "s/^/${dir}: /"; } &
  pids+=($!)
done

failed=0
for pid in "${pids[@]}"; do
  wait "${pid}" || failed=1
done
exit "${failed}"
