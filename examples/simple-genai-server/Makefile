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
# Makefile for building a simple-genai-game-server
#

#  __     __         _       _     _
#  \ \   / /_ _ _ __(_) __ _| |__ | | ___ ___
#   \ \ / / _` | '__| |/ _` | '_ \| |/ _ \ __|
#    \ V / (_| | |  | | (_| | |_) | |  __\__ \
#     \_/ \__,_|_|  |_|\__,_|_.__/|_|\___|___/
#
REPOSITORY ?=
PROD_REPO ?= us-docker.pkg.dev/agones-images/examples

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
project_path := $(dir $(mkfile_path))
version := 0.11
ifeq ($(REPOSITORY),)
	server_tag := simple-genai-game-server:$(version)
else
	server_tag := $(REPOSITORY)/simple-genai-game-server:$(version)
endif
server_tag_linux_amd64 = $(server_tag)-linux-amd64
push_server_manifest = $(server_tag_linux_amd64)
root_path = $(realpath $(project_path)/../..)

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

build: build-linux-image-amd64
push: push-linux-image-amd64
  	# Docker >=24.0.6
	docker buildx imagetools create --tag $(server_tag) \
    	$(push_server_manifest)

# Build a docker image for the server, and tag it
build-linux-image-amd64:
	cd $(root_path) && docker build -f $(project_path)Dockerfile --tag=$(server_tag_linux_amd64) .

push-linux-image-amd64: build
	docker push $(server_tag_linux_amd64)

# check if hosted on Google Artifact Registry
gar-check:
	gcloud container images describe $(PROD_REPO)/$(server_tag)

print-vars:
	$(info project_path is $(project_path))
	$(info server_tag is $(server_tag))
	$(info server_tag_linux_amd64 is $(server_tag_linux_amd64))
	$(info push_server_manifest is $(push_server_manifest))

# build and push the simple-genai-server image with specified tag
cloud-build:
	cd $(root_path) && gcloud builds submit --config=./examples/simple-genai-server/cloudbuild.yaml