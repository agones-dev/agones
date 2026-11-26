---
name: Golang Version and Dependency Upgrades
about: Steps to upgrade Golang version and dependencies
title: "Golang MAJOR.MINOR.PATCH Version and Dependency Upgrades"
labels: kind/cleanup
assignees: ""
---

Steps to determine the Golang version to upgrade to:

- [ ] Go to the Kubernetes repository and find the release branch for the most recent version of
      Kuberntes supported by Agones. Find the `go.mod` file in that branch, and look for the go
      version. This is the minor version that Agones should use. For instance, if the most recent
      Kubernetes version supported by Agones is 1.32, then the minor version of Golang on Agones
      should be 1.23 as is determined by the `go.mod` file in the [release branch of Kubernetes
      1.32](https://github.com/kubernetes/kubernetes/blob/release-1.32/go.mod#L9).
- [ ] Check the [Golang release history](https://go.dev/doc/devel/release) for the most recent patch
      version of the minor version as determined in the previous step.
- [ ] Update this issue title to reflect the Golang `MAJOR.MINOR.PATCH` version determined the in
      the previous step.

Steps to upgrade Golang version and dependencies:

- [ ] Update `go.mod` and `go.sum`. At the root (agones) directory, run:

  - [ ] `find . -name 'go.mod' -not -path '*/\.*' -execdir go mod edit -go=<NEW_GOLANG_VERSION_WITHOUT_PATCH> \;`
  - [ ] `find . -name 'go.mod' -not -path '*/\.*' -execdir go mod tidy \;`

- [ ] Update the Dockerfiles for `build` directory.

  - [ ] At the root directory, run: `find build -type f \( -not -path '*/\.*' -and -not -path 'build/tmp/*' \) -exec sed -i 's/GO_VERSION=[0-9]\+\.[0-9]\+\.[0-9]\+/GO_VERSION=<NEW_GOLANG_VERSION>/g' {} \;`
  - [ ] Update the `golang` version for file `build/agones-bot/Dockerfile` to `golang:<NEW_GOLANG_VERSION_WITHOUT_PATCH>`

- [ ] Update the Dockerfiles for `test` directory.

  - [ ] At the root directory, run: `find test -type f -exec sed -i 's/golang:[0-9]\+\.[0-9]\+\.[0-9]\+/golang:<NEW_GOLANG_VERSION>/g' {} \;`
  - [ ] At the root directory, run: `find test -type f -exec sed -i 's/go [0-9]\+\.[0-9]\+\.[0-9]\+/go <NEW_GOLANG_VERSION>/g' {} \;`
  - [ ] At the root directory, run: `find test -type f -name 'go.mod' -execdir go mod tidy \;`

- [ ] Update the Dockerfiles for `examples` directory. At the root directory, run:

  - [ ] `find examples -name Dockerfile -exec sed -i 's/golang:[0-9]\+\.[0-9]\+-alpine/golang:<NEW_GOLANG_VERSION_WITHOUT_PATCH>-alpine/g' {} \;`
  - [ ] `find examples \( -name Dockerfile -o -name Dockerfile.windows \) -exec sed -i 's/golang:[0-9]\+\.[0-9]\+\.[0-9]\+/golang:<NEW_GOLANG_VERSION>/g' {} \;`

- [ ] Update the example images tag. At `build` directory, run:

  - [ ] `make bump-image IMAGENAME=autoscaler-webhook VERSION=<current-image-version>`
  - [ ] `make bump-image IMAGENAME=crd-client VERSION=<current-image-version>`
  - [ ] `make bump-image IMAGENAME=custom-controller VERSION=<current-image-version>`
  - [ ] `make bump-image IMAGENAME=simple-game-server VERSION=<current-image-version>`
  - [ ] `make bump-image IMAGENAME=supertuxkart-example VERSION=<current-image-version>`
  - [ ] `make bump-image IMAGENAME=xonotic-example VERSION=<current-image-version>`

- [ ] Update Golang dependencies in all `go.mod` files:

  - [ ] At the root directory, run: `find . -name 'go.mod' -not -path '*/\.*' -execdir go get -u \;`
  - [ ] At the root directory, run: `find . -name 'go.mod' -not -path '*/\.*' -execdir go mod tidy \;`

-  [ ] Run `go mod vendor` to ensure all modules are properly updated.
-  [ ] In the `build` directory, run `make lint` to verify code style and linting rules.
-  [ ] In the `build` directory, run `make test` to ensure all tests pass.

- [ ] Run the following to generate and push the new example images:

  - [ ] In `examples/autoscaler-webhook`, run: `make cloud-build`
  - [ ] In `examples/crd-client`, run: `make cloud-build`
  - [ ] In `examples/custom-controller`, run: `make cloud-build`
  - [ ] In `examples/simple-game-server`, run: `make cloud-build`
  - [ ] In `examples/supertuxkart`, run: `make cloud-build`
  - [ ] In `examples/xonotic`, run: `make cloud-build`

- [ ] Create a PR for the above changes and send for review

- [ ] Merge the above PR after it is approved
