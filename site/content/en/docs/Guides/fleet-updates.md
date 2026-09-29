---
title: "Fleet Updates"
date: 2019-08-27T03:58:19Z
weight: 20
description: >
  Common patterns and approaches for updating Fleets with newer and/or different versions of your `GameServer` configuration.
---

## Rolling Update Strategy

When Fleets are edited and updated, the default strategy of Agones is to roll the new version of the `GameServer`
out to the entire `Fleet`, in a step by step increment and decrement by adding a chunk of the new version and removing
a chunk of the current set of `GameServers`.

This is done while ensuring that `Allocated` `GameServers` are not deleted
until they are specifically shutdown through the game servers SDK, as they are expected to have players on them.

You can see this in the `Fleet.Spec.Strategy` [reference]({{< ref "/docs/Reference/fleet.md" >}}), with controls for how
much of the `Fleet` is  incremented and decremented at one time:

```yaml
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
```

So when a Fleet is edited (any field other than `replicas`, see note below), either through `kubectl` `edit`/`apply` or via the Kubernetes API, this performs the following operations:

1. Adds the `maxSurge` number of `GameServers` to the Fleet.
1. Shutdown the `maxUnavailable` number of `GameServers` in the Fleet, skipping `Allocated` `GameServers`.
1. Repeat above steps until all the previous `GameServer` configurations have been `Shutdown` and deleted.

By default, a Fleet will wait for new `GameServers` to become `Ready` during a Rolling Update before continuing to shutdown additional `GameServers`, only counting `GameServers` that are `Ready` as being available when calculating the current `maxUnavailable` value which controls the rate at which `GameServers` are updated.
This ensures that a Fleet cannot accidentally have zero `GameServers` in the `Ready` state if something goes wrong during a Rolling Update or if `GameServers` have a long delay before moving to the `Ready` state.

{{% alert title="Note" color="info" %}}
When `Fleet` update contains only changes to the `replicas` parameter, then new GameServers will be created/deleted straight away,
which means in that case `maxSurge` and `maxUnavailable` parameters for a RollingUpdate will not be used.
The RollingUpdate strategy takes place when you update `spec` parameters other than `replicas`.

If you are using a Fleet which is scaled by a FleetAutoscaler, [read the Fleetautoscaler guide]({{< relref "../Getting Started/create-fleetautoscaler.md#7-change-autoscaling-parameters" >}}) for more details on how RollingUpdates with FleetAutoscalers need to be implemented.

You could also check the behaviour of the Fleet with a RollingUpdate strategy on a test `Fleet` to preview your upcoming updates.
Use `kubectl describe fleet` to track scaling events in a Fleet.
{{% /alert %}}

## Recreate Strategy

This is an optimal `Fleet` update strategy if you want to replace all `GameServers` that are not `Allocated`
with a new version as quickly as possible.

You can see this in the `Fleet.Spec.Strategy` [reference]({{< ref "/docs/Reference/fleet.md" >}}):

```yaml
  strategy:
    type: Recreate
```

So when a Fleet is edited, either through `kubectl` `edit`/`apply` or via the Kubernetes API, this performs the following operations:

1. `Shutdown` all `GameServers` in the Fleet that are not currently `Allocated`.
1. Create the same number of the new version of the `GameServers` that were previously deleted.
1. Repeat above steps until all the previous `GameServer` configurations have been `Shutdown` and deleted.

## Two (or more) Fleets Strategy

If you want very fine-grained control over the rate that new versions of a `GameServer` configuration is rolled out, or
if you want to do some version of A/B testing or smoke test between different versions, running two (or more) `Fleets` at the same time is a
good solution for this.

To do this, create a second `Fleet` inside your cluster, starting with zero replicas. From there you can scale this newer `Fleet`
up and the older `Fleet` down as required by your specific rollout strategy.

This also allows you to rollback if issues arise with the newer version, as you can delete the newer `Fleet`
and scale up the old Fleet to its previous levels, resulting in minimal impact to the players.

{{% alert title="Note" color="info" %}}
For GameServerAllocation, you will need to have at least a single shared label between the `GameServers` in each
Fleet.
{{% /alert %}}

### GameServerAllocation Across Fleets

Since `GameServerAllocation` is powered by label selectors, it is possible to allocate across multiple fleets, and/or
give preference to particular sets of `GameServers` over others. You can see details of this in
the `GameServerAllocation` [reference]({{< ref "/docs/Reference/gameserverallocation.md" >}}).

In a scenario where a new `v2` version of a `Fleet` is being slowly scaled up in a separate Fleet from the previous `v1`
Fleet, we can specify that we `prefer` allocation to occur from the `v2` Fleet, and if none are available, fallback to
the `v1` Fleet, like so:


{{< tabpane persist="header" >}}
  {{< tab header="selectors" lang="yaml" >}}
apiVersion: "allocation.agones.dev/v1"
kind: GameServerAllocation
spec:
  selectors:
    - matchLabels:
        agones.dev/fleet: v2
    - matchLabels:
        game: my-awesome-game
  {{< /tab >}}
  {{< tab header="required & preferred (deprecated)" lang="yaml" >}}
apiVersion: "allocation.agones.dev/v1"
kind: GameServerAllocation
spec:
  # Deprecated, use field selectors instead.
  required:
    matchLabels:
      game: my-awesome-game
  # Deprecated, use field selectors instead.
  preferred:
    - matchLabels:
        agones.dev/fleet: v2
  {{< /tab >}}
{{< /tabpane >}}

In this example, all `GameServers` have the label `game: my-awesome-game`, so the Allocation will search across both
Fleets through that mechanism. The `selectors` label matching selector tells the allocation system to first search
all `GameServers` with the `v2` `Fleet` label, and if not found, search through the rest of the set.

The above `GameServerAllocation` can then be used while you scale up the `v2` Fleet and scale down the original Fleet at
the rate that you deem fit for your specific rollout.

## Notifying GameServers on Fleet Update/Downscale


When `Allocated` `GameServers` are utilised for a long time, such as a Lobby `GameServer`,
or a `GameServer` that is being reused multiple times in a row, it can be useful
to notify an `Allocated` `GameServer` process when its backing Fleet has been updated.
When an update occurs, the `Allocated` `GameServer`, may want to actively perform a graceful shutdown and release its
resources such that it can be replaced by a new version, or similar actions.

To do this, we provide the ability to apply a user-provided set of labels and/or annotations to the `Allocated`
`GameServers` when a `Fleet` update occurs that updates its `GameServer` template, or generally
causes the `Fleet` replica count to drop below the number of currently `Allocated` `GameServers`.

This provides two useful capabilities:

1. The `GameServer` [`SDK.WatchGameServer()`]({{% relref "./Client SDKs/_index.md#watchgameserverfunctiongameserver" %}})
   command can be utilised to react to this annotation and/or label change to
   indicate the Fleet system change, and the game server binary could execute code accordingly.
2. This can also be used to proactively update `GameServer` labels, to effect change in allocation strategy - such as
   preferring the newer `GameServers` when allocating, but falling back to the older version if there aren't enough
   of the new ones yet spun up.

The labels and/or annotations are applied to `GameServers` in a `Fleet` in the order designated by their configured [Fleet scheduling]({{< ref "/docs/Advanced/scheduling-and-autoscaling#fleet-scheduling">}}).

Example yaml configuration:

```yaml
apiVersion: "agones.dev/v1"
kind: Fleet
metadata:
  name: simple-game-server
spec:
  replicas: 2
  allocationOverflow: # This specifies which annotations and/or labels are applied
    labels:
      mykey: myvalue
      version: "" # empty an existing label value, so it's no longer in the allocation selection
    annotations:
      event: overflow
  template:
    spec:
      ports:
        - name: default
          containerPort: 7654
      template:
        spec:
          containers:
            - name: simple-game-server
              image: {{< example-image >}}
```

See the [Fleet reference]({{% relref "../Reference/fleet.md" %}}) for more details.


<!-- This is the only way I could get the alert to work in a feature code -->
{{% alert title="Note" color="info" %}}This works the same across Fleet resizing and Rolling/Recreate Updates, in that the implementation responds to the
underlying `GameServerSet`'s replicas being shrunk to a value smaller than the number of `Allocated`
`GameServers` it controls. Therefore, this functionality works equally well with a rolling update as it does with an
update strategy that requires at least two `Fleets`.
{{% /alert %}}
