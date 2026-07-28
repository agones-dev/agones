---
title: "Installing Agones on AWS Elastic Kubernetes Service using Terraform"
linkTitle: "AWS"
weight: 20
publishDate: 2020-01-21
description: >
  You can use Terraform to provision an EKS cluster and install Agones on it.
---

## Installation

You can use Terraform to provision your Amazon EKS (Elastic Kubernetes Service) cluster and install Agones on it using the Helm Terraform provider.

An example of the EKS submodule config file can be found here:
 {{< ghlink href="examples/terraform-submodules/eks/module.tf" >}}Terraform configuration with Agones submodule{{< /ghlink >}}

Copy this file into a separate folder.

Configure your AWS CLI tool [CLI configure](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html):
```bash
aws configure
```

Initialise your terraform:
```bash
terraform init
```

### Creating Cluster

By editing `modules.tf` you can change the parameters that you need to. For instance, the - `machine_type` variable.

Configurable parameters:

- cluster_name - the name of the EKS cluster (default is "agones-terraform-example")
- agones_version - the version of agones to install (an empty string, which is the default, is the latest version from the [Helm repository](https://agones.dev/chart/stable))
- machine_type - EC2 instance type for hosting game servers (default is "t2.large")
- region - the location of the cluster (default is "us-west-2")
- node_count - count of game server nodes for the default node pool (default is "4")
- log_level - possible values: Fatal, Error, Warn, Info, Debug (default is "info")
- feature_gates - a list of alpha and beta version features to enable. For example, "ContainerPortAllocation=true"
- gameserver_minPort - the lower bound of the port range which gameservers will listen on (default is "7000")
- gameserver_maxPort - the upper bound of the port range which gameservers will listen on (default is "8000")
- gameserver_namespaces - a list of namespaces which will be used to run gameservers (default is `["default"]`). For example `["default", "xbox-gameservers", "mobile-gameservers"]`
- force_update - whether or not to force the replacement/update of resource (default is true, false may be required to prevent immutability errors when updating the configuration)

Now you can create an EKS cluster and deploy Agones on EKS:
```bash
terraform apply [-var agones_version="{{< release-version >}}"]
```

After deploying the cluster with Agones, you can get or update your kubeconfig by using:
```bash
aws eks --region us-west-2 update-kubeconfig --name agones-cluster
```

With the following output:
```
Added new context arn:aws:eks:us-west-2:601646756426:cluster/agones-cluster to /Users/user/.kube/config
```

Switch `kubectl` context to the recently created one:
```bash
kubectl config use-context arn:aws:eks:us-west-2:601646756426:cluster/agones-cluster
```

Check that you are authenticated against the recently created Kubernetes cluster:
```bash
kubectl get nodes
```

### Uninstall the Agones and delete EKS cluster

Run the following commands to delete all Terraform provisioned resources:
```bash
terraform destroy -target module.helm_agones.helm_release.agones -auto-approve && sleep 60
terraform destroy
```
