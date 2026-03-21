---
title: "Installing Agones on Google Kubernetes Engine using Terraform"
linkTitle: "Google Cloud"
weight: 10
description: >
  You can use Terraform to provision a GKE cluster and install Agones on it.
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

Google Cloud Shell is a shell environment for managing resources hosted on Google Cloud Platform (GCP). Cloud Shell comes preinstalled with the [gcloud][gcloud] and [kubectl][kubectl] command-line tools. `gcloud` provides the primary command-line interface for GCP, and `kubectl` provides the command-line interface for running commands against Kubernetes clusters.

If you prefer using your local shell, you must install the gcloud and kubectl command-line tools in your environment.

[cloud-shell]: https://cloud.google.com/shell/
[gcloud]: https://cloud.google.com/sdk/gcloud/
[kubectl]: https://kubernetes.io/docs/reference/kubectl/

#### Cloud shell

To launch Cloud Shell, perform the following steps:

1. Go to [Google Cloud Platform Console][cloud]
1. From the top-right corner of the console, click the 
   **Activate Google Cloud Shell** button: ![cloud shell](../../../../images/cloud-shell.png)
1. A Cloud Shell session opens inside a frame at the bottom of the console. Use this shell to run `gcloud` and `kubectl` commands.
1. Set a compute zone in your geographical region with the following command. The compute zone will be something like `us-west1-a`. A full list can be found [here][zones].
   ```bash
   gcloud config set compute/zone [COMPUTE_ZONE]
   ```

[cloud]: https://console.cloud.google.com/home/dashboard
[zones]: https://cloud.google.com/compute/docs/regions-zones/#available

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

## Installation

An example configuration can be found here:
 {{< ghlink href="examples/terraform-submodules/gke/module.tf" >}}Terraform configuration with Agones submodule{{< /ghlink >}}.

Copy this file into a local directory where you will execute the terraform commands.

The GKE cluster created from the example configuration will contain 3 Node Pools:

- `"default"` node pool with `"game-server"` tag, containing 4 nodes.
- `"agones-system"` node pool for Agones Controller.
- `"agones-metrics"` for monitoring and metrics collecting purpose.

Configurable parameters:

- project - your Google Cloud Project ID (required)
- name - the name of the GKE cluster (default is "agones-terraform-example")
- agones_version - the version of agones to install (an empty string, which is the default, is the latest version from the [Helm repository](https://agones.dev/chart/stable))
- machine_type - machine type for hosting game servers (default is "e2-standard-4")
- node_count - count of game server nodes for the default node pool (default is "4") 
- enable_image_streaming - whether or not to enable image streaming for the `"default"` node pool (default is true) 
- zone - (Deprecated, use location) the name of the [zone](https://cloud.google.com/compute/docs/regions-zones) you want your cluster to be
  created in (default is "us-west1-c")
- network - the name of the VPC network you want your cluster and firewall rules to be connected to (default is "default")
- subnetwork - the name of the subnetwork in which the cluster's instances are launched. (required when using non default network)
- log_level - possible values: Fatal, Error, Warn, Info, Debug (default is "info")
- feature_gates - a list of alpha and beta version features to enable. For example, "PlayerTracking=true&ContainerPortAllocation=true"
- gameserver_minPort - the lower bound of the port range which gameservers will listen on (default is "7000")
- gameserver_maxPort - the upper bound of the port range which gameservers will listen on (default is "8000")
- gameserver_namespaces - a list of namespaces which will be used to run gameservers (default is `["default"]`). For example `["default", "xbox-gameservers", "mobile-gameservers"]`
- force_update - whether or not to force the replacement/update of resource (default is true, false may be required to prevent immutability errors when updating the configuration)
- location - the name of the [location](https://cloud.google.com/compute/docs/regions-zones) you want your cluster  to be created in (default is "us-west1-c")
- autoscale - whether you want to enable autoscale for the gameserver nodepool (default is false)
- min_node_count - the minimum number of nodes for a nodepool when autoscale is enabled (default is "1")
- max_node_count - the maximum number of nodes for a nodepool when autoscale is enabled (default is "5")
- enable_agones_metrics_nodepool - whether you want to enable `"agones-metrics"` for the gameserver nodepool (default is false)

{{% alert title="Warning" color="warning" %}}
On the lines that read `source = "git::https://github.com/agones-dev/agones.git//install/terraform/modules/gke/?ref=main"`
make sure to change `?ref=main` to match your targeted Agones release, as Terraform modules can change between
releases.

For example, if you are targeting {{< release-branch >}}, then you will want to have 
`source = "git::https://github.com/agones-dev/agones.git//install/terraform/modules/gke/?ref={{< release-branch >}}"`
as your source.
{{% /alert %}}

### Creating the cluster

In the directory where you created `module.tf`, run:
```bash
terraform init
```

This will cause terraform to clone the Agones repository and use the `./install/terraform` folder as the starting point of
the Agones submodule, which contains all necessary Terraform configuration files.

Next, make sure that you can authenticate using gcloud:
```bash
gcloud auth application-default login
```
#### Option 1: Creating the cluster in the default VPC
To create your GKE cluster in the default VPC just specify the project variable.

```bash
terraform apply -var project="<YOUR_GCP_ProjectID>"
```

#### Option 2: Creating the cluster in a custom VPC
To create the cluster in a custom VPC you must specify the project, network and subnetwork variables.

```bash
terraform apply -var project="<YOUR_GCP_ProjectID>" -var network="<YOUR_NETWORK_NAME>" -var subnetwork="<YOUR_SUBNETWORK_NAME>"
```

To verify that the cluster was created successfully, set up your kubectl credentials:
```bash
gcloud container clusters get-credentials --zone us-west1-c agones-terraform-example
```

Then check that you have access to the Kubernetes cluster:
```bash
kubectl get nodes
```

You should have 6 nodes in `Ready` state.

### Uninstall the Agones and delete GKE cluster

To delete all resources provisioned by Terraform:
```bash
terraform destroy -var project="<YOUR_GCP_ProjectID>"
```

## Next Steps

- [Confirm Agones is up and running]({{< relref "../confirm.md" >}})
