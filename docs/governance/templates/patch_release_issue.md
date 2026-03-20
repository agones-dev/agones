# Patch Release {patch_version}

<!--
This is the patch release issue template. Make a copy of the markdown in this page
and copy it into a release issue. Fill in relevant values (patch, previous version, and cherry-pick), found inside {}

*** VERSION SHOULD BE IN THE FORMAT OF 1.x.x NOT v1.x.x ***

Note: "If needed" means if this step only needs to be done if it was modified as part of the patch release or otherwise should be updated.
!-->

## Prerequisites

- [ ] Editor-level access to the agones-images project.
- [ ] Permission to publish new versions of the App Engine application.
- [ ] Write access to the Agones GitHub repository.
- [ ] git remote -v should show:
  - [ ] An origin remote pointing to your personal Agones fork (e.g., git@github.com:yourname/agones.git).
  - [ ] An upstream remote pointing to git@github.com:googleforgames/agones.git.
- [ ] A gcloud config configurations configuration named agones-images pointing to the same project.
- [ ] Edit access to the[ Agones Release Calendar](https://calendar.google.com/calendar/u/0?cid=Z29vZ2xlLmNvbV84MjhuOGYxOGhmYnRyczR2dTRoMXNrczIxOEBncm91cC5jYWxlbmRhci5nb29nbGUuY29t).
- [ ] Approval for any modifications to the [Release Title and Description](https://docs.google.com/document/d/1bRZCxYB8lrVcrru41b6s5D_9uU0zS49vVGdBhg0yDIY/edit)

## Steps

- [ ] Run `make shell` and run `gcloud config configurations activate agones-images`.
- [ ] To create a new branch for the patch release, and base it off of the previous patch release run
      `make create-patch-release-branch PREVIOUS_VERSION={previous_version} PATCH_VERSION={patch_version}`.
  - [ ] Run `git status` to confirm you are on the expected branch name.
  - [ ] Run `git log` to confirm the most recent commits are what you expect. (They should be the
        same as the release tagged v{previous_version}.)
- Run `make pre-build-release PATCH_VERSION={patch_version}` to ensure all example images exist on
  the agones-images/examples repository and to deploy the {patch_version} service on GCP/App
  Engine/Services.
- [ ] Run `git cherry-pick {<SHA>}` to pick up the PR changes in the patch release.
- [ ] From within the `agones/build` directory run `make sdk-update-version release_stage=patch version={previous_version}`
      to increment the previous version by 1 to {patch_version} in the build/Makefile as well as the
      SDK files.
- [ ] Update `_BASE_VERSION` in `cloudbuild.yaml` to `{patch_version}`.
- [ ] Create a _draft_ release with the [release template](https://github.com/agones-dev/agones/blob/main/docs/governance/templates/release.md).
  - [ ] Run `make release-example-image-markdown` to populate example images and append the output
        in `Images available with this release` section.
  - [ ] Draft a new release with [release draft](https://github.com/agones-dev/agones/releases).
        Update the `Tag version` and `Release title` with the release version and click on `Generate release notes`
        to generate the release notes with `Full Changelog` info for {patch_version}. Make sure to
        add the description. Include the `Images available with this release` section from the
        previous step that will be present after the `Full Changelog` and save the draft.
  - [ ] Copy the {patch_version} release details from the `Full Changelog` and paste it on top of
        the CHANGELOG.md file
- [ ] Site updated
  - [ ] Create a new file named {patch_version}.md in `/site/content/en/blog/releases`. Copy the
        draft release content in this file (this will be what you send via email)[refer the previous release file].
  - [ ] Run `make site-server` frequently to make sure everything looks fine for the release in your localhost
  - [ ] If needed: In `site/content/en/docs/Installation/_index.md #agones-and-kubernetes-supported-versions`,
        for the current version, replace {{% k8s-version %}} with hardcoded Kubernetes versions
        supported by the current version. And add a row for the Agones release version with
        {{% k8s-version %}} as its supported Kubernetes versions.
  - [ ] Run `make del-data-proofer-ignore FILENAME={patch_version}-1.md` to remove the `data-proofer-ignore` attribute from the previous release blog. Review all occurrences of the link_test and data-proofer-ignore attributes globally. Exclude html and release files.
  - [ ] If needed: Add a link to previous version's documentation to nav dropdown in `site/layouts/partials/navbar.html` on top and Run `make update-navbar-version FILENAME=site/layouts/partials/navbar.html` to remove the older version from the dropdown list.
  - [ ] config.toml updates:
    - [ ] Run `make site-config-update-version release_stage=patch` to update the release version and sync data between dev and prod.
    - [ ] If needed: Update documentation with updated example images tags.
- [ ] If needed: Ensure that the alphaGates and betaGates for "Dev" in `test/upgrade/versionMap.yaml`
      match the Alpha features and Beta features in the patch release branch's `pkg/util/runtime/features.go`.
- [ ] Create PR with these changes, and merge them with an approval.
- [ ] Run `git remote update && git checkout release-{patch_version} && git reset --hard upstream/release-{patch_version}` to ensure your code is in line with the upstream patch release branch.
- [ ] Publish SDK packages
  - [ ] Run `make sdk-shell-node` to get interactive shell to publish node package. Requires Google internal process to publish.
  - [ ] Run `make sdk-publish-csharp` to deploy to NuGet. Requires login credentials. Will need [NuGet API Key](https://www.nuget.org/account/apikeys) from Agones account.
  - [ ] Run `make sdk-publish-rust`. This command executes cargo login for authentication, performs a dry-run publish, and if that succeeds, does the actual publish. Will need [crate's API TOKEN](https://crates.io/settings/tokens) from your crate's account.
- [ ] Run `make post-build-release VERSION={patch_version}` to build the artifacts in
      GCS (These files will be attached in the release notes) and to push the latest images in the
      release repository and push chart on agones-chart.
- [ ] Run `make tag-deprecated-images VERSION={patch_version}` to tag images from the previous
      version with a `deprecated-public-image-<version>` label, indicating they are no longer
      actively maintained.
- [ ] Run `make shell` and run `gcloud config configurations activate <your development project>` to switch Agones development tooling off of the `agones-images` project within the shell. Run `exit` to exit the shell.
- [ ] Run `gcloud config configurations activate <your development project>` to make sure Agones development tooling is off of the agones-images project in your terminal.
- [ ] Smoke Test: run `make install-release` to view helm releases, uninstall agones-system namespace, fetch the latest version of Agones, verify the new version, installing agones-system namespace, and list all the pods of agones-system.
- [ ] Attach all assets found in cloud storage with {patch_version} to the draft GitHub Release.
- [ ] Copy any review changes from the release blog post into the draft GitHub release.
- [ ] Publish the draft GitHub Release.
- [ ] Run `git checkout main && git pull upstream main && git checkout -b post-release-{patch_version}`.
- [ ] In test/sdk/go/Makefile, change release_version to `{patch_version}`.
  - [ ] Run `make shell` and execute `gcloud config configurations activate agones-images`.
  - [ ] Within the shell, cd to the test/sdk/go/ directory and run `make cloud-build`.
- [ ] Verify and update Kubernetes version support and Agones version mappings in `test/upgrade/versionMap.yaml`.
  - [ ] Update ReleaseVersion to the current release `{patch_version}`.
- [ ] Create a PR with these changes and merge into the patch release branch with approval.
- [ ] Email mailing lists with the release details (copy-paste the release blog post). Refer to the [Internal Mailing list posting guide](https://docs.google.com/document/d/1qYR9ccVURgujqFAIpjpSN2GRcCeQ29ow5H_V4sm4RGs/edit#heading=h.zge9gjrt8ws8) for details.
- [ ] Paste the announcement blog post to the #users Slack group.
- [ ] Post to the [agonesdev](https://twitter.com/agonesdev) Twitter account.
- [ ] Close this issue. _Congratulations!_ - the patch release is now complete! :tada: :clap: :smile: :+1:

[release-template]: https://github.com/agones-dev/agones/blob/main/docs/governance/templates/release.md
[release-draft]: https://github.com/agones-dev/agones/releases
[build-makefile]: https://github.com/agones-dev/agones/blob/main/build/Makefile
[Internal Mailing list posting guide]: https://docs.google.com/document/d/1qYR9ccVURgujqFAIpjpSN2GRcCeQ29ow5H_V4sm4RGs/edit#heading=h.zge9gjrt8ws8
