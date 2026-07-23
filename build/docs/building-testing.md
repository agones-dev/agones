# Building and Testing Guide

This guide covers the core workflows for building and testing Agones.

## Getting Started

Make sure you are in the `build` directory to start.

First, let's test the Agones system code. To do this, run `make test-go`, which will execute all the unit tests for
the Go codebase (there are other tests, but they can take a long time to run).

If you haven't run any of Make targets before then this will also create the Docker based build
image, and then run the tests.

Building the build image may take a few minutes to download all the dependencies, so feel
free to make cup of tea or coffee at this point. ☕️

**Note**: If you get build errors and you followed all the instructions so far, consult the [Troubleshooting Guide](troubleshooting.md)

The build image is only created the first time one of the make targets is executed, and will only rebuild if the build
Dockerfile has changed.

## Building Agones

Assuming that the tests all pass, let's go ahead and compile the code and build the Docker images that Agones
consists of.

To compile the Agones images, run `make build-images`. This is often all you need for Agones development.

If you want to compile and build everything (this can take a while), run `make build`, this will:

- Compile the Agones Kubernetes integration code
- Create the Docker images that we will later push
- Build the local development tooling for all supported OS's
- Compile and archive the SDKs in various languages

You may note that docker images, and tar archives are tagged with a concatenation of the
upcoming release number and short git hash for the current commit. This has also been set in
the code itself, so that it can be seen in via log statements.

Congratulations! You have now successfully tested and built Agones!

## Common Development Flows

You can combine some of the above steps into a single one, for example `make build-images push install` is very
common flow, to build you changes on Agones, push them to a container registry and install this development
version to your cluster.

Another would be to run `make lint test-go` to run the golang linter against the Go code, and then run all the unit
tests.

## Testing changes to the `simple-game-server` example image

The [`simple-game-server`](../../examples/simple-game-server) example is used throughout Agones' own examples,
fleets and end-to-end tests. If you make changes to it and want to
test them on Minikube without publishing an image, you can build it and push it straight into your Minikube
cluster, tagged as if it were the production image, so that the existing yaml files (`fleet.yaml`,
`gameserver.yaml`, etc.) pick up your changes without any edits.

From the [`examples/simple-game-server`](../../examples/simple-game-server) directory (this has its own `Makefile`,
separate from the one in `build/`), run:

```bash
make build-linux-image-$(dpkg --print-architecture) && make minikube-push
```

This builds the image for your host machine's architecture, loads it into the `agones` Minikube profile (the same
one created by `make minikube-test-cluster`), and re-tags it to match the production image reference
(e.g. `us-docker.pkg.dev/agones-images/examples/simple-game-server:<version>`).

This can also be useful if the `simple-game-server` was recently edited, and has yet to be published to the production
registry.

## Set Local Make Targets and Variables with `local-includes`

If you want to permanently set `Makefile` variables or add targets to your local setup, all `local-includes/*.mk`
are included into the main `Makefile`, and also ignored by the git repository.

Therefore, you can add your own `.mk` files in the [`local-includes`](./local-includes) directory without affecting
the shared git repository.

For examaple, if I only worked with Linux images for Agones, I could permanently turn off Windows and Arm64 images,
by writing a `images.mk` within that folder, with the following contents:

```makefile
# Just Linux
WITH_WINDOWS=0
WITH_ARM64=0
```

## Running Individual End-to-End Tests

When you are working on an individual feature of Agones, it doesn't make sense to run the entire end-to-end suite of
tests (and it can take a really long time!).

The end-to-end tests within the [`tests/e2e`](../test/e2e) folder are plain
[Go Tests](https://go.dev/doc/tutorial/add-a-test) that will use your local `~/.kube` configuration to connect and
run against the currently active local cluster.

This means you can run individual e2e tests from within your IDE or as a `go` command.

For example:

```shell
go test -race -run ^TestCreateConnect$
```

### Troubleshooting E2E Tests
If you run into cluster connection issues, this can usually be resolved by running any kind of authenticated `kubectl`
command from within `make shell` or locally, to refresh your authentication token.

## Next Steps

- See [Cluster Setup Guide](cluster-setup.md) to set up a development cluster
- See [Make Reference](make-reference.md) for detailed documentation of all make targets
- See [Development Workflow Guide](development-workflow.md) for advanced development patterns