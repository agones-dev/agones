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
# Include for managing the examples
#

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# test all example images exist on Google Artifact Registry (GAR)
test-examples-on-gar: example-image-test.allocation-endpoint
test-examples-on-gar: example-image-test.autoscaler-webhook
test-examples-on-gar: example-image-test.cpp-simple
test-examples-on-gar: example-image-test.nodejs-simple
test-examples-on-gar: example-image-test.rust-simple
test-examples-on-gar: example-image-test.unity-simple
test-examples-on-gar: example-image-test.xonotic
test-examples-on-gar: example-image-test.crd-client
test-examples-on-gar: example-image-test.supertuxkart
test-examples-on-gar: example-image-test.simple-game-server
test-examples-on-gar: example-image-test.simple-genai-server
test-examples-on-gar: example-image-test.custom-controller

push-example-golang-images: example-image-push.allocation-endpoint
push-example-golang-images: example-image-push.autoscaler-webhook
push-example-golang-images: example-image-push.crd-client
push-example-golang-images: example-image-push.simple-game-server
push-example-golang-images: example-image-push.simple-genai-server
push-example-golang-images: example-image-push.supertuxkart
push-example-golang-images: example-image-push.xonotic
push-example-golang-images: example-image-push.custom-controller

# Test to ensure the example image found in the % folder is on GAR. Fails if it is not.
example-image-test.%:
	$(DOCKER_RUN) bash -c "cd examples/$* && make gar-check"

example-image-push.%:
	$(DOCKER_RUN) bash -c "cd examples/$* && make push"

# Perform make build for golang examples
build-go-examples: build-example-allocation-endpoint build-example-autoscaler-webhook build-example-crd-client build-example-simple-game-server build-example-simple-genai-server build-example-supertuxkart build-example-xonotic build-example-custom-controller

# Perform make build for all examples
build-examples: build-example-allocation-endpoint build-example-autoscaler-webhook build-example-cpp-simple build-example-crd-client build-example-nodejs-simple build-example-rust-simple build-example-simple-game-server build-example-simple-genai-server build-example-supertuxkart build-example-xonotic

# Run "make build" command for one example directory
build-example:
	cd  $(examples_folder)/$(EXAMPLE); \
	if [ -f Makefile ] ; then \
		make build; \
	else \
		echo "Makefile was not found in "/examples/$(EXAMPLE)" directory - nothing to execute" ; \
	fi

build-example-allocation-endpoint:
	$(MAKE) build-example EXAMPLE=allocation-endpoint

build-example-autoscaler-webhook:
	$(MAKE) build-example EXAMPLE=autoscaler-webhook

build-example-cpp-simple:
	$(MAKE) build-example EXAMPLE=cpp-simple

build-example-crd-client:
	$(MAKE) build-example EXAMPLE=crd-client

build-example-nodejs-simple:
	$(MAKE) build-example EXAMPLE=nodejs-simple

build-example-rust-simple:
	$(MAKE) build-example EXAMPLE=rust-simple

build-example-simple-game-server:
	$(MAKE) build-example EXAMPLE=simple-game-server

build-example-simple-genai-server:
	$(MAKE) build-example EXAMPLE=simple-genai-server

build-example-supertuxkart:
	$(MAKE) build-example EXAMPLE=supertuxkart

build-example-xonotic:
	$(MAKE) build-example EXAMPLE=xonotic

build-example-custom-controller:
	$(MAKE) build-example EXAMPLE=custom-controller

# ensure a version increase when an example changes
check-example-versions: ensure-build-image
	docker run --rm $(common_mounts) --workdir=$(mount_path)/build/scripts/example-version-checker $(DOCKER_RUN_ARGS) $(build_tag) \
		go run .
