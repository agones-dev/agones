// Copyright 2024 Google LLC All Rights Reserved.
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

# OCI Provider parameters
variable "api_fingerprint" {
  description = "Fingerprint of the API private key to use with OCI API."
  default     = ""
  type        = string
}

variable "api_private_key_path" {
  description = "The path to the OCI API private key."
  default     = ""
  type        = string
}

variable "region" {
  description = "The tenancy's region. Use the short form in lower case e.g. phoenix"
  default     = "ap-singapore-1"
  type        = string
}

variable "home_region" {
  description = "The tenancy's home region. Use the short form in lower case e.g. phoenix"
  default     = "ap-singapore-1"
  type        = string
}

variable "tenancy_id" {
  description = "The tenancy if of the OCI Cloud Account in which to create the resources."
  type        = string
}

variable "user_id" {
  description = "The id of the user that Terraform will use to create the resources."
  default     = ""
  type        = string
}


# General OCI parameters
variable "compartment_id" {
  description = "The compartment id where to create all resources."
  type        = string
}

# ssh keys
variable "ssh_private_key_path" {
  description = "The path to ssh private key."
  default     = "none"
  type        = string
}

variable "ssh_public_key_path" {
  description = "The path to ssh public key."
  default     = "none"
  type        = string
}

# Cluster
variable "kubernetes_version" {
  description = "The version of Kubernetes to use."
  default     = "v1.34.1"
  type        = string
}

variable "cluster_type" {
  description = "Whether to use basic or enhanced OKE clusters."
  default     = "basic"
  type        = string

  validation {
    condition     = contains(["basic", "enhanced"], lower(var.cluster_type))
    error_message = "Accepted values are 'basic' or 'enhanced'."
  }
}

variable "oke_control_plane" {
  description = "Whether to keep all OKE control planes public or private."
  default     = "public"
  type        = string

  validation {
    condition     = contains(["public", "private"], lower(var.oke_control_plane))
    error_message = "Accepted values are 'public' or 'private'."
  }
}

variable "preferred_cni" {
  description = "Whether to use flannel or NPN"
  default     = "flannel"
  type        = string

  validation {
    condition     = contains(["flannel", "npn"], lower(var.preferred_cni))
    error_message = "Accepted values are 'flannel' or 'npn'."
  }
}

variable "nodepools" {
  description = "Node pools for cluster"
  type        = any
  default = {
    np1 = {
      shape            = "VM.Standard.E4.Flex",
      ocpus            = 2,
      memory           = 32,
      size             = 3,
      boot_volume_size = 150,
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

variable "node_count" {
  default = "3"
}

variable "log_level" {
  default = "info"
}

variable "feature_gates" {
  default = ""
}
