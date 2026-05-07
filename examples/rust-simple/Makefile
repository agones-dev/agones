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
# Makefile for building the world's simplest Rust game server
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
root_path := $(realpath $(project_path)/../..)
version := 0.14
ifeq ($(REPOSITORY),)
	server_tag := rust-simple-server:$(version)
else
	server_tag := $(REPOSITORY)/rust-simple-server:$(version)
endif

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# Build the example Gameserver
# alias for 'build-image'
build: build-image

# Build the server binary
build-server:
	cargo build --release

# Build a docker image for the server, and tag it
build-image:
	# Docker does not allow to copy outside files
	mkdir -p $(project_path)sdk
	cp -rf $(project_path)../../sdks/rust/* $(project_path)sdk
	docker build $(project_path) --tag=$(server_tag)

# Run Rust Gameserver binary
# Could be tested with:
# cd ../../build; make run-sdk-conformance-local TIMEOUT=120 TESTS=ready,watch,health,gameserver,shutdown
run:
	./target/release/rust-simple

# Run docker image
# Could be tested with:
# cd ../../build; make run-sdk-conformance-local TIMEOUT=120 TESTS=ready,watch,health,gameserver,shutdown
run-image:
	docker run --network=host $(PROD_REPO)/$(server_tag)

# When you used 'build' or 'build-server' locally without docker
clean:
	cargo clean
	rm -rf $(project_path)sdk

# If you are building with docker clean files in /sdk directory,
# clean leftover files after building an image with Docker
clean-docker:
	rm -rf $(project_path)sdk

# check if hosted on Google Artifact Registry
gar-check:
	gcloud container images describe $(PROD_REPO)/$(server_tag)

#output the tag for this image
echo-image-tag:
	@echo $(PROD_REPO)/$(server_tag)

# Push the docker image
push: build
	docker push $(server_tag)

# build and push the rust-simple image with specified tag
cloud-build:
	cd $(root_path) && gcloud builds submit --config=./examples/rust-simple/cloudbuild.yaml
