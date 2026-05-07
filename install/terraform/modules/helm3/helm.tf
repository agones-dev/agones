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

#
# This is a Helm 3.x module, and is the module that should be actively used.
#
terraform {
  required_version = ">= 1.0.0"
  required_providers {
    helm = {
      version = "~> 2.3"
      source  = "hashicorp/helm"
    }
  }
}

provider "helm" {
  kubernetes {
    host                   = var.host
    token                  = var.token
    cluster_ca_certificate = var.cluster_ca_certificate
  }
}

resource "helm_release" "agones" {
  name             = "agones"
  repository       = "https://agones.dev/chart/stable"
  force_update     = var.force_update
  chart            = var.chart
  timeout          = 420
  version          = var.agones_version
  namespace        = "agones-system"
  create_namespace = true

  # Use terraform of the latest >=1.0.0 version
  values = [
    length(var.values_file) == 0 ? "" : file(var.values_file),
  ]

  dynamic "set" {
    for_each = tolist(var.set_values)
    iterator = set_item

    content {
      name  = set_item.value.name
      type  = set_item.value.type
      value = set_item.value.value
    }
  }

  dynamic "set_list" {
    for_each = tolist(var.set_list_values)
    iterator = set_item

    content {
      name  = set_item.value.name
      value = set_item.value.value
    }
  }

  // Due to a Terraform limitation sensitive values can't be iterated over.
  // See https://github.com/hashicorp/terraform/issues/29744 
  dynamic "set_sensitive" {
    for_each = tolist(nonsensitive(var.set_sensitive_values))
    iterator = set_item

    content {
      name  = set_item.value.name
      type  = set_item.value.type
      value = sensitive(set_item.value.value)
    }
  }

  set {
    name  = "agones.crds.CleanupOnDelete"
    value = var.crd_cleanup
  }

  set {
    name  = "agones.image.registry"
    value = var.image_registry
  }

  set {
    name  = "agones.image.controller.pullPolicy"
    value = var.pull_policy
  }

  set {
    name  = "agones.image.sdk.alwaysPull"
    value = var.always_pull_sidecar
  }

  set {
    name  = "agones.image.controller.pullSecret"
    value = var.image_pull_secret
  }

  set {
    name  = "agones.ping.http.serviceType"
    value = var.ping_service_type
  }

  set {
    name  = "agones.ping.udp.expose"
    value = var.udp_expose
  }

  set {
    name  = "agones.ping.udp.serviceType"
    value = var.ping_service_type
  }

  set {
    name  = "agones.controller.logLevel"
    value = var.log_level
  }

  set {
    name  = "agones.featureGates"
    value = var.feature_gates
  }

  set {
    name  = "gameservers.namespaces"
    value = "{${join(",", var.gameserver_namespaces)}}"
  }

  set {
    name  = "gameservers.minPort"
    value = var.gameserver_minPort
  }

  set {
    name  = "gameservers.maxPort"
    value = var.gameserver_maxPort
  }

  set {
    name  = "agones.allocator.service.loadBalancerIP"
    value = var.load_balancer_ip
  }
}
