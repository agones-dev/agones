# Security policy
## Security considerations
Agones runs on Kubernetes and extends the API with custom resources and controllers. The security of a deployment depends on correct cluster configuration (for example RBAC, network policies, TLS for the allocator service, and node isolation). Operational guidance appears in the [Agones documentation](https://agones.dev/site/docs/), including [best practices](https://agones.dev/site/docs/guides/best-practices/) and topics such as [service accounts](https://agones.dev/site/docs/advanced/service-accounts/) and the [allocator service](https://agones.dev/site/docs/advanced/allocator-service/).

## Security contacts
Security reports are triaged by the [Agones maintainers](https://github.com/agones-dev/agones/blob/main/docs/governance/community_membership.md#current-members). Coordinated fixes and public advisories are published through [GitHub Security Advisories](https://github.com/agones-dev/agones/security/advisories) when appropriate.

## Reporting a vulnerability
Please **do not report security vulnerabilities through public GitHub issues, discussions, or pull requests**.

### Preferred: GitHub
If you believe you have found a security vulnerability in Agones, please report it privately through the following steps:

1. Open a [private vulnerability report](https://github.com/agones-dev/agones/security/advisories/new) for this repository (also reachable from the **Security** tab).
2. Include a clear description, affected version(s) if known, steps to reproduce, and any suspected impact or mitigations.
3. If available, please also include a Proof-of-concept, logs, or other supporting information.

Maintainers will acknowledge receipt as soon as practical, typically within **five business days**, and will work with you on validation, fixes, and disclosure timing.

## Disclosure timeline
We follow **coordinated disclosure**: details stay private until a fix is available or the risk and response have been agreed with the reporter. The exact timeline depends on severity, complexity, and release cadence; we aim to ship security fixes in patch or regular releases and to publish an advisory when users should upgrade.

## Embargo
Information shared with reporters, distributors, or other participants before public disclosure is confidential. It must not be shared more widely than needed to fix or validate the issue, and must not be made public before the agreed disclosure date. 

If you believe embargo terms were broken, contact the maintainers through the same private reporting channel used for the issue.

## Supported versions
Security fixes are published in [releases](https://github.com/agones-dev/agones/releases) as needed. We recommend running a [currently supported Kubernetes version](https://agones.dev/site/docs/installation/#usage-requirements) and upgrading to the latest Agones release you can adopt, especially when release notes or advisories reference security-related changes.
