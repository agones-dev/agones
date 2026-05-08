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

// The performance test cluster is hosted in project `agones-e2e-1`, i.e:
//  terraform apply -var project="agones-e2e-1"

// # GCS bucket for holding the Terraform state of the performance test Terraform config.

terraform {
  required_version = ">= 1.0.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 4.25.0"
    }
  }
}

variable "project" {}

resource "google_storage_bucket" "default" {
  project                     = var.project
  name                        = "${var.project}-performance-infra-bucket-tfstate"
  force_destroy               = false
  uniform_bucket_level_access = true
  location                    = "US"
  storage_class               = "STANDARD"
  versioning {
    enabled = true
  }
}