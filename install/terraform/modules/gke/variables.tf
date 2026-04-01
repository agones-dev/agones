# Copyright 2019 Google LLC All Rights Reserved.
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
    "location"                      = "us-west1-c"
    "name"                          = "test-cluster"
    "machineType"                   = "e2-standard-4"
    "initialNodeCount"              = "4"
    "project"                       = "agones"
    "network"                       = "default"
    "subnetwork"                    = ""
    "releaseChannel"                = "UNSPECIFIED"
    "kubernetesVersion"             = "1.34"
    "windowsInitialNodeCount"       = "0"
    "windowsMachineType"            = "e2-standard-4"
    "autoscale"                     = false
    "workloadIdentity"              = false
    "minNodeCount"                  = "1"
    "maxNodeCount"                  = "5"
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

# enable_agones_metrics_nodepool specifies whether to enable agones-metrics node pool
# By default it is disabled
variable "enable_agones_metrics_nodepool" {
  description = "enable or disable the creation of agones-metrics node pool."
  default     = false
}
