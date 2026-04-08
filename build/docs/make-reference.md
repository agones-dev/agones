# Make Reference Guide

This guide provides comprehensive documentation for all make variables and targets in the Agones build system.

## Make Variable Reference

### VERSION
The version of this build. Version defaults to the short hash of the latest commit.

### REGISTRY
The registry that is being used to store docker images. It doesn't have default value and has to be set explicitly.

### CHARTS_REGISTRY
The chart registry that is being used to store helm charts. It doesn't have default value and has to be set explicitly.
If not set it would use the `GCP_BUCKET_CHARTS`.

### KUBECONFIG
The Kubernetes config file used to access the cluster. Defaults to `~/.kube/config` - the file used by default by kubectl.

### CLUSTER_NAME
The (gcloud) test cluster that is being worked against. Defaults to `test-cluster`.

### GCP_PROJECT
Your GCP project for deploying GKE cluster. Defaults to gcloud default project settings.

### GKE_PASSWORD
If specified basic authentication would be enabled for your cluster with username "admin".
Empty string `""` would disable basic authentication.

### IMAGE_PULL_SECRET
The name of the secret required to pull the Agones images, if needed.
If unset, no pull secret will be used.

### IMAGE_PULL_SECRET_FILE
The full path of the file containing the secret for pulling the Agones images, in case it's needed.

If set, `make install` will install this secret in both the `agones-system` (for pulling the controller image)
and `default` (for pulling the sdk image) repositories.

### WITH_WINDOWS
Build Windows container images for Agones.

This option is enabled by default via implicit `make WITH_WINDOWS=1 build-images`.
To disable, use `make WITH_WINDOWS=0 build-images`.

### WINDOWS_VERSIONS
List of Windows Server versions to build for. Defaults to `ltsc2019` for Windows Server 2019.
See https://hub.docker.com/_/microsoft-windows-servercore for all available Windows versions.

### WITH_ARM64
Build ARM64 container images for Agones

This option is enabled by default via implicit `make WITH_ARM64=1 build-images`.
To disable, use `make WITH_ARM64=0 build-images`.

### MINIKUBE_DRIVER

Which [driver](https://minikube.sigs.k8s.io/docs/drivers/) to use with a Minikube test cluster.

Defaults to "docker".

## Make Target Reference

All targets will create the build image if it is not present.

### Development Targets

Targets for developing with the build image.

#### `make build`
Build all the images required for Agones, as well as the SDKs

#### `make build-images`
Build all the images required for Agones

#### `make build-debug-images`
Build debug-enabled Docker images for all Agones services with Delve debugger support.
These images include the Go debugger and expose debug ports for remote debugging.

#### `make build-sdks`
Build all the sdks required for Agones

#### `make build-sdk`
Next command `make build-sdk SDK_FOLDER=[SDK_TYPE]` will build SDK of `SDK_TYPE`.
For instance, in order to build the cpp sdk static and dynamic libraries (linux libraries only) use `SDK_FOLDER=cpp`

#### `make run-sdk-command`
Next command `make run-sdk-command COMMAND=[COMMAND] SDK_FOLDER=[SDK_TYPE]` will execute command for `SDK_TYPE`.
For instance, in order to generate swagger codes when you change swagger.json definition, use `make run-sdk-command COMMAND=gen SDK_FOLDER=restapi`

#### `make run-sdk-conformance-local`
Run Agones sidecar which would wait for all requests from the SDK client.
Note that annotation should contain UID and label should contain CreationTimestamp values to pass the test.

#### `make run-sdk-conformance-no-build`
Only run a conformance test for a specific Agones SDK.

#### `make run-sdk-conformance-test`
Build, run and clean conformance test for a specific Agones SDK.

#### `make run-sdk-conformance-tests`
Run SDK conformance test.
Run SDK server (sidecar) in test mode (which would record all GRPC requests) versus all SDK test clients which should generate those requests. All methods are verified.

#### `make clean-sdk-conformance-tests`
Clean leftover binary and package files after running SDK conformance tests.

#### `make test`
Run the go tests, the sdk tests, the website tests and yaml tests.

#### `make test-go`
Run only the golang tests

#### `make lint`
Lint the Go code

#### `make build-examples`
Run `make build` for all `examples` subdirectories

#### `make site-server`
Generate `https://agones.dev` website locally and host on `http://localhost:1313`

#### `make hugo-test`
Check the links in a website

#### `make site-test`
Check the links in a website, includes `test-gen-api-docs` target

#### `make site-images`
Create all the site images from dot and puml diagrams in /site/static/diagrams

#### `make gen-api-docs`
Generate Agones CRD reference documentation [Agones CRD API reference](../site/content/en/docs/Reference/agones_crd_api_reference.html). Set `feature` shortcode with proper version automatically

#### `make test-gen-api-docs`
Verifies that there is no changes in generated [Agones CRD API reference](../site/content/en/docs/Reference/agones_crd_api_reference.html) compared to the current one (useful for CI)

#### `make push`
Pushes all built images up to the `$(REGISTRY)`

#### `make install`
Installs the current development version of Agones into the Kubernetes cluster

#### `make uninstall`
Removes Agones from the Kubernetes cluster

#### `make update-allocation-certs`
Updates the Agones installation with the IP of the Allocation LoadBalancer, thereby creating a valid certificate
for the Allocation gRPC endpoints.

The certificates are downloaded from the test kubernetes cluster and stored in ./build/allocation

#### `make test-e2e`
Runs end-to-end tests on the previously installed version of Agones.
These tests validate Agones flow from start to finish.

It uses the KUBECONFIG to target a Kubernetes cluster.

Use `GAMESERVERS_NAMESPACE` flag to provide a namespace or leave it empty in order to create and use a random one.

See [`make minikube-test-e2e`](#make-minikube-test-e2e) to run end-to-end tests on Minikube.

#### `make test-e2e-integration`
Runs integration portion of the end-to-end tests.

Pass flags to [go test](https://golang.org/cmd/go/#hdr-Testing_flags) command
using the `ARGS` parameter. For example, to run only the `TestGameServerReserve` test:

```bash
make test-e2e-integration ARGS='-run TestGameServerReserve'
```

#### `make test-e2e-failure`
Run controller failure portion of the end-to-end tests.

#### `make test-e2e-allocator-crash`
Run allocator failure portion of the end-to-end test.

#### `make setup-prometheus`

Install Prometheus server using [Prometheus Community](https://prometheus-community.github.io/helm-charts)
chart into the current cluster.

By default all exporters and alertmanager is disabled.

You can use this to collect Agones [Metrics](../site/content/en/docs/Guides/metrics.md).

See [`make minikube-setup-prometheus`](#make-minikube-setup-prometheus) and [`make kind-setup-prometheus`](#make-kind-setup-prometheus) to run the installation on Minikube or Kind.

#### make helm-repo-update

Run helm repo update to get the mose recent charts.

#### `make setup-grafana`

Install Grafana server using [grafana community](https://grafana-community.github.io/helm-charts) chart into
the current cluster and setup [Agones dashboards with Prometheus datasource](./grafana/).

You can set your own password using the `PASSWORD` environment variable.

See [`make minikube-setup-grafana`](#make-minikube-setup-grafana) and [`make kind-setup-grafana`](#make-kind-setup-grafana) to run the installation on Minikube or Kind.

#### `make setup-prometheus-stack`

Install Prometheus-stack using [Prometheus Community](https://prometheus-community.github.io/helm-charts) chart into the current cluster.

By default only prometheus and grafana will installed, all exporters and alertmanager is disabled.

You can use this to collect Agones [Metrics](../site/content/en/docs/Guides/metrics.md) using ServiceMonitor.

See [`make minikube-setup-prometheus-stack`](#make-minikube-setup-prometheus-stack) and [`make kind-setup-prometheus-stack`](#make-kind-setup-prometheus-stack) to run the installation on Minikube or Kind.

#### `make prometheus-portforward`

Sets up port forwarding to the
Prometheus deployment (port 9090 Prometheus UI).

On Windows and MacOS you will need to have installed [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

See [`make minikube-prometheus-portforward`](#make-minikube-prometheus-portforward) and [`make kind-prometheus-portforward`](#make-minikube-prometheus-portforward) to run  on Minikube or Kind.

#### `make grafana-portforward`

Sets up port forwarding to the
grafana deployment (port 3000 UI).

On Windows and MacOS you will need to have installed [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

See [`make minikube-grafana-portforward`](#make-minikube-grafana-portforward) and [`make kind-grafana-portforward`](#make-minikube-grafana-portforward) to run  on Minikube or Kind.

#### `make controller-portforward`

Sets up port forwarding to a specified PORT var (defaults to 8080 for controller metrics) to the
controller deployment.

#### `make pprof-cpu-web`

Start the web interface for pprof for cpu profiling.

#### `make pprof-heap-web`

Start the web interface for pprof for heap profiling.

#### `make shell`
Run a bash shell with the developer tools (go tooling, kubectl, etc) and source code in it.

#### `make pkgsite`
Run a container with pkgsite on port 8888

#### `make build-controller-image`
Compile the gameserver controller and then build the docker image

#### `make build-agones-sdk-image`
Compile the gameserver sidecar and then build the docker image

#### `make build-ping-image`
Compile the ping binary and then build the docker image

#### `make gen-install`
Generate the `/install/yaml/install.yaml` from the Helm template

#### `make gen-embedded-openapi`
Generate the embedded OpenAPI specs for existing Kubernetes Objects, such as `PodTemplateSpec` and `ObjectMeta`.

This should be run against a clean or brand new cluster, as external CRD's or schemas could cause errors to occur.

#### `make gen-crd-code`
Generate the Custom Resource Definition client(s), conversions, deepcopy, and defaults code.

#### `make gen-allocation-grpc`
Generate the allocator gRPC code

#### `make gen-all-sdk-grpc`
Generate the SDK gRPC server and client code for all SDKs.

#### `make gen-sdk-grpc`
Generate the SDK gRPC server and client code for a single SDK (specified in the `SDK_FOLDER` variable).

### Build Image Targets

Targets for building the build image

#### `make clean-config`
Cleans the kubernetes and gcloud configurations

#### `make clean-build-image`
Deletes the local build docker image

#### `make build-build-image`
Creates the build docker image

### Google Cloud Platform

A set of utilities for setting up a Kubernetes Engine cluster on Google Cloud Platform,
since it's an easy way to get a test cluster working with Kubernetes.

#### `make gcloud-init`
Initialise the gcloud login and project configuration, if you are working with GCP.

#### `make gcloud-test-cluster`
Creates and authenticates a GKE cluster to work against.

#### `make clean-gcloud-test-cluster`
Delete a GKE cluster previously created with `make gcloud-test-cluster`.

#### `make gcloud-auth-cluster`
Pulls down authentication information for kubectl against a cluster, name can be specified through CLUSTER_NAME
(defaults to 'test-cluster').

#### `make gcloud-auth-docker`
Creates a short lived access to Google Cloud container repositories, so that you are able to call
`docker push` directly. Useful when used in combination with `make push` command.

### Terraform

Utilities for deploying a Kubernetes Engine cluster on Google Cloud Platform using `google` Terraform provider.

#### `make gcloud-terraform-cluster`
Create GKE cluster and install release version of agones.
Run next command to create GKE cluster with agones (version from helm repository):
```
[GKE_PASSWORD="<YOUR_PASSWORD>"] make gcloud-terraform-cluster
```
Where `<YOUR_PASSWORD>` should be at least 16 characters in length. You can omit GKE_PASSWORD and then basic auth would be disabled. Also you change `ports="7000-8000"` setting using tfvars file.
Also you can define password `password=<YOUR_PASSWORD>` string in `build/terraform.tfvars`.
Change AGONES_VERSION to a specific version you want to install.

#### `make gcloud-terraform-install`
Create GKE cluster and install current version of agones.
The current version should be built and pushed to `$(REGISTRY)` beforehand:
```
make build-images
make push
```

#### `make gcloud-terraform-destroy-cluster`
Run `terraform destroy` on your cluster.

#### `make terraform-clean`
Remove .terraform directory with configs as well as tfstate files.

### Minikube

A set of utilities for setting up and running a [Minikube](https://github.com/kubernetes/minikube) instance,
for local development.

Since Minikube runs locally, there are some targets that need to be used instead of the standard ones above.

#### `make minikube-test-cluster`
Switches to an "agones" profile, and starts a kubernetes cluster
of the right version. Uses "docker" as the default driver.

If needed, use MINIKUBE_DRIVER variable to change the VM driver.

#### `make minikube-install`

Installs the current development version of Agones into the Kubernetes cluster.
Use this instead of `make install`, as it disables PullAlways on the install.yaml

#### `make minikube-push`

Push the local Agones Docker images that have already been built
via `make build` or `make build-images` into the "agones" minikube instance with `minikube cache add`

#### `make minikube-setup-prometheus`

Installs prometheus metric backend into the Kubernetes cluster.
Use this instead of `make setup-prometheus`, as it disables Persistent Volume Claim.

#### `make minikube-setup-grafana`

Installs grafana into the Kubernetes cluster.
Use this instead of `make setup-grafana`, as it disables Persistent Volume Claim.

#### `make minikube-setup-prometheus-stack`

Installs prometheus-stack into the Kubernetes cluster.
Use this instead of `make setup-prometheus-stack`, as it disables Persistent Volume Claim.

#### `make minikube-prometheus-portforward`

The minikube version of [`make prometheus-portforward`](#make-prometheus-portforward) to setup
port forwarding to the prometheus deployment.

#### `make minikube-grafana-portforward`

The minikube version of [`make grafana-portforward`](#make-grafana-portforward) to setup
port forwarding to the grafana deployment.

#### `make minikube-test-e2e`
Runs end-to-end tests on the previously installed version of Agones.
These tests validate Agones flow from start to finish.

⚠ Running all the e2e tests can often overwhelm a local minikube cluster, so use at your own risk. You should look at
[Running Individual End-to-End Tests](building-testing.md#running-individual-end-to-end-tests) to run tests on a case by case basis.

#### `make minikube-shell`
Connecting to Minikube requires so enhanced permissions, so use this target
instead of `make shell` to start an interactive shell for development on Minikube.

#### `make minikube-controller-portforward`
The minikube version of [`make controller-portforward`](#make-controller-portforward) to setup
port forwarding to the controller deployment.

#### `make minikube-install-debug`
Install Agones in debug mode with all services set to 1 replica and debug images.
This target is specifically designed for debugging workflows with proper debug configuration.

#### `make minikube-debug-portforward`
Start port forwarding for all Agones services to enable remote debugging.
Sets up the following port forwards:
- Controller: localhost:2346 -> agones-controller:2346
- Extensions: localhost:2347 -> agones-extensions:2346
- Ping: localhost:2348 -> agones-ping:2346  
- Allocator: localhost:2349 -> agones-allocator:2346
- Processor: localhost:2350 -> agones-processor:2346

Use environment variables to customize ports (see [Development Workflow Guide](development-workflow.md)).

#### `make minikube-debug-sdk-portforward`
Start port forwarding for debugging the Agones SDK sidecar in game server pods.
Provides interactive mode to select from available game server pods or specify a pod directly.
Forwards local port 2351 to the debug port 2346 in the selected pod.

### Kind

[Kind - kubernetes in docker](https://github.com/kubernetes-sigs/kind) is a tool for running local Kubernetes clusters using Docker container "nodes".

Since Kind runs locally, there are some targets that need to be used instead of the standard ones above.

#### `make kind-test-cluster`
Starts a local kubernetes cluster, you can delete it with `make kind-delete-cluster`

Use KIND_PROFILE variable to change the name of the cluster.

#### `make kind-push`
Push the local Agones Docker images that have already been built
via `make build` or `make build-images` into the "agones" Kind cluster.

#### `make kind-install`
Installs the current development version of Agones into the Kubernetes cluster.
Use this instead of `make install`, as it disables PullAlways on the install.yaml

#### `make kind-setup-prometheus`

Installs prometheus metric backend into the Kubernetes cluster.
Use this instead of `make setup-prometheus`, as it disables Persistent Volume Claim.

#### `make kind-setup-grafana`

Installs grafana into the Kubernetes cluster.
Use this instead of `make setup-grafana`, as it disables Persistent Volume Claim.

#### `make kind-setup-prometheus-stack`

Installs prometheus-stack into the Kubernetes cluster.
Use this instead of `make setup-prometheus-stack`, as it disables Persistent Volume Claim.

#### `make kind-prometheus-portforward`

The kind version of [`make prometheus-portforward`](#make-prometheus-portforward) to setup
port forwarding to the prometheus deployment.

#### `make kind-grafana-portforward`

The kind version of [`make grafana-portforward`](#make-grafana-portforward) to setup
port forwarding to the grafana deployment.


#### `make kind-test-e2e`
Runs end-to-end tests on the previously installed version of Agones.
These tests validate Agones flow from start to finish.

#### `make kind-shell`
Connecting to Kind requires so enhanced permissions, so use this target
instead of `make shell` to start an interactive shell for development on Kind.

#### `make kind-controller-portforward`
The Kind version of [`make controller-portforward`](#make-controller-portforward) to setup
port forwarding to the controller deployment.

## Next Steps

- See [Building and Testing Guide](building-testing.md) for basic build workflows
- See [Cluster Setup Guide](cluster-setup.md) for setting up clusters to use these targets
- See [Development Workflow Guide](development-workflow.md) for advanced development patterns
