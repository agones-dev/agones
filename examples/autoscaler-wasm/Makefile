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

# build the wasm plugin for testing
build:
	GOOS=wasip1 GOARCH=wasm go build -buildvcs=false -buildmode=c-shared -o plugin.wasm

# simple test to run the plugin with extism
test: build
	extism call plugin.wasm scale $(ARGS) --input '{"request":{"uid":"sample-uid","name":"sample-fleet","namespace":"default","status":{"replicas":5,"readyReplicas":3,"reservedReplicas":1,"allocatedReplicas":2},"labels":{"app":"sample-game"},"annotations":{"description":"sample-description"}}}' --wasi
