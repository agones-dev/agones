// Copyright 2020 Google LLC All Rights Reserved.
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
//  terraform apply [-var agones_version="1.17.0"]

terraform {
  required_version = ">= 1.0.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}

// Install latest version of agones
variable "agones_version" {
  default = ""
}

variable "cluster_name" {
  default = "agones-cluster"
}

variable "region" {
  default = "us-west-2"
}

variable "node_count" {
  default = "4"
}

provider "aws" {
  region = var.region
}

variable "machine_type" { default = "t2.large" }

variable "log_level" {
  default = "info"
}

variable "feature_gates" {
  default = ""
}

module "eks_cluster" {
  // ***************************************************************************************************
  // Update ?ref= to the agones release you are installing. For example, ?ref=release-1.17.0 corresponds
  // to Agones version 1.17.0
  // ***************************************************************************************************
  source = "git::https://github.com/agones-dev/agones.git//install/terraform/modules/eks/?ref=main"

  machine_type = var.machine_type
  cluster_name = var.cluster_name
  node_count   = var.node_count
  region       = var.region
}

data "aws_eks_cluster_auth" "example" {
  name = var.cluster_name
}

// Next Helm module cause "terraform destroy" timeout, unless helm release would be deleted first.
// Therefore "helm delete --purge agones" should be executed from the CLI before executing "terraform destroy".
module "helm_agones" {
  // ***************************************************************************************************
  // Update ?ref= to the agones release you are installing. For example, ?ref=release-1.17.0 corresponds
  // to Agones version 1.17.0
  // ***************************************************************************************************
  source = "git::https://github.com/agones-dev/agones.git//install/terraform/modules/helm3/?ref=main"

  udp_expose             = "false"
  agones_version         = var.agones_version
  values_file            = ""
  feature_gates          = var.feature_gates
  host                   = module.eks_cluster.host
  token                  = data.aws_eks_cluster_auth.example.token
  cluster_ca_certificate = module.eks_cluster.cluster_ca_certificate
  log_level              = var.log_level
}

output "host" {
  value = "${module.eks_cluster.host}"
}
output "cluster_ca_certificate" {
  value = "${module.eks_cluster.cluster_ca_certificate}"
}
