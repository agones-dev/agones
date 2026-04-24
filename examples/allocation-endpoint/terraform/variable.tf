// Copyright 2022 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

variable "project_id" {
  type        = string
  description = "The project ID."
}

variable "ae_proxy_image" {
  type        = string
  description = "The docker image of the allocation proxy."
  default     = "us-docker.pkg.dev/agones-images/examples/allocation-endpoint-proxy:0.16"
}

variable "region" {
  type        = string
  description = "The region."
  default     = "us-central1"
}

variable "authorized_members" {
  type        = list(string)
  description = "The list of the SAs/members that are authorized to call allocation endpoint."
}

variable "clusters_info" {
  type        = string
  description = "The list of allocation endpoints in the form of <endpoint>:<weight> in which weight is between 0 and 1."
}

variable "agones-namespace" {
  type        = string
  description = "The namespace that have the agones-allocator service with ESP container deployed to it."
  default     = "agones-system"
}

variable "workload-pool" {
  type        = string
  description = "The workload pool in the form of my-proj.svc.id.goog."
}
