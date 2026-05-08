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

# Add installation step required to generate your SDK.
# This should contains tooling required to generate GRPC code, build (if distribution is required) 
# and test your integration.

RUN apt-get --allow-releaseinfo-change update && \
    apt-get install -y wget

RUN apt-get update \
    && apt-get install -y apt-transport-https 

RUN wget -q https://packages.microsoft.com/config/debian/13/packages-microsoft-prod.deb -O packages-microsoft-prod.deb \
    && dpkg -i packages-microsoft-prod.deb \
    && rm packages-microsoft-prod.deb

RUN apt-get update \
    && apt-get install -y dotnet-sdk-8.0

# code generation scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh