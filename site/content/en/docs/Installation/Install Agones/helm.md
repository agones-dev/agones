---
title: "Install Agones using Helm"
linkTitle: "Helm"
weight: 20
description: >
  Install Agones on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.
---

## Prerequisites

- [Helm](https://helm.sh/) package manager 3.2.3+
- [Supported Kubernetes Cluster]({{< relref "../_index.md#usage-requirements" >}})

## Helm 3

### Installing the Chart

To install the chart with the release name `my-release` using our stable helm repository:

```bash
helm repo add agones https://agones.dev/chart/stable
helm repo update
helm install my-release --namespace agones-system --create-namespace agones/agones
```

_We recommend installing Agones in its own namespaces, such as `agones-system` as shown above.
If you want to use a different namespace, you can use the helm `--namespace` parameter to specify._

When running in production, Agones should be scheduled on a dedicated pool of nodes, distinct from
where Game Servers are scheduled for better isolation and resiliency. By default Agones prefers to
be scheduled on nodes labeled with `agones.dev/agones-system=true` and tolerates node taint
`agones.dev/agones-system=true:NoExecute`. If no dedicated nodes are available, Agones will
run on regular nodes, but that's not recommended for production use. For instructions on setting up
a dedicated node pool for Agones, see the [Agones installation instructions]({{< relref "../_index.md" >}})
for your preferred environment.

The command deploys Agones on the Kubernetes cluster with the default configuration. The
[configuration](#configuration) section lists the parameters that can be configured during installation.

The Agones chart uses a [Helm Schema](https://helm.sh/docs/topics/charts/#schema-files) to validate
fields set by the user. In the event this validation schema marks a valid edge case as invalid,
please [file a bug](https://github.com/agones-dev/agones/issues), and you can still attempt a
Helm install or Helm upgrade with the Helm flag `--skip-schema-validation`.

{{% alert title="Tip" color="info" %}}
List all releases using `helm list --all-namespaces`
{{% /alert %}}

### Namespaces

By default Agones is configured to work with game servers deployed in the `default` namespace. If
you are planning to use another namespace you can configure Agones via the parameter `gameservers.namespaces`.

For example to use `default` **and** `xbox` namespaces:

```bash
kubectl create namespace xbox
helm install my-release agones/agones --set "gameservers.namespaces={default,xbox}" --namespace agones-system
```

{{% alert title="Note" color="info" %}}
You need to create your namespaces before installing Agones.
{{% /alert %}}

If you want to add a new namespace afterward upgrade your release:

```bash
kubectl create namespace ps4
helm upgrade my-release agones/agones --reuse-values --set "gameservers.namespaces={default,xbox,ps4}" --namespace agones-system
```

### Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
helm uninstall my-release --namespace=agones-system
```

## RBAC

By default, `agones.rbacEnabled` is set to true. This enables RBAC support in Agones and must be true
if RBAC is enabled in your cluster.

The chart will take care of creating the required service accounts and roles for Agones.

If you have RBAC disabled, or to put it another way, ABAC enabled, you should set this value to `false`.

## Configuration

The following tables lists the configurable parameters of the Agones chart and their default values.

### General

| Parameter                        | Description                                                                                                                                                                                                             | Default         |
| -------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------- |
| `agones.featureGates`            | A URL query encoded string of Flags to enable/disable e.g. `Example=true&OtherThing=false`. Any value accepted by [strconv.ParseBool(string)](https://golang.org/pkg/strconv/#ParseBool) can be used as a boolean value | \`\`            |
| `agones.rbacEnabled`             | Creates RBAC resources. Must be set for any cluster configured with RBAC                                                                                                                                                | `true`          |
| `agones.registerWebhooks`        | Registers the webhooks used for the admission controller                                                                                                                                                                | `true`          |
| `agones.registerApiService`      | Registers the apiservice(s) used for the Kubernetes API extension                                                                                                                                                       | `true`          |
| `agones.registerServiceAccounts` | Attempts to create service accounts for the controllers                                                                                                                                                                 | `true`          |
| `agones.createPriorityClass`     | Attempts to create priority classes for the controllers                                                                                                                                                                 | `true`          |
| `agones.priorityClassName`       | Name of the priority classes to create                                                                                                                                                                                  | `agones-system` |
| `agones.requireDedicatedNodes`   | Forces Agones system components to be scheduled on dedicated nodes, only applies to the GKE Standard without node auto-provisioning                                                                                     | `false`         |

### Custom Resource Definitions

| Parameter                     | Description                                                                                                                                                                                                                         | Default |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `agones.crds.install`         | Install the CRDs with this chart. Useful to disable if you want to subchart (since crd-install hook is broken), so you can copy the CRDs into your own chart.                                                                       | `true`  |
| `agones.crds.cleanupOnDelete` | Run the pre-delete hook to delete all GameServers and their backing Pods when deleting the helm chart, so that all CRDs can be removed on chart deletion                                                                            | `true`  |
| `agones.crds.cleanupJobTTL`   | The number of seconds for Kubernetes to delete the associated Job and Pods of the pre-delete hook after it completes, regardless if the Job is successful or not. Set to `0` to disable cleaning up the Job or the associated Pods. | `60`    |

### Metrics

| Parameter                                    | Description                                                                                                                                                                                                         | Default |
| -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `agones.metrics.prometheusServiceDiscovery`  | Adds annotations for Prometheus ServiceDiscovery (and also Strackdriver)                                                                                                                                            | `true`  |
| `agones.metrics.prometheusEnabled`           | Enables controller metrics on port `8080` and path `/metrics`                                                                                                                                                       | `true`  |
| `agones.metrics.stackdriverEnabled`          | Enables Stackdriver exporter of controller metrics                                                                                                                                                                  | `false` |
| `agones.metrics.stackdriverProjectID`        | This overrides the default gcp project id for use with stackdriver                                                                                                                                                  | \`\`    |
| `agones.metrics.stackdriverLabels`           | A set of default labels to add to all stackdriver metrics generated in form of key value pair (`key=value,key2=value2`). By default metadata are automatically added using Kubernetes API and GCP metadata enpoint. | \`\`    |
| `agones.metrics.serviceMonitor.interval`     | Default scraping interval for ServiceMonitor                                                                                                                                                                        | `30s`   |

| Parameter                                        | Description                                      | Default |
| ------------------------------------------------ | ------------------------------------------------ | ------- |
| `agones.metrics.serviceMonitor.additionalLabels` | Extra labels to provide for the ServiceMonitors  |   `{}`  |

### Service Accounts

| Parameter                                      | Description                                                                                                                    | Default             |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ | ------------------- |
| `agones.serviceaccount.controller.name`        | Service account name for the controller                                                                                        | `agones-controller` |
| `agones.serviceaccount.controller.annotations` | [Annotations][annotations] added to the Agones controller service account                                                      | `{}`                |
| `agones.serviceaccount.sdk.name`               | Service account name for the sdk                                                                                               | `agones-sdk`        |
| `agones.serviceaccount.sdk.annotations`        | A map of namespaces to maps of [Annotations][annotations] added to the Agones SDK service account for the specified namespaces | `{}`                |
| `agones.serviceaccount.allocator.name`         | Service account name for the allocator                                                                                         | `agones-allocator`  |
| `agones.serviceaccount.allocator.annotations`  | [Annotations][annotations] added to the Agones allocator service account                                                       | `{}`                |

### Container Images

| Parameter                            | Description                                                                                                                             | Default                                   |
| ------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------- |
| `agones.image.registry`              | Global image registry for all the Agones system images                                                                                  | `us-docker.pkg.dev/agones-images/release` |
| `agones.image.tag`                   | Global image tag for all images                                                                                                         | `{{< release-version >}}`                 |
| `agones.image.controller.name`       | Image name for the controller                                                                                                           | `agones-controller`                       |
| `agones.image.controller.pullPolicy` | Image pull policy for the controller                                                                                                    | `IfNotPresent`                            |
| `agones.image.controller.pullSecret` | Image pull secret for the controller, allocator, sdk and ping image. Should be created both in `agones-system` and `default` namespaces | \`\`                                      |
| `agones.image.sdk.name`              | Image name for the sdk                                                                                                                  | `agones-sdk`                              |
| `agones.image.sdk.tag`               | Image tag for the sdk                                                                                                                   | value of `agones.image.tag`               |
| `agones.image.sdk.cpuRequest`        | The [cpu request][cpu-constraints] for sdk server container                                                                             | `30m`                                     |
| `agones.image.sdk.cpuLimit`          | The [cpu limit][cpu-constraints] for the sdk server container                                                                           | `0` (none)                                |
| `agones.image.sdk.memoryRequest`     | The [memory request][memory-constraints] for sdk server container                                                                       | `0` (none)                                |
| `agones.image.sdk.memoryLimit`       | The [memory limit][memory-constraints] for the sdk server container                                                                     | `0` (none)                                |
| `agones.image.sdk.alwaysPull`        | Tells if the sdk image should always be pulled                                                                                          | `false`                                   |
| `agones.image.ping.name`             | Image name for the ping service                                                                                                         | `agones-ping`                             |
| `agones.image.ping.tag`              | Image tag for the ping service                                                                                                          | value of `agones.image.tag`               |
| `agones.image.ping.pullPolicy`       | Image pull policy for the ping service                                                                                                  | `IfNotPresent`                            |
| `agones.image.extensions.name`       | Image name for extensions                                                                                                               | `agones-extensions`                       |
| `agones.image.extensions.pullPolicy` | Image pull policy for extensions                                                                                                        | `IfNotPresent`                            |

### Agones Controller

| Parameter                                                | Description                                                                                                                                                                                                                         | Default |
| -------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `agones.controller.replicas`                             | The number of replicas to run in the `agones-controller` deployment.                                                                                                                                                                | `2`     |
| `agones.controller.pdb.minAvailable`                     | Description of the number of pods from that set that must still be available after the eviction, even in the absence of the evicted pod. Can be either an absolute number or a percentage. Mutually Exclusive with `maxUnavailable` | `1`     |
| `agones.controller.pdb.maxUnavailable`                   | Description of the number of pods from that set that can be unavailable after the eviction. It can be either an absolute number or a percentage Mutually Exclusive with `minAvailable`                                              | \`\`    |
| `agones.controller.http.port`                            | Port to use for liveness probe service and metrics                                                                                                                                                                                  | `8080`  |
| `agones.controller.healthCheck.initialDelaySeconds`      | Initial delay before performing the first probe (in seconds)                                                                                                                                                                        | `3`     |
| `agones.controller.healthCheck.periodSeconds`            | Seconds between every liveness probe (in seconds)                                                                                                                                                                                   | `3`     |
| `agones.controller.healthCheck.failureThreshold`         | Number of times before giving up (in seconds)                                                                                                                                                                                       | `3`     |
| `agones.controller.healthCheck.timeoutSeconds`           | Number of seconds after which the probe times out (in seconds)                                                                                                                                                                      | `1`     |
| `agones.controller.resources`                            | Controller [resource requests/limit][resources]                                                                                                                                                                                     | `{}`    |
| `agones.controller.generateTLS`                          | Set to true to generate TLS certificates or false to provide your own certificates                                                                                                                                                  | `true`  |
| `agones.controller.tlsCert`                              | Custom TLS certificate provided as a string                                                                                                                                                                                         | \`\`    |
| `agones.controller.tlsKey`                               | Custom TLS private key provided as a string                                                                                                                                                                                         | \`\`    |
| `agones.controller.nodeSelector`                         | Controller [node labels][nodeSelector] for pod assignment                                                                                                                                                                           | `{}`    |
| `agones.controller.tolerations`                          | Controller [toleration][toleration] labels for pod assignment                                                                                                                                                                       | `[]`    |
| `agones.controller.affinity`                             | Controller [affinity][affinity] settings for pod assignment                                                                                                                                                                         | `{}`    |
| `agones.controller.labels`                               | [Labels][labels] added to the Agones controller pods                                                                                                                                                                                | `{}`    |
| `agones.controller.annotations`                          | [Annotations][annotations] added to the Agones controller pods                                                                                                                                                                      | `{}`    |
| `agones.controller.updateStrategy`                             | The [strategy](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#strategy) to apply to the controller deployment                                                                                                 | `{}`           |
| `agones.controller.numWorkers`                           | Number of workers to spin per resource type                                                                                                                                                                                         | `100`   |
| `agones.controller.apiServerQPS`                         | Maximum sustained queries per second that controller should be making against API Server                                                                                                                                            | `400`   |
| `agones.controller.apiServerQPSBurst`                    | Maximum burst queries per second that controller should be making against API Server                                                                                                                                                | `500`   |
| `agones.controller.logLevel`                             | Agones Controller Log level. Log only entries with that severity and above                                                                                                                                                          | `info`  |
| `agones.controller.persistentLogs`                       | Store Agones controller logs in a temporary volume attached to a container for debugging                                                                                                                                            | `true`  |
| `agones.controller.persistentLogsSizeLimitMB`            | Maximum total size of all Agones container logs in MB                                                                                                                                                                               | `10000` |
| `agones.controller.disableSecret`                        | **Deprecated**. Use `agones.extensions.disableSecret` instead. Disables the creation of any allocator secrets. If true, you MUST provide the `{agones.releaseName}-cert` secrets before installation.                               | `false` |
| `agones.controller.customCertSecretPath`                 | Remap cert-manager path to server.crt and server.key                                                                                                                                                                                | `{}`    |
| `agones.controller.allocationApiService.annotations`     | **Deprecated**. Use `agones.extensions.allocationApiService.annotations` instead. [Annotations][annotations] added to the Agones apiregistration                                                                                    | `{}`    |
| `agones.controller.allocationApiService.disableCaBundle` | **Deprecated**. Use `agones.extensions.allocationApiService.disableCaBundle` instead. Disable ca-bundle so it can be injected by cert-manager.                                                                                      | `false` |
| `agones.controller.validatingWebhook.annotations`        | **Deprecated**. Use `agones.extensions.validatingWebhook.annotations` instead. [Annotations][annotations] added to the Agones validating webhook                                                                                    | `{}`    |
| `agones.controller.validatingWebhook.disableCaBundle`    | **Deprecated**. Use `agones.extensions.validatingWebhook.disableCaBundle` instead. Disable ca-bundle so it can be injected by cert-manager                                                                                          | `false` |
| `agones.controller.mutatingWebhook.annotations`          | **Deprecated**. Use `agones.extensions.mutatingWebhook.annotations` instead. [Annotations][annotations] added to the Agones mutating webhook                                                                                        | `{}`    |
| `agones.controller.mutatingWebhook.disableCaBundle`      | **Deprecated**. Use `agones.extensions.mutatingWebhook.disableCaBundle` instead. Disable ca-bundle so it can be injected by cert-manager                                                                                            | `false` |
| `agones.controller.allocationBatchWaitTime`              | Wait time between each allocation batch when performing allocations in controller mode                                                                                                                                              | `500ms` |
| `agones.controller.topologySpreadConstraints`            | Ensures better resource utilization and high availability by evenly distributing Pods in the agones-system namespace                                                                                                                | `{}`    |
| `agones.controller.maxCreationParallelism`               | Maximum number of parallelizing creation calls in GSS controller                                                                                                                                                                    | `16`    |
| `agones.controller.maxGameServerCreationsPerBatch`       | Maximum number of GameServer creation calls per batch                                                                                                                                                                               | `64`    |
| `agones.controller.maxDeletionParallelism`               | Maximum number of parallelizing deletion calls in GSS                                                                                                                                                                               | `64`    |
| `agones.controller.maxGameServerDeletionsPerBatch`       | Maximum number of GameServer deletion calls per batch                                                                                                                                                                               | `64`    |
| `agones.controller.maxPodPendingCount`                   | Maximum number of pending pods per game server set                                                                                                                                                                                  | `5000`  |

### SDK Server

| Parameter                                                | Description                                                                                                                                                                                                                         | Default        |
|----------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------|
| `agones.sdkServer.requestsRateLimit`                | The GameServer sidecar requests rate limit                                                                                                                                                                                          | `500ms` |

### Ping Service


| Parameter                                                | Description                                                                                                                                                                                                                         | Default        |
|----------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------|
| `agones.ping.install`                                    | Whether to install the [ping service][ping]                                                                                                                                                                                         | `true`         |
| `agones.ping.replicas`                                   | The number of replicas to run in the deployment                                                                                                                                                                                     | `2`            |
| `agones.ping.http.expose`                                | Expose the http ping service via a Service                                                                                                                                                                                          | `true`         |
| `agones.ping.http.response`                              | The string response returned from the http service                                                                                                                                                                                  | `ok`           |
| `agones.ping.http.port`                                  | The port to expose on the service                                                                                                                                                                                                   | `80`           |
| `agones.ping.http.serviceType`                           | The [Service Type][service] of the HTTP Service                                                                                                                                                                                     | `LoadBalancer` |
| `agones.ping.http.nodePort`                              | Static node port to use for HTTP ping service. (Only applies when `agones.ping.http.serviceType` is `NodePort`.)                                                                                                                    | `0`            |
| `agones.ping.http.loadBalancerIP`                        | The [Load Balancer IP][loadBalancer] of the HTTP Service load balancer. Only works if the Kubernetes provider supports this option.                                                                                                 | \`\`           |
| `agones.ping.http.externalTrafficPolicy`                 | The `externalTrafficPolicy` for the Agones http ping service                                                                                                                                                                        | `Cluster`      |
| `agones.ping.http.loadBalancerSourceRanges`              | The [Load Balancer SourceRanges][loadBalancer] of the HTTP Service load balancer. Only works if the Kubernetes provider supports this option.                                                                                       | `[]`           |
| `agones.ping.http.ipFamilyPolicy`                        | The `ipFamilyPolicy` setting for the Load Balancer. Kubernetes default is `SingleStack`. Set to `RequireDualStack` or `PreferDualStack` for dual-stack IPv4/IPv6 support if the Kubernetes provider supports this option.           | \`\`           |
| `agones.ping.http.ipFamilies`                            | The `ipFamilies` setting for the Load Balancer. Set to `IPv4` (default) or `IPv6` or both in combination with dual-stack `ipFamilyPolicy` setting.                                                                                  | `[]`           |
| `agones.ping.http.annotations`                           | [Annotations][annotations] added to the Agones ping http service                                                                                                                                                                    | `{}`           |
| `agones.ping.udp.expose`                                 | Expose the udp ping service via a Service                                                                                                                                                                                           | `true`         |
| `agones.ping.udp.rateLimit`                              | Number of UDP packets the ping service handles per instance, per second, per sender                                                                                                                                                 | `20`           |
| `agones.ping.udp.port`                                   | The port to expose on the service                                                                                                                                                                                                   | `80`           |
| `agones.ping.udp.serviceType`                            | The [Service Type][service] of the UDP Service                                                                                                                                                                                      | `LoadBalancer` |
| `agones.ping.udp.nodePort`                               | Static node port to use for UDP ping service. (Only applies when `agones.ping.udp.serviceType` is `NodePort`.)                                                                                                                      | `0`            |
| `agones.ping.udp.loadBalancerIP`                         | The [Load Balancer IP][loadBalancer] of the UDP Service load balancer. Only works if the Kubernetes provider supports this option.                                                                                                  | \`\`           |
| `agones.ping.udp.loadBalancerSourceRanges`               | The [Load Balancer SourceRanges][loadBalancer] of the UDP Service load balancer. Only works if the Kubernetes provider supports this option.                                                                                        | `[]`           |
| `agones.ping.udp.annotations`                            | [Annotations][annotations] added to the Agones ping udp service                                                                                                                                                                     | `{}`           |
| `agones.ping.udp.ipFamilyPolicy`                         | The `ipFamilyPolicy` setting for the Load Balancer. Kubernetes default is `SingleStack`. Set to `RequireDualStack` or `PreferDualStack` for dual-stack IPv4/IPv6 support if the Kubernetes provider supports this option.           | \`\`           |
| `agones.ping.udp.ipFamilies`                             | The `ipFamilies` setting for the Load Balancer. Set to `IPv4` (default) or `IPv6` or both in combination with dual-stack `ipFamilyPolicy` setting.                                                                                  | `[]`           |
| `agones.ping.healthCheck.initialDelaySeconds`            | Initial delay before performing the first probe (in seconds)                                                                                                                                                                        | `3`            |
| `agones.ping.healthCheck.periodSeconds`                  | Seconds between every liveness probe (in seconds)                                                                                                                                                                                   | `3`            |
| `agones.ping.healthCheck.failureThreshold`               | Number of times before giving up (in seconds)                                                                                                                                                                                       | `3`            |
| `agones.ping.healthCheck.timeoutSeconds`                 | Number of seconds after which the probe times out (in seconds)                                                                                                                                                                      | `1`            |
| `agones.ping.resources`                                  | Ping pods [resource requests/limit][resources]                                                                                                                                                                                      | `{}`           |
| `agones.ping.nodeSelector`                               | Ping [node labels][nodeSelector] for pod assignment                                                                                                                                                                                 | `{}`           |
| `agones.ping.tolerations`                                | Ping [toleration][toleration] labels for pod assignment                                                                                                                                                                             | `[]`           |
| `agones.ping.affinity`                                   | Ping [affinity][affinity] settings for pod assignment                                                                                                                                                                               | `{}`           |
| `agones.ping.annotations`                                | [Annotations][annotations] added to the Agones ping pods                                                                                                                                                                            | `{}`           |
| `agones.ping.updateStrategy`                             | The [strategy](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#strategy) to apply to the allocator deployment                                                                                                 | `{}`           |
| `agones.ping.pdb.enabled`                                | Set to `true` to enable the creation of a [PodDisruptionBudget](https://kubernetes.io/docs/tasks/run-application/configure-pdb/) for the ping deployment                                                                            | `false`        |
| `agones.ping.pdb.minAvailable`                           | Description of the number of pods from that set that must still be available after the eviction, even in the absence of the evicted pod. Can be either an absolute number or a percentage. Mutually Exclusive with `maxUnavailable` | `1`            |
| `agones.ping.pdb.maxUnavailable`                         | Description of the number of pods from that set that can be unavailable after the eviction. It can be either an absolute number or a percentage Mutually Exclusive with `minAvailable`                                              | \`\`           |
| `agones.ping.topologySpreadConstraints`                  | Ensures better resource utilization and high availability by evenly distributing Pods in the agones-system namespace                                                                                                                | `{}`           |


### Allocator Service


| Parameter                                             | Description                                                                                                                                                                                                                         | Default                            |
| ----------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------- |
| `agones.allocator.apiServerQPS`                       | Maximum sustained queries per second that an allocator should be making against API Server                                                                                                                                          | `400`                              |
| `agones.allocator.apiServerQPSBurst`                  | Maximum burst queries per second that an allocator should be making against API Server                                                                                                                                              | `500`                              |
| `agones.allocator.remoteAllocationTimeout`            | Remote allocation call timeout.                                                                                                                                                                                                     | `10s`                              |
| `agones.allocator.totalRemoteAllocationTimeout`       | Total remote allocation timeout including retries.                                                                                                                                                                                  | `30s`                              |
| `agones.allocator.logLevel`                           | Agones Allocator Log level. Log only entries with that severity and above                                                                                                                                                           | `info`                             |
| `agones.allocator.install`                            | Whether to install the [allocator service][allocator]                                                                                                                                                                               | `true`                             |
| `agones.allocator.replicas`                           | The number of replicas to run in the deployment                                                                                                                                                                                     | `3`                                |
| `agones.allocator.service.name`                       | Service name for the allocator                                                                                                                                                                                                      | `agones-allocator`                 |
| `agones.allocator.service.serviceType`                | The [Service Type][service] of the HTTP Service                                                                                                                                                                                     | `LoadBalancer`                     |
| `agones.allocator.service.clusterIP`                  | The [Cluster IP][clusterIP] of the Agones allocator. If you want [Headless Service][headless-service] for Agones Allocator, you can set `None` to clusterIP.                                                                        | \`\`                               |
| `agones.allocator.service.loadBalancerIP`             | The [Load Balancer IP][loadBalancer] of the Agones allocator load balancer. Only works if the Kubernetes provider supports this option.                                                                                             | \`\`                               |
| `agones.allocator.service.loadBalancerSourceRanges`   | The [Load Balancer SourceRanges][loadBalancer] of the Agones allocator load balancer. Only works if the Kubernetes provider supports this option.                                                                                   | `[]`                               |
| `agones.allocator.service.annotations`                | [Annotations][annotations] added to the Agones allocator service                                                                                                                                                                    | `{}`                               |
| `agones.allocator.service.externalTrafficPolicy`      | The `externalTrafficPolicy` for the Agones allocator service                                                                                                                                                                        | `Cluster`                          |
| `agones.allocator.service.ipFamilyPolicy`             | The `ipFamilyPolicy` setting for the Load Balancer. Kubernetes default is `SingleStack`. Set to `RequireDualStack` or `PreferDualStack` for dual-stack IPv4/IPv6 support if the Kubernetes provider supports this option.           | \`\`                               |
| `agones.allocator.service.ipFamilies`                 | The `ipFamilies` setting for the Load Balancer. Set to `IPv4` (default) or `IPv6` or both in combination with dual-stack `ipFamilyPolicy` setting.                                                                                  | `[]`                               |
| `agones.allocator.service.http.enabled`               | If true the [allocator service][allocator] will respond to [REST requests][rest-requests]                                                                                                                                           | `true`                             |
| `agones.allocator.service.http.appProtocol`           | The `appProtocol` to set on the Service for the http allocation port. If left blank, no value is set.                                                                                                                               | ``                                 |
| `agones.allocator.service.http.port`                  | The port that is exposed externally by the [allocator service][allocator] for [REST requests][rest-requests]                                                                                                                        | `443`                              |
| `agones.allocator.service.http.portName`              | The name of exposed port                                                                                                                                                                                                            | `http`                             |
| `agones.allocator.service.http.targetPort`            | The port that is used by the allocator pod to listen for [REST requests][rest-requests]. Note that the allocator server cannot bind to low numbered ports.                                                                          | `8443`                             |
| `agones.allocator.service.http.nodePort`              | If the ServiceType is set to "NodePort", this is the NodePort that the allocator http service is exposed on.                                                                                                                        | `30000-32767`                      |
| `agones.allocator.service.http.unallocatedStatusCode` | HTTP status code to return when no GameServer is available for allocation. This setting allows for custom responses when a game server allocation fails, offering flexibility in handling these situations.                         | `429`                              |
| `agones.allocator.service.grpc.enabled`               | If true the [allocator service][allocator] will respond to [gRPC requests][grpc-requests]                                                                                                                                           | `true`                             |
| `agones.allocator.service.grpc.port`                  | The port that is exposed externally by the [allocator service][allocator] for [gRPC requests][grpc-requests]                                                                                                                        | `443`                              |
| `agones.allocator.service.grpc.portName`              | The name of exposed port                                                                                                                                                                                                            | ``                                 |
| `agones.allocator.service.grpc.appProtocol`           | The `appProtocol` to set on the Service for the gRPC allocation port. If left blank, no value is set.                                                                                                                               | ``                                 |
| `agones.allocator.service.grpc.nodePort`              | If the ServiceType is set to "NodePort", this is the NodePort that the allocator gRPC service is exposed on.                                                                                                                        | `30000-32767`                      |
| `agones.allocator.service.grpc.targetPort`            | The port that is used by the allocator pod to listen for [gRPC requests][grpc-requests]. Note that the allocator server cannot bind to low numbered ports.                                                                          | `8443`                             |
| `agones.allocator.generateClientTLS`                  | Set to true to generate client TLS certificates or false to provide certificates in `certs/allocator/allocator-client.default/*`                                                                                                    | `true`                             |
| `agones.allocator.generateTLS`                        | Set to true to generate TLS certificates or false to provide your own certificates                                                                                                                                                  | `true`                             |
| `agones.allocator.disableMTLS`                        | Turns off client cert authentication for incoming connections to the allocator.                                                                                                                                                     | `false`                            |
| `agones.allocator.disableTLS`                         | Turns off TLS security for incoming connections to the allocator.                                                                                                                                                                   | `false`                            |
| `agones.allocator.disableSecretCreation`              | Disables the creation of any allocator secrets. If true, you MUST provide the `allocator-tls`, `allocator-tls-ca`, and `allocator-client-ca` secrets before installation.                                                           | `false`                            |
| `agones.allocator.tlsCert`                            | Custom TLS certificate provided as a string                                                                                                                                                                                         | \`\`                               |
| `agones.allocator.tlsKey`                             | Custom TLS private key provided as a string                                                                                                                                                                                         | \`\`                               |
| `agones.allocator.clientCAs`                          | A map of secret key names to allowed client CA certificates provided as strings                                                                                                                                                     | `{}`                               |
| `agones.allocator.tolerations`                        | Allocator [toleration][toleration] labels for pod assignment                                                                                                                                                                        | `[]`                               |
| `agones.allocator.affinity`                           | Allocator [affinity][affinity] settings for pod assignment                                                                                                                                                                          | `{}`                               |
| `agones.allocator.annotations`                        | [Annotations][annotations] added to the Agones allocator pods                                                                                                                                                                       | `{}`                               |
| `agones.allocator.resources`                          | Allocator pods [resource requests/limit][resources]                                                                                                                                                                                 | `{}`                               |
| `agones.allocator.labels`                             | [Labels][labels] Added to the Agones Allocator pods                                                                                                                                                                                 | `{}`                               |
| `agones.allocator.readiness.initialDelaySeconds`      | Initial delay before performing the first probe (in seconds)                                                                                                                                                                        | `3`                                |
| `agones.allocator.readiness.periodSeconds`            | Seconds between every liveness probe (in seconds)                                                                                                                                                                                   | `3`                                |
| `agones.allocator.readiness.failureThreshold`         | Number of times before giving up (in seconds)                                                                                                                                                                                       | `3`                                |
| `agones.allocator.nodeSelector`                       | Allocator [node labels][nodeSelector] for pod assignment                                                                                                                                                                            | `{}`                               |
| `agones.allocator.serviceMetrics.name`                | Second Service name for the allocator                                                                                                                                                                                               | `agones-allocator-metrics-service` |
| `agones.allocator.serviceMetrics.annotations`         | [Annotations][annotations] added to the Agones allocator second Service                                                                                                                                                             | `{}`                               |
| `agones.allocator.serviceMetrics.http.port`           | The port that is exposed within cluster by the [allocator service][allocator] for http requests                                                                                                                                     | `8080`                             |
| `agones.allocator.serviceMetrics.http.portName`       | The name of exposed port                                                                                                                                                                                                            | `http`                             |
| `agones.allocator.allocationBatchWaitTime`            | Wait time between each allocation batch when performing allocations in allocator mode                                                                                                                                               | `500ms`                            |
| `agones.allocator.updateStrategy`                     | The [strategy](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#strategy) to apply to the ping deployment                                                                                                      | `{}`                               |
| `agones.allocator.pdb.enabled`                        | Set to `true` to enable the creation of a [PodDisruptionBudget](https://kubernetes.io/docs/tasks/run-application/configure-pdb/) for the allocator deployment                                                                       | `false`                            |
| `agones.allocator.pdb.minAvailable`                   | Description of the number of pods from that set that must still be available after the eviction, even in the absence of the evicted pod. Can be either an absolute number or a percentage. Mutually Exclusive with `maxUnavailable` | `1`                                |
| `agones.allocator.pdb.maxUnavailable`                 | Description of the number of pods from that set that can be unavailable after the eviction. It can be either an absolute number or a percentage. Mutually Exclusive with `minAvailable`                                             | \`\`                               |
| `agones.allocator.topologySpreadConstraints`          | Ensures better resource utilization and high availability by evenly distributing Pods in the agones-system namespace                                                                                                                | `{}`                               |


### Extensions

| Parameter                                                | Description                                                                                                                                                                                                                       | Default |
| -------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `agones.extensions.hostNetwork`                          | Determines if the Agones extensions should operate in hostNetwork mode. If running in hostNetwork mode, you should change `agones.extensions.http.port` and `agones.extensions.webhooks.port` to an available port.               | `false` |
| `agones.extensions.http.port`                            | Port to use for liveness probe service and metrics                                                                                                                                                                                | `8080`  |
| `agones.extensions.webhooks.port`                        | Port to use for webhook service                                                                                                                                                                                                   | `8081`  |
| `agones.extensions.healthCheck.initialDelaySeconds`      | Initial delay before performing the first probe (in seconds)                                                                                                                                                                      | `3`     |
| `agones.extensions.healthCheck.periodSeconds`            | Seconds between every liveness probe (in seconds)                                                                                                                                                                                 | `3`     |
| `agones.extensions.healthCheck.failureThreshold`         | Number of times before giving up (in seconds)                                                                                                                                                                                     | `3`     |
| `agones.extensions.healthCheck.timeoutSeconds`           | Number of seconds after which the probe times out (in seconds)                                                                                                                                                                    | `1`     |
| `agones.extensions.resources`                            | Extensions [resource requests/limit][resources]                                                                                                                                                                                   | `{}`    |
| `agones.extensions.generateTLS`                          | Set to true to generate TLS certificates or false to provide your own certificates                                                                                                                                                | `true`  |
| `agones.extensions.nodeSelector`                         | Extensions [node labels][nodeSelector] for pod assignment                                                                                                                                                                         | `{}`    |
| `agones.extensions.tolerations`                          | Extensions [toleration][toleration] labels for pod assignment                                                                                                                                                                     | `[]`    |
| `agones.extensions.affinity`                             | Extensions [affinity][affinity] settings for pod assignment                                                                                                                                                                       | `{}`    |
| `agones.extensions.labels`                               | [Labels][labels] added to the Agones extensions pods                                                                                                                                                                              | `{}`    |
| `agones.extensions.annotations`                          | [Annotations][annotations] added to the Agones extensions pods                                                                                                                                                                    | `{}`    |
| `agones.extensions.updateStrategy`                             | The [strategy](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#strategy) to apply to the extensions deployment                                                                                                 | `{}`           |
| `agones.extensions.numWorkers`                           | Number of workers to spin per resource type                                                                                                                                                                                       | `100`   |
| `agones.extensions.apiServerQPS`                         | Maximum sustained queries per second that extensions should be making against API Server                                                                                                                                          | `400`   |
| `agones.extensions.apiServerQPSBurst`                    | Maximum burst queries per second that extensions should be making against API Server                                                                                                                                              | `500`   |
| `agones.extensions.logLevel`                             | Agones Extensions Log level. Log only entries with that severity and above                                                                                                                                                        | `info`  |
| `agones.extensions.persistentLogs`                       | Store Agones extensions logs in a temporary volume attached to a container for debugging                                                                                                                                          | `true`  |
| `agones.extensions.persistentLogsSizeLimitMB`            | Maximum total size of all Agones container logs in MB                                                                                                                                                                             | `10000` |
| `agones.extensions.disableSecret`                        | Disables the creation of any allocator secrets. You MUST provide the `{agones.releaseName}-cert` secrets before installation if this is set to `true`.                                                                            | `false` |
| `agones.extensions.customCertSecretPath`                 | Remap cert-manager path to server.crt and server.key                                                                                                                                                                              | `{}`    |
| `agones.extensions.allocationApiService.annotations`     | [Annotations][annotations] added to the Agones API registration.                                                                                                                                                                  | `{}`    |
| `agones.extensions.allocationApiService.disableCaBundle` | Disable ca-bundle so it can be injected by cert-manager.                                                                                                                                                                          | `false` |
| `agones.extensions.validatingWebhook.annotations`        | [Annotations][annotations] added to the Agones validating webhook.                                                                                                                                                                | `{}`    |
| `agones.extensions.validatingWebhook.disableCaBundle`    | Disable ca-bundle so it can be injected by cert-manager.                                                                                                                                                                          | `false` |
| `agones.extensions.mutatingWebhook.annotations`          | [Annotations][annotations] added to the Agones mutating webhook.                                                                                                                                                                  | `{}`    |
| `agones.extensions.mutatingWebhook.disableCaBundle`      | Disable ca-bundle so it can be injected by cert-manager.                                                                                                                                                                          | `false` |
| `agones.extensions.allocationBatchWaitTime`              | Wait time between each allocation batch when performing allocations in controller mode                                                                                                                                            | `500ms` |
| `agones.extensions.pdb.minAvailable`                     | Description of the number of pods from that set that must still be available after the eviction, even in the absence of the evicted pod. Can be either an absolute number or a percentage. Mutually Exclusive with maxUnavailable | `1`     |
| `agones.extensions.pdb.maxUnavailable`                   | Description of the number of pods from that set that can be unavailable after the eviction. It can be either an absolute number or a percentage. Mutually Exclusive with `minAvailable`                                           | \`\`    |
| `agones.extensions.replicas`                             | The number of replicas to run in the deployment                                                                                                                                                                                   | `2`     |
| `agones.extensions.topologySpreadConstraints`            | Ensures better resource utilization and high availability by evenly distributing Pods in the agones-system namespace                                                                                                              | `{}`    |

### GameServers

| Parameter                              | Description                                                                                                                             | Default       |
| -------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- | ------------- |
| `gameservers.namespaces`               | a list of namespaces you are planning to use to deploy game servers                                                                     | `["default"]` |
| `gameservers.minPort`                  | Minimum port to use for dynamic port allocation                                                                                         | `7000`        |
| `gameservers.maxPort`                  | Maximum port to use for dynamic port allocation                                                                                         | `8000`        |
| `gameservers.additionalPortRanges`     | Port ranges from which to do named dynamic port allocation. Example: <br /> additionalPortRanges: <br />&nbsp;&nbsp;game: [9000, 10000] | `{}`          |
| `gameservers.podPreserveUnknownFields` | Disable [field pruning][pruning] and schema validation on the Pod template for a [GameServer][gameserver] definition                    | `false`       |
| `gameservers.lists.maxItems`           | The maximum number of items that can be specified for a list.                                                                           | `1000`        |


### Helm Installation

| Parameter           | Description                                                         | Default |
| ------------------- | ------------------------------------------------------------------- | ------- |
| `helm.installTests` | Add an ability to run `helm test agones` to verify the installation | `false` |

[toleration]: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
[nodeSelector]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector
[affinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity
[cpu-constraints]: https://kubernetes.io/docs/tasks/administer-cluster/manage-resources/cpu-constraint-namespace/
[memory-constraints]: https://kubernetes.io/docs/tasks/administer-cluster/manage-resources/memory-constraint-namespace/
[ping]: {{< ref "/docs/Guides/ping-service.md" >}}
[service]: https://kubernetes.io/docs/concepts/services-networking/service/
[clusterIP]: https://kubernetes.io/docs/concepts/services-networking/service/#type-clusterip
[headless-service]: https://kubernetes.io/docs/concepts/services-networking/service/#headless-services
[allocator]: {{< ref "/docs/advanced/allocator-service.md" >}}
[loadBalancer]: https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer
[annotations]: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/
[labels]:https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
[resources]: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
[pruning]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#field-pruning
[gameserver]: {{< ref "/docs/Reference/gameserver.md" >}}
[rest-requests]: {{< ref "/docs/Advanced/allocator-service.md#using-rest" >}}
[grpc-requests]: {{< ref "/docs/Advanced/allocator-service.md#using-grpc" >}}
[split-controller]: {{< ref "/docs/Advanced/high-availability-agones" >}}

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```bash
helm install my-release --namespace agones-system \
  --set gameservers.minPort=1000,gameservers.maxPort=5000 agones
```

The above command will deploy Agones controllers to `agones-system` namespace. Additionally Agones
will use a dynamic GameServers' port allocation range of 1000-5000.

Alternatively, a YAML file that specifies the values for the parameters can be provided while
installing the chart. For example,

```bash
helm install my-release --namespace agones-system -f values.yaml agones/agones
```

{{% alert title="Tip" color="info" %}}
You can use the default {{< ghlink href="install/helm/agones/values.yaml" >}}values.yaml{{< /ghlink >}}
{{% /alert %}}

## Helm test

This test would create a `GameServer` resource and delete it afterwards.

{{% alert title="Note" color="info" %}}
In order to use `helm test` command described in this section you need to set `helm.installTests`
helm parameter to `true`.
{{% /alert %}}

Check the Agones installation by running the following command:

```bash
helm test my-release -n agones-system
```

You should see a successful output similar to this :

```
NAME: my-release
LAST DEPLOYED: Wed Mar 29 06:13:23 2023
NAMESPACE: agones-system
STATUS: deployed
REVISION: 4
TEST SUITE:     my-release-test
Last Started:   Wed Mar 29 06:17:52 2023
Last Completed: Wed Mar 29 06:18:10 2023
Phase:          Succeeded
```

## Controller TLS Certificates

By default agones chart generates tls certificates used by the admission controller, while this is
handy, it requires the agones controller to restart on each `helm upgrade` command.

### Manual

For most use cases the controller would have required a restart anyway (eg: controller image updated).
However if you really need to avoid restarts we suggest that you turn off tls automatic generation
(`agones.controller.generateTLS` to `false`) and provide your own certificates (`certs/server.crt`,`certs/server.key`).

{{% alert title="Tip" color="info" %}}
You can use our script located at {{< ghlink href="install/helm/agones/certs/cert.sh" >}}cert.sh{{< /ghlink >}}
to generate them.
{{% /alert %}}

### Cert-Manager

Another approach is to use [cert-manager.io](https://cert-manager.io/) solution for cluster level
certificate management.

In order to use the cert-manager solution, first [install cert-manager](https://cert-manager.io/docs/installation/kubernetes/)
on the cluster. Then, [configure](https://cert-manager.io/docs/configuration/) an `Issuer`/`ClusterIssuer`
resource and last [configure](https://cert-manager.io/docs/usage/certificate/) a `Certificate`
resource to manage controller `Secret`. Make sure to configure the `Certificate` based on your
system's requirements, including the validity `duration`.

Here is an example of using a self-signed `ClusterIssuer` for configuring controller `Secret` where
secret name is `my-release-cert` or `{{ template "agones.fullname" . }}-cert`:

```bash
#!/bin/bash
# Create a self-signed ClusterIssuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: selfsigned
spec:
  selfSigned: {}
EOF

# Create a Certificate with IP for the my-release-cert )
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: my-release-agones-cert
  namespace: agones-system
spec:
  dnsNames:
    - agones-controller-service.agones-system.svc
  secretName: my-release-agones-cert
  issuerRef:
    name: selfsigned
    kind: ClusterIssuer
EOF
```

After the certificates are generated, we will want to [inject caBundle](https://cert-manager.io/docs/concepts/ca-injector/)
into the controller and extensions webhook and disable the controller and extensions secret creation
through the following values.yaml file.:

```yaml
agones:
  controller:
    disableSecret: true
    customCertSecretPath:
      - key: ca.crt
        path: ca.crt
      - key: tls.crt
        path: server.crt
      - key: tls.key
        path: server.key
    allocationApiService:
      annotations:
        cert-manager.io/inject-ca-from: agones-system/my-release-agones-cert
      disableCaBundle: true
    validatingWebhook:
      annotations:
        cert-manager.io/inject-ca-from: agones-system/my-release-agones-cert
      disableCaBundle: true
    mutatingWebhook:
      annotations:
        cert-manager.io/inject-ca-from: agones-system/my-release-agones-cert
      disableCaBundle: true
  extensions:
    disableSecret: true
    customCertSecretPath:
      - key: ca.crt
        path: ca.crt
      - key: tls.crt
        path: server.crt
      - key: tls.key
        path: server.key
    allocationApiService:
      annotations:
        cert-manager.io/inject-ca-from: agones-system/my-release-agones-cert
      disableCaBundle: true
    validatingWebhook:
      annotations:
        cert-manager.io/inject-ca-from: agones-system/my-release-agones-cert
      disableCaBundle: true
    mutatingWebhook:
      annotations:
        cert-manager.io/inject-ca-from: agones-system/my-release-agones-cert
      disableCaBundle: true
```

After copying the above yaml into a `values.yaml` file, use below command to install Agones:

```bash
helm install my-release --namespace agones-system --create-namespace --values values.yaml agones/agones
```

## Reserved Allocator Load Balancer IP

In order to reuse the existing load balancer IP on upgrade or install the `agones-allocator` service
as a `LoadBalancer` using a reserved static IP, a user can specify the load balancer's IP with the
`agones.allocator.http.loadBalancerIP` helm configuration parameter value. By setting the
`loadBalancerIP` value:

1. The `LoadBalancer` is created with the specified IP, if supported by the cloud provider.
2. A self-signed server TLS certificate is generated for the IP, used by the `agones-allocator` service.

## Next Steps

- [Confirm Agones is up and running]({{< relref "../confirm.md" >}})
