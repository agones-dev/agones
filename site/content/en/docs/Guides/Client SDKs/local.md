---
title: "Local Development"
linkTitle: "Local Development"
date: 2019-01-29T10:18:08Z
weight: 1000
description: >
  Working against the SDK without having to run a full kubernetes stack
---

When the game server is running on Agones, the SDK communicates over TCP to a small
gRPC server that Agones coordinated to run in a container in the same network
namespace as it - usually referred to in Kubernetes terms as a "sidecar".

Therefore, when developing locally, we also need a process for the SDK to connect to!

To do this, we can run the same binary (the SDK Server) that runs inside Agones, but pass in a flag
to run it in "local mode". Local mode means that the sidecar binary
will not try to connect to anything, and will just send log messages to stdout and persist local state in memory so
that you can see exactly what the SDK in your game server is doing, and can
confirm everything works.

## Running the SDK Server

To run the SDK Server, you will need a copy of the binary.
This can either be done by downloading a prebuilt binary or running from source code.
This guide will focus on running from the prebuilt binary, but details about running from source code can be found [below](#running-from-source-code-instead-of-prebuilt-binary).

To run the prebuilt binary, for the latest release, you will need to download {{% ghrelease file_prefix="agonessdk-server" %}}, and unzip it.
You will find the executables for the SDK server, for each type of operating system.

* MacOS
  * sdk-server.darwin.amd64
  * sdk-server.darwin.arm64
* Linux
  * sdk-server.linux.amd64
  * sdk-server.linux.arm64
* Windows
  * sdk-server.windows.amd64.exe

### Running In "Local Mode"

To run in local mode, pass the flag `--local` to the executable.

For example:

```bash
./sdk-server.linux.amd64 --local
```

You should see output similar to the following:
```
{"ctlConf":{"Address":"localhost","IsLocal":true,"LocalFile":"","Delay":0,"Timeout":0,"Test":"","GRPCPort":9357,"HTTPPort":9358},"message":"Starting sdk sidecar","severity":"info","source":"main","time":"2019-10-30T21:44:37.973139+03:00","version":"1.1.0"}
{"grpcEndpoint":"localhost:9357","message":"Starting SDKServer grpc service...","severity":"info","source":"main","time":"2019-10-30T21:44:37.974585+03:00"}
{"httpEndpoint":"localhost:9358","message":"Starting SDKServer grpc-gateway...","severity":"info","source":"main","time":"2019-10-30T21:44:37.975086+03:00"}
{"message":"Ready request has been received!","severity":"info","time":"2019-10-30T21:45:47.031989+03:00"}
{"message":"gameserver update received","severity":"info","time":"2019-10-30T21:45:47.03225+03:00"}
{"message":"Shutdown request has been received!","severity":"info","time":"2019-10-30T21:46:18.179341+03:00"}
{"message":"gameserver update received","severity":"info","time":"2019-10-30T21:46:18.179459+03:00"}
```

### Enabling Feature Gates

For development and testing purposes, you might want to enable specific [features gates]({{% ref "/docs/Guides/feature-stages.md#feature-gates" %}}) in the local SDK Server. 

To do this, you can either set the `FEATURE_GATES` environment variable or use the `--feature-gates` command line parameter like so, with the same format as utilised when [configuring it on a Helm install]({{< ref "/docs/Installation/Install Agones/helm.md#configuration" >}}).

For example:

```bash
./sdk-server.linux.amd64 --local --feature-gates Example=true
```
or 
```bash
FEATURE_GATES=Example=true ./sdk-server.linux.amd64 --local
```

## Providing your own `GameServer` configuration for local development

By default, the local sdk-server will create a default `GameServer` configuration that is used for `GameServer()`
and `WatchGameServer()` SDK calls. If you wish to provide your own configuration, as either yaml or json, this
can be passed through as either `--file` or `-f` along with the `--local` flag.

If the `GamerServer` configuration file is changed while the local server is running,
this will be picked up by the local server, and will change the current active configuration, as well as sending out
events for `WatchGameServer()`. This is a useful way of testing functionality, such as changes of state from `Ready` to
`Allocated` in your game server code.

It's important to note that during local development, only specific parts of the GameServer configuration can be modified through SDK calls. For instance, `counters` and `lists` should be placed within the `gameserver.status` section of the configuration file. By making this change, the relevant parts of the configuration are properly exposed and can be accessed through the SDK calls.

{{% alert title="Note" color="info" %}}
File modification events can fire more than one for each save (for a variety of reasons),
but it's best practice to ensure handlers that implement `WatchGameServer()` be idempotent regardless, as repeats can
happen when live as well.
{{% /alert %}}

For example:

```bash
wget https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/gameserver.yaml
./sdk-server.linux.amd64 --local -f ./gameserver.yaml
```
```
{"ctlConf":{"Address":"localhost","IsLocal":true,"LocalFile":"./gameserver.yaml","Delay":0,"Timeout":0,"Test":"","GRPCPort":9357,"HTTPPort":9358},"message":"Starting sdk sidecar","severity":"info","source":"main","time":"2019-10-30T21:47:45.742776+03:00","version":"1.1.0"}
{"filePath":"/Users/alexander.apalikov/Downloads/agonessdk-server-1.1.0/gameserver.yaml","message":"Reading GameServer configuration","severity":"info","time":"2019-10-30T21:47:45.743369+03:00"}
{"grpcEndpoint":"localhost:9357","message":"Starting SDKServer grpc service...","severity":"info","source":"main","time":"2019-10-30T21:47:45.759692+03:00"}
{"httpEndpoint":"localhost:9358","message":"Starting SDKServer grpc-gateway...","severity":"info","source":"main","time":"2019-10-30T21:47:45.760312+03:00"}
```

## Changing State of a Local GameServer

Some SDK calls would change the GameServer state according to [GameServer State Diagram]({{< ref "../../Reference/gameserver.md#gameserver-state-diagram" >}}). Also local SDK server would persist labels and annotations updates.

Here is a complete list of these commands: ready, allocate, setlabel, setannotation, shutdown, reserve.

For example call to Reserve() for 30 seconds would change the GameServer state to Reserve and if no call to Allocate() occurs it would return back to Ready state after this period.

{{% alert title="Note" color="info" %}}
All state transitions are supported for local SDK server, however not all of them are valid in the real scenario. For instance, you cannot make a transition of a GameServer from Shutdown to a Ready state, but can do using local SDK server.
{{% /alert %}}

All changes to the GameServer state could be observed and retrieved using Watch() or GameServer() methods using GameServer SDK.

Example of using HTTP gateway locally:

```bash
curl -X POST "http://localhost:9358/ready" -H "accept: application/json" -H "Content-Type: application/json" -d "{}"
```
```
{}
```
```bash
curl -GET "http://localhost:9358/gameserver" -H "accept: application/json"
```
```
{"object_meta":{"creation_timestamp":"-62135596800"},"spec":{"health":{}},"status":{"state":"Ready"}}
```
```bash
curl -X PUT "http://localhost:9358/metadata/label" -H "accept: application/json" -H "Content-Type: application/json" -d "{ \"key\": \"foo\", \"value\": \"bar\"}"
curl -GET "http://localhost:9358/gameserver" -H "accept: application/json"
```
```
{"object_meta":{"creation_timestamp":"-62135596800","labels":{"agones.dev/sdk-foo":"bar"}},"spec":{"health":{}},"status":{"state":"Ready"}}
```

## Running Local Mode in a Container

Once you have your game server process in a container, you may also want to test the container build locally as well.

Since the production agones-sdk binary has the `--local` mode built in, you can also use the production container image
locally as well!

Since the SDK and your game server container need to share a port on `localhost`, one of the easiest ways to do that
is to have them both run using the host network, like so:

In one shell run:

```shell
docker run --network=host --rm us-docker.pkg.dev/agones-images/release/agones-sdk:{{< release-version >}} --local
```

You should see a similar output to what you would if you were running the binary directly, i.e. outside a container.

Then in another shell, start your game server container:

```shell
docker run --network=host --rm <your image here>
```

If you want to [mount a custom `gameserver.yaml`](#providing-your-own-gameserver-configuration-for-local-development),
this is also possible:

```bash
wget https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/gameserver.yaml
# required so that the `agones` user in the container can read the file
chmod o+r gameserver.yaml
docker run --network=host --rm -v $(pwd)/gameserver.yaml:/tmp/gameserver.yaml us-docker.pkg.dev/agones-images/release/agones-sdk:{{<release-version>}} --local -f /tmp/gameserver.yaml
```

If you run Docker on a OS that doesn't run Docker natively or in a VM, such as on Windows or macOS, you may want to to run the ClientSDK and your game server container together with [Docker Compose](https://docs.docker.com/compose/). To do so, create a `docker-compose.yaml` file setup with a network overlay shared between them:

```yaml
version: '3'
services:
  gameserver:
    build: . # <path to build context>
    ports:
      - "127.0.0.1:7777:7777/udp"

  sdk-server:
    image: "us-docker.pkg.dev/agones-images/release/agones-sdk:{{<release-version>}}"
    command: --local -f /gs_config
    network_mode: service:gameserver # <shared network between sdk and game server>
    configs:
      - gs_config

configs:
  gs_config:
    file: ./gameserver.yaml
```

Run `docker-compose`

```shell
docker-compose up --build
```

## Running from source code instead of prebuilt binary

If you wish to run from source rather than pre-built binaries, that is an available alternative.
You will need [Go installed](https://go.dev/doc/install) and will need to clone the [Agones GitHub repo](https://github.com/agones-dev/agones).

**Disclaimer:** Agones is run and tested with the version of Go specified by the `GO_VERSION` variable in the project's [build Dockerfile](https://github.com/agones-dev/agones/blob/main/build/build-image/Dockerfile). Other versions are not supported, but may still work.

Your cloned repository is best switched to the latest specific release's branch/tag. For example:
```bash
git clone https://github.com/agones-dev/agones.git
cd agones
git checkout release-{{< release-version >}}
```

With Go installed and the Agones repository cloned, the SDK Server can be run with the following command (from the Agones clone directory):
```bash
go run cmd/sdk-server/main.go --local
```

Commandline flags (e.g. `--local`) are exactly the same as command line flags when utilising a pre-built binary.


## Next Steps:

- Learn how to connect your local development game server binary into a running Agones Kubernetes cluster for even more live development options with an [out of cluster dev server]({{< ref "/docs/Advanced/out-of-cluster-dev-server.md" >}}).
