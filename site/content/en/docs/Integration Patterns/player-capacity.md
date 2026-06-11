---
title: "Allocating based on GameServer Player Capacity"
linkTitle: "Player Capacity"
date: 2021-08-31
weight: 90
description: >
  Find a `GameServer` that has room for a specific number of players.
---

Using this approach, we are able to be able to make an Allocation request that is akin to: "Find me a `GameServer` 
that is already allocated, with room for _n_ number of players, and if one is not available, allocate me a `Ready` 
`GameServer`".

Common applications of this type of allocation are Lobby servers where players await matchmaking, or a
persistent world server where players connect and disconnect from a large map.

## Tracking Players Through Lists

{{< beta title="Counters And Lists" gate="CountsAndLists" >}}

<a href="../../../diagrams/allocation-player-capacity-list.puml.png" target="_blank">
<img src="../../../diagrams/allocation-player-capacity-list.puml.png" alt="Player Capacity Allocation Diagram" />
</a>

### Example `GameServerAllocation`

The below allocation will attempt to find an already Allocated `GameServer` from the `Fleet` "simple-game-server" with 
room for at least 10, and if it cannot find one, will allocate a Ready one from the same `Fleet`.

```yaml
apiVersion: allocation.agones.dev/v1
kind: GameServerAllocation
spec:
  selectors:
    - matchLabels:
        agones.dev/fleet: simple-game-server
      gameServerState: Allocated # check for Allocated first
      lists:
        players:
          minAvailable: 10 # at least 10 players in available capacity
    - matchLabels:
        agones.dev/fleet: simple-game-server
      lists:
        players:
          minAvailable: 10 # not required, since our GameServers start with empty lists, but a good practice
```

## Consistency

Agones, and Kubernetes itself are built as eventually consistent, self-healing systems. To that end, it is worth
noting that there may be minor delays between each of the operations in either of the above flows. For example,
depending on the cluster load, it may take approximately a second for an SDK driven
[list change]({{% ref "/docs/Guides/Client SDKs/_index.md#counters-and-lists" %}}) on a `GameServer` record to be visible to the Agones
allocation system. We recommend building your integrations with Agones with this in mind.

## Next Steps

- Have a look at all commands the [Client SDK]({{< ref "/docs/Guides/Client SDKs/_index.md" >}}) provides.
- Check all the options available on [`GameServerAllocation`]({{% ref "/docs/Reference/gameserverallocation.md" %}}).
- If you aren't familiar with the term [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/), this should
  provide a reference.
