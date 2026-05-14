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
ARG BASE_IMAGE=agones-build-sdk-base:latest
FROM $BASE_IMAGE

## --allow-releaseinfo-change is a workaround for the new debian release
## Can be removed when upgrading to bullseye
RUN apt-get --allow-releaseinfo-change update && apt-get install -y curl && apt-get install -y && \
    apt-get install -y gnupg && apt-get clean

# Import the Nodesource GPG key
RUN apt-get update && \
    apt-get install -y ca-certificates curl gnupg && \
    mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg

# Create deb repository for a specific Node major version
ARG NODE_MAJOR=20
RUN echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_$NODE_MAJOR.x nodistro main" > /etc/apt/sources.list.d/nodesource.list

# Update and install Node.js
RUN apt-get update && apt-get install -y nodejs

# Installing request is to address a bug with node-pre-gyp
RUN npm install --unsafe-perm --global request@2.88.2 grpc-tools@1.11.3

# code generation scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh