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

#Helm variables

variable "chart" {
  default = "agones"
}

variable "force_update" {
  default = "true"
}

variable "agones_version" {
  default = ""
}

variable "udp_expose" {
  default = "true"
}

variable "log_level" {
  default = "info"
}

variable "feature_gates" {
  default = ""
}

variable "host" {}

variable "token" {}

variable "cluster_ca_certificate" {}

variable "crd_cleanup" {
  default = "true"
}

variable "image_registry" {
  default = "us-docker.pkg.dev/agones-images/release"
}

variable "pull_policy" {
  default = "IfNotPresent"
}

variable "always_pull_sidecar" {
  default = "false"
}

variable "image_pull_secret" {
  default = ""
}

variable "ping_service_type" {
  default = "LoadBalancer"
}

variable "values_file" {
  default = ""
}

variable "gameserver_minPort" {
  default = "7000"
}

variable "gameserver_maxPort" {
  default = "8000"
}

variable "gameserver_namespaces" {
  default = ["default"]
  type    = list(string)
}

variable "load_balancer_ip" {
  default = ""
}

variable "set_values" {
  type = set(object({
    name  = string
    type  = string
    value = string
  }))
  default = []
}

variable "set_list_values" {
  type = set(object({
    name  = string
    value = list(string)
  }))
  default = []
}

variable "set_sensitive_values" {
  type = set(object({
    name  = string
    type  = string
    value = string
  }))
  default   = []
  sensitive = true
}