---
title: "Best Practices" 
linkTitle: "Best Practices"
date: 2023-05-12T00:00:00Z
weight: 9
description: "Best practices for running Agones in production."
---

## Overview

Running Agones in production takes consideration, from planning your launch to figuring
out the best course of action for cluster and Agones upgrades. On this page, we've collected
some general best practices. We also have cloud specific pages for:

* [Google Kubernetes Engine (GKE)]({{< relref "gke.md" >}})

If you are interested in submitting best practices for your cloud prodiver / on-prem, [please contribute!]({{< relref "Contribute" >}})

## Separation of Agones from GameServer nodes

When running in production, Agones should be scheduled on a dedicated pool of nodes, distinct from where Game Servers
are scheduled for better isolation and resiliency. By default Agones prefers to be scheduled on nodes labeled with
`agones.dev/agones-system=true` and tolerates the node taint `agones.dev/agones-system=true:NoExecute`.
If no dedicated nodes are available, Agones will run on regular nodes. See [taints and tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
for more information about Kubernetes taints and tolerations.

If you are collecting [Metrics]({{< relref "metrics" >}}) using our standard Prometheus installation, see
[the installation guide]({{< relref "metrics#prometheus-installation" >}}) for instructions on configuring a separate node pool for the `agones.dev/agones-metrics=true` taint.

See [Creating a Cluster]({{< relref "Creating Cluster" >}}) for initial set up on your cloud provider.

## Redundant Clusters

### Allocate Across Clusters

Agones supports Multi-cluster Allocation to avoid a single point of failure when allocating game servers. While earlier versions of Agones included a custom multi-cluster allocation solution, the current best practice is to use a **Service Mesh** (e.g., Istio, Linkerd, [Google Cloud Service Mesh](https://cloud.google.com/service-mesh/docs/overview)) to handle allocation traffic between clusters.

By deploying a Service Mesh across your Agones clusters and backend services, you can expose and route traffic to each cluster’s agones-allocator endpoint based on cluster priority, latency, or other criteria.

To implement this approach, refer to the full setup and guidance in the [Multi-cluster Allocation documentation]({{< relref "multi-cluster-allocation" >}}).

You can also explore the [Global Multiplayer Demo](https://github.com/googleforgames/global-multiplayer-demo) for a working example using Google Cloud Service Mesh with Istio.

### Spread

You should consider spreading your game servers in two ways:
* **Across geographic fault domains** ([GCP regions](https://cloud.google.com/compute/docs/regions-zones), [AWS availability zones](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html), separate datacenters, etc.): This is desirable for geographic fault isolation, but also for optimizing client latency to the game server.
* **Within a fault domain**: Kubernetes Clusters are single points of failure. A single misconfigured RBAC rule, an overloaded Kubernetes Control Plane, etc. can prevent new game server allocations, or worse, disrupt existing sessions. Running multiple clusters within a fault domain also allows for [easier upgrades]({{< relref "Upgrading#upgrading-agones-multiple-clusters" >}}).
