---
title: "High Availability Agones"
date: 2023-02-10
weight: 20
description: >
  Learn how to configure your Agones services for high availability and resiliency to disruptions.
publishDate: 2023-02-28
---


## High Availability for Agones Controller


The `agones-controller` responsibility is split up into `agones-controller`, which enacts the Agones control loop, and `agones-extensions`, which acts as a service endpoint for webhooks and the allocation extension API. Splitting these responsibilities allows the `agones-extensions` pod to be **horizontally scaled**, making the Agones control plane **highly available** and more **resilient to disruption**.

Multiple `agones-controller` pods enabled, with a primary controller selected via leader election. Having multiple `agones-controller` minimizes downtime of the service from pod disruptions such as deployment updates, autoscaler evictions, and crashes.

## Extension Pod Configurations 

The `agones-extensions` binary has a similar `helm` configuration to `agones-controller`, see [here]({{< relref "/docs/Installation/Install Agones/helm.md" >}}). If you previously overrode `agones.controller.*` settings, you may need to override the same `agones.extensions.*` setting.

To change `controller.numWorkers` to 200 from 100 values and through the use of `helm --set`, add the following to the `helm` command:

{{% alert color="warning" %}} Important: This will not have any effect on any `extensions` values! {{% /alert %}}
```
 ...
 --set agones.controller.numWorkers=200
 ...
```

An important configuration to note is the PodDisruptionBudget fields, `agones.extensions.pdb.minAvailable` and `agones.extensions.pdb.maxUnavailable`. Currently, the `agones.extensions.pdb.minAvailable` field is set to 1. 

## Deployment Considerations


Leader election will automatically be enabled and `agones.controller.replicas` is > 1. [`agones.controller.replicas`]({{< relref "/docs/Installation/Install Agones/helm.md#configuration" >}}) defaults to 2.

The default configuration now deploys 2 `agones-controller` pods and 2 `agones-extensions` pods, replacing the previous single `agones-controller` pod setup. For example:

```
NAME                                 READY   STATUS    RESTARTS   AGE
agones-allocator-78c6b8c79-h9nqc     1/1     Running   0          23h
agones-allocator-78c6b8c79-l2bzp     1/1     Running   0          23h
agones-allocator-78c6b8c79-rw75j     1/1     Running   0          23h
agones-controller-fbf944f4-vs9xx     1/1     Running   0          23h
agones-controller-fbf944f4-sjk3t     1/1     Running   0          23h
agones-extensions-5648fc7dcf-hm6lk   1/1     Running   0          23h
agones-extensions-5648fc7dcf-qbc6h   1/1     Running   0          23h
agones-ping-5b9647874-2rrl6          1/1     Running   0          27h
agones-ping-5b9647874-rksgg          1/1     Running   0          27h
```

The number of replicas for `agones-extensions` can be set using helm variable [`agones.extensions.replicas`]({{< relref "/docs/Installation/Install Agones/helm.md#configuration" >}}), but the default is `2`. 

## High Availability for Supporting Services

The `agones-allocator` service is also deployed with multiple replicas by default. This makes allocation requests resilient to pod failures, rolling updates, and node shutdowns, since other allocator pods can continue serving traffic.

Similarly, the `agones-ping` service runs multiple replicas, allowing health checks and ping responses to remain available even if an individual pod or node becomes unavailable.

The example deployment above shows three `agones-allocator` pods and two `agones-ping` pods running simultaneously to provide redundancy.

## Topology Spread Considerations

For additional resiliency, Agones controller and extension pods can be configured with Kubernetes `topologySpreadConstraints` to distribute replicas across different nodes and zones.

This reduces the risk of multiple replicas being scheduled onto the same node or failure domain, helping ensure that a single node shutdown does not take down all replicas of a service.

We expect the aggregate memory consumption of the pods will be slightly higher than the previous singleton pod, but as the responsibilities are now split across the pods, the aggregate CPU consumption should also be similar.

## Feature Design

Please see [HA Agones](https://github.com/agones-dev/agones/issues/2797).