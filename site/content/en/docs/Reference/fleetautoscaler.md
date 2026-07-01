---
title: "Fleet Autoscaler Specification"
linkTitle: "Fleet Autoscaler"
date: 2019-01-03T03:58:55Z
description: "A `FleetAutoscaler`'s job is to automatically scale up and down a `Fleet` in response to demand."
weight: 30
---

A full `FleetAutoscaler` specification is available below and in the
{{< ghlink href="examples/fleetautoscaler.yaml" >}}example folder{{< /ghlink >}} for reference, but here are several
examples that show different autoscaling policies.

## Ready Buffer Autoscaling

Fleet autoscaling with a buffer can be used to maintain a configured number of game server instances ready to serve
players based on number of allocated instances in a Fleet. The buffer size can be specified as an absolute number or a
percentage of the desired number of Ready game server instances over the Allocated count.

```yaml
apiVersion: "autoscaling.agones.dev/v1"
kind: FleetAutoscaler
# FleetAutoscaler Metadata
# {{< k8s-api-version href="#objectmeta-v1-meta" >}}
metadata:
  name: fleet-autoscaler-example
spec:
  # The name of the fleet to attach to and control. Must be an existing Fleet in the same namespace
  # as this FleetAutoscaler
  fleetName: fleet-example
  # The autoscaling policy
  policy:
    # type of the policy. for now, only Buffer is available
    type: Buffer
    # parameters of the buffer policy
    buffer:
      # Size of a buffer of "ready" game server instances
      # The FleetAutoscaler will scale the fleet up and down trying to maintain this buffer, 
      # as instances are being allocated or terminated
      # it can be specified either in absolute (i.e. 5) or percentage format (i.e. 5%)
      bufferSize: 5
      # minimum fleet size to be set by this FleetAutoscaler. 
      # if not specified, the actual minimum fleet size will be bufferSize
      minReplicas: 10
      # maximum fleet size that can be set by this FleetAutoscaler
      # required
      maxReplicas: 20
  # The autoscaling sync strategy
  sync:
    # type of the sync. for now, only FixedInterval is available
    type: FixedInterval
    # parameters of the fixedInterval sync
    fixedInterval:
      # the time in seconds between each auto scaling
      seconds: 30
```

## Counter and List Autoscaling

{{< beta title="Counters and Lists" gate="CountsAndLists" >}}

A Counter based autoscaler can be used to autoscale `GameServers` based on a Count and Capacity set on each of the 
GameServers in a Fleet to ensure there is always a buffer of available capacity available.

For example, if you have a game server that can support 10 rooms, and you want to ensure that there are always at least
5 rooms available, you could use a counter-based autoscaler with a buffer size of 5. The autoscaler would then scale the
`Fleet` up or down based on the difference between the count of rooms across the `Fleet` and the capacity of
rooms across the Fleet to ensure the buffer is maintained.

Counter-based `FleetAutoscaler` specification below and in the {{< ghlink href="examples/counterfleetautoscaler.yaml" >}}example folder{{< /ghlink >}}:

```yaml
apiVersion: autoscaling.agones.dev/v1
kind: FleetAutoscaler
metadata:
  name: fleet-autoscaler-counter
spec:
  fleetName: fleet-example
  policy:
    type: Counter  # Counter based autoscaling
    counter:
      # Key is the name of the Counter. Required field.
      key: rooms
      # BufferSize is the size of a buffer of counted items that are available in the Fleet (available capacity).
      # Value can be an absolute number (ex: 5) or a percentage of the Counter available capacity (ex: 5%).
      # An absolute number is calculated from percentage by rounding up. Must be bigger than 0. Required field.
      bufferSize: 5
      # MinCapacity is the minimum aggregate Counter total capacity across the fleet.
      # If BufferSize is specified as a percentage, MinCapacity is required and cannot be 0.
      # If non zero, MinCapacity must be smaller than MaxCapacity and must be greater than or equal to BufferSize.
      minCapacity: 10
      # MaxCapacity is the maximum aggregate Counter total capacity across the fleet.
      # MaxCapacity must be greater than or equal to both MinCapacity and BufferSize. Required field.
      maxCapacity: 100
```

A List based autoscaler can be used to autoscale `GameServers` based on the List length and Capacity set on each of the
GameServers in a Fleet to ensure there is always a buffer of available capacity available.

For example, if you have a game server that can support 10 players, and you want to ensure that there are always 
room for at least 5 players across `GameServers` in a `Fleet`, you could use a list-based autoscaler with a buffer size 
of 5. The autoscaler would then scale the `Fleet` up or down based on the difference between the total length of 
the `players` and the total `players` capacity across the Fleet to ensure the buffer is maintained.

List-based `FleetAutoscaler` specification below and in the {{< ghlink href="examples/listfleetautoscaler.yaml" >}}example folder{{< /ghlink >}}:

```yaml
apiVersion: autoscaling.agones.dev/v1
kind: FleetAutoscaler
metadata:
  name: fleet-autoscaler-list
spec:
  fleetName: fleet-example
  policy:
    type: List  # List based autoscaling.
    list:
      # Key is the name of the List. Required field.
      key: players
      # BufferSize is the size of a buffer based on the List capacity that is available over the current
      # aggregate List length in the Fleet (available capacity).
      # It can be specified either as an absolute value (i.e. 5) or percentage format (i.e. 5%).
      # Must be bigger than 0. Required field.
      bufferSize: 5
      # MinCapacity is the minimum aggregate List total capacity across the fleet.
      # If BufferSize is specified as a percentage, MinCapacity is required must be greater than 0.
      # If non-zero, MinCapacity must be smaller than MaxCapacity and must be greater than or equal to BufferSize.
      minCapacity: 10
      # MaxCapacity is the maximum aggregate List total capacity across the fleet.
      # MaxCapacity must be greater than or equal to both MinCapacity and BufferSize. Required field.
      maxCapacity: 100
```

## Webhook Autoscaling

A webhook-based `FleetAutoscaler` can be used to delegate the scaling logic to a separate http based service. This
can be useful if you want to use a custom scaling algorithm or if you want to integrate with other systems. For
example, you could use a webhook-based `FleetAutoscaler` to scale your fleet based on data from a match-maker or player
authentication system or a combination of systems.

Webhook based autoscalers have the added benefit of being able to scale a Fleet to 0 replicas, since they are able to
scale up on demand based on an external signal before a `GameServerAllocation` is executed from a match-maker or 
similar system.

In order to define the path to your Webhook you can use either `URL` or `service`.

The `caBundle` parameter is available if you use custom certificates for HTTPS webhooks or want specifically to
validate the HTTPS certificate with a specific certificate authority.

For Webhook FleetAutoscaler below and in {{< ghlink href="examples/webhookfleetautoscaler.yaml" >}}example folder{{< /ghlink >}}:

```yaml
apiVersion: "autoscaling.agones.dev/v1"
kind: FleetAutoscaler
metadata:
  name: webhook-fleet-autoscaler
spec:
  fleetName: simple-game-server
  policy:
    # type of the policy - this example is Webhook
    type: Webhook
    # parameters for the webhook policy - this is a WebhookClientConfig, as per other K8s webhooks
    webhook:
      # use a service, or URL
      service:
        name: autoscaler-webhook-service
        namespace: default
        path: scale
      # optional for URL defined webhooks
      # url: ""
      # caBundle:  optional, used for HTTPS webhook type
  # The autoscaling sync strategy
  sync:
    # type of the sync. for now, only FixedInterval is available
    type: FixedInterval
    # parameters of the fixedInterval sync
    fixedInterval:
      # the time in seconds between each auto scaling
      seconds: 30
```

See the [Webhook Endpoint Specification](#webhook-endpoint-specification) for the specification of the incoming and 
outgoing JSON packet structure for the webhook endpoint.

### Webhook Endpoint Specification

A webhook based `FleetAutoscaler` sends an HTTP POST request to the webhook endpoint every sync period (default is 30s)
with a JSON body, and scale the target fleet based on the data that is returned.

The JSON payload that is sent is a `FleetAutoscaleReview` data structure and a `FleetAutoscaleReview` with a populated
`FleetAutoscaleResponse` data structure is expected to be returned.

The `FleetAutoscaleResponse`'s `Replica` field is used to set the target `Fleet` count with each sync interval, thereby
providing the autoscaling functionality.

{{% feature expiryVersion="1.54.0" %}}
{{< alpha title="Fleet Metadata in Webhook Requests" gate="FleetAutoscaleRequestMetaData" >}}
{{% /feature %}}

{{% feature publishVersion="1.54.0" %}}
{{< beta title="Fleet Metadata in Webhook Requests" gate="FleetAutoscaleRequestMetaData" >}}
{{% /feature %}}

When the `FleetAutoscaleRequestMetaData` feature gate is enabled, the Fleet's `labels` and 
`annotations` are included in the `FleetAutoscaleRequest`, allowing webhooks to make scaling decisions based on 
Fleet-specific metadata.

```go
// FleetAutoscaleReview is passed to the webhook with a populated Request value,
// and then returned with a populated Response.
type FleetAutoscaleReview struct {
	Request  *FleetAutoscaleRequest  `json:"request"`
	Response *FleetAutoscaleResponse `json:"response"`
}

type FleetAutoscaleRequest struct {
	// UID is an identifier for the individual request/response. It allows us to distinguish instances of requests which are
	// otherwise identical (parallel requests, requests when earlier requests did not modify etc)
	// The UID is meant to track the round trip (request/response) between the Autoscaler and the WebHook, not the user request.
	// It is suitable for correlating log entries between the webhook and apiserver, for either auditing or debugging.
	UID types.UID `json:"uid"`
	// Name is the name of the Fleet being scaled
	Name string `json:"name"`
	// Namespace is the namespace associated with the request (if any).
	Namespace string `json:"namespace"`
	// The Fleet's status values
	Status v1.FleetStatus `json:"status"`
	// Standard map labels; More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels.
	Labels map[string]string `json:"labels,omitempty"`
	// Standard map annotations; More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}

type FleetAutoscaleResponse struct {
	// UID is an identifier for the individual request/response.
	// This should be copied over from the corresponding FleetAutoscaleRequest.
	UID types.UID `json:"uid"`
	// Set to false if no scaling should occur to the Fleet
	Scale bool `json:"scale"`
	// The targeted replica count
	Replicas int32 `json:"replicas"`
}

// FleetStatus is the status of a Fleet
type FleetStatus struct {
	// Replicas the total number of current GameServer replicas
	Replicas int32 `json:"replicas"`
	// ReadyReplicas are the number of Ready GameServer replicas
	ReadyReplicas int32 `json:"readyReplicas"`
	// ReservedReplicas are the total number of Reserved GameServer replicas in this fleet.
	// Reserved instances won't be deleted on scale down, but won't cause an autoscaler to scale up.
	ReservedReplicas int32 `json:"reservedReplicas"`
	// AllocatedReplicas are the number of Allocated GameServer replicas
	AllocatedReplicas int32 `json:"allocatedReplicas"`
    // Allocations is a counter of the number of allocations observed.
    Allocations int64 `json:"allocations"`
}
```

For Webhook Fleetautoscaler Policy either HTTP or HTTPS could be used. Switching between them occurs depending on https presence in `URL` or by the presence of `caBundle`.
The example of the webhook written in Go could be found {{< ghlink href="examples/autoscaler-webhook/main.go" >}}here{{< /ghlink >}}.

It implements the {{< ghlink href="examples/autoscaler-webhook/" >}}scaling logic{{< /ghlink >}} based on the percentage of allocated gameservers in a fleet.



## Wasm Autoscaling

{{< alpha title="Wasm Autoscaler" gate="WasmAutoscaler" >}}

A Wasm-based `FleetAutoscaler` can be used to implement custom scaling logic using WebAssembly modules. This
provides a flexible and secure way to run custom autoscaling algorithms without requiring external services. Wasm
modules are sandboxed and can be written in multiple languages that compile to WebAssembly, such as Rust, Go, or
C++.

Agones uses the [Extism](https://extism.org/) WebAssembly plugin framework to load and execute Wasm modules, providing
a secure and performant runtime environment with support for multiple programming languages.

Wasm based autoscalers offer the benefits of custom scaling logic similar to webhooks, but with the added advantages
of running within the cluster without having to spin up dedicated service within your architecture.

In order to define the source of your Wasm module you can use either `URL` or `service`. The Wasm module is downloaded
from the specified location and executed locally.

For Wasm FleetAutoscaler below and in {{< ghlink href="examples/wasmfleetautoscaler.yaml" >}}example folder{{< /ghlink >}}:

```yaml
apiVersion: "autoscaling.agones.dev/v1"
kind: FleetAutoscaler
metadata:
  name: wasm-fleet-autoscaler
spec:
  fleetName: simple-game-server
  policy:
    # type of the policy - this example is Wasm
    type: Wasm
    # parameters for the wasm policy
    wasm:
      # The exported function to call in the wasm module, defaults to 'scale'
      function: 'scale'
      # Config values to pass to the wasm program on startup
      config:
        buffer_size: 10
      from:
        url:
          # use a service, or direct URL
          service:
            name: fileserver
            namespace: default
            path: /wasm/plugin.wasm
          # optionally can define a full URL if not hosted on cluster
          # url: "https://my-bucket-storage.cloud/wasm/plugin.wasm"
          # caBundle:  optional, used for HTTPS paths with custom certs
      # optional hex encoded sha256 hash to match against wasm file (recommended)
      hash: "df7199d01a25bf34b3d650c7e6f685736b2c794e6a526d86b2e55bf074df3f36"
```

See the [Wasm Function Specification](#wasm-function-specification) for the specification of the incoming and 
outgoing JSON data structure for the Wasm function.

### Wasm Function Specification

A Wasm-based `FleetAutoscaler` calls the configured exported function in the Wasm module every sync period (default is 30s)
with JSON input data, and scales the target fleet based on the JSON output data that is returned.

The JSON input data is provided via the [Extism PDK's](https://extism.org/docs/quickstart/plugin-quickstart) input 
functions (e.g., `pdk.InputJSON()` in Go) as a `FleetAutoscaleReview` 
data structure with a populated `Request` field. The Wasm function must return a `FleetAutoscaleReview` with a populated 
`Response` field via the Extism PDK's output functions (e.g., `pdk.OutputJSON()` in Go). As you may note, this is the
exact same structure as the [Webhook Autoscaler Specification](#webhook-endpoint-specification).

The `FleetAutoscaleResponse`'s `Replicas` field is used to set the target `Fleet` count with each sync interval, thereby
providing the autoscaling functionality. The `Scale` field indicates whether scaling should occur.

```go
// FleetAutoscaleReview is passed to the Wasm function with a populated Request value,
// and then returned with a populated Response.
type FleetAutoscaleReview struct {
	Request  *FleetAutoscaleRequest  `json:"request"`
	Response *FleetAutoscaleResponse `json:"response"`
}

type FleetAutoscaleRequest struct {
	// UID is an identifier for the individual request/response. It allows us to distinguish instances of requests which are
	// otherwise identical (parallel requests, requests when earlier requests did not modify etc)
	// The UID is meant to track the round trip (request/response) between the Autoscaler and the Wasm function, not the user request.
	// It is suitable for correlating log entries between the autoscaler and the Wasm function, for either auditing or debugging.
	UID string `json:"uid"`
	// Name is the name of the Fleet being scaled
	Name string `json:"name"`
	// Namespace is the namespace associated with the request (if any).
	Namespace string `json:"namespace"`
	// The Fleet's status values
	Status FleetStatus `json:"status"`
	// Standard map labels; More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels.
	Labels map[string]string `json:"labels,omitempty"`
	// Standard map annotations; More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}

type FleetAutoscaleResponse struct {
	// UID is an identifier for the individual request/response.
	// This should be copied over from the corresponding FleetAutoscaleRequest.
	UID string `json:"uid"`
	// Set to false if no scaling should occur to the Fleet
	Scale bool `json:"scale"`
	// The targeted replica count
	Replicas int32 `json:"replicas"`
}

// FleetStatus is the status of a Fleet
type FleetStatus struct {
	// Replicas the total number of current GameServer replicas
	Replicas int32 `json:"replicas"`
	// ReadyReplicas are the number of Ready GameServer replicas
	ReadyReplicas int32 `json:"readyReplicas"`
	// ReservedReplicas are the total number of Reserved GameServer replicas in this fleet.
	// Reserved instances won't be deleted on scale down, but won't cause an autoscaler to scale up.
	ReservedReplicas int32 `json:"reservedReplicas"`
	// AllocatedReplicas are the number of Allocated GameServer replicas
	AllocatedReplicas int32 `json:"allocatedReplicas"`
    // Allocations is a counter of the number of allocations observed.
    Allocations int64 `json:"allocations"`
}
```

The example of a Wasm autoscaler written in Go using the Extism PDK can be found {{< ghlink href="examples/autoscaler-wasm/main.go" >}}here{{< /ghlink >}}.

It implements {{< ghlink href="examples/autoscaler-wasm/" >}}scaling logic{{< /ghlink >}} that maintains a buffer of available replicas based on allocated game servers.


## Schedule and Chain Autoscaling

A schedule-based autoscaler can automatically scale GameServers at a specific time based on a FleetAutoscaler policy.

For example, if you have an in-game event happening at 12:00 AM PST on October 31, 2024 and you want to ensure that there are always at least 5 ready game servers at that specific time, you could use a schedule-based autoscaler that applies a buffer policy with a buffer size of 5. 

Schedule-based `FleetAutoscaler` specification below and in the {{< ghlink href="examples/schedulefleetautoscaler.yaml" >}}example folder{{< /ghlink >}}:

```yaml
apiVersion: autoscaling.agones.dev/v1
kind: FleetAutoscaler
metadata:
  name: schedule-fleet-autoscaler
spec:
  fleetName: fleet-example
  policy:
    # Schedule based policy for autoscaling.
    type: Schedule
    schedule:
      between:
        # The policy becomes eligible for application starting on October 31st, 2024 at 12:00 AM PST. If not set, the policy will immediately be eligible for application.
        start: "2024-10-31T00:00:00-07:00"
        # The policy is never ineligible for application. If not set, the policy will always be eligible for application (after the start time).
        end: ""
      activePeriod:
        # Use PST time for the startCron field. Defaults to UTC if not set.
        timezone: "America/Los_Angeles"
        # Start applying the policy everyday at 12:00 AM PST. If not set, the policy will always be applied in the .between window.
        # (Only eligible starting on October 31, 2024 at 12:00 AM PST).
        startCron: "0 0 * * *"
        # Apply this policy indefinitely. If not set, the duration will be defaulted to always/indefinite.
        duration: ""
      # Policy to be applied during the activePeriod. Required.
      policy:
        type: Buffer
        buffer:
          bufferSize: 5
          minReplicas: 10
          maxReplicas: 20
  # The autoscaling sync strategy, this will determine how frequent the schedule is evaluated.
  sync:
    # type of the sync. for now, only FixedInterval is available
    type: FixedInterval
    # parameters of the fixedInterval sync
    fixedInterval:
      # the time in seconds between each auto scaling
      seconds: 30
```

{{% alert title="Note" color="info" %}}
While it's possible to use multiple FleetAutoscalers to perform the same actions on a fleet, this approach can result in unpredictable scaling behavior and overwhelm the controller. Multiple FleetAutoscalers pointing to the same fleet can lead to conflicting schedules and complex management. To avoid these issues, it's recommended to use the Chain Policy for defining scheduled scaling. This simplifies management and prevents unexpected scaling fluctuations.
{{% /alert %}}

A Chain-based autoscaler can be used to autoscale `GameServers` based on a list of policies. The index of each policy within the list determines its priority, with the first item in the chain having the highest priority and being evaluated first, followed by the second item, and so on. The order in the list is crucial as it directly affects the sequence in which policies are evaluated and applied.

For example, if you have an in-game event happening between 12:00 AM and 10:00 PM on October 31, 2024 EST, and you want to ensure there are always at least 5 ready game servers during that time, you chain add a schedule policy that applies a buffer policy with a buffer size of 5. Now, let's say you want to adjust the number of game servers after the event based on external factors after the in-game event, you could chain a webhook policy to dynamically. Finally as a fallback, for if the webhook invocation fails, you can chain another buffer policy to default to.

Chain-based `FleetAutoscaler` specification below and in the {{< ghlink href="examples/chainfleetautoscaler.yaml" >}}example folder{{< /ghlink >}}:

```yaml
apiVersion: autoscaling.agones.dev/v1
kind: FleetAutoscaler
metadata:
  name: chain-fleet-autoscaler
spec:
  fleetName: fleet-example
  policy:
    # Chain based policy for autoscaling.
    type: Chain
    chain:
      # Id of chain entry. If not set, the Id will be defaulted to the index of the entry within the chain.
      - id: "in-game-event"
        type: Schedule
        schedule:
          between:
            # The policy becomes eligible for application starting on October 31st, 2024 at 12:00 AM PST. If not set, the policy will immediately be eligible for application.
            start: "2024-10-31T00:00:00-07:00"
            # The policy is no longer eligible for application starting on  October 31st, 2024 at 10:00 PM PST. If not set, the policy will always be eligible for application (after the start time).
            end: "2024-10-31T22:00:00-07:00"
          activePeriod:
            # Use PST time for the startCron field. Defaults to UTC if not set.
            timezone: "America/Los_Angeles"
            # Start applying the policy everyday at 1:00 AM PST. If not set, the policy will always be applied in the .between window.
            # (Only eligible starting on October 31, 2024 at 12:00 AM PST).
            startCron: "0 0 * * *"
            # Apply this policy indefinitely. If not set, the duration will be defaulted to always/indefinite.
            duration: ""
          # Policy to be applied during the activePeriod. Required.
          policy:
            type: Buffer
            buffer:
              bufferSize: 5
              minReplicas: 10
              maxReplicas: 20
      # Id of chain entry. If not set, the Id will be defaulted to the index of the entry within the chain list.
      - id: "webhook"
         type: Webhook
          webhook:
            service:
              name: autoscaler-webhook-service
              namespace: default
              path: scale
      # Id of chain entry. If not set, the Id will be defaulted to the index of the entry within the chain list.
      - id: "default"
        # Policy will always be applied when no other policy is applicable. Required.
        type: Buffer
        buffer:
          bufferSize: 2
          minReplicas: 5
          maxReplicas: 10
  # The autoscaling sync strategy, this will determine how frequent the chain is evaluated.
  sync:
    # type of the sync. for now, only FixedInterval is available
    type: FixedInterval
    # parameters of the fixedInterval sync
    fixedInterval:
      # the time in seconds between each auto scaling
      seconds: 30
```

## Spec Field Reference

The `spec` field of the `FleetAutoscaler` is composed as follows:

- `fleetName` is name of the fleet to attach to and control. Must be an existing `Fleet` in the same namespace
   as this `FleetAutoscaler`.
- `policy` is the autoscaling policy
  - `type` is type of the policy. "Buffer", "Webhook", "Counter", "List", "Schedule", "Chain", and "Wasm" are available
  - `buffer` parameters of the buffer policy type
    - `bufferSize`  is the size of a buffer of "ready" and "reserved" game server instances.
                    The FleetAutoscaler will scale the fleet up and down trying to maintain this buffer, 
                    as instances are being allocated or terminated.
                    Note that "reserved" game servers could not be scaled down.
                    It can be specified either in absolute (i.e. 5) or percentage format (i.e. 5%)
    - `minReplicas` is the minimum fleet size to be set by this FleetAutoscaler. 
                    if not specified, the minimum fleet size will be bufferSize if absolute value is used.
                    When `bufferSize` in percentage format is used, `minReplicas` should be more than 0.
    - `maxReplicas` is the maximum fleet size that can be set by this FleetAutoscaler. Required. 
  - `webhook` parameters of the webhook policy type
    - `service` is a reference to the service for this webhook. Either `service` or `url` must be specified. If the webhook is running within the cluster, then you should use `service`. Port 8000 will be used if it is open, otherwise it is an error.
      - `name`  is the service name bound to Deployment of autoscaler webhook. Required {{< ghlink href="examples/autoscaler-webhook/autoscaler-service.yaml" >}}(see example){{< /ghlink >}}
                      The FleetAutoscaler will scale the fleet up and down based on the response from this webhook server
      - `namespace` is the kubernetes namespace where webhook is deployed. Optional
                      If not specified, the "default" would be used
      - `path` is an optional URL path which will be sent in any request to this service. (i. e. /scale)
      - `port` is optional, it is the port for the service which is hosting the webhook. The default is 8000 for backward compatibility. If given, it should be a valid port number (1-65535, inclusive).
    - `url` gives the location of the webhook, in standard URL form (`[scheme://]host:port/path`). Exactly one of `url` or `service` must be specified. The `host` should not refer to a service running in the cluster; use the `service` field instead.  (optional, instead of service)
    - `caBundle` is a PEM encoded certificate authority bundle which is used to issue and then validate the webhook's server certificate. Base64 encoded PEM string. Required only for HTTPS. If not present HTTP client would be used.
  - Note: only one `buffer` or `webhook` could be defined for FleetAutoscaler which is based on the `type` field.
  - `counter` parameters of the counter policy type
    - `counter` contains the settings for counter-based autoscaling:
      - `key` is the name of the counter to use for scaling decisions.
      - `bufferSize` is the size of a buffer of counted items that are available in the Fleet (available capacity). Value can be an absolute number or a percentage of desired game server instances. An absolute number is calculated from percentage by rounding up. Must be bigger than 0.
      - `minCapacity` is the minimum aggregate Counter total capacity across the fleet. If zero, MinCapacity is ignored. If non zero, MinCapacity must be smaller than MaxCapacity and bigger than BufferSize.
      - `maxCapacity` is the maximum aggregate Counter total capacity across the fleet. It must be bigger than both MinCapacity and BufferSize.
  - `list` parameters of the list policy type
    - `list` contains the settings for list-based autoscaling:
      - `key` is the name of the list to use for scaling decisions.
      - `bufferSize` is the size of a buffer based on the List capacity that is available over the current aggregate List length in the Fleet (available capacity). It can be specified either as an absolute value or percentage format.
      - `minCapacity` is the minimum aggregate List total capacity across the fleet. If zero, it is ignored. If non zero, it must be smaller than MaxCapacity and bigger than BufferSize.
      - `maxCapacity` is the maximum aggregate List total capacity across the fleet. It must be bigger than both MinCapacity and BufferSize. Required field.
  - `wasm` parameters of the wasm policy type
    - The following are subfields of the `wasm` field, which contains the settings for WebAssembly-based autoscaling:
      - `function` is the exported function to call in the wasm module. Optional, defaults to 'scale'.
      - `config` is a map of key-value pairs to pass to the wasm program on startup. Optional.
      - `from` defines the source of the Wasm module. Required.
        - `url` is the URL configuration for the Wasm module location. Required.
          - `service` is a reference to the service hosting the Wasm module. Either `service` or `url` must be specified.
            - `name` is the service name. Required if using service.
            - `namespace` is the kubernetes namespace where the service is deployed. Optional, defaults to "default".
            - `path` is the URL path to the Wasm module file (e.g., /wasm/plugin.wasm). Optional.
            - `port` is the port for the service. Optional, defaults to 8000.
          - `url` gives the direct URL location of the Wasm module, in standard URL form (`[scheme://]host:port/path`). Exactly one of `url` or `service` must be specified. Optional.
          - `caBundle` is a PEM encoded certificate authority bundle for HTTPS. Base64 encoded PEM string. Required only for HTTPS with custom certificates.
      - `hash` is an optional hex encoded sha256 hash to verify the integrity of the Wasm module. Optional but recommended.
- `sync` is autoscaling sync strategy. It defines when to run the autoscaling
  - `type` is type of the sync. For now only "FixedInterval" is available
  - `fixedInterval` parameters of the fixedInterval sync
    - `seconds` is the time in seconds between each autoscaling

