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
# Include for Windows, operating under WSL
#

#  __     __         _       _     _
#  \ \   / /_ _ _ __(_) __ _| |__ | | ___ ___
#   \ \ / / _` | '__| |/ _` | '_ \| |/ _ \ __|
#    \ V / (_| | |  | | (_| | |_) | |  __\__ \
#     \_/ \__,_|_|  |_|\__,_|_.__/|_|\___|___/
#

# Get the sha for a file
sha = $(shell sha256sum $(1) | head -c 10)

# Get the sha of all files in a directory using wildcard in $(1)
sha_dir = $(shell sha256sum  $(1) | cut -d' ' -f1 | sha256sum | head -c 10 )

# Minikube executable
MINIKUBE ?= minikube.exe

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# port forward the agones controller.
# useful for pprof and stats viewing, etc
controller-portforward: PORT ?= 8080
controller-portforward:
	kubectl port-forward deployments/agones-controller -n agones-system $(PORT)

# portforward prometheus web ui
prometheus-portforward:
	kubectl port-forward deployments/prom-prometheus-server 9090 -n metrics

# portforward prometheus web ui
grafana-portforward:
	kubectl port-forward deployments/grafana 3000 -n metrics