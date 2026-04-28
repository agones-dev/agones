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
GO111MODULE=off
DIR=/go/src/agones.dev/agones/test/sdk/cpp/
echo "$DIR"/sdk

# Copy all CPP SDK files into a new directory
if [[ ! -d $DIR/sdk ]]
then
    mkdir -p "$DIR"/sdk/.build
    cp -r /go/src/agones.dev/agones/sdks/cpp/* $DIR/sdk
    cd $DIR/sdk/.build
    cmake .. -DCMAKE_BUILD_TYPE=Release -DAGONES_SILENT_OUTPUT=OFF \
      -DCMAKE_INSTALL_PREFIX=$DIR/sdk/.build -G "Unix Makefiles" -Wno-dev
else
    echo "Directory with cpp SDK third party dependencies \
has already built - using cached version. \
Use make clean-sdk-conformance-tests if you want to start from scratch"
fi
cd $DIR/sdk/.build
cmake --build .  --target install -j$(nproc)
cd $DIR && mkdir -p .build && cd .build
cmake .. -G "Unix Makefiles" \
   -DCMAKE_PREFIX_PATH=$DIR/sdk/.build \
   -Dagones_DIR=$DIR/sdk/.build/agones/cmake \
   -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=.bin
cmake --build . --target install -j$(nproc)
