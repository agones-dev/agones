---
title: "GameServer Counters and Lists"
linkTitle: "Counters and Lists"
date: 2024-01-08
weight: 25
description: >
  Track, allocate and auto-scale based on user defined counters and lists stored on a `GameServer`.
---

{{< beta title="Counters and Lists" gate="CountsAndLists" >}}

Counters and Lists is provided as a way to track arbitrary integer counter values as well as
lists of values against a `GameServer` by a user provided key. 

Combined with the ability to set and manipulate max capacity values at runtime for each counter and list, allows
Agones to also provide Allocation, Fleet scheduling and Fleet autoscaling based on this functionality, such that it
supports a wide variety of use cases, including, but not limited to:

* Connected player listing tracking, Allocation filtering and autoscaling based on available capacity.
* Multi-tenant server room counting, Allocation filtering and autoscaling based on available capacity.
* Game specific GameServer weighting on Allocation.
* ...any other use case that requires a list of values or a counter aligned with a GameServer for tracking purposes.

## Declaration

All keys for either Counters and Lists must be declared at creation time within the 
[GameServerSpec]({{< ref "/docs/Reference/agones_crd_api_reference.html#agones.dev/v1.GameServerSpec" >}}) before being
utilised, and keys cannot be added or deleted from `GameServers` past their initial creation.

For example, if we want to use a Counter of `rooms` to track the number of game session rooms that exist in
this `GameServer` instance, while also tracking a List of currently connected `players`, we could
implement the following:

```yaml
apiVersion: agones.dev/v1
kind: Fleet
metadata:
  name: simple-game-server
spec:
  replicas: 2
  template:
    spec:
      ports:
        - name: default
          containerPort: 7654
      counters:
        rooms: # room counter
          count: 0
          capacity: 10
      lists:
        players: # players list
          values: []
      template:
        spec:
          containers:
            - name: simple-game-server
              image: {{< example-image >}}
```

Both Counters and Lists can have a `capacity` value, which indicated the maximum counter value, or number of items that
can be stored in the list.

In the above example, the `room` Counter has a capacity of 10, which means that the count cannot go past that value.

See the [GameServer]({{< ref "/docs/Reference/gameserver.md" >}}) reference for all configurable options.

## Retrieval

We now have several ways to retrieve this Counter and List information from a GameServer, depending on your use case.

### Kubernetes API

If you wish to retrieve or view current Counter or List values from outside your `GameServer`, you are able to do this
through the Kubernetes API, or similarly through `kubectl`.

Counter values and capacities for are stored on a `GameServer` resource instance
under [`GameServer.Status.Counters`][gameserverstatus] by key, and [`GameServer.Status.Lists`][gameserverstatus] stores
List value arrays and capacities by key as well.

Therefore, in the above examples, the `GameServer.Status.Counters[rooms].Count`
and `GameServer.Status.Counters[rooms].Capacity` would have the current Counter value and capacity for the room
counters.

Subsequently `GameServer.Status.Lists[players].Values` stores the array of values for the list
and `GameServer.Status.Lists[players].Capacity` is the current capacity for the `players` List.

Check the API reference for [`GameServerStatus`][gameserverstatus] for all the details on the data structure.

### SDK

Counter and Lists values can be accessed through the Agones SDK when that information is required within your game
server process.

For example, to retrieve the above `room` counter, each language SDK has some implementation of the
[`SDK.Beta().GetCounterCount("room")`]({{< ref "/docs/Guides/Client SDKs/_index.md#betagetcountercountkey" >}}) 
function 
that returns
the current value of the counter. Similarly, to retrieve the `players` list, we can use the
[`SDK.Beta().GetListValues("players")`]({{< ref "/docs/Guides/Client SDKs/_index.md#betagetlistvalueskey" >}})
function.

The special ability of the SDK retrieval operations, is that is also tracks any modifications that have been made
through the SDK, that have yet to be persisted to the `GameServer`, and will return values with that data included.

This means that any modifications made to Counters and Lists sent through the SDK can be immediately and atomically
retrieved from the SDK from the game server binary with accurate information without having to wait for it be persisted
to the `GameServer`.

See the [SDK Guide]({{< ref "/docs/Guides/Client SDKs/_index.md" >}}) for the full set of data retrieval functions 
that are available.

## Manipulation

We also have several ways to manipulate the Counter and List information stored on a `GameServer` instance, depending on your use case.

### SDK

Counter and Lists values can be modified through the Agones SDK when you wish to be able to edit that information 
within your game server process.

For example, to increment the above `room` counter by 1, each language SDK has some implementation of the
[`SDK.Beta().IncrementCounter("room", 1)`]({{< ref "/docs/Guides/Client SDKs/_index.md#betaincrementcounterkey-amount" >}}) 
function thatincrements the counter by 1. Similarly, to add the value `player1` to the `players` list, we can use the
[`SDK.Beta().AppendListValue("players", "player1")`]({{< ref "/docs/Guides/Client SDKs/_index.md#betaappendlistvaluekey-value" >}})
function.

See the [SDK Guide]({{< ref "/docs/Guides/Client SDKs/_index.md" >}}) for the full set of modification functions 
that are available.
### Allocation

When performing a `GameServer` allocation, you may want to manipulate the Counter and/or List information atomically 
on a `GameServer` allocation operation.

For example, you may wish to increment the `room` counter to indicate a new game session has started on the returned 
`GameServer` instance, or provide the connecting player information to the `player` list to make that available to 
the game server binary before the player connects.

This can be done through [`GameServerAllocation.Spec.Counters`][gameserverallocation] and 
[`GameServerAllocation.Spec.Lists`][gameserverallocation], which provide
[`CounterAction`]({{< ref "/docs/Reference/agones_crd_api_reference.html#allocation.agones.dev/v1.CounterAction" >}}) and 
[`ListAction`]({{< ref "/docs/Reference/agones_crd_api_reference.html#allocation.agones.dev/v1.ListAction" >}}) 
configuration respectively.

For example, if on allocation we wished to increment the `room` counter by 1 and add `player1` to the `players` list,
this could be done with the following `GameServerAllocation` specification:

```yaml
apiVersion: allocation.agones.dev/v1
kind: GameServerAllocation
spec:
  selectors:
    - matchLabels:
        agones.dev/fleet: simple-game-server
  counters:
    rooms:
      action: Increment
      amount: 1
  lists:
    players:
      addValues:
        - player1
```

Counter and List changes made through the `GameServerAllocation` functionality, will be eventually consistent in 
their availability via the [SDK retrieval functions](#sdk) for performance reasons.

Performing these data changes as part of the Allocation also **does not** guarantee that the
counter or list is past its capacity value. In the event that the capacity is exceeded, the allocation will succeed, but
the Counter or List values will not be updated. If you want to ensure there is capacity available for the Allocation
operation, see [Allocation Filtering and Prioritisation](#allocation-filtering-and-prioritisation) below.

See the [Allocation Reference]({{< ref "/docs/Reference/gameserverallocation.md" >}}) for all the Allocation 
configuration options.

### Kubernetes API

Counter values and capacities are stored on a `GameServer` resource instance
under [`GameServer.Status.Counters`][gameserverstatus] by key, and [`GameServer.Status.Lists`][gameserverstatus] stores
List value arrays and capacities by key as well. Therefore, they can be modified through the Kubernetes API either 
through [Kubernetes client libraries](https://kubernetes.io/docs/reference/using-api/client-libraries/) or manually 
through `kubectl edit`.

Counter and List changes made through the `GameServerAllocation` functionality, will be eventually consistent in
their availability via the [SDK retrieval functions](#sdk) for performance reasons.

Check out the [Access Agones via the Kubernetes API]({{% relref "access-api.md" %}}) guide for a more in depth guide 
on how to interact with Agones resources through the Kubernetes API programmatically.

## Allocation Filtering and Prioritisation

Counters and Lists can also be used as filtering properties when performing an allocation through either the
[GameServerAllocation.Spec.Selectors.Counts][gameserverallocation] or 
[GameServerAllocation.Spec.Selectors.Lists][gameserverallocation] properties.

If we want to expand the above example to ensure there is always room for the `room` Counter increment and room
for the `player` List to add `player1`, we can add the following to the `GameServerAllocation` specification:

```yaml
apiVersion: allocation.agones.dev/v1
kind: GameServerAllocation
spec:
  selectors:
    - matchLabels:
        agones.dev/fleet: simple-game-server
      counters:
        rooms:
          minAvailable: 1
      lists:
        players:
          minAvailable: 1
  counters:
    rooms:
      action: Increment
      amount: 1
  lists:
    players:
      addValues:
        - player1
```

Only `GameServer` instances that have a `capacity` value of at least one more than the Counter value, or List length
will be included in the potential allocation result - thereby ensuring that both `room` and `player` operations 
will succeed, assuming a GameServer is found.

This can be combined with the ability to define multiple `selectors` to create quite sophisticated `GameServer` 
selection options that can attempt to find the most appropriate `GameServer`.

See the [Allocation Reference]({{< ref "/docs/Reference/gameserverallocation.md" >}}) for more details.

The `priorities` block also gives you options to prioritise specific `GameServers` over others when performing an 
Allocation, depending on the Allocation Strategy.

* Packed: The [usual infrastructure optimisation strategy]({{< ref "/docs/Advanced/scheduling-and-autoscaling.md#allocation-scheduling-strategy" >}})
  still applies, but the `priorities` block is used as a tie-breaker within the least utilised infrastructure, to ensure
  optimal infrastructure usage while also allowing some custom prioritisation of `GameServers`.
* Distributed: The entire selection of `GameServers` will be sorted by this priority list to provide the
  order that `GameServers` will be allocated by.

For example, if we wanted to select a `GameServer` that had:

1. At least one `room` available in the GameServer capacity.
2. First check Allocated `GameServers` and falling back to `Ready` `GameServers` if there aren't any.
3. Include general infrastructure optimisation for least usage.
4. And also ensure the `GameServers` that were most full are allocated first (after infrastructure optimisations
   where applied).

We could implement the following `GameServerAllocation`:

```yaml
apiVersion: allocation.agones.dev/v1
kind: GameServerAllocation
spec:
  scheduling: Packed
  # Choose most full `GameServers` first
  priorities:
    - type: Counter
      key: rooms
      order: Ascending
  selectors:
    # First check to see if we can back-fill an already allocated `GameServer`
    - gameServerState: Allocated
      matchLabels:
        agones.dev/fleet: simple-game-server
      counters:
        rooms:
          minAvailable: 1
    # If we can't, then go get a `Ready` `GameServer`.
    - gameServerState: Ready
      matchLabels:
        agones.dev/fleet: simple-game-server
      counters:
        rooms:
          minAvailable: 1
  counters:
    rooms:
      action: Increment
      amount: 1
```

For more details on how Agones implements infrastructure optimisation, see the documentation on
[Scheduling and Autoscaling]({{< ref "/docs/Advanced/scheduling-and-autoscaling.md" >}}).

## Fleet Scale Down Prioritisation

Another optimisation control you can apply with `Fleets` when Agones' scaled them down, is to also set `priorities` 
to influence the order in which `GameServers` are shutdown and deleted.


While neither `players` or `rooms` are particularly good examples for this functionality, if we wanted to ensure 
that `Ready` `GameServers` with the most available capacity `rooms` were a factor when scaling down a `Fleet` we could 
implement the following:

```yaml
apiVersion: agones.dev/v1
kind: Fleet
metadata:
  name: simple-game-server
spec:
  replicas: 2
  priorities:
    - type: Counter
      key: rooms
      order: Descending
  template:
    spec:
      ports:
        - name: default
          containerPort: 7654
      counters:
        rooms: # room counter
          count: 0
          capacity: 10
      template:
        spec:
          containers:
            - name: simple-game-server
              image: {{< example-image >}}
```

Depending on the Fleet allocation strategy, the `priorities` block will influence scale down logic as follows:

* Packed: The [usual infrastructure optimisation strategy]({{< ref "/docs/Advanced/scheduling-and-autoscaling.md#fleet-scheduling" >}})
  still applies, but the `priorities` block is used as a tie-breaker within the least utilised infrastructure, to ensure
  optimal infrastructure usage while also allowing some custom prioritisation of `GameServers`.
* Distributed: The entire selection of `GameServers` will be sorted by this priority list to provide the
  order that `GameServers` will be scaled down by.

See [Fleet Reference]({{< ref "/docs/Reference/fleet.md" >}}) for all the configuration options.

## Autoscaling

Counters and Lists expands on Fleet Autoscaling capabilities, by allowing you to autoscale based on available capacity
across the `Fleet` as a unit, rather than the less granular unit of individual `GameServers` instances.

This means, we can implement an Autoscaling strategy of "always make sure there are 5 free `rooms` available for new
players at all times" like so:

```yaml
apiVersion: autoscaling.agones.dev/v1
kind: FleetAutoscaler
metadata:
  name: simple-game-server
spec:
  fleetName: fleet-example
  policy:
    type: Counter
    counter:
      key: rooms
      bufferSize: 5
      maxCapacity: 100
```

See the [Fleet Autoscaling Reference]({{< ref "/docs/Reference/fleetautoscaler.md" >}}) for all the configuration 
options.

## Metrics

Metrics are exported, using the `key` that the metric is stored under as a label on the metrics, in aggregate across
all `GameServers` within a `Fleet`, exporting aggregate numeric totals for Counters and Lists as gauge metrics.

| Name                  | Description                                                                              | Type  |
|-----------------------|------------------------------------------------------------------------------------------|-------|
| agones_fleet_counters | Aggregate Metrics for Counters within a Fleet, including total capacity and count values | gauge |
| agones_fleet_lists    | Aggregate Metrics for Lists within a Fleet, including total capacity and List lengths    | gauge |

See [Metrics available]({{< ref "/docs/Guides/metrics.md#metrics-available" >}}) for the full list of available metrics
and how to use them.

## Next Steps

* Check out how to access [Agones resource via the Kubernetes API]({{% relref "access-api.md" %}}).
* Have a look at the external [Allocator Service]({{< ref "/docs/Advanced/allocator-service.md" >}}) to make integrating
  Allocation into your workflow easier.

[gameserverstatus]: {{< ref "/docs/Reference/agones_crd_api_reference.html#agones.dev/v1.GameServerStatus" >}}
[gameserverallocation]: {{< ref "/docs/Reference/agones_crd_api_reference.html#allocation.agones.dev/v1.GameServerAllocationSpec" >}}
