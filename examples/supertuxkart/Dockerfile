# Copyright 2020 Google LLC All Rights Reserved.
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

# build go
FROM golang:1.26.0 AS go-builder
WORKDIR /go/src/supertuxkart

COPY main.go go.mod ./
RUN go mod tidy &&\
    go build -o wrapper .

# Installing STK - forky has 1.5, so we'll jump ahead a little.
FROM debian:forky

WORKDIR /home/supertuxkart
RUN apt update && apt-get install -y supertuxkart && apt clean

RUN useradd -m  -u 1000 supertuxkart

COPY --from=go-builder /go/src/supertuxkart/wrapper .
COPY entrypoint.sh server_config.xml ./

RUN chown -R supertuxkart:supertuxkart /home/supertuxkart && chmod +x wrapper

ENV ENABLE_PLAYER_TRACKING=false
USER 1000
ENTRYPOINT ["./entrypoint.sh"]
