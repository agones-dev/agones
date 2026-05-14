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
//  terraform apply -var project="<YOUR_GCP_ProjectID>"

terraform {
  required_version = ">= 1.0.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 4.25.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.3"
    }
  }
}

variable "project" {}
variable "kubernetesVersion" {}
variable "location" {}
variable "releaseChannel" {}

module "gke_cluster" {
  source = "../../../../install/terraform/modules/gke-autopilot"

  cluster = {
    "name"                          = format("gke-autopilot-e2e-test-cluster-%s", replace(var.kubernetesVersion, ".", "-"))
    "project"                       = var.project
    "location"                      = var.location
    "releaseChannel"                = var.releaseChannel
    "kubernetesVersion"             = var.kubernetesVersion
    "deletionProtection"            = false
    "maintenanceExclusionStartTime" = timestamp()
    "maintenanceExclusionEndTime"   = timeadd(timestamp(), "2640h") # 110 days
  }

  udpFirewall = false // firewall is created at the project module level
}
