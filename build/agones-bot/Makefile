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

# To redeploy agones-bot:
#   make

STORAGE ?= gs://agones-bot-pr-commenter-config

current_dir := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

all: build deploy

# build: creates the project Cloud Build custom build step and hosts it on gcr.io
build:
	gcloud builds submit . --project=agones-images --config=cloudbuild.yaml $(ARGS)

# deploy pushes a new version to Cloud Run
deploy:
	gsutil cp $(current_dir)/agones-bot-pr-commenter.yaml $(STORAGE)
	gcloud run deploy agones-bot-pr-commenter \
		--project=agones-images \
		--image="us-docker.pkg.dev/agones-images/build/agones-bot" \
		--service-account=agones-bot-pr-commenter@agones-images.iam.gserviceaccount.com \
		--platform managed \
		--no-allow-unauthenticated \
		--region=us-west3 \
		--update-env-vars="CONFIG_PATH=$(STORAGE)/github.yaml"
