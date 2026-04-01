# Copyright 2023 Google LLC All Rights Reserved.
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

# Set of GKE cluster parameters which defines its name, zone
# and primary node pool configuration.
# It is crucial to set valid ProjectID for "project".
variable "cluster" {
  description = "Set of GKE cluster parameters."
  type        = map(any)

  default = {
    "name"                          = "test-cluster"
    "project"                       = "agones"
    "location"                      = "us-west1"
    "network"                       = "default"
    "subnetwork"                    = ""
    "releaseChannel"                = "REGULAR"
    "kubernetesVersion"             = "1.34"
    "deletionProtection"            = true
    "maintenanceExclusionStartTime" = null
    "maintenanceExclusionEndTime"   = null
  }
}

# udpFirewall specifies whether to create a UDP firewall named
# `firewallName` with port range `ports`, source range `sourceRanges` 
variable "udpFirewall" {
  default = true
}

# Ports can be overriden using tfvars file
variable "ports" {
  default = "7000-8000"
}

# SourceRanges can be overriden using tfvars file
variable "sourceRanges" {
  default = "0.0.0.0/0"
}

variable "firewallName" {
  description = "name for the cluster firewall. Defaults to 'game-server-firewall-{local.name}' if not set."
  type        = string
  default     = ""
}
