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

// If you are getting `Error: googleapi: Error 409: Your previous request to create the named bucket
// succeeded and you already own it., conflict` this means that your local tfstate file has
// divereged from the tfstate file in Google Cloud Storage (GCS). To use the GCS version of the
// tfstate, delete your local .terraform and .tfstate files. You may need to run
// `sudo chown -R yourusername .` to be able to delete them. Then navigate to this directory and run
// `terraform init`. Pull in the tfstate file from gcloud with
// `terraform import google_storage_bucket.default "<YOUR_GCP_ProjectID>"-e2e-infra-bucket-tfstate`.

// # GCS bucket for holding the Terraform state of the e2e Terraform config.

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
  name                        = "${var.project}-e2e-infra-bucket-tfstate"
  force_destroy               = false
  uniform_bucket_level_access = true
  location                    = "US"
  storage_class               = "STANDARD"
  versioning {
    enabled = true
  }
}
