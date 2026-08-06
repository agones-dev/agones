---
title: "Limiting CPU & Memory"
date: 2019-01-03T05:45:15Z
weight: 30
description: >
  Kubernetes natively has inbuilt capabilities for requesting and limiting both CPU and Memory usage of running containers.
---

As a short description:

- CPU `Requests` are limits that are applied when there is CPU congestion, and as such can burst above their set limits.
- CPU `Limits` are hard limits on how much CPU time the particular container gets access to.

This is useful for game servers, not just as a mechanism to distribute compute resources evenly, but also as a way
to advice the Kubernetes scheduler how many game server processes it is able to fit into a given node in the cluster.

It's worth reading the [Managing Compute Resources for Containers](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/)
Kubernetes documentation for more details on "requests" and "limits" to both CPU and Memory, and how to configure them.

## GameServers

Since the `GameServer` specification provides a full [`PodSpecTemplate`]({{% k8s-api-version href="#podtemplatespec-v1-core" %}}),
we can take advantage of both resource limits and requests in our `GameServer` configurations.

For example, to set a CPU limit on our `GameServer` configuration of `250m/0.25` of a CPU,
we could do so as followed:

```yaml
apiVersion: "agones.dev/v1"
kind: GameServer
metadata:
  name: "simple-game-server"
spec:
  ports:
  - name: default
    containerPort: 7654
  template:
    spec:
      containers:
      - name: simple-game-server
        image: {{% example-image %}}
        resources:
          limits:
            cpu: "250m" #this is our limit here
```

If you do not set a limit or request, the default is set by Kubernetes at a 100m CPU request.

## SDK GameServer sidecar

You may also want to tweak the CPU request or limits on the SDK `GameServer` sidecar process that spins up alongside
each game server container.

You can do this through the [Helm configuration]({{< ref "/docs/Installation/Install Agones/helm.md" >}}) when installing Agones.

By default, this is set to having a CPU request value of 30m, with no hard CPU limit. This ensures that the sidecar always has enough CPU
to function, but it is configurable in case a lower, or higher value is required on your clusters, or if you desire
hard limit.

When a memory request is configured, Agones sets the SDK sidecar's `GOMEMLIMIT` to 90% of that request so the Go runtime
keeps its managed memory close to the amount scheduled for the sidecar.
