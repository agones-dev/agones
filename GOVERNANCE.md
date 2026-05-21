# Agones Project Governance

The Agones project is dedicated to creating an open source, batteries-included, dedicated game server hosting platform built on Kubernetes.
This governance explains how the project is run.

- [Values](#values)
- [Approvers](#approvers)
- [Becoming an Approver](#becoming-an-approver)
- [Meetings](#meetings)
- [CNCF Resources](#cncf-resources)
- [Code of Conduct Enforcement](#code-of-conduct)
- [Security Response Team](#security-response-team)
- [Voting](#voting)
- [Modifications](#modifying-this-charter)

## Values

The Agones project and its leadership embrace the following values:

* Openness: Communication and decision-making happens in the open and is discoverable for future
  reference. As much as possible, all discussions and work take place in public
  forums and open repositories.

* Fairness: All stakeholders have the opportunity to provide feedback and submit
  contributions, which will be considered on their merits.

* Community over Product or Company: Sustaining and growing our community takes
  priority over shipping code or sponsors' organizational goals.  Each
  contributor participates in the project as an individual.

* Inclusivity: We innovate through different perspectives and skill sets, which
  can only be accomplished in a welcoming and respectful environment.

* Participation: Responsibilities within the project are earned through
  participation, and there is a clear path up the contributor ladder into leadership
  positions.

## Approvers

Agones Approvers have write access to the [project GitHub repository](https://github.com/agones-dev/agones).
They can merge their own patches or patches from others. The current approvers
can be found in [community_membership.md](./docs/governance/community_membership.md).  Approvers collectively manage the project's
resources and contributors.

This privilege is granted with some expectation of responsibility: approvers
are people who care about the Agones project and want to help it grow and
improve. An approver is not just someone who can make changes, but someone who
has demonstrated their ability to collaborate with the team, get the most
knowledgeable people to review code and docs, contribute high-quality code, and
follow through to fix issues (in code or tests).

An approver is a contributor to the project's success and a citizen helping
the project succeed.

The collective team of all Approvers is known as the Approver Council, which
is the governing body for the project.

### Becoming an Approver

Requirements and the nomination process for becoming an Approver are defined in
[community_membership.md](./docs/governance/community_membership.md#approver).

Approvers who are selected will be granted the necessary GitHub rights,
invited to the [private approver mailing list](mailto:agones-admin@googlegroups.com), and provided access to all appropriate
resources.

### Removing an Approver

Resignation, removal criteria, and Emeritus status are defined in
[community_membership.md](./docs/governance/community_membership.md#resignation-and-removal).

## Meetings

Time zones permitting, Approvers are expected to participate in the public
community meeting as often as they are able, which occurs monthly. 
Meeting details and calendar invites are available via the [mailing list](https://groups.google.com/forum/#!forum/agones-discuss)
and the [community calendar](https://calendar.google.com/calendar/embed?src=google.com_828n8f18hfbtrs4vu4h1sks218%40group.calendar.google.com&ctz=America%2FLos_Angeles).
Past meeting recordings are available on the [community YouTube playlist](https://www.youtube.com/playlist?list=PLhkWKwFGACw2dFpdmwxOyUCzlGP2-n7uF).

Approvers will also have closed meetings in order to discuss security reports
or Code of Conduct violations.  Such meetings should be scheduled by any
Approver on receipt of a security issue or CoC report.  All current Approvers
must be invited to such closed meetings, except for any Approver who is
accused of a CoC violation.

## CNCF Resources

Any Approver may suggest a request for CNCF resources, either in the
[mailing list](https://groups.google.com/forum/#!forum/agones-discuss), the #development Slack channel, or during a
meeting.  A simple majority of Approvers approves the request.  The Approvers
may also choose to delegate working with the CNCF to non-Approver community
members, who will then be added to the [CNCF's Maintainer List](https://github.com/cncf/foundation/blob/main/project-maintainers.csv)
for that purpose.

## Code of Conduct

[Code of Conduct](./code-of-conduct.md)
violations by community members will be discussed and resolved
on the [private Approver mailing list](mailto:agones-admin@googlegroups.com) or on a private Slack channel.  
If an Approver is directly involved in the report, the Approvers will instead designate two Approvers to work
with the CNCF Code of Conduct Committee in resolving it.

## Security Response Team

The Approvers will appoint a Security Response Team to handle security reports.
This committee may simply consist of the Approver Council themselves.  If this
responsibility is delegated, the Approvers will appoint a team of at least two
contributors to handle it.  The Approvers will review who is assigned to this
at least once a year.

The Security Response Team is responsible for handling all reports of security
holes and breaches according to the [security policy](.github/SECURITY.md).

## Voting

While most business in Agones is conducted by "[lazy consensus](https://community.apache.org/committers/lazyConsensus.html)",
periodically the Approvers may need to vote on specific actions or changes.
A vote can be taken on [the developer mailing list](https://groups.google.com/forum/#!forum/agones-discuss), private Slack channels,
[the private Approver mailing list](mailto:agones-admin@googlegroups.com) for security or conduct matters, or via a GitHub issue for public matters.
Votes may also be taken at [the community meeting](https://calendar.google.com/calendar/embed?src=google.com_828n8f18hfbtrs4vu4h1sks218%40group.calendar.google.com&ctz=America%2FLos_Angeles).  Any Approver may
demand a vote be taken.

Most votes require a simple majority of all Approvers to succeed, except where
otherwise noted.  Two-thirds majority votes mean at least two-thirds of all
existing approvers.

## Modifying this Charter

Changes to this Governance and its supporting documents may be approved by
a 2/3 vote of the Approvers. Proposed changes should be submitted as a GitHub Pull Request or Issue,
with the vote conducted on the PR or issue itself.
