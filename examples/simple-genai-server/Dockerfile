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

# build
FROM golang:1.26.0 as builder
WORKDIR /go/src
COPY . agones.dev/agones

WORKDIR /go/src/agones.dev/agones/examples/simple-genai-server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .

# final image
FROM gcr.io/distroless/static-debian13:nonroot
WORKDIR /

COPY --from=builder /go/src/agones.dev/agones/examples/simple-genai-server/server /server

USER nonroot:nonroot
ENTRYPOINT ["/server"]
