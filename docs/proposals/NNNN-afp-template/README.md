<!--
**Note:** When your AFP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Create an issue in agones repository**
  When filing an feature tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the AFP. You
  can leave that blank until this AFP is filed, and then go back to the
  feature and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning docs/proposals directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your proposal above.
- [ ] **Fill out as much of the afp.yaml file as you can.**
  At minimum, you should fill in the "Title", "AFP-Number", "Authors",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the AFP.
- [ ] **Create a PR for this AFP.**
  Assign it to the relevant people for approval.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the AFP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a AFP is merged does not mean it is complete or approved. Any AFP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing AFPs, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/docs/proposals/NNNN-afp-template/README.md).

**Note:** Any PRs to move a AFP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the AFP reviewers.
If none of those reviewers are still appropriate, then changes to that list
should be approved by the remaining reviewers).
-->

# AFP-NNNN: Your short, descriptive title

<!--
This is the title of your AFP. Keep it short, simple, and descriptive. A good
title can help communicate what the AFP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a AFP and for
highlighting any additional information provided beyond the standard AFP
template.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [e2e tests](#e2e-tests)
- [API Impact surfaces](#api-impact-surfaces)
- [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [agones/proposals] referencing this AFP and targeting a release
version **before the [Feature Freeze](https://github.com/agones-dev/agones/tree/main/site/content/en/blog/releases)
of the targeted release**.

For features that make changes to code or processes/procedures in core
Agones.e., [googleforgames/agones], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the proposal to be released.
-->

Items marked with (R) are required *prior to targeting to a release version / release*.

- [ ] (R) Proposal issue linked to AFP directory in [agones/proposals] (not initial AFP PR)
- [ ] (R) AFP reviewers have approved the AFP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan in place, including input from Agones Architecture and Testing
  - [ ] (R) End-to-End (E2E) Tests must cover all key Agones API operations (e.g., GameServer, GameServerSet, Fleet, FleetAllocation, and Controller).
- [ ] (R) Implementation review completed
- [ ] (R) Implementation review approved
- [ ] "Implementation History" section is up-to-date for release target
- [ ] User-facing documentation has been created in [agones/site], for publication to [agones.dev]
- [ ] Supporting documentation (e.g., design docs, mailing list discussions, relevant PRs/issues, release notes) prepared

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this proposals is being considered for a release target.
-->

[agones.dev]: https://agones.dev
[agones/proposals]: https://github.com/agones-dev/agones/tree/main/docs/proposals
[googleforgames/agones]: https://github.com/agones-dev/agones
[agones/site]: https://github.com/agones-dev/agones/tree/main/site

## Summary

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. AFP editors and Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this AFP. Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a AFP within the wider Agones community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

<!--
List the specific goals of the AFP. What is it trying to achieve? How will we
know that this has succeeded?
-->

### Non-Goals

<!--
What is out of scope for this AFP? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this AFP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

#### Story 2

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Agones ecosystem.

How will security be reviewed, and by whom?

How will performance be evaluated, and by whom?

Consider including folks who also contribute outside the team or workgroup.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Agones testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://github.com/agones-dev/agones/tree/main/build#testing-and-building
-->

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this proposal to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

- `<package>`: `<date>` - `<test coverage>`

##### e2e tests

<!--
This question should be filled when targeting a release.
Describe what tests will be added to ensure proper quality of the proposal.
-->

- <test>: <link to test coverage>


### API Impact surfaces

<!--
This section provides detailed information regarding the impact of enabling or 
using a specific feature, particularly in relation to API calls and types. 
The answers to the following questions are crucial for understanding the changes 
to the system and ensuring efficient integration and usage.
-->

###### Will enabling / using this feature result in introducing new API?

<!--
Describe them, providing:
  - API details
-->

### Troubleshooting

<!--
Please provide details for this section. For final releases, 
reviewers will confirm answers based on practical experience. This information 
may later be expanded into a dedicated playbook with monitoring details.
-->

## Implementation History

<!--
Major milestones in the lifecycle of a AFP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Agones release where an initial version of the AFP was available
- the version of Agones where the AFP graduated to general availability
- when the AFP was retired or superseded
-->

## Drawbacks

<!--
Why should this AFP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!-- 
Use this section to specify any infrastructure needs for the project. 
Examples include: - Requesting a new subproject or repository - Setting up 
GitHub permissions or automation - Provisioning CI/CD resources - Any additional 
tooling or integrations. Listing these here allows a proposal to get 
the process for these resources started right away.
-->
