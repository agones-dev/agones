# Copyright 2017 Google LLC All Rights Reserved.
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

RUN apt-get update && apt-get install -y unzip curl

WORKDIR /go/src/examples/xonotic
COPY . /go/src/examples/xonotic
RUN go build -o wrapper .

WORKDIR /go/src/xonotic
RUN curl -o xonotic.zip https://dl.xonotic.org/xonotic-0.8.6.zip && \
    unzip -qq xonotic.zip && rm xonotic.zip

# final image
FROM debian:trixie

WORKDIR /home/xonotic

RUN useradd -u 1000 -m xonotic

#
# To learn about setting up a Xonotic dedicated server, read the `readme.txt` in the
# `server` directory after downloading the archive from xonotic.org
#

COPY --from=builder /go/src/examples/xonotic .
COPY --from=builder /go/src/xonotic/Xonotic ./Xonotic

RUN mkdir -p ./.xonotic/data && mv /home/xonotic/Xonotic/server/server_linux.sh /home/xonotic/Xonotic/
COPY server.cfg ./.xonotic/data

RUN chown -R xonotic:xonotic . && chmod +x wrapper

USER 1000
ENTRYPOINT /home/xonotic/wrapper -i /home/xonotic/Xonotic/server_linux.sh
