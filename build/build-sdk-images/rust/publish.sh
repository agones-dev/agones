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

cd ./sdks/rust/proto/sdk

# Authenticate with crates.io
read -p 'Crates.io API Token: ' TOKEN
cargo login $TOKEN

# Perform a dry run of cargo publish
dry_run_output=$(cargo publish --dry-run 2>&1)

# Check if the dry run output contains the warning about aborting upload
if echo "$dry_run_output" | grep -q "warning: aborting upload due to dry run"; then
    echo "Actual Cargo Publish begins.."
    # Dry run succeeded, proceed to actual publishing
    cargo publish
else
    echo "Dry run failed. Aborting actual publishing."
fi
