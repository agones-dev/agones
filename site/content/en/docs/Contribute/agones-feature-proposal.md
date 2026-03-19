---
title: "Agones Feature Proposal (AFP)"
linkTitle: "Agones Feature Proposal"
weight: 1000
description: >
  Suggest a new feature or enhancement for the Agones project.
---

## Overview of the Agones Feature Proposal (AFP) Process

The **Agones Feature Proposal (AFP)** process provides a standardized way to propose, discuss, and track new features, enhancements, or architectural changes within the Agones project. This process is intended to ensure clarity, transparency, and collaborative decision-making, allowing stakeholders to effectively contribute to and communicate about important changes in the project.

### Key Objectives of the AFP Process

- **Standardized Proposals**: Offers a common structure for proposing changes, ensuring that the motivation for a feature, its implementation, and its impact are clearly defined.
- **Cross-Team Communication**: Facilitates collaboration across different teams within the Agones community, ensuring that all relevant stakeholders are informed and involved in the decision-making process.
- **Roadmap Integration**: Supports the creation of a user-facing development roadmap that outlines upcoming features and changes.
- **Feature Tracking**: Allows features and major changes to be tracked across one or more releases, providing a historical record of decisions and progress.

### AFP Structure

Each AFP consists of several key sections that are defined incrementally as the feature progresses. These sections typically include:

- **Motivation**: The reasoning behind the proposed change, including its expected impact on users and developers.
- **Reference-Level Explanation**: Detailed technical information about how the feature will be implemented and integrated.
- **Stewardship**: Clarification of the roles and responsibilities of key participants in the AFP process.
- **Drawbacks**: Acknowledgment of potential issues or downsides of the proposal.
- **Alternatives**: Discussion of alternative approaches to achieving the desired outcome.
- **Unresolved Questions**: A place to document any open questions or issues that need further clarification.

### AFP Workflow and States

The AFP process follows a series of stages to ensure that the proposal is well-defined, reviewed, and implemented in a structured manner. These stages include:

1. **Provisional**: The AFP is actively being discussed and defined.
2. **Implementable**: The AFP has been reviewed and approved for implementation.
3. **Implemented**: The feature has been implemented and is no longer actively changing.
4. **Deferred**: The AFP is proposed but not actively worked on.
5. **Rejected**: The AFP is no longer considered viable.
6. **Withdrawn**: The authors have decided to withdraw the AFP.
7. **Replaced**: The AFP has been replaced by a new AFP.

### How to Submit an AFP

To submit a new AFP, follow the instructions provided in the official AFP template. This template outlines the required sections and metadata, such as the title, authors, reviewers, and approval statuses. AFPs are stored in the project's **[docs/proposals](https://github.com/agones-dev/agones/blob/main/docs/proposals)** directory, with each folder named following the format `00x-feature-name`, where "00x" represents an incremental AFP number. For example, you can refer to the [NNNN-afp-template](https://github.com/agones-dev/agones/blob/main/docs/proposals/NNNN-afp-template) folder for a detailed proposal template. The folder name is also used in the corresponding pull request (PR) title, which follows the format "AFP-00x" to match the folder naming convention. The document undergoes iterative updates, with changes tracked via version control.

### Why AFPs Are Important

The AFP process helps reduce **tribal knowledge** within the community by creating a centralized, transparent way to communicate significant changes. It encourages comprehensive discussions and allows for the smooth transition of features from initial idea to production.

It is particularly beneficial for tracking:

- **Large Features**: Features that impact a wide range of users or developers.
- **Major Refactorings**: Significant changes to the architecture that require widespread consensus.
- **Cross-Team Initiatives**: Efforts that involve multiple stakeholders and require coordination across teams.

### How AFPs Benefit the Agones Project

- **Improved Communication**: Provides a clear, centralized mechanism for proposing and tracking changes.
- **Better Decision-Making**: Ensures that all stakeholders, including maintainers, contributors, and users, are informed and involved in the decision-making process.
- **Long-Term Roadmap Clarity**: Helps create a roadmap for future releases, aligning community expectations and improving planning for feature adoption.

### Learn More About AFPs

To get started with contributing to the AFP process, check out the following resources:

- **[AFP Template](https://github.com/agones-dev/agones/blob/main/docs/proposals/NNNN-afp-template/README.md)**: A detailed template for creating an AFP.
- **[AFP Metadata](https://github.com/agones-dev/agones/blob/main/docs/proposals/README.md#afp-metadata)**: Information on the metadata required for each AFP.
- **[How to Propose a Feature](https://github.com/agones-dev/agones/issues/new?template=feature_request.md)**: Step-by-step instructions on how to submit a new AFP.

<!-- ### Examples of AFPs

Here are some example AFPs that have been proposed or implemented in Agones:

- **[123-AFP: Example Feature](#)**: Description of a feature proposal.
- **[124-AFP: Example Architecture Change](#)**: Description of a major architectural change proposal. -->

### How to Get Involved

We welcome contributions from all members of the Agones community! If you're interested in proposing a new feature or contributing to the ongoing development of Agones, you can start by submitting an AFP or reviewing and providing feedback on existing proposals.

---

For further information or assistance, please reach out to the Agones maintainers via our [Agones Community Hub](https://agones.dev/site/community/).

