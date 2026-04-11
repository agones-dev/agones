# Copyright 2022 Google LLC All Rights Reserved.
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
# Makefile for building a multi-cluster allocation proxy
#

#  __     __         _       _     _
#  \ \   / /_ _ _ __(_) __ _| |__ | | ___ ___
#   \ \ / / _` | '__| |/ _` | '_ \| |/ _ \ __|
#    \ V / (_| | |  | | (_| | |_) | |  __\__ \
#     \_/ \__,_|_|  |_|\__,_|_.__/|_|\___|___/
#

REPOSITORY ?=
PROD_REPO ?= us-docker.pkg.dev/agones-images/examples

# Directory that this Makefile is in.
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
project_path := $(dir $(mkfile_path))
root_path = $(realpath $(project_path)/../..)
version := 0.16
ifeq ($(REPOSITORY),)
	server_tag := allocation-endpoint-proxy:$(version)
else
	server_tag := $(REPOSITORY)/allocation-endpoint-proxy:$(version)
endif

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# build the allocation endpoint proxy binary
build:
	cd $(project_path)/server/ && docker build --tag=$(server_tag) .

run:
	docker run --network=host $(PROD_REPO)/$(server_tag) $(args)

# check if hosted on Google Artifact Registry
gar-check:
	gcloud container images describe $(PROD_REPO)/$(server_tag)

#output the tag for this image
echo-image-tag:
	@echo $(PROD_REPO)/$(server_tag)

# Push the docker image
push: build
	docker push $(server_tag)

# build and push the allocation-endpoint image with specified tag
cloud-build:
	gcloud builds submit --config=cloudbuild.yaml
