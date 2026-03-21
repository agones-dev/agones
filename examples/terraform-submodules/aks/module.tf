// Copyright 2019 Google LLC All Rights Reserved.
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


// Run:
//  terraform apply [-var agones_version="1.17.0"]

terraform {
  required_version = ">= 1.0.0"
}

// Install latest version of agones
variable "agones_version" {
  default = ""
}

variable "client_id" {
  default = ""

}
variable "client_secret" {
  default = ""
}

variable "cluster_name" {
  default = "test-cluster"
}

variable "disk_size" {
  default = 30
}

variable "feature_gates" {
  default = ""
}

variable "log_level" {
  default = "info"
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

module "aks_cluster" {
  // ***************************************************************************************************
  // Update ?ref= to the agones release you are installing. For example, ?ref=release-1.17.0 corresponds
  // to Agones version 1.17.0
  // ***************************************************************************************************
  source = "git::https://github.com/agones-dev/agones.git//install/terraform/modules/aks/?ref=main"

  client_id               = var.client_id
  client_secret           = var.client_secret
  cluster_name            = var.cluster_name
  disk_size               = var.disk_size
  machine_type            = var.machine_type
  node_count              = var.node_count
  resource_group_location = var.resource_group_location
  resource_group_name     = var.resource_group_name
}

module "helm_agones" {
  // ***************************************************************************************************
  // Update ?ref= to the agones release you are installing. For example, ?ref=release-1.17.0 corresponds
  // to Agones version 1.17.0
  // ***************************************************************************************************
  source = "git::https://github.com/agones-dev/agones.git//install/terraform/modules/helm3/?ref=main"

  agones_version         = var.agones_version
  cluster_ca_certificate = module.aks_cluster.cluster_ca_certificate
  feature_gates          = var.feature_gates
  host                   = module.aks_cluster.host
  log_level              = var.log_level
  token                  = module.aks_cluster.token
  values_file            = ""
}

output "cluster_ca_certificate" {
  value = module.aks_cluster.cluster_ca_certificate
}

output "host" {
  value = module.aks_cluster.host
}

output "token" {
  value = module.aks_cluster.token
}
