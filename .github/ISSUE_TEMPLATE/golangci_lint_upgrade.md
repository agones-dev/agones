---
name: golangci-lint Upgrade
about: Steps to upgrade golangci-lint to a new release
title: "Upgrade golangci-lint to {{ env.LATEST_VERSION }}"
labels: kind/cleanup
assignees: ""
---

A new golangci-lint release is out. Agones pins golangci-lint in `build/build-image/Dockerfile`.

- Pinned: `{{ env.PINNED_VERSION }}`
- Latest: `{{ env.LATEST_VERSION }}`
- Release notes: https://github.com/golangci/golangci-lint/releases

Steps to upgrade:

- [ ] Read the release notes for every version between the pinned and latest release. Look for:
  - [ ] New linters, or new analyzers/checks in the linters we already enable (for example gosec,
        staticcheck, testifylint, modernize), that are worth turning on in `.golangci.yml`.
  - [ ] Changes to defaults or deprecated settings that affect `.golangci.yml`.
- [ ] Update the `golangci-lint` version in `build/build-image/Dockerfile`. If the major version
      changed, the `go install` module path (`.../golangci-lint/vN/...`) changes with it.
- [ ] Run `make -C build lint` and fix any new findings, or explain each new exclusion in
      `.golangci.yml`.
- [ ] Open the upgrade PR with `/kind cleanup`, and open separate issues for any new linters that
      are worth enabling but too big to do in the upgrade PR.
