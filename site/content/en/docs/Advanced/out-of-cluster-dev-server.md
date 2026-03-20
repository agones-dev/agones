---
title: "Out of Cluster Dev Server"
linkTitle: "Out of Cluster Dev Server"
date: 2023-07-22T17:21:25Z
publishDate: 2023-08-15T07:00:00Z
weight: 1000
description: >
  Running and debugging server binary locally while connected to a full kubernetes stack
---

This section builds upon the topics discussed in [local SDK Server]({{< ref "/docs/Guides/Client SDKs/local.md" >}}), [Local Game Server]({{< ref "/docs/Guides/local-game-server.md" >}}), and `GameServer` allocation (discussed [here]({{< ref "/docs/Integration Patterns/allocation-from-fleet.md" >}}), [here]({{< ref "/docs/Reference/gameserverallocation.md" >}}), and [here]({{< ref "/docs/Advanced/allocator-service.md" >}})).
Having a firm understanding of those concepts will be necessary for running an "out of cluster" local server.

Running an "out of cluster" dev server combines the best parts of local debugging and being a part of a cluster.
A developer will be able to run a custom server binary on their local machine, even within an IDE with breakpoints.
The server would also be allocatable within a cluster, allowing integration with the project's full stack for handling game server lifetime.

For each run, the only manual steps required by the developer is to manually run the local SDK Server and to run their custom gameplay binary (each can easily be reused/restarted).
All other state progression will be automatically handled by the custom gameplay server (calling the SDK API), the SDK Server (handling the SDK calls), the cluster `GameServer` Controller (progressing specific [states]({{< ref "/docs/Reference/gameserver.md#gameserver-state-diagram" >}})), and the cluster's allocation system (whether be through `GameServerAllocation` or via the Allocator Service) -- just as it would when running in a pod in a cluster!

Out of cluster development is a fantastic option during early prototyping, as it can (optionally) all be run on a single machine with tools such as [Minikube]({{< ref "/docs/Installation/Creating Cluster/minikube.md" >}}).

The name "out of cluster" is to contrast [InClusterConfig](https://pkg.go.dev/k8s.io/client-go/tools/clientcmd#InClusterConfig) which is used in the internal golang kubeconfig API.

## Prerequisite steps

To be able to run an "out of cluster" local game server, one needs to first complete a few prerequisite steps.

### Cluster created

First, a cluster must have been created that the developer has access to through commands like `kubectl`.
This cluster could be running on a provider or locally (e.g. on Minikube).
See [Create Kubernetes Cluster]({{< ref "/docs/Installation/Creating Cluster/_index.md" >}}) for more details on how to create a cluster, if not already done so.

### Agones `GameServer` resource created

Out of cluster dev servers make use of [local dev servers]({{< ref "/docs/Guides/local-game-server.md" >}}).
Follow the instructions there to create a `GameServer` resource for use with a local game server.
Note that the `metadata:annotations:agones.dev/dev-address` should be updated to point to the local machine, more details [below](#forwarded-ports) around port forwarding.

### SDK Server available

An "out of cluster" dev server requires the need to also run the SDK Server locally.

When a `GameServer` runs normally in a prod-like environment, the Agones cluster controller will handle initializing the containers which contain the SDK Server and the game server binary.
The game server binary will be able to connect over gRPC to the SDK Server running in the sidecar container.
When the game server binary makes SDK calls (e.g. `SDK.Ready()`), those get sent to the SDK Server via gRPC and the SDK Server as able to modify the `GameServer` resource in the cluster.
When the `GameServer` resource gets modified (either by the Agones cluster controller, by the Agones Allocation Service, or by the K8s API), the SDK Server is monitoring and sends update events over gRPC to the SDK API, resulting in a callback in the game server binary logic.

The goal of an "out of cluster" dev server is to keep all this prod-like functionality, even in a debuggable context.
To do so, the developer must run the SDK Server locally such that the (also local) game server binary can connect via gRPC.
Instructions for downloading and running the SDK Server can be found [here]({{< ref "/docs/Guides/Client SDKs/local.md" >}}).
However, instead of using `--local` or `--file`, the SDK Server will need to be run in "out of cluster" mode by providing a kubeconfig file to connect to the cluster. This section is focusing on getting the SDK Server ready to run locally, more detail about running it can be found [below](#running-sdk-server-locally).

### Game server binary available

When running Agones normally, the game server binary is inside a prebuilt docker image which is loaded into a container in a `GameServer`'s pod.
This can either be a custom, developer-created, docker image and contained binary or a sample image/binary from an external source.
This document will use the sample `simple-game-server`, which follows suit from various other documentation pages (e.g. [Quickstart: Create a Game Server]({{< ref "/docs/Getting Started/create-gameserver.md" >}})).

The `simple-game-server` can be run from the docker image `{{< example-image >}}`.
The game server binary can either be run within a docker container or run locally, so long as all ports are published/forward -- more on this [below](#forwarded-ports).

Alternatively, the `simple-game-server` can also be run from source code; see `examples/simple-game-server/main.go`. More details about running from source can be found [here]({{< ref "/docs/Guides/Client SDKs/local.md#running-from-source-code-instead-of-prebuilt-binary" >}}).

**Disclaimer:** Agones is run and tested with the version of Go specified by the `GO_VERSION` variable in the project's [build Dockerfile](https://github.com/agones-dev/agones/blob/main/build/build-image/Dockerfile). Other versions are not supported, but may still work.

If a developer has their own game server logic, written in the language of their choice, that would be perfectly fine.
A custom game server can be similarly run within a docker container, run directly on commandline, or run via an IDE/debugger.

### Forwarded Ports

As the game server binary will be run on the developer's machine and a requesting client will attempt to connect to the game server via the `GameServer`'s `metadata:annotations:agones.dev/dev-address` and `spec:ports:hostPort` fields, the developer needs to ensure that connection can take place.

If the game server binary and the arbitrary connecting client logic are both on the same network, then connecting should work without any extra steps.
However, if the developer has a more complicated network configuration or if they are attempting to connect over the public internet, extra steps may be required.

Obviously, this document does not know what every developer's specific network configuration is, how their custom game client(s) work, their development environment, and/or various other factors.
The developer will need to figure out which steps are necessary for their specific configuration.

If attempting to connect via the internet, the developer needs to set the `GameServer`'s `metadata:annotations:agones.dev/dev-address` field to their public IP.
This can be found by going to [whatsmyip.org](https://www.whatsmyip.org/) or <a href="https://www.whatismyip.com/" data-proofer-ignore>whatismyip.com</a> in a web browser.

The  `GameServer`'s `spec:ports:hostPort`/`spec:ports:containerPort` should be set to whichever port the game server binary's logic will bind to -- the port used by `simple-game-server` is 7654 (by default).
The local network's router must also be configured to forward this port to the desired machine; allowing inbound external requests (from the internet) to be directed to the machine on the network that is running the game server.

If the SDK Server is run on the same machine as the game server binary, no extra steps are necessary for the two to connect.
By default, the SDK API (in the game server binary) will attempt to gRPC connect to the SDK Server on `localhost` on the port `9357`.
If the SDK Server is run on another machine, or if the SDK Server is set to use different ports (e.g. via commandline arguments), the developer will need to also take appropriate steps to ensure that the game server can connect to the SDK Server.
As discussed [further below](#running-sdk-server-locally) running the SDK Server with `--address 0.0.0.0` can be quite helpful with various setups.

If the developer is running the SDK Server or the game server binary within docker container(s), then publishing ports and/or connecting to a docker network may be necessary.
Again, these configurations can vary quite dramatically and the developer will need to find the necessary steps for their specific setup.

## Running "out of cluster" local game server

Now that all prerequisite steps have been completed, the developer should have:
  * a [cluster](#cluster-created) with a configured [`GameServer` resource](#agones-gameserver-resource-created).
  * the [SDK Server](#sdk-server-available) ready to run.
  * a [game server binary](#game-server-binary-available) ready to run.

### Optional `GameServer` state monitoring

A helpful (optional) step to see progress when running is to watch the `GameServer` resource.

This can be done with the command:
```bash
kubectl get --watch -n default gs my-local-server
```
It may be necessary to replace `default` and `my-local-server` with whichever namespace/name values are used by the dev `GameServer` created [above](#agones-gameserver-resource-created)).

With this command running, the terminal will automatically show updates to the `GameServer`'s state -- however, this is not necessary to proceed.

### Running SDK Server locally

The first step is to run the SDK Server, making it available for the (later run) game server binary to connect.
Here is a sample command to run the SDK Server, with each argument discussed after.
```bash
./sdk-server.linux.amd64 \
  --gameserver-name my-local-server \
  --pod-namespace default \
  --kubeconfig "$HOME/.kube/config" \
  --address 0.0.0.0 \
  --graceful-termination false
```

* `--gameserver-name` is a necessary arg, passed instead of the `GAMESERVER_NAME` enviroment variable.
  * It is set to the name of the dev `GameServer` k8s resource.
  * It tells the SDK Sever which resource to read/write to on the k8s cluster.
  * This example value of `my-local-server` matches to the instructions for setting up a [Local Game Server]({{< ref "/docs/Guides/local-game-server.md" >}}).
* `--pod-namespace` is a necessary arg, passed instead of the `POD_NAMESPACE` enviroment variable.
  * It is set set to the namespace which the dev `GameServer` resides in.
  * It tells the SDK Sever which namespace to look under for the `GameServer` to read/write to on the k8s cluster.
  * This example value of `default` is used as most instructions in this documentation assumes `GameServers` to be created in the `default` namespace.
* `--kubeconfig` tells the SDK Server how to connect to the k8s cluster.
  * This actually does not trigger any special flow (unlike `--local` or `--file`).
  The SDK Server will run just as it would when created in a sidecar container in a k8s cluster.
  * Passing this argument simply provides where to connect along with the credentials to do so.
  * This example value of `"$HOME/.kube/config"` is the default location for k8s authentication information. This requires the developer be logged in via `kubectl` and have the desired cluster selected via [`kubectl config use-context`](https://jamesdefabia.github.io/docs/user-guide/kubectl/kubectl_config_use-context/).
* `--address` specifies the binding IP address for the SDK Server's SDK API.
  * By default, the binding address is `localhost`. This may be difficult for some development setups.
  * Overriding this value changes which IP address(es) the server will bind to for receiving gRPC/REST SDK API calls.
  * This example value of `0.0.0.0` sets the SDK Server to receive API calls that are sent to any IP address (that reach the machine).
* `--graceful-termination` set to false will disable some smooth state transitions when exiting.
  * By default, the SDK Server will wait until the `GameServer` has reached the `Shutdown` state before exiting ("graceful termination").
  * This will cause the SDK Server to hang (waiting on state update) when attempting to terminate (e.g. with `^C`).
  * When running binaries in a development context, quickly exiting and restarting the SDK Server is handy.

This can easily be terminated with `^C` and restarted as necessary.
Note that terminating the SDK Server while the game server binary (discussed in the [next section](#running-game-server-binary-locally)) is using it may result in failure to update/watch `GameServer` state and may result in a runtime error in the game server binary.

### Running game server binary locally

Now that the SDK Server is running locally with k8s credentials, the game server binary can run in an integrated fashion.
The game server binary's SDK calls will reach the local SDK Server, which will then interact with the `GameServer` resource on the k8s cluster.

Again, this document will make use of `simple-game-server` via its docker image, but running directly or use of a custom game server binary is just as applicable.
Run the game server binary with the command:
```
docker run --rm --network="host" {{< example-image >}}
```

The `--rm` flag will nicely autoclean up the docker container after exiting.
The `--network="host"` flag will tell the docker container to use the host's network stack directly; this allows calls to `localhost` to reach the SDK Server.
The commands and flags used will likely differ if running a custom game server binary.

If the earlier `kubectl get --watch` command was run, it will now show the `GameServer` progressed to the `RequestReady` state, which will automatically be progressed to the `Ready` state by the Agones controller on the cluster.

The `GameServer` state can further be modified by SDK calls, gRPC/REST calls, allocation via either [`GameServerAllocation`]({{< ref "/docs/Reference/gameserverallocation.md" >}}) or [Allocator Service]({{< ref "/docs/Advanced/allocator-service.md" >}}), K8s API calls, etc.
These changes will be shown by the `kubectl get --watch` command.
These changes will also be picked up by the game server binary, if there is a listener registered through the SDK API.
This means that this `GameServer` can be allocated just as it would be when running completely on k8s, but it can be locally debugged.

If the server crashes or is killed by the developer, it can easily be restarted.
This can be done without restarting the SDK Server or any other manual intevention with the `GameServer` resource.
Naturally, this may have implications on any connected clients, but that is project specific and left to the developer to handle.
