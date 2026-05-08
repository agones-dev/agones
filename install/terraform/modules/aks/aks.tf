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
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 2.66"
    }
  }
}

provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "agones" {
  location = var.resource_group_location
  name     = var.resource_group_name
}

resource "azurerm_kubernetes_cluster" "agones" {
  name                = var.cluster_name
  location            = azurerm_resource_group.agones.location
  resource_group_name = azurerm_resource_group.agones.name
  # don't change dns_prefix as node pool Network Security Group name uses a hash of dns_prefix on on its name
  dns_prefix = "agones"

  kubernetes_version = var.kubernetes_version

  default_node_pool {
    name                  = "default"
    node_count            = var.node_count
    vm_size               = var.machine_type
    os_disk_size_gb       = var.disk_size
    enable_auto_scaling   = false
    enable_node_public_ip = var.enable_node_public_ip
  }

  service_principal {
    client_id     = var.client_id
    client_secret = var.client_secret
  }
  tags = {
    Environment = "Production"
  }
}

resource "azurerm_kubernetes_cluster_node_pool" "system" {
  name                  = "system"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.agones.id
  vm_size               = var.machine_type
  node_count            = 1
  os_disk_size_gb       = var.disk_size
  enable_auto_scaling   = false
  node_taints = [
    "agones.dev/agones-system=true:NoExecute"
  ]
  node_labels = {
    "agones.dev/agones-system" : "true"
  }
}

resource "azurerm_kubernetes_cluster_node_pool" "metrics" {
  name                  = "metrics"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.agones.id
  vm_size               = var.machine_type
  node_count            = 1
  os_disk_size_gb       = var.disk_size
  enable_auto_scaling   = false
  node_taints = [
    "agones.dev/agones-metrics=true:NoExecute"
  ]
  node_labels = {
    "agones.dev/agones-metrics" : "true"
  }
}

resource "azurerm_network_security_rule" "gameserver" {
  name                       = "gameserver"
  priority                   = 100
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "Udp"
  source_port_range          = "*"
  destination_port_range     = "7000-8000"
  source_address_prefix      = "*"
  destination_address_prefix = "*"
  resource_group_name        = azurerm_kubernetes_cluster.agones.node_resource_group
  # We don't use azurerm_resources datasource to get the security group as it's not reliable: random empty resource array
  # 55978144 are the first 8 characters of the fnv64a hash's UInt32 of master node's dns prefix ("agones")
  network_security_group_name = "aks-agentpool-55978144-nsg"

  depends_on = [
    azurerm_kubernetes_cluster.agones,
    azurerm_kubernetes_cluster_node_pool.metrics,
    azurerm_kubernetes_cluster_node_pool.system
  ]

  # Ignore resource_group_name changes because of random case returned by AKS Api (MC_* or mc_*)
  lifecycle {
    ignore_changes = [
      resource_group_name
    ]
  }
}
