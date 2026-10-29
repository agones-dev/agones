# Performance Testing Guide

This guide is a manual runbook for provisioning, running, and tearing down the Agones allocation
performance tests.

> [!WARNING]
> **The performance test cluster is large and expensive.** The default configuration in
> `build/terraform/performance/module.tf` creates a regional GKE cluster with 200 `e2-standard-4`
> nodes **per zone** (roughly 600 nodes in a three-zone region). There is no continuously running
> performance cluster and no automated Cloud Build trigger. Plan and budget each run, and tear the
> cluster down as soon as you are done.

The following are kept in the repository so the tests can be run when there is a planned need:

* `build/terraform/performance/` - Terraform for the cluster (`module.tf`) and its state bucket (`state-bucket/`)
* `ci/perf-test-cloudbuild.yaml` - Cloud Build config that builds, installs Agones and runs the test
* `build/performance-test.sh` - entrypoint script that creates the fleet and autoscaler and runs the scenario
* `test/load/allocation/` - the allocation load test itself

The examples below use `<PROJECT>` as a placeholder for the GCP project you will run the tests in.
Choose the project deliberately: these commands provision a very large cluster.

## 1. Bootstrap the Terraform state bucket

The cluster's Terraform state is stored in a GCS bucket named `<PROJECT>-performance-infra-bucket-tfstate`.
Create it first:

```bash
cd build; make shell
cd build/terraform/performance/state-bucket
terraform init
terraform apply -var project="<PROJECT>"
```

## 2. Create the performance test cluster

Check `kubernetes_versions` in `build/terraform/performance/module.tf` (the region should be
`us-central1`), then initialize Terraform:

```bash
cd build; make shell
cd build/terraform/performance
terraform init -backend-config="bucket=<PROJECT>-performance-infra-bucket-tfstate" -backend-config="prefix=terraform/state"
```

`module.tf` also declares the `gke-game-server-firewall` and `gke-game-server-firewall-tcp`
firewall rules, which `build/terraform/e2e/` declares as well. How you apply depends on the project:

* **Project where the e2e firewall rules already exist** (e.g. `agones-images`): target only the
  cluster module, so the existing rules are never created or deleted:

  ```bash
  terraform apply -var project="<PROJECT>" -target=module.gke_standard_cluster
  ```

* **Fresh project**: apply everything, which also creates the firewall rules:

  ```bash
  terraform apply -var project="<PROJECT>"
  ```

The cluster is named `standard-performance-test-cluster-<version>`, for example
`standard-performance-test-cluster-1-35`. It must match `_TEST_CLUSTER_NAME` in
`ci/perf-test-cloudbuild.yaml`.

## 3. Run the performance test

### Option A: Cloud Build (recommended)

From the repository root, submit the Cloud Build config manually. It builds and pushes the images,
installs Agones on the performance cluster, and runs `build/performance-test.sh`.

The defaults in `ci/perf-test-cloudbuild.yaml` point at the `agones-images` project, so override
`_TEST_PROJECT_ID` and `_REGISTRY` with your project and an Artifact Registry repository in it:

```bash
gcloud builds submit . --project=<PROJECT> --config=ci/perf-test-cloudbuild.yaml \
  --substitutions=_TEST_PROJECT_ID=<PROJECT>,_REGISTRY=us-docker.pkg.dev/<PROJECT>/<REPOSITORY>
```

Any of the other `_TEST_*` substitutions can be overridden in the same way, for example:

```bash
gcloud builds submit . --project=<PROJECT> --config=ci/perf-test-cloudbuild.yaml \
  --substitutions=_TEST_PROJECT_ID=<PROJECT>,_REGISTRY=us-docker.pkg.dev/<PROJECT>/<REPOSITORY>,_TEST_DURATION=10m,_TEST_CLIENTS=50,_TEST_INTERVAL=1000
```

Follow the build in the Cloud Build console or with `gcloud builds log --stream <BUILD_ID>`.
The scenario results are printed at the end of the `performance-test` step.

### Option B: Run `build/performance-test.sh` against your own cluster

You can also run the test against any standard GKE cluster with Agones installed:

1. Create a standard GKE cluster: https://agones.dev/site/docs/installation/creating-cluster/gke/
2. Install Agones: https://agones.dev/site/docs/installation/install-agones/helm/
3. Optionally, add a node pool for Prometheus and install Prometheus and Grafana
   (https://agones.dev/site/docs/guides/metrics/#installation):

   ```bash
   gcloud container node-pools create agones-metrics --cluster={CLUSTER_NAME} --zone={REGION} \
     --node-taints agones.dev/agones-metrics=true:NoExecute \
     --node-labels agones.dev/agones-metrics=true \
     --num-nodes=1 \
     --machine-type=e2-standard-4
   ```

4. The script is written to run inside Cloud Build. To run it locally, comment out the lines that
   come after the variables are set, and change the `cd` directory:

   ```bash
   # export SHELL="/bin/bash"
   # mkdir -p /go/src/agones.dev/
   # ln -sf /workspace /go/src/agones.dev/agones
   # cd /go/src/agones.dev/agones/build

   # gcloud config set project $PROJECT
   # gcloud container clusters get-credentials $CLUSTER_NAME \
   #        --zone=$CLUSTER_LOCATION --project=$PROJECT

   # make install LOG_LEVEL=info REGISTRY='"'$REGISTRY'"' DOCKER_RUN=""

   # cd /go/src/agones.dev/agones/test/load/allocation
   cd ../test/load/allocation
   ```

5. From `agones/build`, run the script. It takes its settings as positional arguments:

   ```bash
   # Arguments: CLUSTER_NAME CLUSTER_LOCATION REGISTRY PROJECT REPLICAS AUTO_SHUTDOWN_DELAY
   #            BUFFER_SIZE MAX_REPLICAS DURATION CLIENTS INTERVAL
   ./performance-test.sh agones-standard us-central1-c us-docker.pkg.dev/<PROJECT>/<REPOSITORY> <PROJECT> \
     10000 60 9900 20000 10m 50 1000
   ```

   If you see timeout issues, re-run the command.

### Test configuration

The script is an entrypoint for the allocation performance test in `test/load/allocation`. The
fleet and autoscaler configuration (buffer size, min/max replicas, etc.) is in:

* [performance-test-fleet-template.yaml](https://github.com/agones-dev/agones/blob/main/test/load/allocation/performance-test-fleet-template.yaml)
* [performance-test-autoscaler-template.yaml](https://github.com/agones-dev/agones/blob/main/test/load/allocation/performance-test-autoscaler-template.yaml)

If `AUTO_SHUTDOWN_DELAY` is greater than zero, the simple game server shuts itself down that many
seconds after it is allocated.

`CLIENTS` and `INTERVAL` set the offered load. For example, 50 clients with a 500ms interval means
each client sends 2 allocation requests per second, so the allocator receives 50 * 2 = 100
requests per second.

## 4. Tear down the cluster and state bucket

Destroy the cluster as soon as the run is finished, using the same form you used to create it:

```bash
cd build; make shell
cd build/terraform/performance
terraform init -backend-config="bucket=<PROJECT>-performance-infra-bucket-tfstate" -backend-config="prefix=terraform/state"
```

Then run only the destroy command that matches how you applied:

* **Project where the e2e firewall rules already exist** (e.g. `agones-images`): target only the
  cluster module, so the existing firewall rules are left in place:

  ```bash
  terraform destroy -var project="<PROJECT>" -target=module.gke_standard_cluster
  ```

* **Fresh project**: destroy everything, including the firewall rules:

  ```bash
  terraform destroy -var project="<PROJECT>"
  ```

Then delete the state bucket. It is versioned and `force_destroy` is `false`, so empty it
(including old object versions) before deleting it:

```bash
gcloud storage rm --recursive --all-versions 'gs://<PROJECT>-performance-infra-bucket-tfstate/**'
gcloud storage buckets delete gs://<PROJECT>-performance-infra-bucket-tfstate
```

Confirm that nothing is left behind:

```bash
gcloud container clusters list --project=<PROJECT> --filter="name~performance"
gcloud compute disks list --project=<PROJECT> --filter="name~performance"
gcloud compute forwarding-rules list --project=<PROJECT>
```

## Next Steps

- See [Cluster Setup Guide](cluster-setup.md) for setting up GKE clusters
- See [Make Reference](make-reference.md) for performance-related make targets
