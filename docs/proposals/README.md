# Agones Feature Proposals (AFPs)

A Agones Feature Proposal (AFP) is a way to propose, communicate and coordinate on new efforts for the Agones project.

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
- [Stewardship](#stewardship)
- [Reference-Level Explanation](#reference-level-explanation)
  - [What Type of Work Should Be Tracked by a AFP](#what-type-of-work-should-be-tracked-by-a-afp)
  - [AFP Template](#afp-template)
  - [AFP Metadata](#afp-metadata)
  - [AFP Workflow](#afp-workflow)
  - [Git and GitHub Implementation](#git-and-github-implementation)
  - [AFP Editor Role](#afp-editor-role)
- [Drawbacks](#drawbacks)
- [FAQs](#faqs)
<!-- /toc -->

## Summary

A standardized development process for Agones is proposed, in order to:

- provide a common structure for proposing changes to Agones
- ensure that the motivation for a change is clear
- persist project information in a Version Control System (VCS) for future
  Agones
- support the creation of _high-value, user-facing_ information such as:
  - an overall project development roadmap
  - motivation for impactful user-facing changes
- reserve GitHub issues for tracking work in flight, instead of creating "umbrella"
  issues
- enable community participants to drive changes effectively across multiple releases 
  while ensuring stakeholder representation throughout the process.

This process is supported by a unit of work called a Agones Feature Proposal, or AFP.
A AFP attempts to combine aspects of

- a feature, and effort-tracking document
- a product requirements document
- a design document

These are created incrementally through collaboration with relevant stakeholders and teams.

## Motivation

Currently, design discussions in Agones primarily occur as comments on GitHub issues. 
While this provides flexibility, it can also lead to confusion and inefficiencies. 
For example, design discussions may be fragmented, causing difficulty in tracking 
updates and providing feedback on lengthy designs.

To improve clarity, the AFP process introduces a more structured way to propose and 
track changes. This process is inspired by Kubernetes' KEP process and is designed 
to be lightweight while supporting high-quality, uniform design and implementation documents.

## Stewardship
The following DACI model identifies the responsible parties for AFPs.

**Workstream** | **Driver** | **Approver** | **Contributor** | **Informed**
--- | --- | --- | --- | ---
| AFP Process Stewardship | - | - |  Project Maintainers | Community |
| Proposal Delivery | Proposal Owner |  Project Maintainers  | Proposal Implementer(s) (may overlap with Driver) | Community |

## Reference-Level Explanation

### What Type of Work Should Be Tracked by a AFP

The definition of what constitutes a "feature" is essential for the Agones project. 
Generally, any Agones user or operator-facing feature should follow the AFP process. This includes:

- Features that impact users or operators.
- Technical efforts such as refactoring or major architectural changes that impact a broad section of the development community.

The AFP process should be used to communicate changes that impact multiple areas of Agones or 
involve major cuts across the entire project.

### AFP Template

**The template for a AFP is precisely defined [here](/docs/proposals/NNNN-afp-template).**

### AFP Metadata

There is a place in each AFP for a YAML document that has standard metadata.
This will be used to support tooling around filtering and display. It is also
critical to clearly communicate the status of a AFP.

<b>
While this defines the metadata schema for now, these things tend to evolve.
The AFP template is the authoritative definition of things like the metadata
schema.
</b>

Metadata items:
* **title** Required
  * The title of the AFP in plain language. The title will also be used in the
    AFP filename.  See the template for instructions and details.
* **status** Required
  * The current state of the AFP.
  * Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
* **authors** Required
  * A list of authors of the AFP.
    This is simply the GitHub ID.
    In the future we may enhance this to include other types of identification.
* **reviewers** Required
  * Reviewer(s) chosen after triage, according to the proposal process.
  * Reviewer(s) selected based on the impacted areas or components of the project.
    It is up to the individual areas to determine how they pick reviewers for AFPs impacting them.
    The reviewers are speaking for the respective areas in the process of approving this AFP
    The impacted areas can modify this list as necessary.
  * The reviewers are the individuals who decide when to move this AFP to the `implementable` state.
  * If not yet chosen, replace with `TBD`.
  * Same name/contact scheme as `authors`.
* **editor** Required
  * Someone to keep things moving forward.
  * If not yet chosen, replace with `TBD`.
  * Same name/contact scheme as `authors`.
* **creation-date** Required
  * The date that the AFP was first submitted in a PR.
  * In the form `yyyy-mm-dd`.
  * While this info will also be in source control, it is helpful to have the set of AFP files stand on their own.
* **last-updated** Optional
  * The date that the AFP was last changed significantly.
  * In the form `yyyy-mm-dd`.
* **see-also** Optional
  * A list of other AFPs that are relevant to this AFP.
  * In the form `AFP-123`.
* **replaces** Optional
  * A list of AFPs that this AFP replaces. Those AFPs should list this AFP in
    their `superseded-by`.
  * In the form `AFP-123`.
* **superseded-by**
  * A list of AFPs that supersede this AFP. Use of this should be paired with
    this AFP moving into the `Replaced` status.
  * In the form `AFP-123`.

### AFP Workflow

A AFP has the following states:

- `provisional`: The AFP has been proposed and is actively being defined.
  This is the starting state while the AFP is being fleshed out and actively defined and discussed.
  The maintainer has accepted that this work must be done.
- `implementable`: The reviewers have approved this AFP for implementation.
- `implemented`: The AFP has been implemented and is no longer actively changed.
- `deferred`: The AFP is proposed but not actively being worked on.
- `rejected`: The reviewers and authors have decided that this AFP is not moving forward.
  The AFP is kept around as a historical document.
- `withdrawn`: The authors have withdrawn the AFP.
- `replaced`: The AFP has been replaced by a new AFP.
  The `superseded-by` metadata value should point to the new AFP.

### Git and GitHub Implementation

AFPs are checked into the repository under the `/docs/proposals` directory.

New AFPs should be checked in with a folder name in the form of `00x-feature-name`, where 
"00x" represents an incremental AFP number. The corresponding pull request (PR) should be titled 
in the format `AFP-00x`, matching the folder naming convention.

As significant work is done on the AFP, the authors can assign an AFP number. No other changes 
should be included in that PR so that it can be approved quickly and minimize merge conflicts. 
The AFP number can also be assigned as part of the initial submission if the PR is likely to 
be uncontested and merged quickly.

### AFP Editor Role

Taking a cue from the [Kubernetes KEP process][] & [Python PEP process][], we define the role of a AFP editor.
The job of an AFP editor is likely very similar to the [PEP editor responsibilities][] and will hopefully provide another opportunity for people who do not write code daily to contribute to Agones.

In keeping with the PEP editors, who:

> Read the PEP to check if it is ready: sound and complete. The ideas must make
> technical sense, even if they don't seem likely to be accepted.
> The title should accurately describe the content.
> Edit the PEP for language (spelling, grammar, sentence structure, etc.), markup
> (for reST PEPs), code style (examples should match PEP 8 & 7)

AFP editors should generally not pass judgement on a AFP beyond editorial corrections.
AFP editors can also help inform authors about the process and otherwise help things move smoothly.

[Kubernetes KEP process]: https://github.com/kubernetes/enhancements/tree/master/keps
[Python PEP process]: https://www.python.org/dev/peps/pep-0001/
[PEP editor responsibilities]: https://www.python.org/dev/peps/pep-0001/#pep-editor-responsibilities-workflow

## Drawbacks

Introducing the AFP process may initially add extra steps that could frustrate some community members. 
It could also introduce bottlenecks, particularly during the review and approval stages. 
However, the structure and clarity provided by AFPs will ultimately benefit larger, more complex projects like Agones.

While GitHub issues serve as a way to track changes, they can become cumbersome for managing 
long-term proposals and discussions. The AFP process aims to address these limitations by providing 
a more organized and structured approach.

## FAQs

### Do I have to use the AFP process?

More or less, yes.

Having a rich set of AFPs in one place will make it easier for people to track
what is going in the community and find a structured historical record.

AFPs are required for most non-trivial changes.  Specifically:
* Anything that may be controversial
* Most new features (except the very smallest)
* Major changes to existing features
* Changes that are wide ranging or impact most of the project (these changes
  are usually coordinated through the relevant maintainers)

Beyond these, it is up to the team to decide when they want to use the AFP 
process. It should be light-weight enough that AFPs are the default position.

### Why would I want to use the AFP process?

Our aim with AFPs is to clearly communicate new efforts to the Agones contributor community.
As such, we want to build a well curated set of clear proposals in a common format with useful metadata.

Benefits to AFP users (in the limit):
* Cross indexing of AFPs so that users can find connections and the current status of any AFP.
* A clear process with and reviewers for making decisions.
  This will lead to more structured decisions that stick as there is a discoverable record around the decisions.

### What will it take for AFPs to "graduate" out of "beta"?

Things we'd like to see happen to consider AFPs well on their way:
* A set of AFPs that show healthy process around describing an effort and recording decisions in a reasonable amount of time.
* AFPs indexed and maintained in a structured process for easy reference.
* Presubmit checks for AFPs around metadata format and markdown validity.

Even so, the process can evolve. As we find new techniques we can improve our processes.

### What is the number at the beginning of the AFP name?

AFPs are now prefixed with their associated tracking issue number. This gives
both the AFP a unique identifier and provides an easy breadcrumb for people to
find the issue where the current state of the AFP is being updated.

### My FAQ isn't answered here!

The AFP process is still evolving!
If something is missing or not answered here feel free to reach out to [Slack](https://join.slack.com/t/agones/shared_invite/zt-2mg1j7ddw-0QYA9IAvFFRKw51ZBK6mkQ).
If you want to propose a change to the AFP process you can open a PR on [AFP-NNNN](https://github.com/agones-dev/agones/issues/new?template=feature_request.md) with your proposal.
