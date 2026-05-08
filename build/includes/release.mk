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

#   ____      _
#  |  _ \ ___| | ___  __ _ ___  ___
#  | |_) / _ \ |/ _ \/ _` / __|/ _ \
#  |  _ <  __/ |  __/ (_| \__ \  __/
#  |_| \_\___|_|\___|\__,_|___/\___|
#

#
# targets for an Agones release
#

# agones image release registry
release_registry = us-docker.pkg.dev/agones-images/release

# outputs the markdown for the example images section of the release template
release-example-image-markdown: example-image-markdown.allocation-endpoint
release-example-image-markdown: example-image-markdown.autoscaler-webhook
release-example-image-markdown: example-image-markdown.cpp-simple
release-example-image-markdown: example-image-markdown.crd-client
release-example-image-markdown: example-image-markdown.nodejs-simple
release-example-image-markdown: example-image-markdown.rust-simple
release-example-image-markdown: example-image-markdown.simple-game-server
release-example-image-markdown: example-image-markdown.supertuxkart
release-example-image-markdown: example-image-markdown.unity-simple
release-example-image-markdown: example-image-markdown.xonotic

example-image-markdown.%:
	@cd $(agones_path)/examples/$* && \
    tag=$$(make -silent echo-image-tag) && \
    echo "- [$$tag](https://$$tag)"


# Deploys the site by taking in the base version and deploying the previous minor or patch version
release-deploy-site: $(ensure-build-image)
release-deploy-site: DOCKER_RUN_ARGS += -e GOFLAGS="-mod=mod" --entrypoint=/usr/local/go/bin/go
release-deploy-site:
	version_to_process=$(if $(VERSION),$(VERSION),$(base_version)) && \
	version=$$($(DOCKER_RUN) run $(mount_path)/build/scripts/previousversion/main.go -version $$version_to_process) && \
	echo "Deploying Site Version: $$version" && \
	$(MAKE) ENV=HUGO_ENV=snapshot site-deploy SERVICE=$$version

# Creates, switches, and pushes a new minor version release branch based off of the main branch.
# The should be run before pre_cloudbuild.yaml. This means base_version has not yet been updated.
create-minor-release-branch: RELEASE_VERSION ?= $(base_version)
create-minor-release-branch:
	@echo "Starting creating release branch for minor version: $(RELEASE_VERSION)"

	# switch to the right project
	$(DOCKER_RUN) gcloud config configurations activate agones-images

	git remote update -p
	git fetch --all --tags
	git checkout -b release-$(RELEASE_VERSION) upstream/main
	git push -u upstream release-$(RELEASE_VERSION)

# Creates, switches, and pushes a new patch version release branch based off of the release branch.
# The should be run before pre_cloudbuild.yaml. Require user to the specify both the patch version,
# and the version to base the release-branch off of.
create-patch-release-branch: PREVIOUS_VERSION ?=
create-patch-release-branch: PATCH_VERSION ?=
create-patch-release-branch:
	$(if $(PREVIOUS_VERSION),,$(error PREVIOUS_VERSION is not set. Please provide the version to branch from.))
	$(if $(PATCH_VERSION),,$(error PATCH_VERSION is not set. Please provide the new patch version number.))

	@echo "Creating new patch release branch release-$(PATCH_VERSION) from tag v$(PREVIOUS_VERSION)"

	# switch to the right project
	$(DOCKER_RUN) gcloud config configurations activate agones-images

	git remote update -p
	git fetch upstream --tags
	git checkout -b release-$(PATCH_VERSION) v$(PREVIOUS_VERSION)
	git push -u upstream release-$(PATCH_VERSION)

# push the current chart to google cloud storage and update the index
# or push the current charts to the helm registry `CHARTS_REGISTRY`
push-chart: $(ensure-build-image) build-chart
ifneq ($(CHARTS_REGISTRY),)
	docker run --rm $(common_mounts) -w $(workdir_path) $(build_tag) bash -c \
		"helm push ./install/helm/bin/*.* $(CHARTS_REGISTRY)"
else
	docker run $(DOCKER_RUN_ARGS) --rm $(common_mounts) -w $(workdir_path) $(build_tag) bash -c \
		"gsutil copy gs://$(GCP_BUCKET_CHARTS)/index.yaml ./install/helm/bin/index.yaml || /bin/true && \
		helm repo index --merge ./install/helm/bin/index.yaml ./install/helm/bin && \
		cat ./install/helm/bin/index.yaml && ls ./install/helm/bin/ && \
		cp ./install/helm/bin/index.yaml ./install/helm/bin/index-$(VERSION).yaml && \
		gsutil copy ./install/helm/bin/*.* gs://$(GCP_BUCKET_CHARTS)/"
endif

# Ensure the example images exists for a release and deploy the previous version's website.
pre-build-release: VERSION ?=
pre-build-release:
	$(if $(VERSION),,$(error VERSION is not set. Please provide the current release version.))
	docker run --rm $(common_mounts) -w $(workdir_path)/build/release $(build_tag) \
		gcloud builds submit . --substitutions _BRANCH_NAME=release-$(VERSION),_VERSION=$(VERSION) --config=./pre_cloudbuild.yaml $(ARGS)

# Build and push the images in the release repository, stores artifacts,
# Pushes the current chart version to the helm repository hosted on gcs.
post-build-release: VERSION ?=
post-build-release:
	$(if $(VERSION),,$(error VERSION is not set. Please provide the current release version.))
	docker run --rm $(common_mounts) -w $(workdir_path)/build/release $(build_tag) \
		gcloud builds submit . --substitutions _VERSION=$(VERSION),_BRANCH_NAME=release-$(VERSION) --config=./post_cloudbuild.yaml $(ARGS)

# Tags images from the previous release as deprecated.
# The tr -d '-' command is used to remove the dashes from the output of the script
# (e.g., 1-52-1 becomes 1.52.1), which is the format needed for the Docker image tag.
tag-deprecated-images: VERSION ?=
tag-deprecated-images: $(ensure-build-image)
tag-deprecated-images: DOCKER_RUN_ARGS += -e GOFLAGS="-mod=mod" --entrypoint=/usr/local/go/bin/go
tag-deprecated-images:
	$(if $(VERSION),,$(error VERSION is not set. Please provide the current release version.))
	previous_version=$$($(DOCKER_RUN) run $(mount_path)/build/scripts/previousversion/main.go -version $(VERSION)| tr '-' '.') && \
	images="agones-controller agones-extensions agones-sdk agones-allocator agones-ping agones-processor" && \
	for image in $$images; do \
		echo "Tagging ${release_registry}/$$image:$$previous_version as deprecated..."; \
		gcloud artifacts docker tags add ${release_registry}/$$image:$$previous_version ${release_registry}/$$image:deprecated-public-image-$$previous_version; \
	done
