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

output "cluster_ca_certificate" {
  value = base64decode(azurerm_kubernetes_cluster.agones.kube_config.0.cluster_ca_certificate)
  depends_on = [
    # Helm would be invoked only after all node pools would be created
    # This way taints and tolerations for Agones controller would work properly
    azurerm_kubernetes_cluster_node_pool.system,
    azurerm_kubernetes_cluster_node_pool.metrics
  ]
}

output "client_certificate" {
  value = azurerm_kubernetes_cluster.agones.kube_config.0.client_certificate
}

output "kube_config" {
  value = azurerm_kubernetes_cluster.agones.kube_config_raw
}

output "host" {
  value = azurerm_kubernetes_cluster.agones.kube_config.0.host
}

output "token" {
  value = azurerm_kubernetes_cluster.agones.kube_config.0.password
}
