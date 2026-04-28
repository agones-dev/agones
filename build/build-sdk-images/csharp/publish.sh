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
cd ./sdks/csharp/sdk

dotnet pack csharp-sdk.csproj --configuration Release --include-source --output ./pkgs
read -p 'NuGet API Key: ' TOKEN && \
dotnet nuget push ./pkgs/AgonesSDK.$VERSION.nupkg -k $TOKEN -s https://api.nuget.org/v3/index.json
