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
set -e
FEATURES=$1
CLOUD_PRODUCT=$2
TEST_CLUSTER_NAME=$3
TEST_CLUSTER_LOCATION=$4
REGISTRY=$5
GS_TEST_IMAGE=$6

echo $FEATURES
export SHELL="/bin/bash"
export KUBECONFIG="/root/.kube/config"
mkdir -p /go/src/agones.dev/ /root/.kube/
ln -s /workspace /go/src/agones.dev/agones
cd /go/src/agones.dev/agones/build
if [ "$1" = 'local' ]
then
        gcloud auth login
fi
gcloud container clusters get-credentials $TEST_CLUSTER_NAME \
        --zone=${TEST_CLUSTER_LOCATION} --project=agones-images

echo /root/e2e.sh "${FEATURES}" "${CLOUD_PRODUCT}" "${REGISTRY}" "${GS_TEST_IMAGE}"
/root/e2e.sh "${FEATURES}" "${CLOUD_PRODUCT}" "${REGISTRY}" "${GS_TEST_IMAGE}"
