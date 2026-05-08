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
//  terraform apply -var "project_id=<project id>" -var "authorized_members=[\"serviceAccount:<service-account-email>\"]" -var "clusters_info=[{\"name\":\"cluster1\",\"endpoint\":\"<agones-allocator-ip>\",\"namespace\":\"default\",\"allocation_weight\":100}]"  -var "workload-pool=<gke-project-id>.svc.id.goog"

locals {
  aep_endpoints_name = "agones-allocation.endpoints.${var.project_id}.cloud.goog"
}

provider "google" {
  project     = var.project_id
  region      = var.region
}

data "template_file" "api_config" {
  template = file("api_config.yaml.tpl")

  vars = {
    service-name    = local.aep_endpoints_name
    service-account = google_service_account.ae_sa.email
  }
}

resource "google_endpoints_service" "endpoints_service" {
  service_name         = local.aep_endpoints_name
  grpc_config          = data.template_file.api_config.rendered
  protoc_output_base64 = filebase64("agones_allocation_api_descriptor.pb")
}

resource "google_endpoints_service_iam_binding" "endpoints_service_binding" {
  service_name = google_endpoints_service.endpoints_service.service_name
  role         = "roles/servicemanagement.serviceController"
  members = [
    "serviceAccount:ae-esp-sa@${var.project_id}.iam.gserviceaccount.com",
  ]
  depends_on = [google_project_service.allocator-service]
}

resource "google_service_account_iam_binding" "workload-identity-binding" {
  service_account_id = google_service_account.ae_sa.name
  role = "roles/iam.workloadIdentityUser"

  members = [
    "serviceAccount:${var.workload-pool}[${var.agones-namespace}/agones-allocator]",
  ]
}

resource "google_service_account" "ae_sa" {
  account_id   = "ae-esp-sa"
  display_name = "Service Account for Allocation Endpoint"
}

resource "google_service_account_key" "ae_sa_key" {
  service_account_id = google_service_account.ae_sa.name
}

resource "google_cloud_run_service_iam_binding" "binding" {
  service  = google_cloud_run_service.aep_cloud_run.name
  project = google_cloud_run_service.aep_cloud_run.project
  location = google_cloud_run_service.aep_cloud_run.location
  role     = "roles/run.invoker"
  members  = var.authorized_members
}


resource "google_cloud_run_service" "aep_cloud_run" {
  project = var.project_id
  name     = "allocation-endpoint-proxy"
  location = var.region

  template {
    spec {
      container_concurrency = 80
      timeout_seconds       = 30
      containers {
        image = var.ae_proxy_image
        env {
          name  = "CLUSTERS_INFO"
          value = var.clusters_info
        }
        env {
          name  = "AUDIENCE"
          value = local.aep_endpoints_name
        }
        env {
          name  = "SA_KEY"
          value_from {
            secret_key_ref {
              name = google_secret_manager_secret.ae-sa-key.secret_id
              key = "latest"
            }
          }
        }
        ports {
          container_port = 8080
          # this enables the http/2 support. h2c: https://cloud.google.com/run/docs/configuring/http2
          name = "h2c"
        }
        resources {
          limits = {
            "cpu"    = "2000m"
            "memory" = "256Mi"
          }
        }
      }
    }
    metadata {
      annotations = {
        "autoscaling.knative.dev/maxScale" = "1000"
        "autoscaling.knative.dev/minScale" = "0"
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }

  metadata {
    annotations = {
      "run.googleapis.com/ingress"     = "all"
      "run.googleapis.com/client-name" = "terraform"
    }
  }

  lifecycle {
    ignore_changes = [
      # Ignore changes for the values set by GCP
      metadata[0].annotations,
      # This is currently not working and the fix is available in TF 0.14
      # https://github.com/hashicorp/terraform/pull/27141
      template[0].metadata[0].annotations["run.googleapis.com/sandbox"],
    ]
  }

  depends_on = [
    google_secret_manager_secret_version.ae-sa-key-secret,
    google_secret_manager_secret_iam_member.secret-access,
    google_project_service.run,
  ]
}

resource "google_secret_manager_secret" "ae-sa-key" {
  secret_id = "ae-sa-key"
  
  replication {
    automatic = true
  }
  depends_on = [google_project_service.secretmanager]
}

resource "google_secret_manager_secret_version" "ae-sa-key-secret" {
  secret      = google_secret_manager_secret.ae-sa-key.id
  secret_data = base64decode(google_service_account_key.ae_sa_key.private_key)
}

resource "google_secret_manager_secret_iam_member" "secret-access" {
  secret_id = google_secret_manager_secret.ae-sa-key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${data.google_project.project.number}-compute@developer.gserviceaccount.com"
  depends_on = [google_secret_manager_secret.ae-sa-key]
}

// get project details
data "google_project" "project" {
}

# Enables the Secret Manager API
resource "google_project_service" "secretmanager" {
  service  = "secretmanager.googleapis.com"
}

# Enables the Service Control API
resource "google_project_service" "servicecontrol" {
  service  = "servicecontrol.googleapis.com"
}

# Enables the Cloud Run API
resource "google_project_service" "run" {
  service = "run.googleapis.com"
}

resource "google_project_service" "allocator-service" {
  service                    = google_endpoints_service.endpoints_service.id
  disable_dependent_services = true
}
