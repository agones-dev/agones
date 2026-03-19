---
title: "Fleet Specification"
linkTitle: "Fleet"
date: 2019-01-03T03:58:52Z
description: "A `Fleet` is a set of warm GameServers that are available to be allocated from."
weight: 20
---

To allocate a `GameServer` from a `Fleet`, use a `GameServerAllocation`.

Like any other Kubernetes resource you describe a `Fleet`'s desired state via a specification written in YAML or JSON to the Kubernetes API. The Agones controller will then change the actual state to the desired state.

A full `Fleet` specification is available below and in the {{< ghlink href="examples/fleet.yaml" >}}example folder{{< /ghlink >}} for reference :


```yaml
apiVersion: "agones.dev/v1"
kind: Fleet
# Fleet Metadata
# {{< k8s-api-version href="#objectmeta-v1-meta" >}}
metadata:
  name: fleet-example
spec:
  # the number of GameServers to keep Ready or Allocated in this Fleet
  replicas: 2
  # defines how GameServers are organised across the cluster.
  # Options include:
  # "Packed" (default) is aimed at dynamic Kubernetes clusters, such as cloud providers, wherein we want to bin pack
  # resources
  # "Distributed" is aimed at static Kubernetes clusters, wherein we want to distribute resources across the entire
  # cluster
  scheduling: Packed
  # a GameServer template - see:
  # https://agones.dev/site/docs/reference/gameserver/ for all the options
  strategy:
    # The replacement strategy for when the GameServer template is changed. Default option is "RollingUpdate",
    # "RollingUpdate" will increment by maxSurge value on each iteration, while decrementing by maxUnavailable on each
    # iteration, until all GameServers have been switched from one version to another.
    # "Recreate" terminates all non-allocated GameServers, and starts up a new set with the new details to replace them.
    type: RollingUpdate
    # Only relevant when `type: RollingUpdate`
    rollingUpdate:
      # the amount to increment the new GameServers by. Defaults to 25%
      maxSurge: 25%
      # the amount to decrements GameServers by. Defaults to 25%
      maxUnavailable: 25%
  # Labels and/or Annotations to apply to overflowing GameServers when the number of Allocated GameServers is more
  # than the desired replicas on the underlying `GameServerSet`
  allocationOverflow:
    labels:
      mykey: myvalue
      version: "" # empty an existing label value
    annotations:
      otherkey: setthisvalue
  # [Stage:Beta]
  # [FeatureFlag:CountsAndLists]
  # Which gameservers in the Fleet are most important to keep around - impacts scale down logic.
  # Now in Beta, and enabled by default
  priorities:
    - type: Counter # Sort by a “Counter”
      key: rooms # The name of the Counter. No impact if no GameServer found.
      order: Descending # Default is "Ascending" so smaller available capacity will be removed first on down scaling.
    - type: List # Sort by a “List”
      key: players # The name of the List. No impact if no GameServer found.
      order: Ascending # Default is "Ascending" so smaller available capacity will be removed first on down scaling.
  template:
    # GameServer metadata
    metadata:
      labels:
        foo: bar
    # GameServer specification
    spec:
      ports:
      - name: default
        portPolicy: Dynamic
        containerPort: 26000
      health:
        initialDelaySeconds: 30
        periodSeconds: 60
      # Parameters for game server sidecar
      sdkServer:
        logLevel: Info
        grpcPort: 9357
        httpPort: 9358
      #
      # [Stage:Beta]
      # [FeatureFlag:CountsAndLists]
      # Counts and Lists provides the configuration for generic (player, room, session, etc.) tracking features.
      # Now in Beta, and enabled by default
      counters:
        rooms:
          count: 0 # Initial value
          capacity: 10
      lists:
        players:
          values: []
      # The GameServer's Pod template
      template:
        spec:
          containers:
          - name: simple-game-server
            image: {{< example-image >}}
```

Since Agones defines a new 
[Custom Resources Definition (CRD)](https://kubernetes.io/docs/concepts/api-extension/custom-resources/) 
we can define a new resource using the kind `Fleet` with the custom group `agones.dev` and API 
version `v1`.

You can use the metadata field to target a specific 
[namespaces](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) but also 
attach specific [annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) 
and [labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) to your resource. 
This is a very common pattern in the Kubernetes ecosystem.

The length of the `name` field of the fleet should be at most 63 characters.

The `spec` field is the actual `Fleet` specification and it is composed as follow:

- `replicas` is the number of `GameServers` to keep Ready or Allocated in this Fleet
- `scheduling` defines how GameServers are organised across the cluster. Affects backing Pod scheduling, as well as scale
                 down mechanics.
                 "Packed" (default) is aimed at dynamic Kubernetes clusters, such as cloud providers, wherein we want to bin pack
                 resources. "Distributed" is aimed at static Kubernetes clusters, wherein we want to distribute resources across the entire
                 cluster. See [Scheduling and Autoscaling]({{< relref "../Advanced/scheduling-and-autoscaling.md" >}}) for more details.
- `strategy` is the `GameServer` replacement strategy for when the `GameServer` template is edited.
  - `type` is replacement strategy for when the GameServer template is changed. Default option is "RollingUpdate", but "Recreate" is also available.
    - `RollingUpdate` will increment by `maxSurge` value on each iteration, while decrementing by `maxUnavailable` on each iteration, until all GameServers have been switched from one version to another.   
    - `Recreate` terminates all non-allocated `GameServers`, and starts up a new set with the new `GameServer` configuration to replace them.
  - `rollingUpdate` is only relevant when `type: RollingUpdate`
    - `maxSurge` is the amount to increment the new GameServers by. Defaults to 25%
    - `maxUnavailable` is the amount to decrements GameServers by. Defaults to 25%
- `allocationOverflow` (Beta, requires `FleetAllocationOverflow` flag) The labels and/or Annotations to apply to 
  GameServers when the number of Allocated GameServers exceeds the desired replicas in the underlying 
  `GameServerSet`.
  - `labels` the map of labels to be applied
  - `annotations` the map of annotations to be applied
  - `Fleet's Scheduling Strategy`: The GameServers associated with the GameServerSet are sorted based on either `Packed` or `Distributed` strategy.
      - `Packed`: Agones maximizes resource utilization by trying to populate nodes that are already in use before allocating GameServers to other nodes.
      - `Distributed`: Agones employs this strategy to spread out GameServer allocations, ensuring an even distribution of GameServers across the available nodes.
- `priorities`: (Beta, requires `CountsAndLists` feature flag): Defines which `GameServers` in the Fleet are most
  important to keep around - impacts scale down logic.
  - `type`: Sort by a "Counter" or a "List".
  - `key`: The name of the Counter or List. If not found on the GameServer, has no impact.
  - `order`: Order: Sort by “Ascending” or “Descending”. “Descending” a bigger available capacity is preferred. “Ascending” would be smaller available capacity is preferred.
- `template` a full `GameServer` configuration template.
   See the [GameServer]({{< relref "gameserver.md" >}}) reference for all available fields.

## Fleet Scale Subresource Specification

Scale subresource is defined for a Fleet. Please refer to [Kubernetes docs](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#subresources).

You can use the following command to scale the fleet with name simple-game-server:

```bash
kubectl scale fleet simple-game-server --replicas=10
```
```
fleet.agones.dev/simple-game-server scaled
```

You can also use [Kubernetes API]({{< ref "/docs/Guides/access-api.md" >}}) to get or update the Replicas count:

```bash
curl http://localhost:8001/apis/agones.dev/v1/namespaces/default/fleets/simple-game-server/scale
```
```
{
  "kind": "Scale",
  "apiVersion": "autoscaling/v1",
  "metadata": {
    "name": "simple-game-server",
    "namespace": "default",
    "selfLink": "/apis/agones.dev/v1/namespaces/default/fleets/simple-game-server/scale",
    "uid": "4dfaa310-2566-11e9-afd1-42010a8a0058",
    "resourceVersion": "292652",
    "creationTimestamp": "2019-01-31T14:41:33Z"
  },
  "spec": {
    "replicas": 10
  },
  "status": {
    "replicas": 10
  }
```

Also exposing a Scale subresource would allow you to configure HorizontalPodAutoscaler and PodDisruptionBudget for a fleet in the future. However these features have not been tested, and are not currently supported - but if you are looking for these features, please be sure to let us know in the [ticket](https://github.com/agones-dev/agones/issues/553). 
