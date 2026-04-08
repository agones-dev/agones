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

variable "client_id" {
}

variable "client_secret" {
}

variable "cluster_name" {
  default = "test-cluster"
}

variable "disk_size" {
  default = 30
}

# VMSS is used, so it is unpredictable how NICs will be given to VMs
# So let Azure to create NICs with Public IPs as gameservers require
# Azure Managment SDK can be used to obtain these IPs and map Agones GameServers internal IPs to public
variable "enable_node_public_ip" {
  default = true
}

variable "kubernetes_version" {
  default = "1.35"
}

variable "machine_type" {
  default = "Standard_D2_v2"
}

variable "node_count" {
  default = 4
}

variable "resource_group_location" {
  default = "East US"
}

variable "resource_group_name" {
  default = "agonesRG"
}
