# Copyright 2024 Google LLC All Rights Reserved.
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
# Makefile for building a test game-server
#

#  __     __         _       _     _
#  \ \   / /_ _ _ __(_) __ _| |__ | | ___ ___
#   \ \ / / _` | '__| |/ _` | '_ \| |/ _ \ __|
#    \ V / (_| | |  | | (_| | |_) | |  __\__ \
#     \_/ \__,_|_|  |_|\__,_|_.__/|_|\___|___/
#

REGISTRY ?=
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
project_path := $(dir $(mkfile_path))
root_path = $(realpath $(project_path)/)
# Because go mod init in the Dockerfile installs the most recently released version of Agones, this
# will need to be built and pushed post-release. During DEV it will be built at DEV - 1.
release_version = 1.57.0
server_tag := $(REGISTRY)/sdk-client-test:$(release_version)

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# Build a docker image for the server, and tag it
build:
	cd $(root_path) && docker build -f $(project_path)Dockerfile --tag=$(server_tag) .

push: build
	docker push $(server_tag)

# build and push the sdk-client-test image with specified tag
cloud-build:
	cd $(root_path) && gcloud builds submit --config=cloudbuild.yaml
