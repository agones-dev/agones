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
# Makefile for building the SuperTuxKart server
#

#  __     __         _       _     _
#  \ \   / /_ _ _ __(_) __ _| |__ | | ___ ___
#   \ \ / / _` | '__| |/ _` | '_ \| |/ _ \ __|
#    \ V / (_| | |  | | (_| | |_) | |  __\__ \
#     \_/ \__,_|_|  |_|\__,_|_.__/|_|\___|___/
#

REPOSITORY ?=
PROD_REPO ?= us-docker.pkg.dev/agones-images/examples
MINIKUBE ?= minikube
MINIKUBE_PROFILE ?= agones

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
project_path := $(dir $(mkfile_path))
version := 0.25
ifeq ($(REPOSITORY),)
	image_tag := supertuxkart-example:$(version)
else
	image_tag := $(REPOSITORY)/supertuxkart-example:$(version)
endif

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# Build a docker image for the server, and tag it
build:
	docker build -f $(project_path)/Dockerfile --tag=$(image_tag) .

# Push the docker image
push: build
	docker push $(image_tag)

# Pushes the image (built via `make build`) into the "agones" minikube instance,
# and re-tags it with the production tag (i.e. $(PROD_REPO)/$(image_tag)), so
# that it can be used by the example gameserver.yaml and fleet.yaml without
# modification.
minikube-push:
	$(MINIKUBE) image load $(image_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image tag $(image_tag) $(PROD_REPO)/$(image_tag) -p $(MINIKUBE_PROFILE)

# Run tests
test:
	go test -v ./...

# check if hosted on Google Artifact Registry
gar-check:
	gcloud container images describe $(PROD_REPO)/$(image_tag)

#output the tag for this image
echo-image-tag:
	@echo $(PROD_REPO)/$(image_tag)

# build and push the supertuxkart image with specified tag
cloud-build:
	gcloud builds submit --config=cloudbuild.yaml