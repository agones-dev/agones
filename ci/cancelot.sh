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

#
# Provides automation for cancelling Cloud Builds
# Use as a first step to cancel previous builds currently in progress or queued for the same branch name and trigger id.
# Similar to: https://github.com/GoogleCloudPlatform/cloud-builders-community/tree/master/cancelot
#
# Usage within Cloud Build step:
#    steps:
#    - name: 'gcr.io/cloud-builders/gcloud-slim:latest'
#      entrypoint: 'bash'
#      args: ['./cancelot.sh', '--current_build_id', '$BUILD_ID']

# Exit script when command fails
set -o errexit
# Return value of a pipeline is the value of the last (rightmost) command to exit with a non-zero status
set -o pipefail

CMDNAME=${0##*/}
echoerr() { echo "$@" 1>&2; }

usage() {
    cat <<USAGE >&2
Usage:
    $CMDNAME --current_build_id \$BUILD_ID
    --current_build_id \$BUILD_ID  Current Build Id
USAGE
    exit 1
}

# Process arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
    --current_build_id)
        CURRENT_BUILD_ID="$2"
        if [[ $CURRENT_BUILD_ID == "" ]]; then break; fi
        shift 2
        ;;
    --help)
        usage
        ;;
    *)
        echoerr "Unknown argument: $1"
        usage
        ;;
    esac
done

if [[ "$CURRENT_BUILD_ID" == "" ]]; then
    echo "Error: you need to provide Build Id"
    usage
fi

# Note BUILD_BRANCH and BUILD_TRIGGER_ID could be empty
QUERY_BUILD=$(gcloud builds describe "$CURRENT_BUILD_ID" --format="csv[no-heading](createTime, buildTriggerId, substitutions.BRANCH_NAME)")
IFS="," read -r BUILD_CREATE_TIME BUILD_TRIGGER_ID BUILD_BRANCH <<<"$QUERY_BUILD"

FILTERS="id!=$CURRENT_BUILD_ID AND createTime<$BUILD_CREATE_TIME AND substitutions.BRANCH_NAME=$BUILD_BRANCH AND buildTriggerId=$BUILD_TRIGGER_ID"

echo "Filtering ongoing builds for branch '$BUILD_BRANCH' trigger id '$BUILD_TRIGGER_ID' created before: $BUILD_CREATE_TIME"

# Get ongoing build ids to cancel (+status)
while IFS=$'\n' read -r line; do CANCEL_BUILDS+=("$line"); done < <(gcloud builds list --ongoing --filter="$FILTERS" --format="value(id, status)")

BUILDS_COUNT=${#CANCEL_BUILDS[@]}
echo "Found $BUILDS_COUNT builds to cancel"
if [[ $BUILDS_COUNT -eq 0 ]]; then
    exit 0
fi

# Cancel builds one by one to get output for each
# printf '%s\n' "${CANCEL_BUILDS[@]}"
echo "BUILD ID                                CURRENT STATUS"
for build in "${CANCEL_BUILDS[@]}"; do
    echo "$build"
    ID=$(echo "$build" | awk '{print $1;}')
    gcloud builds cancel "$ID"  >/dev/null || true
done