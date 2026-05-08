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

#   ____        _ _     _   ___
#  | __ ) _   _(_) | __| | |_ _|_ __ ___   __ _  __ _  ___
#  |  _ \| | | | | |/ _` |  | || '_ ` _ \ / _` |/ _` |/ _ \
#  | |_) | |_| | | | (_| |  | || | | | | | (_| | (_| |  __/
#  |____/ \__,_|_|_|\__,_| |___|_| |_| |_|\__,_|\__, |\___|
#                                               |___/

build_remote_tag = $(REGISTRY)/$(build_tag)

# Creates the build docker image
build-build-image:
	docker build --tag=$(build_tag) $(build_path)/build-image $(DOCKER_BUILD_ARGS)

# Deletes the local build docker image
clean-build-image:
	docker rmi $(build_tag)

ensure-arm-builder:
	 docker run --privileged --rm tonistiigi/binfmt:qemu-v6.2.0 --install arm64
	 
ensure-build-config:
	-mkdir -p $(kubeconfig_path)
	-mkdir -p $(build_path)/.gocache
	-mkdir -p $(build_path)/.config/gcloud
	-mkdir -p $(helm_config)
	-mkdir -p $(helm_cache)

# create the build image if it doesn't exist
ensure-build-image: ensure-build-config
	$(MAKE) ensure-image IMAGE_TAG=$(build_tag) BUILD_TARGET=build-build-image
ifeq ($(WITH_ARM64), 1) 
ensure-build-image: ensure-arm-builder
endif

# attempt to pull the image, if it exists and rename it to the local tag
# exit's clean if it doesn't exist, so can be used on CI
pull-build-image:
	$(MAKE) pull-remote-build-image REMOTE_TAG=$(build_remote_tag) LOCAL_TAG=$(build_tag)

tag-build-image:
	docker tag $(build_tag) $(CUSTOM_LOCAL_TAG)

# push the local build image up to your repository
push-build-image:
	$(MAKE) push-remote-build-image REMOTE_TAG=$(build_remote_tag) LOCAL_TAG=$(build_tag)

# ensure passed in image exists, if not, run the target
ensure-image:
	@if [ -z $$(docker images -q $(IMAGE_TAG)) ]; then\
		echo "Could not find $(IMAGE_TAG) image. Building...";\
		$(MAKE) $(BUILD_TARGET);\
	fi

# push a local build image up to a remote repository
push-remote-build-image:
	docker tag $(LOCAL_TAG) $(REMOTE_TAG)
	docker push $(REMOTE_TAG)

# pull a local image that may exist on a remote repository, to a local image
pull-remote-build-image:
	-docker pull $(REMOTE_TAG) && docker tag $(REMOTE_TAG) $(LOCAL_TAG)

ensure-agones-sdk-image:
	$(MAKE) ensure-image IMAGE_TAG=$(sidecar_linux_amd64_tag) BUILD_TARGET=build-agones-sdk-image
