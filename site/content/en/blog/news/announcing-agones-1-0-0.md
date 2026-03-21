---
title: "Announcing Agones 1.0.0"
date: 2019-09-25T22:52:15Z
---

## History

In 2017, Agones started as a collaboration between [Ubisoft](https://www.ubisoft.com) and 
[Google Cloud](https://cloud.google.com/) to develop a turn-key, open source, solution for running, scaling and
orchestrating multiplayer dedicated game servers on top of Kubernetes. They shared an equal vision of expanding the
impact of open source in game development, especially in the infrastructure and multiplayer space, and an equal 
understanding of the power and capabilities of Kubernetes and how it could be extended to work with dedicated, 
authoritative, simulation game servers.

Almost two years later, with the release of the 1.0.0 of Agones, there are now 50 contributors from a wide variety of
game studios, and several games in production using the platform and many more in development. Not only that, the
product has developed and matured with the communities feedback and help in ways none of us ever envisioned from the
outset.

## Features

The main feature of Agones 1.0.0 is that we now have a stable API surface! Users can take advantage of Agones without 
fear of breaking changes.

Agones itself has a wide set of features, including, but not limited to:

*   Define single a `GameServer`, or large pre-spun game server `Fleets`, either through kubectl + yaml or via the 
    Kubernetes API.
*   Manage GameServer life cycles - including health checking and connection information through configuration and an
    integrated SDK.
*   Game server `Fleet` autoscaling capabilities that integrate with Kubernetes' native cluster autoscaling.
*   Game server specific metric exports and dashboards for operations teams.
*   Allocate `GameServers` out of a set for players to play on, even while scaling or updating backing Fleet
    configuration and rollout.
*   Optimisation patterns for both Cloud and On-Premises to ensure cost effective usage of your infrastructure.
*   Modular architecture that can be further customised to the needs of your game.
*   Local development tools for fast development interaction without the need of a full Kubernetes cluster.
*   … and even more!

## What’s Next

We have more exciting this on the horizon! Items on the roadmap include:

*   Scheduled Agones stress testing, and public performance dashboards.
*   Player tracking and metrics.
*   Common and custom in-game metrics.
*   Windows support.

But we also want to hear from our users and testers of Agones -- what would you like to see in the project? If there
is a feature that would be useful for you, please 
[file a feature request](https://github.com/agones-dev/agones/issues/new?assignees=&labels=kind%2Ffeature&template=feature_request.md&title=),
or talk about it in our
[Slack channel](https://join.slack.com/t/agones/shared_invite/enQtMzE5NTE0NzkyOTk1LWU3ODAyZjdjMjNlYWIxZTAwODkxMGY3YWEyZjNjMjc4YWM1Zjk0OThlMGU2ZmUyMzRlMDljNDJiNmZlMGQ1M2U)!

## Getting Started

If you want to get started, have a look at the [installation guide](https://agones.dev/site/docs/installation/) for
your preferred Kubernetes platform, and then follow our [quickstarts](https://agones.dev/site/docs/getting-started/) to
get a simple UDP server up and running, perform autoscaling, allocate game servers, and more.

## Finally

A massive thanks to everyone in the Agones community - from our users, to people that have submitted bugs and feature
requests, to contributors, approvers and more. This has truly been a group effort, and it wouldn’t have been
possible without the time and effort that many people have put into this project.
