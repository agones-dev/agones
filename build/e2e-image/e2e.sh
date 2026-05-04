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

FEATURES=$1
CLOUD_PRODUCT=$2
REGISTRY=$3
GS_TEST_IMAGE=$4

echo $FEATURES
echo $REGISTRY
set -e
echo "installing current release"
DOCKER_RUN= make install FEATURE_GATES='"'$FEATURES'"' REGISTRY='"'$REGISTRY'"' 
echo "starting e2e test"
DOCKER_RUN= make test-e2e ARGS=-parallel=16 E2E_USE_GOTESTSUM=true GOTESTSUM_VERBOSE=true FEATURE_GATES='"'$FEATURES'"' CLOUD_PRODUCT='"'$CLOUD_PRODUCT'"' GS_TEST_IMAGE='"'$GS_TEST_IMAGE'"'
echo "completed e2e test"
