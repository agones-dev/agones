// Copyright Contributors to Agones a Series of LF Projects, LLC.
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
      source  = "hashicorp/google"
      version = "~> 4.25.0"
    }
  }
}

/////////////////////
// Cluster parameters

variable "name" {
  default = "agones-tf-cluster"
}

variable "project" {
  default = ""
}

variable "location" {
  default     = "us-west1"
  description = "The GCP location to create the cluster in"
}

variable "network" {
  default     = "default"
  description = "The name of the VPC network to attach the cluster and firewall rule to"
}

////////////////////
// Agones parameters

// Install latest version of agones
variable "agones_version" {
  default = ""
}

variable "values_file" {
  default = "../../../install/helm/agones/values.yaml"
}

variable "chart" {
  default = "agones"
}

variable "crd_cleanup" {
  default = "true"
}

variable "ping_service_type" {
  default = "LoadBalancer"
}

variable "pull_policy" {
  default = "Always"
}

variable "image_registry" {
  default = "us-docker.pkg.dev/agones-images/release"
}

variable "always_pull_sidecar" {
  default = "true"
}

variable "image_pull_secret" {
  default = ""
}

variable "log_level" {
  default = "info"
}

variable "feature_gates" {
  default = ""
}

module "gke_autopilot_cluster" {
  source = "../../../install/terraform/modules/gke-autopilot"

  cluster = {
    "name"     = var.name
    "project"  = var.project
    "location" = var.location
    "network"  = var.network
  }
}

module "helm_agones" {
  source = "../../../install/terraform/modules/helm3"

  agones_version         = var.agones_version
  values_file            = var.values_file
  chart                  = var.chart
  feature_gates          = var.feature_gates
  host                   = module.gke_autopilot_cluster.host
  token                  = module.gke_autopilot_cluster.token
  cluster_ca_certificate = module.gke_autopilot_cluster.cluster_ca_certificate
  image_registry         = var.image_registry
  image_pull_secret      = var.image_pull_secret
  crd_cleanup            = var.crd_cleanup
  ping_service_type      = var.ping_service_type
  log_level              = var.log_level
}

output "host" {
  value = module.gke_autopilot_cluster.host
}
output "token" {
  value     = module.gke_autopilot_cluster.token
  sensitive = true
}
output "cluster_ca_certificate" {
  value = module.gke_autopilot_cluster.cluster_ca_certificate
}
