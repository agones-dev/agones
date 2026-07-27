---
title: "Installing Agones on OCI Kubernetes Engine using Terraform"
linkTitle: "OCI"
weight: 30
publishDate: 2024-10-21
description: >
  You can use Terraform to provision an OKE cluster and install Agones on it.
---

## Installation

You can use Terraform to provision your OKE (Oracle Kubernetes Engine) cluster and install Agones on it using the Helm Terraform provider.

An example of the OKE submodule script files can be found here:
 {{< ghlink href="examples/terraform-submodules/oke/" >}}Terraform configuration with Agones submodule{{< /ghlink >}}

Copy these files into a separate folder.

Configure your OCI CLI tool [CLI configure](https://docs.oracle.com/en-us/iaas/Content/API/SDKDocs/cliinstall.htm):
```bash
oci setup config
```

Initialise your terraform:
```bash
terraform init
```

### Creating Cluster

By editing `terraform.auto.tfvars` you can change the parameters that you need to. For instance, the - `kubernetes_version` variable.

Configurable parameters:

- cluster_name - the name of the OKE cluster (default is "agones-cluster")
- cluster_type - the OKE cluster type, basic or enhanced
- agones_version - the version of agones to install (an empty string, which is the default, is the latest version from the [Helm repository](https://agones.dev/chart/stable))
- region - the location of the cluster (default is "us-ashburn-1")
- home_region - the tenancy's home region. Required to perform identity operations
- tenancy_id - the tenancy id of the OCI Cloud Account in which to create the resources
- user_id - the id of the user that terraform will use to create the OCI resources
- api_fingerprint - fingerprint of the API private key to user with OCI API
- api_private_key_path - the path to the OCI API private key
- compartment_id - the compartment id where resources will be created
- ssh_private_key_path - a path on the local filesystem to the SSH private key
- ssh_public_key_path - a path on the local filesystem to the SSH public key
- node_count - count of game server nodes for the default node pool (default is "3")
- log_level - possible values: Fatal, Error, Warn, Info, Debug (default is "info")
- feature_gates - a list of alpha and beta version features to enable. For example, "ContainerPortAllocation=true"

Now you can create an OKE cluster and deploy Agones on OKE:
```bash
terraform apply [-var agones_version="{{< release-version >}}"]
```

Check that you are authenticated against the recently created Kubernetes cluster:
```bash
kubectl get nodes
```

### Uninstall the Agones and delete OKE cluster

Run the following commands to delete all Terraform provisioned resources:
```bash
terraform destroy -target module.helm_agones.helm_release.agones -auto-approve && sleep 60
terraform destroy
```
