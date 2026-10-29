# Agones Development Documentation

Tooling for building and developing against Agones, with only dependencies being
[Make](https://www.gnu.org/software/make/) and [Docker](https://www.docker.com)

Rather than installing all the dependencies locally, you can test and build Agones using the Docker image that is
built from the Dockerfile in this directory. There is an accompanying Makefile for all the common
tasks you may wish to accomplish.

## Quick Start

1. **Platform Setup**: Set up your development environment → [Platform Setup Guide](docs/platform-setup.md)
2. **Build & Test**: Learn basic build workflows → [Building and Testing Guide](docs/building-testing.md)
3. **Cluster Setup**: Set up a development cluster → [Cluster Setup Guide](docs/cluster-setup.md)

## Documentation

### Guides
- **[Platform Setup](docs/platform-setup.md)** - Set up development environment on Linux, Windows, macOS
- **[Building and Testing](docs/building-testing.md)** - Core build workflows and testing procedures
- **[Cluster Setup](docs/cluster-setup.md)** - Set up GKE, Minikube, Kind, or custom clusters
- **[Development Workflow](docs/development-workflow.md)** - Advanced development patterns and remote debugging
- **[Dependencies](docs/dependencies.md)** - Go modules and vendoring
- **[Performance Testing](docs/performance-testing.md)** - Manual runbook for the performance tests, which require provisioning a large, costly dedicated cluster
- **[Troubleshooting](docs/troubleshooting.md)** - Common issues and solutions

### Reference
- **[Make Reference](docs/make-reference.md)** - Complete documentation of all make variables and targets

## Common Quick Commands

```bash
# Run tests and build images
make lint test-go build-images

# Set up Minikube cluster (with 2 nodes) and install Agones
make minikube-test-cluster MINIKUBE_NODES=2
make build-images minikube-push minikube-install

# Debug with Minikube
make build-debug-images minikube-push minikube-install-debug
make minikube-debug-portforward

# Run end-to-end tests
# This takes a _while_, so run at your peril!
make minikube-test-e2e
# You can specify ARGS to run specific tests
make minikube-test-e2e ARGS='-run TestAllocatorWithSelectors'
```

For detailed command documentation, see the [Make Reference](docs/make-reference.md).

## Need Help?

- **Issues**: Check the [Troubleshooting Guide](docs/troubleshooting.md)
- **Build Problems**: See [Building and Testing Guide](docs/building-testing.md)
- **Setup Issues**: Check [Platform Setup Guide](docs/platform-setup.md) or [Cluster Setup Guide](docs/cluster-setup.md)
- **Remote Debugging**: See [Development Workflow Guide](docs/development-workflow.md)
- **#development Slack**: [Join the development channel on Slack](https://join.slack.com/t/agones/shared_invite/zt-2mg1j7ddw-0QYA9IAvFFRKw51ZBK6mkQ) and talk to fellow Agones developers.

