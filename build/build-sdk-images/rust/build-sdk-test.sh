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

mkdir -p /go/src/agones.dev/agones/test/sdk/rust/.cargo
mkdir -p /go/src/agones.dev/agones/test/sdk/rust/.cargo-targets
cd /go/src/agones.dev/agones/test/sdk/rust
export CARGO_HOME=/go/src/agones.dev/agones/test/sdk/rust/.cargo
export CARGO_TARGET_DIR=/go/src/agones.dev/agones/test/sdk/rust/.cargo-targets
cargo fetch
cargo build
