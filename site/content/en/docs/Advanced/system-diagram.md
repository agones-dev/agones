---
title: "System Diagram"
date: 2024-04-18
weight: -100
description: >
  A pictoral overview of the Agones component relationships.
---

![System Diagram](../../../diagrams/system-diagram.dot.png)

# Agones Control Plane 

The Agones Control Plane consists of 4 `Deployments`:
```
NAME                READY   UP-TO-DATE   AVAILABLE   AGE
agones-allocator    3/3     3            3           40d
agones-controller   2/2     2            2           40d
agones-extensions   2/2     2            2           40d
agones-ping         2/2     2            2           40d
```

## `agones-allocator`

`agones-allocator` provides a gRPC/REST service that translates allocation requests into `GameServerAllocations`. See [Allocator Service]({{< relref "allocator-service.md">}}) for more information.

## `agones-controller`

`agones-controller` maintains various control loops for all Agones CRDs (`GameServer`, `Fleet`, etc.). A single leader-elected `Pod` of the `Deployment`
is active at any given time (see [High Availability]({{< relref "high-availability-agones.md">}})).

## `agones-extensions`

`agones-extensions` is the endpoint for:
* Agones-installed Kubernetes webhooks, which handle defaulting and validation for Agones CRs,
* and the `GameServerAllocation` `APIService`, which handles `GameServer` allocations (either from the Allocator Service or the Kubernetes API).

## `agones-ping` (not pictured)

`agones-ping` is a simple ping service for latency testing from your game client - see [Latency Testing]({{< relref "ping-service.md">}}).

# Agones CRDs

See [Create a Game Server]({{< relref "create-gameserver.md">}}), [Create a Game Server]({{< relref "create-fleet.md">}}) for examples of Agones CRDs in action, or the [API Reference]({{< ref "/docs/Reference" >}}) for more detail.

All of the Agones CRDs are controlled and updated by `agones-controller`. `GameServer` is additionally updated by the SDK Sidecar and `agones-extensions` (moving a `GameServer` from [`Ready` to `Allocated`]({{< ref "/docs/Reference/gameserver.md#gameserver-state-diagram" >}})

# Game Server Pod

Also pictured is an example `Pod` owned by a `GameServer`. Game clients typically connect to the Dedicated Game Server directly, or via a proxy like [Quilkin](https://github.com/EmbarkStudios/quilkin/blob/main/docs/src/introduction.md). The game server server interfaces with Agones using the [Client SDK]({{< relref "Client SDKs">}}), which is a thin wrapper around the [SDK gRPC protocol](https://github.com/agones-dev/agones/blob/main/proto/sdk/sdk.proto). The SDK connects to the SDK Sidecar (`sdk-server`) in the same Pod, which handles the SDK business logic for e.g. [health checks]({{< relref "health-checking.md">}}), [Counters and Lists]({{< relref "counters-and-lists.md">}}), etc.