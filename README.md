<img src="./docs/agones.png" alt="Agones logo" width="250px" height="250px" />

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/agones.dev/agones)
[![GoDoc](https://godoc.org/agones.dev/agones?status.svg)](https://godoc.org/agones.dev/agones)
[![Go Report Card](https://goreportcard.com/badge/github.com/googleforgames/agones)](https://goreportcard.com/report/github.com/googleforgames/agones)
[![GitHub release](https://img.shields.io/github/release/googleforgames/agones.svg)](https://github.com/agones-dev/agones/releases)
[![Follow on Twitter](https://img.shields.io/twitter/follow/agonesdev.svg?style=social&logo=twitter)](https://twitter.com/intent/follow?screen_name=agonesdev)

Agones is a library for hosting, running and scaling [dedicated game servers](https://en.wikipedia.org/wiki/Game_server#Dedicated_server) on [Kubernetes](https://kubernetes.io).

_Agones, is derived from the Greek word agōn which roughly translates to “contest”, “competition at games” and “gathering”.
([source](https://www.merriam-webster.com/dictionary/agones))_

## Why does this project exist?
Agones replaces usual bespoke or proprietary cluster management and game server scaling solutions with a [Kubernetes](https://kubernetes.io/) cluster
that includes the Agones custom _[Kubernetes Controller](https://kubernetes.io/docs/concepts/api-extension/custom-resources/#custom-controllers)_ and matching [Custom Resource Definitions](https://kubernetes.io/docs/concepts/api-extension/custom-resources/#customresourcedefinitions) for _GameServers_, _Fleets_ and more.

With Agones, Kubernetes gets native abilities to create, run, manage and scale dedicated game server processes within Kubernetes clusters using standard Kubernetes tooling and APIs. This model also allows any matchmaker to interact directly with Agones via the Kubernetes API to provision a dedicated game server.

For more details on why this project was written, read the
[announcement blog post](https://cloudplatform.googleblog.com/2018/03/introducing-Agones-open-source-multiplayer-dedicated-game-server-hosting-built-on-Kubernetes.html).

## Major Features
- Define a single `GameServer`, and/or large game server `Fleets` within Kubernetes - either through yaml or via the API
- Manage GameServer lifecycles - including health checking and connection information.
- `Fleet` Autoscaling capabilities that integrate with Kubernetes' native cluster autoscaling
- Gameserver specific metric exports and dashboards for ops teams

## Usage

Documentation can be found on the [Agones website](https://agones.dev/site/docs/).

## Get involved

- [Slack](https://join.slack.com/t/agones/shared_invite/zt-2mg1j7ddw-0QYA9IAvFFRKw51ZBK6mkQ)
- [Twitter](https://twitter.com/agonesdev)
- [Mailing List](https://groups.google.com/forum/#!forum/agones-discuss)
- [Community Meetings](https://www.youtube.com/playlist?list=PLhkWKwFGACw2dFpdmwxOyUCzlGP2-n7uF) (Join the mailing
  list for details)

## Code of Conduct

Participation in this project comes under the [Contributor Covenant Code of Conduct](code-of-conduct.md)

## Reporting Security Issues

To report a security issue for this project, please follow the instructions in
the [Project Security Policy](.github/SECURITY.md)

## Development and Contribution

Please read the [contributing](CONTRIBUTING.md) guide for directions on submitting Pull Requests to Agones, and community membership governance.

See the [Developing, Testing and Building Agones](build/README.md) documentation for developing, testing and building Agones from source. For adding a new field to one of the Custom Resource Definitions refer to [Development Work on Agones Custom Resource Definitions (CRDs)](install/helm/agones/README.md).

For Agones client SDK development refer to [Developing, Testing, and Building Agones Client SDKs](sdks/README.md).

The [Release Process](docs/governance/release_process.md) documentation displays the project's upcoming release calendar and release process.

Agones is in active development - we would love your help in shaping its future!

## This all sounds great, but can you explain Docker and/or Kubernetes to me?

### Docker
- [Docker's official "Getting Started" guide](https://docs.docker.com/get-started/)

### Kubernetes
- [You should totally read this comic, and interactive tutorial](https://cloud.google.com/kubernetes-engine/kubernetes-comic/)

## Licence

Apache 2.0
