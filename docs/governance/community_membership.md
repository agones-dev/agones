# Community Membership

This document outlines the responsibilities of contributor roles in Agones.

This is based on the [Kubernetes Community Membership](https://github.com/kubernetes/community/blob/master/community-membership.md).

There are currently two roles for this project, but that may grow in the future.

| Role         | Responsibilities                 | Requirements                                                       | Defined by                                                              |
|--------------|----------------------------------|--------------------------------------------------------------------|-------------------------------------------------------------------------|
| Collaborator | Can have tickets assigned        | Have one PR merged                                                 | Read access to the Agones repository                                    |
| Releaser     | Create Agones releases           | Sponsored by 2 approvers                                           | Commit access to the Agones repository                                  |
| Approver     | Review and approve contributions | Sponsored by 2 approvers and multiple contributions to the project | Commit access to the Agones repository and [OWNERS] file approver entry |


## New contributors

New contributors should be welcomed to the community by existing members,
helped with PR workflow, and directed to relevant documentation and
communication channels.

## Established community members

Established community members are expected to demonstrate their adherence to the
principles in this document, familiarity with project organization, roles,
policies, procedures, conventions, etc., and technical and/or writing ability.
Role-specific expectations, responsibilities, and requirements are enumerated
below.

## Collaborator

For regular contributors that wish to have issues assigned to them, we have the collaborator role.

To become a collaborator, create an issue using the 
[become a repo collaborator issue template](https://github.com/agones-dev/agones/issues/new?assignees=thisisnotapril&labels=area%2Fcommunity&projects=&template=become-a-repo-collaborator.md&title=Collaborator+Request)
and we will review it as a team.

**Defined by:** Read access to the Agones repository.

### Requirements

- Have at least one merged Pull Request.
- Have reviewed the [contribution guidelines](https://github.com/agones-dev/agones/blob/main/CONTRIBUTING.md)
- Have enabled [2FA on my GitHub account](https://github.com/settings/security)
- Have joined the [Agones Slack workspace](https://join.slack.com/t/agones/shared_invite/zt-2mg1j7ddw-0QYA9IAvFFRKw51ZBK6mkQ)

## Releaser

Releasers are engineers that are able to commit to the Agones repository with
the express purpose of publishing Agones releases. They are **not** approvers
and as such are not expected to perform code reviews or merge PRs. Using their
commit privileges above and beyond creating releases is grounds for revoking
commit access and being demoted from being a releaser.

**Defined by:** Commit access to the Agones repository.

### Requirements

- Enabled [two-factor authentication](https://help.github.com/articles/about-two-factor-authentication)
  on their GitHub account
- Work at Google
    - This is due to some steps in the [release process](release_process.md) that require Google internal processes
- Have read the [contributor guide](../../CONTRIBUTING.md)
- Sponsored by 2 approvers. **Note the following requirements for sponsors**:
    - Unlike for the Approver role, sponsors should both be from Google
- **[Open an issue](./templates/membership.md) against the Agones repo**
   - Ensure your sponsors are @mentioned on the issue
   - Label the issue with the `meta` tag
   - Complete every item on the checklist ([preview the current version of the template](./templates/membership.md))
   - There is no need to include any contributions to the Agones project
- Have your sponsoring approvers reply confirmation of sponsorship: `+1`

### Responsibilities and privileges

- Responsible for creating Agones releases
- Granted commit access to Agones repo

## Approver

Code approvers are able to both review and approve code contributions.  While
code review is focused on code quality and correctness, approval is focused on
holistic acceptance of a contribution including: backwards / forwards
compatibility, adhering to API and flag conventions, subtle performance and
correctness issues, interactions with other parts of the system, etc.

**Defined by:** Commit access to the Agones repository and [OWNERS] file approver entry.

**Note:** Acceptance of code contributions requires at least one approver.

### Requirements

- Enabled [two-factor authentication](https://help.github.com/articles/about-two-factor-authentication)
  on their GitHub account
- Have made multiple contributions to Agones.  Contribution must include:
    - Authored at least 3 PRs on GitHub
    - Provided reviews on at least 4 PRs they did not author
    - Filing or commenting on issues on GitHub
- Have read the [contributor guide](../../CONTRIBUTING.md)
- Sponsored by 2 approvers. **Note the following requirements for sponsors**:
    - Sponsors must have close interactions with the prospective member - e.g. code/design/proposal review, coordinating
      on issues, etc.
    - Sponsors must be from multiple companies to demonstrate integration across community.
- **[Open an issue](./templates/membership.md) against the Agones repo**
   - Ensure your sponsors are @mentioned on the issue
   - Label the issue with the `meta` tag
   - Complete every item on the checklist ([preview the current version of the template](./templates/membership.md))
   - Make sure that the list of contributions included is representative of your work on the project
- Have your sponsoring approvers reply confirmation of sponsorship: `+1`

### Responsibilities and privileges

- Responsible for project quality control via code reviews
  - Focus on code quality and correctness, including testing and factoring
  - May also review for more holistic issues, but not a requirement
- Expected to be responsive to review requests in a timely manner
- Assigned PRs to review related based on expertise
- Granted commit access to Agones repo

[OWNERS]: https://github.com/kubernetes/community/blob/master/contributors/guide/owners.md

