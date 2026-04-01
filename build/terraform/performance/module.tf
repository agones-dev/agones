// Copyright 2023 Google LLC All Rights Reserved.
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
//  terraform init -backend-config="bucket=<YOUR_GCP_ProjectID>-performance-infra-bucket-tfstate" -backend-config="prefix=terraform/state"
//  terraform apply -var project="<YOUR_GCP_ProjectID>"
// The performance test cluster is hosted in project `agones-images`, i.e:
//  terraform init -backend-config="bucket=agones-images-performance-infra-bucket-tfstate" -backend-config="prefix=terraform/state"
//  terraform apply -var project="agones-images"

terraform {
  required_version = ">= 1.0.0"
  required_providers {
    google = {
      source = "hashicorp/google"
      version = "~> 4.25.0"
    }
    helm = {
      source = "hashicorp/helm"
      version = "~> 2.3"
    }
  }
  backend "gcs" {
  }
}

variable "project" {}
variable "kubernetes_versions" {
  description = "Create performance test clusters with these k8s versions in these regions"
  type        = map(list(string))
  default     = {
    "1.34" = ["us-central1", "RAPID"]
  }
}

module "gke_standard_cluster" {
  for_each = var.kubernetes_versions
  source = "../e2e/gke-standard"
  project = var.project
  kubernetesVersion = each.key
  location = each.value[0]
  overrideName = format("standard-performance-test-cluster-%s", replace(each.key, ".", "-"))
  releaseChannel = each.value[1]
  machineType = "e2-standard-4"
  initialNodeCount = 200
}

resource "google_compute_firewall" "udp" {
  name    = "gke-game-server-firewall"
  project = var.project
  network = "default"

  allow {
    protocol = "udp"
    ports    = ["7000-8000"]
  }

  target_tags = ["game-server"]
  source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "tcp" {
  name    = "gke-game-server-firewall-tcp"
  project = var.project
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["7000-8000"]
  }

  target_tags = ["game-server"]
  source_ranges = ["0.0.0.0/0"]
}
