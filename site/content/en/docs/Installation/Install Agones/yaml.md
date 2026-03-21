---
title: "Install Agones using YAML"
linkTitle: "YAML"
weight: 10
description: >
  We can install Agones to the cluster using an install.yaml file.
---

## Installing Agones

{{% alert title="Warning" color="warning" %}}
Installing Agones with the `install.yaml` file will use pre-generated, well known TLS
certificates stored in this repository for securing Kubernetes webhooks communication.

For production workloads, we **strongly** recommend using the
[helm installation]({{< relref "helm.md" >}}) which allows you to generate
new, unique certificates or provide your own certificates. Alternatively,
you can use `helm template` as described [below]({{< relref "yaml.md" >}}#customizing-your-install)
to generate a custom yaml installation file with unique certificates.
{{% /alert %}}

Installing Agones using the pre-generated `install.yaml` file is the quickest,
simplest way to get Agones up and running in your Kubernetes cluster:

```bash
kubectl create namespace agones-system
kubectl apply --server-side -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/install/yaml/install.yaml
```

You can also find the `install.yaml` in the latest `agones-install` zip from the [releases](https://github.com/agones-dev/agones/releases) archive.

### Customizing your install

To change the [configurable parameters](https://agones.dev/site/docs/installation/install-agones/helm/#configuration)
in the `install.yaml` file, you can use `helm template` to generate a custom file locally
without needing to use helm to install Agones into your cluster.

The following example sets the `featureGates` and `generateTLS` helm parameters
and creates a customized `install-custom.yaml` file (note that the `pull`
command was introduced in Helm version 3):

```bash
helm pull --untar https://agones.dev/chart/stable/agones-{{< release-version >}}.tgz && \
cd agones && \
helm template agones-manual --namespace agones-system  . \
  --set agones.controller.generateTLS=false \
  --set agones.allocator.generateTLS=false \
  --set agones.allocator.generateClientTLS=false \
  --set agones.crds.cleanupOnDelete=false \
  --set agones.featureGates="PlayerTracking=true" \
  > install-custom.yaml
```

## Uninstalling Agones

To uninstall/delete the `Agones` deployment and delete `agones-system` namespace:

```bash
kubectl delete fleets --all --all-namespaces
kubectl delete gameservers --all --all-namespaces
kubectl delete -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/install/yaml/install.yaml
kubectl delete namespace agones-system
```

Note: It may take a couple of minutes until all resources described in `install.yaml` file are deleted.

## Next Steps

- [Confirm Agones is up and running]({{< relref "../confirm.md" >}})
