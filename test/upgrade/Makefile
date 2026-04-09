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
base_version = 1.58.0
# Version defaults to the short hash of the latest commit
VERSION ?= $(base_version)-dev-$(shell git rev-parse --short=7 HEAD)
server_tag := $(REGISTRY)/upgrade-test-controller:$(VERSION)
cwd:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# Using .ONESHELL allows us to `cd` to the parent directory agones. This gives the Dockerfile the
# context of the agones directory, which allows it to COPY files from any child directory.
.ONESHELL:
# Build a docker image for the server, and tag it
build:
	cd "$(cwd)/../.." && DOCKER_BUILDKIT=1 docker build -f $(cwd)/Dockerfile --tag=$(server_tag) .

push: build
	docker push $(server_tag)
