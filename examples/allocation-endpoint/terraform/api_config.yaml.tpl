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

type: google.api.Service
config_version: 3

#
# Name of the Agones Allocation Service's configuration.
#
name: ${service-name}

#
# API title to appear in the user interface (Google Cloud Console).
#
title: Agones Allocation Endpoints gRPC API
apis:
- name: allocation.AllocationService

usage:
  rules:
  - selector: "*"
    allow_unregistered_calls: true

authentication:
  providers:
  - id: google_service_account
    audiences: ${service-name}
    issuer: ${service-account}
    jwks_uri: https://www.googleapis.com/robot/v1/metadata/x509/${service-account}
  rules:
  # This auth rule will apply to all methods.
  - selector: "*"
    requirements:
      - provider_id: google_service_account
