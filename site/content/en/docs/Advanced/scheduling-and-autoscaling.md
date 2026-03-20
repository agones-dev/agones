---
title: "Scheduling and Autoscaling"
date: 2019-01-03T05:45:05Z
weight: 10
description: >
  Scheduling and autoscaling go hand in hand, as where in the cluster `GameServers` are provisioned
  impacts how to autoscale fleets up and down (or if you would even want to)
---

## Cluster Autoscaler

Kubernetes has a [cluster node autoscaler that works with a wide variety of cloud providers](https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler).

The default scheduling strategy (`Packed`) is designed to work with the Kubernetes autoscaler out of the box.

The autoscaler will automatically add Nodes to the cluster when `GameServers` don't have room to be scheduled on the
clusters, and then scale down when there are empty Nodes with no `GameServers` running on them.

This means that scaling `Fleets` up and down can be used to control the size of the cluster, as the cluster autoscaler
will adjust the size of the cluster to match the resource needs of one or more `Fleets` running on it.

To enable and configure autoscaling on your cloud provider, check their [connector implementation](https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler/cloudprovider),
or their cloud specific documentation.

### Google Kubernetes Engine
* [Administering Clusters: Autoscaling a Cluster](https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-autoscaler)
* [Cluster Autoscaler](https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-autoscaler)

### Amazon Elastic Kubernetes Service
* [Cluster Autoscaler for EKS](https://docs.aws.amazon.com/eks/latest/userguide/cluster-autoscaler.html)

### Azure Kubernetes Service
* [Cluster Autoscaler on Azure Kubernetes Service (AKS) - Preview](https://docs.microsoft.com/en-us/azure/aks/autoscaler)

## Fleet Autoscaling

Fleet autoscaling is the only type of autoscaling that exists in Agones. It is currently available as a
buffer autoscaling strategy or as a webhook driven strategy, such that you can provide your own autoscaling logic.

Have a look at the [Create a Fleet Autoscaler]({{< relref "../Getting Started/create-fleetautoscaler.md" >}}) quickstart, the
[Create a Webhook Fleet Autoscaler]({{< relref "../Getting Started/create-webhook-fleetautoscaler.md" >}}) quickstart,
and the [Fleet Autoscaler Specification]({{< relref "../Reference/fleetautoscaler.md" >}}) for details.

## Autoscaling Concepts

To facilitate autoscaling, we need to combine several concepts and functionality, as described below.

### Allocation Scheduling

Allocation scheduling refers to the order in which `GameServers`, and specifically their backing `Pods` are chosen
from across the Kubernetes cluster within a given `Fleet` when [allocation]({{< relref "../Getting Started/create-fleet.md#4-allocate-a-game-server-from-the-fleet"  >}}) occurs.

### Pod Scheduling

Each `GameServer` is backed by a Kubernetes [`Pod`](https://kubernetes.io/docs/concepts/workloads/pods/pod/). Pod scheduling
refers to the strategy that is in place that determines which node in the Kubernetes cluster the Pod is assigned to,
when it is created.

### Fleet Scale Down Strategy

Fleet Scale Down strategy refers to the order in which the `GameServers` that belong to a `Fleet` are deleted,
when Fleets are shrunk in size.

## Fleet Scheduling

There are two scheduling strategies for Fleets - each designed for different types of Kubernetes Environments.

### Packed

```yaml
apiVersion: "agones.dev/v1"
kind: Fleet
metadata:
  name: simple-game-server
spec:
  replicas: 100
  scheduling: Packed
  template:
    spec:
      ports:
      - containerPort: 7654
      template:
        spec:
          containers:
          - name: simple-game-server
            image: {{% example-image %}}
```

This is the *default* Fleet scheduling strategy. It is designed for dynamic Kubernetes environments, wherein you wish
to scale up and down as load increases or decreases, such as in a Cloud environment where you are paying
for the infrastructure you use.

It attempts to _pack_ as much as possible into the smallest set of nodes, to make
scaling infrastructure down as easy as possible.

This affects the Cluster autoscaler, Allocation Scheduling, Pod Scheduling and Fleet Scale Down Scheduling.

#### Cluster Autoscaler

When using the "Packed" strategy, Agones will ensure that the Cluster Autoscaler doesn't attempt to evict and move `GameServer` `Pods` onto new Nodes during
gameplay.

If a gameserver can tolerate [being evicted](https://kubernetes.io/docs/concepts/scheduling-eviction/api-eviction/#how-api-initiated-eviction-works)
(generally in combination with setting an appropriate graceful termination period on the gameserver pod) and you
want the Cluster Autoscaler to compact your cluster by evicting game servers when it would allow the Cluster
Autoscaler to reduce the number of nodes in the cluster, [Controlling Disruption]({{< relref "controlling-disruption.md" >}}) describes
how to choose the `.eviction` setting appropriate for your `GameServer` or `Fleet`.

#### Allocation Scheduling Strategy

Under the "Packed" strategy, allocation will prioritise allocating `GameServers` to nodes that are running on
Nodes that already have allocated `GameServers` running on them.

##### Counter and List Priorities

{{< beta title="Counters And Lists" gate="CountsAndLists" >}}

When `priorities` are configured on a `GameServerAllocation`, they act as a tie-breaker for `GameServers` on Nodes with equal utilisation. `Ascending` order (the default) prioritises `GameServers` with less available `Counter` or `List` capacity, and `Descending` order prioritises those with greater available capacity. See the [GameServerAllocation]({{< relref "../Reference/gameserverallocation.md" >}}) reference for configuration details.

#### Pod Scheduling Strategy

Under the "Packed" strategy, Pods will be scheduled using the [`PodAffinity`](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity-beta-feature)
with a `preferredDuringSchedulingIgnoredDuringExecution` affinity with [hostname](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#interlude-built-in-node-labels)
topology. This attempts to group together `GameServer` Pods within as few nodes in the cluster as it can.

{{% alert title="Note" color="info" %}}
The default Kubernetes scheduler doesn't do a perfect job of packing, but it's a good enough job for what we need -
  at least at this stage.
{{% /alert %}}

#### Fleet Scale Down Strategy

With the "Packed" strategy, Fleets will remove `Ready` `GameServers` from Nodes with the _least_ number of `Ready` and
`Allocated` `GameServers` on them. Attempting to empty Nodes so that they can be safely removed.

##### Counter and List Priorities

{{< beta title="Counters And Lists" gate="CountsAndLists" >}}

When `priorities` are configured on the `Fleet`, they act as a tie-breaker for `GameServers` on the same `Node` with equal total counts. `Ascending` order (the default) removes `GameServers` with less available `Counter` or `List` capacity first, and `Descending` order removes those with greater available capacity first. See the [Fleet]({{< relref "../Reference/fleet.md" >}}) reference for configuration details.

### Distributed

```yaml
apiVersion: "agones.dev/v1"
kind: Fleet
metadata:
  name: simple-game-server
spec:
  replicas: 100
  scheduling: Distributed
  template:
    spec:
      ports:
      - containerPort: 7654
      template:
        spec:
          containers:
          - name: simple-game-server
            image: {{% example-image %}}
```

This Fleet scheduling strategy is designed for static Kubernetes environments, such as when you are running Kubernetes
on bare metal, and the cluster size rarely changes, if at all.

This attempts to distribute the load across the entire cluster as much as possible, to take advantage of the static
size of the cluster.

{{% alert title="Note" color="info" %}}
`Distributed` scheduling does not set
a [`PodAffinity`](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity-beta-feature)
on `GameServer` `Pods`, and instead assumes that the default scheduler for your cluster will distribute the 
`GameServer` `Pods` across the cluster by default.

If your default scheduler does not do this, you may wish to set your own `PodAffinity` to spread the load across the
cluster, or update the default scheduler to provide this functionality.
{{% /alert %}}

This affects Allocation Scheduling, Pod Scheduling and Fleet Scale Down Scheduling.

#### Cluster Autoscaler

Since this strategy is not aimed at clusters that autoscale, this strategy does nothing for the cluster autoscaler.

#### Allocation Scheduling Strategy

Under the "Distributed" strategy, allocation will prioritise allocating `GameServers` to nodes that have the least
number of allocated `GameServers` on them.

##### Counter and List Priorities

{{< beta title="Counters And Lists" gate="CountsAndLists" >}}

When `priorities` are configured on a `GameServerAllocation`, instead of random `GameServer` selection, `GameServers` are sorted and selected in priority order. `Ascending` order (the default) prioritises `GameServers` with less available `Counter` or `List` capacity, and `Descending` order prioritises those with greater available capacity. See the [GameServerAllocation]({{< relref "../Reference/gameserverallocation.md" >}}) reference for configuration details.

#### Pod Scheduling Strategy

Under the "Distributed" strategy, `Pod` scheduling is provided by the default Kubernetes scheduler, which will attempt
to distribute the `GameServer` `Pods` across as many nodes as possible.

#### Fleet Scale Down Strategy

With the "Distributed" strategy, Fleets will remove `Ready` `GameServers` from Nodes with at random, to ensure
a distributed load is maintained.

##### Counter and List Priorities

{{< beta title="Counters And Lists" gate="CountsAndLists" >}}

When `priorities` are configured on the `Fleet`, instead of random removal, `GameServers` are removed in priority order (with creation timestamp as a tie-breaker, newest first). `Ascending` order (the default) removes `GameServers` with less available `Counter` or `List` capacity first, and `Descending` order removes those with greater available capacity first. See the [Fleet]({{< relref "../Reference/fleet.md" >}}) reference for configuration details.
