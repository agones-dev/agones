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
//  terraform apply -var project="<YOUR_GCP_ProjectID>" [-var agones_version="1.17.0"]

terraform {
  required_version = ">= 1.0.0"
  required_providers {
    google = {
      source = "hashicorp/google"
      version = "~> 4.25.0"
    }
  }
}

variable "project" {
  default = ""
}

variable "name" {
  default = "agones-terraform-example"
}

// Install latest version of agones
variable "agones_version" {
  default = ""
}

variable "machine_type" {
  default = "e2-standard-4"
}

// Note: This is the number of gameserver nodes. The Agones module will automatically create an additional
// two node pools with 1 node each for "agones-system" and "agones-metrics".
variable "node_count" {
  default = "4"
}

variable "enable_image_streaming" {
  default = "true"
}

variable "location" {
  default     = "us-west1-c"
  description = "The GCP location to create the cluster in"
}

variable "zone" {
  default     = ""
  description = "The GCP zone to create the cluster in (deprecated, use `location`)"
}

variable "network" {
  default     = "default"
  description = "The name of the VPC network to attach the cluster and firewall rule to"
}

variable "subnetwork" {
  default     = ""
  description = "The subnetwork to host the cluster in. Required field if network value isn't 'default'."
}

variable "log_level" {
  default = "info"
}

variable "feature_gates" {
  default = ""
}

variable "windows_node_count" {
  default = "0"
}

variable "windows_machine_type" {
  default = "e2-standard-4"
}

variable "autoscale" {
  default = "false"
}

variable "min_node_count" {
  default = "1"
}

variable "max_node_count" {
  default = "5"
}

module "gke_cluster" {
  // ***************************************************************************************************
  // Update ?ref= to the agones release you are installing. For example, ?ref=release-1.17.0 corresponds
  // to Agones version 1.17.0
  // ***************************************************************************************************
  source = "git::https://github.com/agones-dev/agones.git//install/terraform/modules/gke/?ref=main"

  cluster = {
    "name"                    = var.name
    "location"		      = var.location
    "zone"                    = var.zone
    "machineType"             = var.machine_type
    "initialNodeCount"        = var.node_count
    "enableImageStreaming"    = var.enable_image_streaming
    "project"                 = var.project
    "network"                 = var.network
    "subnetwork"              = var.subnetwork
    "windowsInitialNodeCount" = var.windows_node_count
    "windowsMachineType"      = var.windows_machine_type
    "autoscale"		      = var.autoscale
    "mindNodeCount"	      = var.min_node_count
    "maxNodeCount"	      = var.max_node_count
  }
}

module "helm_agones" {
  // ***************************************************************************************************
  // Update ?ref= to the agones release you are installing. For example, ?ref=release-1.17.0 corresponds
  // to Agones version 1.17.0
  // ***************************************************************************************************
  source = "git::https://github.com/agones-dev/agones.git//install/terraform/modules/helm3/?ref=main"

  agones_version         = var.agones_version
  values_file            = ""
  feature_gates          = var.feature_gates
  host                   = module.gke_cluster.host
  token                  = module.gke_cluster.token
  cluster_ca_certificate = module.gke_cluster.cluster_ca_certificate
  log_level              = var.log_level
}

output "host" {
  value = module.gke_cluster.host
}
output "token" {
  value     = module.gke_cluster.token
  sensitive = true
}
output "cluster_ca_certificate" {
  value = module.gke_cluster.cluster_ca_certificate
}
