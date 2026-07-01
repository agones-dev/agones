---
title: "Windows Gameservers"
date: 2021-04-19T21:14:19Z
publishDate: 2021-04-27T23:00:00Z
weight: 20
description: >
  Run `GameServers` on Kubernetes nodes with the Windows operating system.
---

{{% alert title="Note" color="info" %}}
Windows GameServer support is in Beta. Only the SDK sidecar (`sdk-server`)
runs on Windows nodes. All Agones control-plane components remain Linux-only.
{{% /alert %}}

## Prerequisites

{{< gs-prerequisites >}}

Ensure that you have some nodes to your cluster that are running Windows.

## Objectives

- Create a GameServer on a Windows node.
- Connect to the GameServer.

### 1. Create a GameServer

{{% alert title="Note" color="info" %}}
Starting with version 0.3, the {{< ghlink href="examples/simple-game-server/" >}}simple-game-server{{< /ghlink >}} example is compiled as a multi-arch docker image that will run on both Linux and Windows. To ensure that the game server runs on a Windows node, a nodeSelector of `"kubernetes.io/os": windows` must be added to the game server specification.
{{% /alert %}}

Create a GameServer using the following command:

```bash
kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/simple-game-server/gameserver-windows.yaml
```

You should see a successful output similar to this:

```
gameserver.agones.dev/simple-game-server-4ss4j created
```

Verify that the GameServer becomes Ready by running:

```bash
kubectl get gameservers
```
It should look something like this:

```
NAME                       STATE     ADDRESS       PORT   NODE     AGE
simple-game-server-7pjrq   Ready   35.233.183.43   7190   agones   3m
```

Take a note of the Game Server IP address and ports.

For the full details of the YAML file head to the [GameServer Specification Guide]({{< ref "/docs/Reference/gameserver.md" >}})


### 2. Connect to the GameServer

{{% alert title="Note" color="info" %}}
If you have Agones installed on Google Kubernetes Engine, and are using
  Cloud Shell for your terminal, UDP is blocked. For this step, we recommend
  SSH'ing into a running VM in your project, such as a Kubernetes node.
  You can click the 'SSH' button on the [Google Compute Engine Instances](https://console.cloud.google.com/compute/instances)
  page to do this.
  Run `toolbox` on GKE Node to run docker container with tools and then `nc` command would be available.
{{% /alert %}}

You can now communicate with the Game Server:

{{% alert title="Note" color="info" %}}
If you do not have netcat installed
  (i.e. you get a response of `nc: command not found`),
  you can install netcat by running `sudo apt install netcat`.

If you are on Windows, you can alternatively install netcat on
[WSL](https://docs.microsoft.com/en-us/windows/wsl/install-win10),
or download a version of netcat for Windows from [nmap.org](https://nmap.org/ncat/).
{{% /alert %}}

```
nc -u {IP} {PORT}
Hello World !
ACK: Hello World !
EXIT
```

You can finally type `EXIT` which tells the SDK to run the [Shutdown command]({{< ref "/docs/Guides/Client SDKs/_index.md#shutdown" >}}), and therefore shuts down the `GameServer`.

If you run `kubectl describe gameserver` again - either the GameServer will be gone completely, or it will be in `Shutdown` state, on the way to being deleted.


## Next Steps

- Make a local copy of the simple-game-server {{< ghlink href="examples/simple-game-server/fleet.yaml" >}}fleet configuration{{< /ghlink >}},
modify it to include a node selector, and use it to go through the [Quickstart: Create a Game Server Fleet]({{< ref "/docs/Getting Started/create-fleet.md" >}}).
- If you want to use your own GameServer container make sure you have properly integrated the [Agones SDK]({{< ref "/docs/Guides/Client SDKs/_index.md" >}}).


