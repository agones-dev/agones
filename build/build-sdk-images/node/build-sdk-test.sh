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

cd /go/src/agones.dev/agones/test/sdk/nodejs
npm install --quiet
npm rebuild --quiet

# If first 'npm install' attempt fails, which could occur for a variety of reasons,
# do one more attempt
if [ $? -gt 0 ]
then
    echo "Run npm install one more time"
    rm -rf /go/src/agones.dev/agones/sdks/nodejs/node_modules
    rm -rf /go/src/agones.dev/agones/test/sdk/nodejs/node_modules
    rm /go/src/agones.dev/agones/test/sdk/nodejs/package-lock.json
    npm cache clean
    npm rebuild --quiet
    npm install --quiet
fi
