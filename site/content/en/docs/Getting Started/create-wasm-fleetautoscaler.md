---
title: "Quickstart: Create a Fleet Autoscaler with Wasm"
linkTitle: "Create a Wasm Fleetautoscaler"
date: 2025-12-09
weight: 50
description: >
  This guide covers how you can create a wasm fleet autoscaler policy.
---

{{< alpha title="Wasm Autoscaler" gate="WasmAutoscaler" >}}

In some cases, your game servers may need to use custom logic for scaling your fleet that is more complex than what
can be expressed using the Buffer policy in the fleetautoscaler. This guide shows how you can extend Agones
with a WebAssembly (Wasm) autoscaler to implement a custom autoscaling policy.

When you use a Wasm autoscaler, the logic computing the number of target replicas is executed by a WebAssembly module
that you provide. The fleetautoscaler will call the Wasm module's exported function every sync period (configurable, 
default 30s) with fleet data, and scale the target fleet based on the data that is returned.

Agones uses the [Extism](https://extism.org/) WebAssembly plugin framework to load and execute Wasm modules, providing
a secure and performant runtime environment with support for multiple programming languages.

Wasm autoscalers offer several advantages over webhook autoscalers:
- **Performance**: No network calls required - the Wasm module runs directly in the autoscaler process
- **Simplicity**: No need to deploy and manage separate webhook services
- **Security**: Sandboxed execution environment with no network access
- **Portability**: Write your autoscaling logic once and run it anywhere

## Prerequisites

It is assumed that you have completed the instructions to [Create a Game Server Fleet]({{< relref "./create-fleet.md" >}}) and have a running fleet of game servers.

## Objectives

- Run a fleet
- Create a Fleet Autoscaler with Wasm policy type in Kubernetes using Agones custom resource
- Watch the Fleet scale up when allocating GameServers
- Watch the Fleet scale down after GameServer shutdown

## Step 1: Deploy the Fleet

Run a fleet in a cluster:
```bash
kubectl apply -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/fleet.yaml
```

## Step 2: Create a Fleet Autoscaler with Wasm Policy

The Wasm autoscaler policy allows you to specify a WebAssembly module that will be executed to determine the desired
number of replicas for your fleet. The Wasm module receives information about the current fleet state and returns
the desired replica count.

Let's create a Fleet Autoscaler using the following command:

```bash
kubectl apply -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/autoscaler-wasm/autoscaler.yaml
```

You should see a successful output similar to this:

```
fleetautoscaler.autoscaling.agones.dev/wasm-fleet-autoscaler created
```

This has created a FleetAutoscaler record inside Kubernetes that references a Wasm module hosted at a URL.

Let's examine the key parts of the autoscaler configuration:

```yaml
apiVersion: autoscaling.agones.dev/v1
kind: FleetAutoscaler
metadata:
  name: wasm-fleet-autoscaler
spec:
  fleetName: simple-game-server
  policy:
    type: Wasm
    wasm:
      # The exported function to call in the wasm module, defaults to 'scale'
      function: 'scale'
      # Config values to pass to the wasm program on startup
      config:
        buffer_size: "5"
      from:
        url:
          # Direct URL to the wasm plugin
          url: "https://github.com/agones-dev/agones/raw/refs/heads/main/examples/autoscaler-wasm/plugin.wasm"
```

### Understanding the Wasm Policy Configuration

- **type: Wasm**: Specifies that this autoscaler uses a WebAssembly module
- **function**: The name of the exported function in the Wasm module to call (defaults to `scale`)
- **config**: Key-value pairs passed to the Wasm module on startup, allowing you to configure the autoscaling behavior
- **from.url.url**: A direct URL to download the Wasm module from

Alternatively, you can reference a Wasm module served by a Kubernetes service:

```yaml
from:
  url:
    service:
      namespace: default
      name: wasm-server-service
      path: /plugin.wasm
      port: 80
```

### Optional: Hash Verification

For additional security, you can specify a SHA256 hash to verify the integrity of the downloaded Wasm module:

```yaml
wasm:
  function: 'scale'
  hash: 'a1b2c3d4e5f6...'  # SHA256 hash of the wasm file
  from:
    url:
      url: "https://example.com/plugin.wasm"
```

If the hash doesn't match, the autoscaler will not load the module and will log an error.

## Step 3: See the Fleet and Autoscaler Status

In order to track the list of gameservers which run in your fleet you can run this command in a separate terminal tab:

```bash
watch "kubectl get gs -n default"
```

In order to get autoscaler status use the following command:

```bash
kubectl describe fleetautoscaler wasm-fleet-autoscaler
```

It should look something like this:

```yaml
Name:         wasm-fleet-autoscaler
Namespace:    default
Labels:       <none>
Annotations:  <none>
API Version:  autoscaling.agones.dev/v1
Kind:         FleetAutoscaler
Metadata:
  Creation Timestamp:  2025-12-10T00:30:28Z
  Generation:          1
  Resource Version:    3457
  UID:                 6296f283-4a0e-4470-8d8e-5897faf5815b
Spec:
  Fleet Name:  simple-game-server
  Policy:
    Type:  Wasm
    Wasm:
      Config:
        buffer_size:  5
      From:
        URL:
          URL:   https://github.com/agones-dev/agones/raw/refs/heads/main/examples/autoscaler-wasm/plugin.wasm
      Function:  scale
  Sync:
    Fixed Interval:
      Seconds:  30
    Type:       FixedInterval
Status:
  Able To Scale:        true
  Current Replicas:     5
  Desired Replicas:     5
  Last Applied Policy:  Wasm
  Last Scale Time:      2025-12-10T00:30:30Z
  Scaling Limited:      false
Events:
  Type    Reason            Age   From                        Message
  ----    ------            ----  ----                        -------
  Normal  AutoScalingFleet  81s   fleetautoscaler-controller  Scaling fleet simple-game-server from 2 to 5

```

You can see the status (able to scale, not limited), the last time the fleet was scaled (nil for never), current and 
desired fleet size.

The Events stream also gives you an audit trail of what scaling has happened when, or if something has gone wrong!

The autoscaler executes the Wasm module's function periodically, and based on the returned value, it changes the target
replica size. The fleet then creates/deletes game server instances to achieve that number. The convergence is achieved
in time, which is usually measured in seconds.

## Step 4: Allocate Game Servers from the Fleet to Trigger Scale Up

If you're interested in more details for game server allocation, you should consult the [Create a Game Server Fleet]({{< relref "../Getting Started/create-fleet.md" >}}) page.
Here we are only interested in triggering allocations to see the autoscaler in action.

```bash
kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/gameserverallocation.yaml -o yaml
```

You should get in return the allocated game server details, which should end with something like:

```yaml
status:
  address: 192.168.49.2
  addresses:
  - address: 192.168.49.2
    type: InternalIP
  - address: agones
    type: Hostname
  - address: 10.244.0.17
    type: PodIP
  gameServerName: simple-game-server-8pbz5-4fths
  metadata:
    annotations:
      agones.dev/last-allocated: "2025-12-10T00:42:39.311833025Z"
      agones.dev/sdk-version: 1.55.0-dev-aff8248
    labels:
      agones.dev/fleet: simple-game-server
      agones.dev/gameserverset: simple-game-server-8pbz5
  nodeName: agones
  ports:
  - name: default
    port: 7506
  source: local
  state: Allocated
```

Note the address and port, you might need them later to connect to the server.

## Step 5: Check New Autoscaler and Fleet Status

Now let's wait a few seconds to allow the autoscaler to detect the change in the fleet and check again its status:

```bash
kubectl describe fleetautoscaler wasm-fleet-autoscaler
```

The last part should look similar to this:

```yaml
Spec:
  Fleet Name:  simple-game-server
  Policy:
    Type:  Wasm
    Wasm:
      Config:
        buffer_size:  5
      From:
        URL:
          URL:   https://github.com/agones-dev/agones/raw/refs/heads/main/examples/autoscaler-wasm/plugin.wasm
      Function:  scale
  Sync:
    Fixed Interval:
      Seconds:  30
    Type:       FixedInterval
Status:
  Able To Scale:        true
  Current Replicas:     6
  Desired Replicas:     6
  Last Applied Policy:  Wasm
  Last Scale Time:      2025-12-10T00:43:00Z
  Scaling Limited:      false
Events:
  Type    Reason            Age   From                        Message
  ----    ------            ----  ----                        -------
  Normal  AutoScalingFleet  14m   fleetautoscaler-controller  Scaling fleet simple-game-server from 2 to 5
  Normal  AutoScalingFleet  100s  fleetautoscaler-controller  Scaling fleet simple-game-server from 5 to 6

```

You can see that the fleet size has increased (based on the custom logic in the Wasm module), the autoscaler having
compensated for the allocated instance while maintaining the configured buffer.

Double-check the actual number of game server instances and status by running:

```bash
kubectl get gs -n default
```

This will get you a list of all the current `GameServers` and their `Status > State`.

```
NAME                             STATE       ADDRESS        PORT   NODE     AGE
simple-game-server-8pbz5-2w72n   Ready       192.168.49.2   7238   agones   2m49s
simple-game-server-8pbz5-4fths   Allocated   192.168.49.2   7506   agones   15m
simple-game-server-8pbz5-b4pwb   Ready       192.168.49.2   7012   agones   25m
simple-game-server-8pbz5-cq8wf   Ready       192.168.49.2   7011   agones   15m
simple-game-server-8pbz5-lk7l9   Ready       192.168.49.2   7068   agones   25m
simple-game-server-8pbz5-tb4qk   Ready       192.168.49.2   7055   agones   15m
```

## Step 6: Check Downscaling

The example Wasm autoscaler maintains a buffer of ready game servers. When you shut down an allocated game server,
the autoscaler will scale down the fleet to maintain the configured buffer size.

We need to run the EXIT command on the allocated gameserver (use the IP address and port from Step 4) to shut it down:

```bash
nc -u 192.168.49.2 7506
EXIT
```

The server will enter a shutdown state and be removed from the fleet.

Wait about 30 seconds, then check the autoscaler status:

```bash
kubectl describe fleetautoscaler wasm-fleet-autoscaler
```

You should see a scaling event in the events section:

```
Events:
  Type    Reason            Age   From                        Message
  ----    ------            ----  ----                        -------
  Normal  AutoScalingFleet  16m   fleetautoscaler-controller  Scaling fleet simple-game-server from 2 to 5
  Normal  AutoScalingFleet  4m4s  fleetautoscaler-controller  Scaling fleet simple-game-server from 5 to 6
  Normal  AutoScalingFleet  4s    fleetautoscaler-controller  Scaling fleet simple-game-server from 6 to 5

```

And verify the gameserver count:

```bash
kubectl get gs -n default
```

```
NAME                             STATE   ADDRESS        PORT   NODE     AGE
simple-game-server-8pbz5-2w72n   Ready   192.168.49.2   7238   agones   4m32s
simple-game-server-8pbz5-b4pwb   Ready   192.168.49.2   7012   agones   27m
simple-game-server-8pbz5-cq8wf   Ready   192.168.49.2   7011   agones   17m
simple-game-server-8pbz5-lk7l9   Ready   192.168.49.2   7068   agones   27m
simple-game-server-8pbz5-tb4qk   Ready   192.168.49.2   7055   agones   17m
```

## Step 7: Cleanup

You can delete the autoscaler and fleet with the following commands:

```bash
kubectl delete fleetautoscaler wasm-fleet-autoscaler
```

```bash
kubectl delete fleet simple-game-server
```

## Creating Your Own Wasm Autoscaler

To create your own Wasm autoscaler module, you need to:

1. **Write a function** in a language that compiles to WebAssembly (such as Rust, Go, or AssemblyScript) with the 
   [Extism](https://extism.org/) WebAssembly plugin framework.
2. **Export the function** with a specific signature that receives fleet data and returns the desired replica count
3. **Compile to Wasm** and make the `.wasm` file accessible via URL or Kubernetes service
4. **Configure the FleetAutoscaler** to reference your Wasm module

The Wasm module receives information about the fleet including:
- Current replica count
- Ready replica count
- Allocated replica count
- Reserved replica count
- Counter and List values (if using Counters and Lists features)

Your function should return the desired number of replicas based on your custom logic.

For a complete example of a Wasm autoscaler implementation, see the {{< ghlink href="examples/autoscaler-wasm/" >}}autoscaler-wasm example{{< /ghlink >}} in the Agones repository
and the [FleetAutoscaler Reference]({{< ref "/docs/Reference/fleetautoscaler.md#wasm-autoscaling" >}}).

## Troubleshooting Guide

If you run into problems with the configuration of your Wasm fleetautoscaler, the easiest way to debug is to run:

```bash
kubectl describe fleetautoscaler wasm-fleet-autoscaler
```

and inspect the events at the bottom of the output.

### Common Error Messages

**Invalid Wasm module or function not found:**
```
Error: failed to call Wasm plugin function missing: unknown function: scale
```
This means the function name specified in the configuration doesn't exist in the Wasm module. Check that the `function` field matches an exported function in your Wasm module.

**Hash verification failed:**
```
Error: failed to create Wasm plugin from https://example.com/plugin.wasm: hash mismatch for module 'main'             
```
The downloaded Wasm module's SHA256 hash doesn't match the hash specified in the configuration. Verify that you're using the correct hash for your Wasm file.

**Failed to download Wasm module:**
```
Error: bad status code 404 from the server: https://example.com/autoscaler-wasm/pluginZZZ.wasm
```
The autoscaler cannot reach the URL specified in the configuration. Check that the URL is accessible from within the cluster and that any required services are running.

## Next Steps

For a complete reference of all Wasm autoscaler configuration options and specifications, see the [Fleet Autoscaler Specification]({{< relref "../Reference/fleetautoscaler.md#wasm-autoscaling" >}}).

Read the advanced [Scheduling and Autoscaling]({{< relref "../Advanced/scheduling-and-autoscaling.md" >}}) guide, for more details on autoscaling.

If you want to use your own GameServer container make sure you have properly integrated the [Agones SDK]({{< relref "../Guides/Client SDKs/_index.md" >}}).
