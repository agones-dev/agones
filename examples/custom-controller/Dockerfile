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

# Gather dependencies and build the executable
FROM golang:1.26.0 as builder
WORKDIR /go/src/custom-controller
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o client .

# Create the final image that will run the webhook server for FleetAutoscaler webhook policy
FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=builder /go/src/custom-controller/client /

USER nonroot:nonroot
ENTRYPOINT ["/client"]
