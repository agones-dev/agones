---
title: "Feature Stages"
linkTitle: "Feature Stages"
date: 2019-09-26T01:20:41Z
weight: 5
description: >
  This page provides a description of the various stages that Agones features can be in, and the
  relative maturity and support level expected for each level.
---

## Supported Versions

Agones versions are expressed as x.y.z, where x is the major version, y is the minor version, and z is the patch version
following [Semantic Versioning](http://semver.org/) terminology.

## Agones Features

A feature within Agones can be in `Alpha`, `Beta` or `Stable` stage.

## Feature Gates

`Alpha` and `Beta` features can be enabled or disabled through the `agones.featureGates` configuration option
that can be found in the [Helm configuration]({{< ref "/docs/Installation/Install Agones/helm.md#configuration" >}})
documentation. `Stable` features are always enabled, and exist for upgrade compatibility, and cannot
be disabled.

The current set of `alpha`, `beta` and `stable` feature gates include:

### Alpha

| Feature Name                                                                                                      | Gate                     | Default  | Stage   | Since  |
|-------------------------------------------------------------------------------------------------------------------|--------------------------|----------|---------|--------|
| [Player Tracking]({{< ref "/docs/Guides/player-tracking.md" >}})                                                  | `PlayerTracking`         | Disabled | `Alpha` | 1.6.0  |
| Example Gate (not in use)                                                                                         | `Example`                | Disabled | None    | 0.13.0 |
| [GameServer player capacity filtering on GameServerAllocations](https://github.com/agones-dev/agones/issues/1239) | `PlayerAllocationFilter` | Disabled | `Alpha` | 1.14.0 |
| [Wasm Autoscaler](https://github.com/agones-dev/agones/issues/4080)                                               | `WasmAutoscaler`         | Disabled | `Alpha` | 1.54.0 |

### Beta

| Feature Name                                                                                                           | Gate                               | Default | Stage  | Since  |
|------------------------------------------------------------------------------------------------------------------------|------------------------------------|---------|--------|--------|
| [CountsAndLists](https://github.com/agones-dev/agones/issues/2716)                                                     | `CountsAndLists`                   | Enabled | `Beta` | 1.41.0 |
| [Support for Extended Duration Pods on GKE Autopilot (*1.28+ only*)](https://github.com/agones-dev/agones/issues/3386) | `GKEAutopilotExtendedDurationPods` | Enabled | `Beta` | 1.44.0 |
| [Port Policy None](https://github.com/agones-dev/agones/issues/3804)                                                   | `PortPolicyNone`                   | Enabled | `Beta` | 1.49.0 |
| [Multiple dynamic port ranges](https://github.com/agones-dev/agones/issues/1911)                                       | `PortRanges`                       | Enabled | `Beta` | 1.49.0 |
| [Rolling Update Fixes](https://github.com/agones-dev/agones/issues/3688)                                               | `RollingUpdateFix`                 | Enabled | `Beta` | 1.50.0 |
| [Scheduled Fleet Autoscaling](https://github.com/agones-dev/agones/issues/3008)                                        | `ScheduledAutoscaler`              | Enabled | `Beta` | 1.51.0 |
| [Extend Webhook autoscaler to send fleet metadata with the request](https://github.com/agones-dev/agones/issues/3951)  | `FleetAutoscaleRequestMetaData`    | Enabled | `Beta` | 1.54.0 |
| [Sidecar Containers](https://github.com/agones-dev/agones/issues/3642)                                                 | `SidecarContainers`                | Enabled | `Beta` | 1.56.0 |

### Stable Features

| Feature Name                                                                             | Gate                       | Default | Stage    | Since  |
|------------------------------------------------------------------------------------------|----------------------------|---------|----------|--------|
| [DisableResyncOnSDKServer](https://github.com/agones-dev/agones/issues/3377)             | `DisableResyncOnSDKServer` | Enabled | `Stable` | 1.49.0 |
| [Support Passthrough on GKE Autopilot](https://github.com/agones-dev/agones/issues/3721) | `AutopilotPassthroughPort` | Enabled | `Stable` | 1.54.0 |

[fleet-updates]: {{% relref "./fleet-updates.md#notifying-gameservers-on-fleet-updatedownscale" %}}

{{% alert title="Note" color="info" %}}
If you aren't sure if Feature Flags have been set correctly, have a look at the
_[The Feature Flag I enabled/disabled isn't working as expected]({{% relref "troubleshooting.md#the-feature-flag-i-enableddisabled-isnt-working-as-expected" %}})_
troubleshooting section.
{{% /alert %}}

## Description of Stages

### Alpha

An `Alpha` feature means:

* Disabled by default.
* Might be buggy. Enabling the feature may expose bugs.
* Support for this feature may be dropped at any time without notice.
* The API may change in incompatible ways in a later software release without notice.
* Recommended for use only in short-lived testing clusters, due to increased risk of bugs and lack of long-term support.

{{% alert title="Note" color="info" %}}
Please do try `Alpha` features and give feedback on them. This is important to ensure less breaking changes
through the `Beta` period.
{{% /alert %}}

### Beta

A `Beta` feature means:

* Enabled by default, but able to be disabled through a feature gate.
* The feature is well tested. Enabling the feature is considered safe.
* Support for the overall feature will not be dropped, though details may change.
* The schema and/or semantics of objects may change in incompatible ways in a subsequent beta or stable releases. When
  this happens, we will provide instructions for migrating to the next version. This may require deleting, editing,
  and re-creating API objects. The editing process may require some thought. This may require downtime for
  applications that rely on the feature.
* Recommended for only non-business-critical uses because of potential for incompatible changes in subsequent releases.
  If you have multiple clusters that can be upgraded independently, you may be able to relax this restriction.

{{% alert title="Note" color="info" %}}
Note: Please do try `Beta` features and give feedback on them! After they exit beta, it may not be practical for us
to make more changes.
{{% /alert %}}

### Stable

A `Stable` feature means:

* The feature is enabled and the corresponding feature gate no longer exists.
* Stable versions of features will appear in released software for many subsequent versions.

## Feature Stage Indicators

There are a variety of features with Agones, how can we determine what stage each feature is in?

Below are indicators for each type of functionality that can be used to determine the feature stage for a given aspect
of Agones.

### Custom Resource Definitions (CRDs)

This refers to Kubernetes resource for Agones, such as `GameServer`, `Fleet` and `GameServerAllocation`.

#### New CRDs

For new resources, the stage of the resource will be indicated by the `apiVersion` of the resource.

For example: `apiVersion: "agones.dev/v1"` is a `stable` resource, `apiVersion: "agones.dev/v1beta1"` is a `beta`
 stage resource, and `apiVersion: "agones.dev/v1alpha1"` is an `alpha` stage resource.

#### New CRD attributes

When `alpha` and `beta` attributes are added to an existing stable Agones CRD, we will follow the Kubernetes [_Adding
 Unstable Features to Stable Versions_](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#adding-unstable-features-to-stable-versions)
Guide to optimise on the least amount of breaking changes for users as attributes progress through feature stages.

`alpha` and `beta` attributes will be added to the existing CRD as `optional` and documented with their feature stage.
Attempting to populate these `alpha` and `beta` attributes on an Agones CRD will return a validation error if their
 accompanying Feature Flag is not enabled.

`alpha` and `beta` attributes can be subject to change of name and structure, and will result in breaking changes
 before moving to a `stable` stage. These changes will be outlined in release notes and feature documentation.

### Agones Game Server SDK

Any `alpha` or `beta` Game Server SDK functionality will be a subpackage of the `sdk` package. For example
, functionality found in a `sdk.alphav1` package should be considered at the `alpha` feature stage.

Only experimental functionality will be found in any `alpha` and `beta` SDK packages, and as such may change as
development occurs.

As SDK features move to through feature stages towards `stable`, the previous version of the SDK API
will remain for at least one release to enable easy migration to the more stable feature stage (i.e. from `alpha
` -> `beta`, `beta` -> `stable`)

Any other SDK functionality not marked as `alpha` or `beta` is assumed to be `stable`.

### REST & gRPC APIs

REST and gRPC API will have versioned paths where appropriate to indicate their feature stage.

For example, a REST API with a prefix of `v1alpha1` is an `alpha` stage feature:
`http://api.example.com/v1alpha1/exampleaction`.

Similar to the SDK, any `alpha` or `beta` gRPC functionality will be a subpackage of the main API package.
For example, functionality found in a `api.alphav1` package should be considered at the `alpha` feature stage.
