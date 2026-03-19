# How to Contribute

We'd love to accept your patches and contributions to this project. There are
just a few small guidelines you need to follow.

## Contributor License Agreement

Contributions to this project must be accompanied by a Contributor License
Agreement. You (or your employer) retain the copyright to your contribution,
this simply gives us permission to use and redistribute your contributions as
part of the project. Head over to <https://cla.developers.google.com/> to see
your current agreements on file or to sign a new one.

You generally only need to submit a CLA once, so if you've already submitted one
(even if it was for a different project), you probably don't need to do it
again.

## Code of Conduct

Participation in this project comes under the [Contributor Covenant Code of Conduct](code-of-conduct.md)

## Finding Issues to Work On

If you're new to the project and want to help, but don't know where to start, look for issues labeled with:

- **[good first issue](https://github.com/agones-dev/agones/labels/good%20first%20issue)** - These are great for
  people who are new to the project. They are typically smaller in scope and easier to tackle.
- **[help wanted](https://github.com/agones-dev/agones/labels/help%20wanted)** - These issues are ready for
  contributions and often include good first issues, but may also involve more complex work for those who want to dig a
  little deeper.

These labels help you find issues that are well-suited for external contributions and where maintainers are actively
looking for help.

## Submitting code via Pull Requests

*Thank you* for considering submitting code to Agones!

- We follow the [GitHub Pull Request Model](https://help.github.com/articles/about-pull-requests/) for
  all contributions.
- For large bodies of work, we recommend creating an issue and labelling it
  "[kind/design](https://github.com/googleprivate/agones/issues?q=is%3Aissue+is%3Aopen+label%3Akind%2Fdesign)"
  outlining the feature that you wish to build, and describing how it will be implemented. This gives a chance
  for review to happen early, and ensures no wasted effort occurs.
- For new features, documentation *must* be included. Review the [Documentation Editing and Contribution](https://agones.dev/site/docs/contribute/)
  guide for details.
- It is strongly recommended that new API design follows the [Google AIPs](https://google.aip.dev/) design guidelines.  
- All submissions, including submissions by project members, will require review before being merged.
- Once review has occurred, please rebase your PR down to a single commit. This will ensure a nice clean Git history.
- If you are unable to access build errors from your PR, make sure that you have joined the [agones-discuss mailing list](https://groups.google.com/forum/#!forum/agones-discuss).
- Please follow the code formatting instructions below.

### Additional Instructions for Unreal Plugin Pull Requests

As there is no CI for the Unreal plugin, the following checklist should be run
manually before the PR is approved, using the latest released version of UE4.

1. Create default C++ template project in UE4.
1. Create a Plugins folder under the project directory (should be a sibling of the .uproject file).
1. Copy the [sdks/unreal/Agones](sdks/unreal/Agones) directory into the Plugins folder.
1. Build the UE4 project.
1. If the build succeeded, paste the build logs into the PR.

## Formatting

When submitting pull requests, make sure to do the following:

- Format all Go code with [gofmt](https://golang.org/cmd/gofmt/). Many people
  use [goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) which
  fixes import statements and formats code in the same style of `gofmt`.
- C++ code should follow the [Google C++ Style
  Guide](https://google.github.io/styleguide/cppguide.html), which can be
  applied automatically using the
  [ClangFormat](https://clang.llvm.org/docs/ClangFormat.html) command-line tool
  (e.g., `clang-format -style=Google foo.cc`). The exception to this is
  the [Unreal Engine plugin code](sdks/unreal/Agones), which should follow the
  [Unreal Engine 4 Coding Standard](https://docs.unrealengine.com/en-US/Programming/Development/CodingStandard/index.html).
- Remove trailing whitespace. Many editors will do this automatically.
- Ensure any new files have [a trailing newline](https://stackoverflow.com/questions/5813311/no-newline-at-end-of-file)

## Feature Stages

Often, new features will need to go through experimental stages so that we can gather feedback and adjust as necessary.

You can see this project's [feature stage documentation](https://agones.dev/site/docs/guides/feature-stages/) on the Agones
website.

If you are working on a new feature, you may need to take feature stages into account. This should be discussed on a
 design ticket prior to commencement of work. 

## Continuous Integration

Continuous integration is provided by [Google Cloud Container Builder](https://cloud.google.com/container-builder/),
through the [cloudbuilder.yaml](./cloudbuild.yaml) file found at the root of the directory.

Build success/failure with relevant details are pushed automatically to pull requests via [agones-bot](./build/agones-bot/README.md).

See the [Container Builder documentation](https://cloud.google.com/container-builder/docs/) for more details on
how to edit and expand the build process.

## Kubernetes Versions Update
### When to update supported Kubernetes Versions
As documented in the [version update policy](https://agones.dev/site/docs/installation/#agones-and-kubernetes-supported-versions), each version of Agones supports 3 releases of Kubernetes. The newest supported version is the latest available version in the GKE Rapid channel and at least one of the 3 supported version is supported by each of the major cloud providers (EKS and AKS). This means whenever a new minor version is available in the [GKE Rapid channel](https://cloud.google.com/kubernetes-engine/docs/release-notes-rapid), we should check whether we can roll forward the supported versions.
### How to update supported Kubernetes Versions
Please follow the steps below to update the Kubernetes versions supported.

1. Create a Issue from the [kubernetes update issue template](./.github/ISSUE_TEMPLATE/kubernetes_update.md) with the newly supported versions.
2. Complete all items in the issue checklist.
3. Close the issue.


## Community Meetings

Community meetings occur every month, and are open to all who wish to attend!

You can see them on our calendar 
([web](https://calendar.google.com/calendar/embed?src=google.com_828n8f18hfbtrs4vu4h1sks218%40group.calendar.google.com&ctz=America%2FLos_Angeles), 
[ical](https://calendar.google.com/calendar/ical/google.com_828n8f18hfbtrs4vu4h1sks218%40group.calendar.google.com/public/basic.ics)) and/or join the 
[mailing list or Slack](https://agones.dev/site/community/)
for notifications.

## Becoming a Collaborator on Agones

If you have submitted at least one Pull Request and had it merged, you may wish to become an official collaborator.
This will give you the ability to have tickets assigned to you (or you can assign tickets to yourself!).

We have a [community membership guide](./docs/governance/community_membership.md), that outlines the process.

## Becoming an Approver on Agones

If you are interested in becoming an Approver on the Agones project and getting commit access to the
repository, we have a [community membership guide](./docs/governance/community_membership.md), that outlines the process.

### Additional Resources

#### Extending Kubernetes

- [Kubernetes Custom Resources](https://kubernetes.io/docs/concepts/api-extension/custom-resources/) -
  This is how we define our own resource names (`GameServer`, etc) within Kubernetes.
- [Kubernetes Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) -
  Kubernetes documentation on writing controllers.
- [Extend the Kubernetes API with CustomResourceDefinitions](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/) -
  This page shows how to install a custom resource into the Kubernetes API by creating a CustomResourceDefinition.
- [Joe Beda's TGIK Controller](https://github.com/jbeda/tgik-controller) -
  [Joe Beda](https://twitter.com/jbeda) did a video series on writing controllers for Kubernetes.
  **This is the best resource for learning about controllers and Kubernetes.**
- [Kubernetes Sample Controller](https://github.com/kubernetes/sample-controller) -
  Example of a Custom Resources with a Kubernetes Controller.
- [Kubernetes Code Generator](https://github.com/kubernetes/code-generator) -
  The tooling that generated the Go libraries for the Custom Resource we define
- [Kubernetes Controller Best Practices](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/controllers.md) -
  Set of best practices written for writing Controllers inside Kubernetes. Also a great list for everywhere else too.
- [Writing Kube Controllers for Everyone - Maciej Szulik, Red Hat](https://www.youtube.com/watch?v=AUNPLQVxvmw) -
  A great intro video into coding for Controllers, and explaining Informers and Listers.
- [@markmandel](https://github.com/markmandel) regularly streams his development of Agones on [Twitch](https://www.twitch.tv/markmandel).
  You can find the full archive on [YouTube](https://www.youtube.com/playlist?list=PLqqp1QEhKwa5aNivDIE4SS21ehE9Zt0VZ)


#### Coding and Development

- [How to write a good Git Commit message](https://chris.beams.io/posts/git-commit/) -
  Great way to make sure your Pull Requests get accepted.
- **Log levels usage:**
  - Fatal - a critical error has happened and the application can not perform subsequent work anymore. Examples: missing configuration information in case there are no default values provided, one of the services can not start normally, etc.
  - Error - a serious issue has happened, users are affected without having a way to work around one, but an application may continue to work. This error usually requires someone’s attention. Examples: a file cannot be opened, cannot respond to HTTP request properly, etc.
  - Warn - something bad has happened, but the application still has the chance to heal itself or the issue can wait for some time to be fixed. Example: a system has failed to connect to an external resource but will try again automatically.
  - Info - should be used to document state changes in the application or some entity within the application. These logs provide the skeleton of what has happened. Examples: system started/stopped, remote API calls, a new user has been created/updated, etc.
  - Debug - diagnostic information goes here and everything that can help to troubleshoot an application. Examples: any values in business logic, detailed information about the data flow.

  More details can be found in [this article](https://medium.com/@tom.hombergs/tip-use-logging-levels-consistently-913b7b8e9782).
