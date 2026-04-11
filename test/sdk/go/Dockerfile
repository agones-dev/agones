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

# Build the Go image from source
FROM golang:1.26.0 AS build-stage

WORKDIR /agones.dev

COPY *.go ./

# Note: because this installs the most recently released version of Agones, this will need to be
# built and pushed post-release.
RUN go mod init agones.dev/agones/test/sdk/go/sdk-client-test
RUN go mod tidy
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /sdk-client-test

# Copy the above binary into a lean image
FROM gcr.io/distroless/static-debian13:nonroot AS build-release-stage

WORKDIR /

COPY --from=build-stage /sdk-client-test /sdk-client-test

USER nonroot:nonroot

ENTRYPOINT ["/sdk-client-test"]
