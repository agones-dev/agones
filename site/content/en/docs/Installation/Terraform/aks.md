---
title: "Installing Agones on Azure Kubernetes Service using Terraform"
linkTitle: "Azure"
weight: 20
description: >
  You can use Terraform to provision an AKS cluster and install Agones on it.
---

## Installation

Install `az` utility by following [these instructions](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest).

The example of AKS submodule configuration could be found here:
 {{< ghlink href="examples/terraform-submodules/aks/module.tf" >}}Terraform configuration with Agones submodule{{< /ghlink >}}

Copy `module.tf` file into a separate folder.

Log in to Azure CLI:
```bash
az login
```

Configure your terraform:
```bash
terraform init
```

Create a service principal and configure its access to Azure resources: 
```bash
az ad sp create-for-rbac
```

Now you can deploy your cluster (use values from the above command output):
```bash
terraform apply -var client_id="<appId>" -var client_secret="<password>"
```

Once you created all resources on AKS you can get the credentials so that you can use `kubectl` to configure your cluster:
```bash
az aks get-credentials --resource-group agonesRG --name test-cluster
```

Check that you have access to the Kubernetes cluster:
```bash
kubectl get nodes
```

Configurable parameters:

- log_level - possible values: Fatal, Error, Warn, Info, Debug (default is "info")
- cluster_name - the name of the AKS cluster (default is "agones-terraform-example")
- agones_version - the version of agones to install (an empty string, which is the default, is the latest version from the [Helm repository](https://agones.dev/chart/stable))
- machine_type - node machine type for hosting game servers (default is "Standard_D2_v2")
- disk_size - disk size of the node
- region - the location of the cluster
- node_count - count of game server nodes for the default node pool (default is "4")
- feature_gates - a list of alpha and beta version features to enable. For example, "ContainerPortAllocation=true"
- gameserver_minPort - the lower bound of the port range which gameservers will listen on (default is "7000")
- gameserver_maxPort - the upper bound of the port range which gameservers will listen on (default is "8000")
- gameserver_namespaces - a list of namespaces which will be used to run gameservers (default is `["default"]`). For example `["default", "xbox-gameservers", "mobile-gameservers"]`
- force_update - whether or not to force the replacement/update of resource (default is true, false may be required to prevent immutability errors when updating the configuration)

## Uninstall the Agones and delete AKS cluster

Run next command to delete all Terraform provisioned resources:
```bash
terraform destroy
```

## Reference
Details on how you can authenticate your AKS terraform provider using official [instructions](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/guides/service_principal_client_secret).

## Next Steps

- [Confirm Agones is up and running]({{< relref "../confirm.md" >}})
