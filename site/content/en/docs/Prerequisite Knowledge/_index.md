---
title: "Prerequisite Knowledge"
linkTitle: "Prerequisite Knowledge"
weight: 3
description: >
  Foundational knowledge you should know before starting working with Agones.
---

Agones is built on top of the foundation of multiple open source projects, as well as utilising
several architectural patterns across both distributed and multiplayer game systems -- which can
make it complicated to get started with, if they are things you are not familiar with or have
experience with already.

To make getting started easier to break down and digest, this guide was written to outline what concepts and
technology that the documentation assumes that you have knowledge of, and the
depth of that knowledge, as well as providing resource to help fill those knowledge gaps.

## Docker and Containerisation

Docker and containerisation is the technological foundation of Agones, so if you aren't familiar,
we recommend you have knowledge in the following areas before getting started with Agones:

* Containers as a concept
* Running Docker containers
* Building your own container
* Registries as a concept
* Pushing and pulling containers from a registry

### Resources

The following resources are great for learning these concepts:

* [Get started with Docker containers](https://docs.docker.com/learn/paths/get-started-with-docker-containers/)
* [Docker "Quickstart" guide](https://docs.docker.com/get-started/)

## Kubernetes

Kubernetes builds on top of Docker to run containers at scale, on lots of machines.
If you have yet to learn about Kubernetes, we recommend that you have knowledge in the following
areas before getting started with Agones:

* Kubernetes as a concept - you should take the [basics tutorial](https://kubernetes.io/docs/tutorials/kubernetes-basics/)
* Pods
* Deployments
* Services
* Creating a Deployment with a Service

### Mappings in Agones

Agones extends the Kubernetes API to include new [custom resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) such as `GameServer` and `Fleet`. See the [Reference documentation](https://agones.dev/site/docs/reference/) for more information.

Agones creates a backing Pod with the appropriate configuration parameters for
each `GameServer` that is configured in a cluster. They both have the same name.

### Resources

* [You should totally read this comic and interactive tutorial](https://cloud.google.com/kubernetes-engine/kubernetes-comic/)
* [Kubernetes concepts, explained](https://kubernetes.io/docs/concepts/)

## Dedicated Game Servers

Agones is a platform for dedicated game servers for multiplayer games. If "dedicated game servers" is a term that is not
something you are familiar with, we recommend checking out some of the resources below, before getting started with
Agones:

### Resources

* [Dedicated Game Servers, Drawn Badly (video)](https://www.youtube.com/watch?v=Nl_FIGFtYdc)
* [What Every Programmer Needs To Know About Game Networking](https://gafferongames.com/post/what_every_programmer_needs_to_know_about_game_networking/)
* [Fast-Paced Multiplayer (Part I): Client-Server Game Architecture](https://www.gabrielgambetta.com/client-server-game-architecture.html)
* [Game Server (wikipedia)](https://en.wikipedia.org/wiki/Game_server)
* {{< ghlink href="examples/simple-game-server" >}}Example simple gameserver that responds to UDP and/or
 TCP commands{{< /ghlink >}}

## Game Engine

If you are building a multiplayer game, you will eventually need to understand how your
[game engine](https://en.wikipedia.org/wiki/Game_engine) will integrate with Agones.
There are multiple possible solutions, but the engines that have out of the box SDK's for Agones are:

* [Unity](https://unity.com/)
* <a href="https://www.unrealengine.com/" data-proofer-ignore>Unreal</a>

While you can integrate other engines with Agones, if you are new to developing on a game engine, you may want to
start with the above.

## Additional Resources

* [Game Network Resources](https://gamenetcode.com/) is a curated list of multiplayer game network programming (netcode) resources. Includes articles, talks, and tools covering various aspects of multiplayer game development. Use this resource to learn more about fundamental subjects in multiplayer game programming from companies such as Riot, Bungie, and Activision.
