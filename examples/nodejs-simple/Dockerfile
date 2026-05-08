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

FROM debian:trixie
RUN useradd -u 1000 -m server
RUN apt-get update && apt-get install -y curl gnupg  && \
    apt-get clean

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

WORKDIR /home/server

COPY ./sdks/nodejs sdks/nodejs
RUN cd sdks/nodejs && \
    npm ci --production
COPY ./examples/nodejs-simple examples/nodejs-simple
RUN cd examples/nodejs-simple && \
    npm ci --production
RUN chown -R server /home/server
USER 1000

WORKDIR /home/server/examples/nodejs-simple
ENTRYPOINT ["npm", "start", "--"]
