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

output "cluster_endpoints" {
  description = "Endpoints for the OKE cluster"
  value       = module.oke_cluster.cluster_endpoints
}

output "cluster_kubeconfig" {
  description = "OKE kebuconfig"
  value       = module.oke_cluster.cluster_kubeconfig
}

output "cluster_ca_cert" {
  description = "OKE cluster CA certificate"
  value       = module.oke_cluster.cluster_ca_cert
}

output "bastion_public_ip" {
  description = "Public IP address of bastion host"
  value       = module.oke_cluster.bastion_public_ip
}

output "operator_private_ip" {
  description = "Private IP address of operator host"
  value       = module.oke_cluster.operator_private_ip
}

output "ssh_to_bastion" {
  description = "SSH command for bastion host"
  value       = module.oke_cluster.ssh_to_bastion
}

output "ssh_to_operator" {
  description = "SSH command for operator host"
  value       = module.oke_cluster.ssh_to_operator
}
