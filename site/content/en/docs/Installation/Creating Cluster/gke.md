---
title: "Google Kubernetes Engine"
linkTitle: "Google Cloud"
weight: 10
description: >
    Follow these steps to create a [Google Kubernetes Engine (GKE)](https://cloud.google.com/kubernetes-engine/)
    cluster for your Agones install.
---

## Before you begin

Take the following steps to enable the Kubernetes Engine API:

1. Visit the [Kubernetes Engine][kubernetes] page in the Google Cloud Platform Console.
1. Create or select a project.
1. Wait for the API and related services to be enabled. This can take several minutes.
1. [Enable billing][billing] for your project.
  * If you are not an existing GCP user, you may be able to enroll for a $300 US [Free Trial][trial] credit.

[kubernetes]: https://console.cloud.google.com/kubernetes/list
[billing]: https://support.google.com/cloud/answer/6293499#enable-billing
[trial]: https://cloud.google.com/free/

### Choosing a shell

To complete this quickstart, we can use either [Google Cloud Shell][cloud-shell] or a local shell.

Google Cloud Shell is a shell environment for managing resources hosted on Google Cloud Platform (GCP). Cloud Shell comes preinstalled with the [`gcloud`][gcloud] and [`kubectl`][kubectl] command-line tools. `gcloud` provides the primary command-line interface for GCP, and `kubectl` provides the command-line interface for running commands against Kubernetes clusters.

If you prefer using your local shell, you must install the `gcloud` and `kubectl` command-line tools in your environment.

[cloud-shell]: https://cloud.google.com/shell/
[gcloud]: https://cloud.google.com/sdk/gcloud/
[kubectl]: https://kubernetes.io/docs/reference/kubectl/

#### Cloud shell

To launch Cloud Shell, perform the following steps:

1. Go to [Google Cloud Platform Console][cloud]
1. From the top-right corner of the console, click the
   **Activate Google Cloud Shell** button: ![cloud shell](../../../../images/cloud-shell.png)
1. A Cloud Shell session opens inside a frame at the bottom of the console. Use this shell to run `gcloud` and `kubectl` commands.

[cloud]: https://console.cloud.google.com/home/dashboard

#### Local shell

To install `gcloud` and `kubectl`, perform the following steps:

1. [Install the Google Cloud SDK][gcloud-install], which includes the `gcloud` command-line tool.
1. Initialize some default configuration by running the following command.
   * When asked `Do you want to configure a default Compute Region and Zone? (Y/n)?`, enter `Y` and choose a zone in your geographical region of choice.
   ```bash
   gcloud init
   ```
1. Install the `kubectl` command-line tool by running the following command:
   ```bash
   gcloud components install kubectl
   ```

[gcloud-install]: https://cloud.google.com/sdk/docs/quickstarts

### Choosing a Regional or Zonal Cluster

You will need to pick a geographical region or zone where you want to deploy your cluster, and whether to
create a [regional or zonal cluster](https://cloud.google.com/kubernetes-engine/docs/concepts/types-of-clusters).
We recommend using a Regional cluster, as the zonal GKE control plane can go down temporarily to adjust for cluster resizing,
[automatic upgrades](https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-upgrades) and
[repairs](https://cloud.google.com/kubernetes-engine/docs/concepts/maintenance-windows-and-exclusions#repairs).

After choosing a cluster type, [choose a region or zone][regions]. The region you chose is `COMPUTE_REGION` below.
(Note that if you chose a zone, replace `--region=[COMPUTE_REGION]` with `--zone=[COMPUTE_ZONE]` in commands below.)

[regions]: https://cloud.google.com/compute/docs/regions-zones/#available

### Choosing a Release Channel and Optional Version

We recommend using the `regular` release channel, which offers a balance between stability and freshness.
If you'd like to read more, see our guide on [Release Channels]({{< ref "/docs/Guides/Best Practices/gke.md#release-channels" >}}).
The release channel you chose is `RELEASE_CHANNEL` below.

(Optional) During cluster creation, to set a specific available version in the release channel, use the `--cluster-version=[VERSION]` flag, e.g. `--cluster-version={{% gke-example-cluster-version %}}`. Be sure to choose a [version supported by Agones]({{< relref "../../Installation/#usage-requirements" >}}). (If you rely on release channels, the latest Agones release [should be supported]({{< relref "../../Installation/#agones-and-kubernetes-supported-versions" >}}) by the default versions of all channels.)

### Choosing a GKE cluster mode

A [cluster][cluster] consists of at least one *control plane* machine and multiple worker machines called *nodes*. In Google Kubernetes Engine, nodes are [Compute Engine virtual machine][vms] instances that run the Kubernetes processes necessary to make them part of the cluster.

Agones supports both GKE Standard mode and GKE Autopilot mode.
Agones `GameServer` and `Fleet` manifests that work on Standard are compatible
on Autopilot with some constraints, described in the following section. We recommend
running GKE Autopilot clusters, if you meet the constraints.

If you choose GKE Autopilot mode, then you should skip the following sections on
`NodePool` configuration. GKE Autopilot handles this for you.

You can't convert existing Standard clusters to Autopilot; create new Autopilot
clusters instead.


#### Agones on GKE Autopilot

Autopilot is GKE's fully-managed mode. GKE configures, maintains, scales, and
upgrades nodes for you, which can reduce your maintenance and operating
overhead. You only pay for the resources requested by your running Pods, and
you don't pay for unused node capacity or Kubernetes system workloads.

This section describes the Agones-specific considerations in Autopilot
clusters. For a general comparison between Autopilot and Standard, refer to
[Choose a GKE mode of operation](https://cloud.google.com/kubernetes-engine/docs/concepts/choose-cluster-mode).

Autopilot nodes are, by default, optimized for most workloads. If some of your
workloads have broad compute requirements such as Arm architecture or a minimum
CPU platform, you can also choose a
[compute class](https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-compute-classes)
that meets that requirement. However, if you have specialized hardware needs
that require fine-grained control over machine configuration, consider using
GKE Standard.

Agones on Autopilot has pre-configured opinionated constraints. Evaluate
whether these constraints impact your workloads:

*  **Operating system:** No Windows containers.
*  **Resource requests:** Autopilot has pre-determined
   [minimum Pod resource requests](https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-resource-requests#min-max-requests).
   If your game servers require less than those minimums, use GKE Standard.
*  **[Scheduling strategy]({{<ref "/docs/Advanced/scheduling-and-autoscaling#fleet-scheduling">}}):**
   `Packed` is supported, which is the Agones default. `Distributed` is not
   supported.
* **Node pools:** Autopilot manages nodes for you. If you require node pools with different configurations, use GKE Standard.
*  **[Host port policy]({{<ref "/docs/reference/agones_crd_api_reference#agones.dev/v1.GameServerPort">}}):** `Dynamic` is supported, which is the Agones default.
   `Static` is not supported. `Passthrough` is supported with the feature gate `AutopilotPassthroughPort` set to `true`.
*  **[Port range]({{<ref "/docs/reference/agones_crd_api_reference#agones.dev/v1.GameServerPort">}}):** `default` is supported, which is the Agones default.
   Additional port ranges are not supported.
*  **Seccomp profile:** Agones sets the seccomp profile to `Unconfined` to
   avoid unexpected container creation delays that might occur because
   Autopilot enables the
   [`RuntimeDefault` seccomp profile](https://cloud.google.com/kubernetes-engine/docs/concepts/seccomp-in-gke).
*  **[Pod disruption policy]({{<ref "/docs/Advanced/controlling-disruption#eviction-api">}}):**
   `eviction.safe: Never` is supported, which is the Agones
   default. `eviction.safe: Always` is supported. `eviction.safe: OnUpgrade` is
   not supported. If your game sessions exceed one hour, refer to
   [Considerations for long sessions]({{<ref "/docs/Advanced/controlling-disruption#considerations-for-long-sessions">}}).

### Choosing a GCP network

By default, `gcloud` and the Cloud Console use the VPC named `default` for all new resources. If you
plan to create a [dual-stack IPv4/IPv6 cluster][dual-stack] cluster, special considerations need to
be made. Dual-stack clusters require a [dual-stack subnet][subnet-types], which are only supported in
[_custom mode_][vpc-mode] VPC networks. For a new dual-stack cluster, you can either:

* create a new _custom mode_ VPC,

* or if you wish to continue using the `default` network, you must [switch it to _custom mode_][switch-mode].
After switching a network to custom mode, you will need to manually manage subnets within the `default` VPC.

Once you have a _custom mode_ VPC, you will need to choose whether to use an existing subnet or create a
new one - read [VPC-native guide on creating a dual-stack cluster][dual-stack], but don't create the cluster
just yet - we'll create the cluster later in this guide. To use the network and/or subnetwork you just created,
you'll need to add `--network` and `--subnetwork`, and for GKE Standard, possibly `--stack-type` and
`--ipv6-access-type`, depending on whether you created the subnet simultaneously with the cluster.

[dual-stack]: https://cloud.google.com/kubernetes-engine/docs/how-to/alias-ips#dual-stack
[subnet-types]: https://cloud.google.com/vpc/docs/subnets#subnet-types
[vpc-mode]: https://cloud.google.com/vpc/docs/vpc#subnet-ranges
[switch-mode]: https://cloud.google.com/vpc/docs/create-modify-vpc-networks#switch-network-mode

## Creating the firewall

We need a firewall to allow UDP traffic to nodes tagged as `game-server` via ports 7000-8000. These firewall rules apply to cluster nodes you will create in the
next section.

```bash
gcloud compute firewall-rules create gke-agones-game-server-firewall \
  --allow udp:7000-8000 \
  --target-tags game-server \
  --description "Firewall to allow game server udp traffic"
```

## Creating the cluster

Create a GKE cluster in which you'll install Agones. You can use
[GKE Standard mode](#create-a-standard-mode-cluster-for-agones)
or [GKE Autopilot mode](#create-an-autopilot-mode-cluster-for-agones).
You can read more about choosing a [cluster mode above](#choosing-a-gke-cluster-mode).

### Create an Autopilot mode cluster for Agones

1. Choose a [Release Channel]({{<ref "/docs/Guides/Best Practices/gke.md#release-channels" >}}) (Autopilot clusters must be on a Release Channel).

1. Create the cluster:

    ```bash
    gcloud container clusters create-auto [CLUSTER_NAME] \
      --region=[COMPUTE_REGION] \
      --release-channel=[RELEASE_CHANNEL] \
      --autoprovisioning-network-tags=game-server
    ```

Replace the following:
* `[CLUSTER_NAME]`: The name of your cluster.
* `[COMPUTE_REGION]`: the GCP region to create the cluster in.
* `[RELEASE_CHANNEL]`: one of `rapid`, `regular`, or `stable`, chosen [above](#choosing-a-release-channel-and-optional-version). The default is `regular`.

Flag explanations:
* `--region`: The compute region [you chose above](#choosing-a-gke-cluster-mode).
* `--release-channel`: The release channel [you chose above](#choosing-a-release-channel-and-optional-version).
* `--autoprovisioning-network-tags`: Defines the tags that will be attached to new nodes in the cluster. This is to grant access through ports via the [firewall created above](#creating-the-firewall).


### Create a Standard mode cluster for Agones

Create the cluster:
```bash
gcloud container clusters create [CLUSTER_NAME] \
  --region=[COMPUTE_REGION] \
  --release-channel=[RELEASE_CHANNEL] \
  --tags=game-server \
  --scopes=gke-default \
  --num-nodes=4 \
  --enable-image-streaming \
  --machine-type=e2-standard-4
```

Replace the following:
* `[CLUSTER_NAME]`: The name of the cluster you want to create
* `[COMPUTE_REGION]`: The GCP region to create the cluster in, [chosen above](#choosing-a-gke-cluster-mode)
* `[RELEASE_CHANNEL]`: The GKE release channel, [chosen above](#choosing-a-release-channel-and-optional-version)

Flag explanations:
* `--region`: The compute region [you chose above](#choosing-a-gke-cluster-mode).
* `--release-channel`: The release channel [you chose above](#choosing-a-release-channel-and-optional-version).
* `--tags`: Defines the tags that will be attached to new nodes in the cluster. This is to grant access through ports via the [firewall created above](#creating-the-firewall).
* `--scopes`: Defines the Oauth scopes required by the nodes.
* `--num-nodes`: The number of nodes to be created in each of the cluster's zones. Default: 4. Depending on the needs of your game, this parameter should be adjusted.
* `--enable-image-streaming`: Use [Image streaming](https://cloud.google.com/kubernetes-engine/docs/how-to/image-streaming) to pull container images, which leads to significant improvements in initialization times. [Limitations](https://cloud.google.com/kubernetes-engine/docs/how-to/image-streaming#limitations) apply to enable this feature.
* `--machine-type`: The type of machine to use for nodes. Default: `e2-standard-4`. Depending on the needs of your game, you may wish to [have smaller or larger machines](https://cloud.google.com/compute/docs/machine-types).

#### (Optional) Creating a dedicated node pool

Create a [dedicated node pool](https://cloud.google.com/kubernetes-engine/docs/concepts/node-pools)
for the Agones resources to be installed in. If you skip this step, the Agones controllers will
share the default node pool with your game servers, which is fine for experimentation but not
recommended for a production deployment. You must also skip this step if you are using GKE Autopilot.

```bash
gcloud container node-pools create agones-system \
  --cluster=[CLUSTER_NAME] \
  --region=[COMPUTE_REGION] \
  --node-taints agones.dev/agones-system=true:NoExecute \
  --node-labels agones.dev/agones-system=true \
  --num-nodes=1 \
  --machine-type=e2-standard-4
```

Replace the following:
* `[CLUSTER_NAME]`: The name of the cluster you created
* `[COMPUTE_REGION]`: The GCP region to create the cluster in, [chosen above](#choosing-a-gke-cluster-mode)

Flag explanations:
* `--cluster`: The name of the cluster you created.
* `--region`: The compute region [you chose above](#choosing-a-gke-cluster-mode).
* `--node-taints`: The Kubernetes taints to automatically apply to nodes in this node pool.
* `--node-labels`: The Kubernetes labels to automatically apply to nodes in this node pool.
* `--num-nodes`: The number of nodes per cluster zone. For regional clusters, `--num-nodes=1` creates one node in 3 separate zones in the region, giving you faster recovery time in the event of a node failure.
* `--machine-type`: The type of machine to use for nodes. Default: `e2-standard-4`. Depending on the needs of your game, you may wish to [have smaller or larger machines](https://cloud.google.com/compute/docs/machine-types).

#### (Optional) Creating a metrics node pool

Create a node pool for [Metrics]({{< relref "../../Guides/metrics.md" >}}) if you want to monitor the
 Agones system using Prometheus with Grafana or Cloud Logging and Monitoring. You must skip this step
if you are using GKE Autopilot.

```bash
gcloud container node-pools create agones-metrics \
  --cluster=[CLUSTER_NAME] \
  --region=[COMPUTE_REGION] \
  --node-taints agones.dev/agones-metrics=true:NoExecute \
  --node-labels agones.dev/agones-metrics=true \
  --num-nodes=1 \
  --machine-type=e2-standard-4
```

Replace the following:
* `[CLUSTER_NAME]`: The name of the cluster you created
* `[COMPUTE_REGION]`: The GCP region to create the cluster in, [chosen above](#choosing-a-gke-cluster-mode)

Flag explanations:
* `--cluster`: The name of the cluster you created.
* `--region`: The compute region [you chose above](#choosing-a-gke-cluster-mode).
* `--node-taints`: The Kubernetes taints to automatically apply to nodes in this node pool.
* `--node-labels`: The Kubernetes labels to automatically apply to nodes in this node pool.
* `--num-nodes`: The number of nodes per cluster zone. For regional clusters, `--num-nodes=1` creates one node in 3 separate zones in the region, giving you faster recovery time in the event of a node failure.
* `--machine-type`: The type of machine to use for nodes. Default: `e2-standard-4`. Depending on the needs of your game, you may wish to [have smaller or larger machines](https://cloud.google.com/compute/docs/machine-types).

#### (Optional) Creating a node pool for Windows

If you run game servers on Windows, you
need to create a dedicated node pool for those servers. Windows Server 2019 (`WINDOWS_LTSC_CONTAINERD`) is the recommended image for Windows
game servers. Note that GKE Autopilot does not support Windows nodes.

{{% alert title="Warning" color="warning" %}}
Running `GameServers` on Windows nodes is currently Alpha. Feel free to file feedback
through [Github issues](https://github.com/agones-dev/agones/issues).
{{% /alert %}}

```bash
gcloud container node-pools create windows \
  --cluster=[CLUSTER_NAME] \
  --region=[COMPUTE_REGION] \
  --image-type WINDOWS_LTSC_CONTAINERD \
  --machine-type e2-standard-4 \
  --num-nodes=4
```

Replace the following:
* `[CLUSTER_NAME]`: The name of the cluster you created
* `[COMPUTE_REGION]`: The GCP region to create the cluster in, [chosen above](#choosing-a-gke-cluster-mode)

Flag explanations:
* `--cluster`: The name of the cluster you created.
* `--region`: The compute region [you chose above](#choosing-a-gke-cluster-mode).
* `--image-type`: The image type of the instances in the node pool - `WINDOWS_LTSC_CONTAINERD` in this case.
* `--machine-type`: The type of machine to use for nodes. Default: `e2-standard-4`. Depending on the needs of your game, you may wish to [have smaller or larger machines](https://cloud.google.com/compute/docs/machine-types).
* `--num-nodes`: The number of nodes per cluster zone. For regional clusters, `--num-nodes=1` creates one node in 3 separate zones in the region, giving you faster recovery time in the event of a node failure.

## Setting up cluster credentials

`gcloud container clusters create` configurates credentials for `kubectl` automatically. If you ever lose those, run:

```bash
gcloud container clusters get-credentials [CLUSTER_NAME] --region=[COMPUTE_REGION]
```

[cluster]: https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture
[vms]: https://cloud.google.com/compute/docs/instances/

## Next Steps

* Continue to [Install Agones]({{< relref "../Install Agones/_index.md" >}}).
* Have a look at the
   [Best practices for running Agones on Google Kubernetes Engine]({{< ref "/docs/Guides/Best Practices/gke.md" >}}).
