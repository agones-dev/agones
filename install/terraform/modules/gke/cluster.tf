# Copyright Contributors to Agones a Series of LF Projects, LLC.
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


terraform {
  required_version = ">= 1.0.0"
}

data "google_client_config" "default" {}

# A list of all parameters used in interpolation var.cluster
# Set values to default if not key was not set in original map
locals {
  project                       = lookup(var.cluster, "project", "agones")
  location                      = lookup(var.cluster, "location", "us-west1-c")
  zone                          = lookup(var.cluster, "zone", "")
  name                          = lookup(var.cluster, "name", "test-cluster")
  machineType                   = lookup(var.cluster, "machineType", "e2-standard-4")
  initialNodeCount              = lookup(var.cluster, "initialNodeCount", "4")
  enableImageStreaming          = lookup(var.cluster, "enableImageStreaming", true)
  network                       = lookup(var.cluster, "network", "default")
  subnetwork                    = lookup(var.cluster, "subnetwork", "")
  releaseChannel                = lookup(var.cluster, "releaseChannel", "UNSPECIFIED")
  kubernetesVersion             = lookup(var.cluster, "kubernetesVersion", "1.33")
  windowsInitialNodeCount       = lookup(var.cluster, "windowsInitialNodeCount", "0")
  windowsMachineType            = lookup(var.cluster, "windowsMachineType", "e2-standard-4")
  autoscale                     = lookup(var.cluster, "autoscale", false)
  workloadIdentity              = lookup(var.cluster, "workloadIdentity", false)
  minNodeCount                  = lookup(var.cluster, "minNodeCount", "1")
  maxNodeCount                  = lookup(var.cluster, "maxNodeCount", "5")
  maintenanceExclusionStartTime = lookup(var.cluster, "maintenanceExclusionStartTime", null)
  maintenanceExclusionEndTime   = lookup(var.cluster, "maintenanceExclusionEndTime", null)
}

data "google_container_engine_versions" "version" {
  project        = local.project
  provider       = google-beta
  location       = local.location
  version_prefix = format("%s.", local.kubernetesVersion)
}

# echo command used for debugging purpose
# Run `terraform taint null_resource.test-setting-variables` before second execution
resource "null_resource" "test-setting-variables" {
  provisioner "local-exec" {
    command = <<EOT
    ${format("echo Current variables set as following - name: %s, project: %s, machineType: %s, initialNodeCount: %s, network: %s, zone: %s, location: %s, windowsInitialNodeCount: %s, windowsMachineType: %s, releaseChannel: %s, kubernetesVersion: %s",
    local.name,
    local.project,
    local.machineType,
    local.initialNodeCount,
    local.network,
    local.zone,
    local.location,
    local.windowsInitialNodeCount,
    local.windowsMachineType,
    local.releaseChannel,
    local.kubernetesVersion,
)}
    EOT
}
}

resource "google_container_cluster" "primary" {
  name       = local.name
  location   = local.zone != "" ? local.zone : local.location
  project    = local.project
  network    = local.network
  subnetwork = local.subnetwork

  networking_mode = "VPC_NATIVE"
  ip_allocation_policy {}

  # https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/container_cluster#example-usage---with-a-separately-managed-node-pool-recommended
  remove_default_node_pool = true
  initial_node_count       = 1
  release_channel {
    channel = local.releaseChannel
  }

  min_master_version = local.kubernetesVersion

  dynamic "maintenance_policy" {
    for_each = (local.releaseChannel != "UNSPECIFIED" && local.maintenanceExclusionStartTime != null && local.maintenanceExclusionEndTime != null) ? [1] : []
    content {
      # When exclusions and maintenance windows overlap, exclusions have precedence.
      daily_maintenance_window {
        start_time = "03:00"
      }
      maintenance_exclusion {
        exclusion_name = format("%s-%s", local.name, "exclusion")
        start_time     = local.maintenanceExclusionStartTime
        end_time       = local.maintenanceExclusionEndTime
        exclusion_options {
          scope = "NO_MINOR_UPGRADES"
        }
      }
    }
  }

  dynamic "ip_allocation_policy" {
    for_each = tonumber(local.windowsInitialNodeCount) > 0 ? [1] : []
    content {
      # Enable Alias IPs to allow Windows Server networking.
      cluster_ipv4_cidr_block  = "/14"
      services_ipv4_cidr_block = "/20"
    }
  }
  dynamic "workload_identity_config" {
    for_each = local.workloadIdentity ? [1] : []
    content {
      workload_pool = "${local.project}.svc.id.goog"
    }
  }
  timeouts {
    create = "30m"
    update = "40m"
  }
}

# create a nodepool for the above cluster named "default"
resource "google_container_node_pool" "default" {
  name       = "default"
  cluster    = google_container_cluster.primary.id
  node_count = local.autoscale ? null : local.initialNodeCount
  version    = local.releaseChannel == "UNSPECIFIED" ? data.google_container_engine_versions.version.latest_node_version : data.google_container_engine_versions.version.release_channel_latest_version[local.releaseChannel]

  dynamic "autoscaling" {
    for_each = local.autoscale ? [1] : []
    content {
      min_node_count = local.minNodeCount
      max_node_count = local.maxNodeCount
    }
  }

  management {
    auto_upgrade = local.releaseChannel == "UNSPECIFIED" ? false : true
  }

  node_config {
    machine_type = local.machineType

    oauth_scopes = [
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
      "https://www.googleapis.com/auth/service.management.readonly",
      "https://www.googleapis.com/auth/servicecontrol",
      "https://www.googleapis.com/auth/trace.append",
    ]

    tags = ["game-server"]

    gcfs_config {
      enabled = local.enableImageStreaming
    }
  }
}

# create agones-system nodepool
resource "google_container_node_pool" "agones-system" {
  name       = "agones-system"
  cluster    = google_container_cluster.primary.id
  node_count = 1
  version    = local.releaseChannel == "UNSPECIFIED" ? data.google_container_engine_versions.version.latest_node_version : data.google_container_engine_versions.version.release_channel_latest_version[local.releaseChannel]

  management {
    auto_upgrade = local.releaseChannel == "UNSPECIFIED" ? false : true
  }

  node_config {
    machine_type = "e2-standard-4"

    oauth_scopes = [
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
      "https://www.googleapis.com/auth/service.management.readonly",
      "https://www.googleapis.com/auth/servicecontrol",
      "https://www.googleapis.com/auth/trace.append",
    ]

    labels = {
      "agones.dev/agones-system" = "true"
    }

    taint {
      key    = "agones.dev/agones-system"
      value  = "true"
      effect = "NO_EXECUTE"
    }

    gcfs_config {
      enabled = true
    }
  }
}

resource "google_container_node_pool" "agones-metrics" {
  count      = var.enable_agones_metrics_nodepool ? 1 : 0
  name       = "agones-metrics"
  cluster    = google_container_cluster.primary.id
  node_count = 1
  version    = local.releaseChannel == "UNSPECIFIED" ? data.google_container_engine_versions.version.latest_node_version : data.google_container_engine_versions.version.release_channel_latest_version[local.releaseChannel]

  management {
    auto_upgrade = local.releaseChannel == "UNSPECIFIED" ? false : true
  }

  node_config {
    machine_type = "e2-standard-4"

    oauth_scopes = [
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
      "https://www.googleapis.com/auth/service.management.readonly",
      "https://www.googleapis.com/auth/servicecontrol",
      "https://www.googleapis.com/auth/trace.append",
    ]

    labels = {
      "agones.dev/agones-metrics" = "true"
    }

    taint {
      key    = "agones.dev/agones-metrics"
      value  = "true"
      effect = "NO_EXECUTE"
    }

    gcfs_config {
      enabled = true
    }
  }
}

resource "google_container_node_pool" "windows" {
  count = tonumber(local.windowsInitialNodeCount) > 0 ? 1 : 0

  name       = "windows"
  cluster    = google_container_cluster.primary.id
  node_count = local.windowsInitialNodeCount
  version    = local.releaseChannel == "UNSPECIFIED" ? data.google_container_engine_versions.version.latest_node_version : data.google_container_engine_versions.version.release_channel_latest_version[local.releaseChannel]

  management {
    auto_upgrade = local.releaseChannel == "UNSPECIFIED" ? false : true
  }

  node_config {
    image_type   = "WINDOWS_LTSC_CONTAINERD"
    machine_type = local.windowsMachineType

    oauth_scopes = [
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
      "https://www.googleapis.com/auth/service.management.readonly",
      "https://www.googleapis.com/auth/servicecontrol",
      "https://www.googleapis.com/auth/trace.append",
    ]

    tags = ["game-server"]
  }
}

# create firewall rule for the cluster

resource "google_compute_firewall" "default" {
  count   = var.udpFirewall ? 1 : 0
  name    = length(var.firewallName) == 0 ? "game-server-firewall-${local.name}" : var.firewallName
  project = local.project
  network = local.network

  allow {
    protocol = "udp"
    ports    = [var.ports]
  }

  target_tags   = ["game-server"]
  source_ranges = [var.sourceRanges]
}
