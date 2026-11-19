# Load Tests for gRPC Allocation Service

[Allocation Load Test](#allocation-load-test) and [Scenario Tests](#scenario-tests)
for testing the performance of the gRPC allocation service.

## Prerequisites

1. A [Kubernetes cluster](https://agones.dev/site/docs/installation/creating-cluster/) with [Agones](https://agones.dev/site/docs/installation/install-agones/)
    - We recommend installing Agones using the [Helm](https://agones.dev/site/docs/installation/install-agones/helm/) package manager.
    - If you are running in GCP, use a regional cluster instead of a zonal cluster to ensure high availability of the cluster control plane.
    - Use a dedicated node pool for the Agones controllers with multiple CPUs per node, e.g. 'e2-standard-4'.
    - For Allocation Load Test:
      - In the default node pool (where the Game Server pods are created), 75 nodes are required to make sure there are enough nodes available for all game servers to move into the ready state. When using a regional GKE cluster with three zones that will require a configuration of 25 nodes per zone.
    - For Scenario Tests:
      - See [Kubernetes Cluster Setup for Scenario Tests](#kubernetes-cluster-setup-for-scenario-tests)
2. A configured [Allocator Service](https://agones.dev/site/docs/advanced/allocator-service/)
    - The allocator service uses gRPC. In order to be able to call the service, TLS
and mTLS have to be set up on the Server and Client.
3. (Optional) [Metrics](https://agones.dev/site/docs/guides/metrics/) for monitoring Agones workloads

# Allocation Load Test

This load test aims to validate performance of the gRPC allocation service.

## Fleet Setting

We used the sample [fleet configuration](./fleet.yaml). We set the `automaticShutdownDelaySec` parameter to 600 so simple-game-server game servers shutdown after 10
minutes. This helps to easily re-run the test without having to delete the game servers and allows to run tests continously.


## Running the test

```
kubectl apply -f ./fleet.yaml
````
Wait until the fleet shows 4000 ready game servers before running the allocation script.
```
kubectl get fleet
NAME              SCHEDULING   DESIRED   CURRENT   ALLOCATED   READY   AGE
load-test-fleet   Packed       4000      4000      0           4000    2m38s
```

You can use the provided runAllocation.sh script by providing two parameters:
- number of clients (to do parallel allocations)
- number of allocations for client

For making 4000 allocations calls, you can provide 40 and 100

```
./runAllocation.sh 40 100
```

Script will print out the start and end date/time:
```
started: 2020-10-22 23:33:25.828724579 -0700 PDT m=+0.005921014
finished: 2020-10-22 23:34:18.381396416 -0700 PDT m=+52.558592912
```

If some errors occurred, the error message will be printed:
```
started: 2020-10-22 22:16:47.322731849 -0700 PDT m=+0.002953843
(failed(client=3,allocation=43): rpc error: code = Unknown desc = error updating allocated gameserver: Operation cannot be fulfilled on gameservers.agones.dev "simple-game-server-mlljx-g9crp": the object has been modified; please apply your changes to the latest version and try again
(failed(client=2,allocation=47): rpc error: code = Unknown desc = error updating allocated gameserver: Operation cannot be fulfilled on gameservers.agones.dev "simple-game-server-mlljx-rxflv": the object has been modified; please apply your changes to the latest version and try again
(failed(client=7,allocation=45): rpc error: code = Unknown desc = error updating allocated gameserver: Operation cannot be fulfilled on gameservers.agones.dev "simple-game-server-mlljx-x4khw": the object has been modified; please apply your changes to the latest version and try again
finished: 2020-10-22 22:17:18.822039094 -0700 PDT m=+31.502261092
```

You can use environment variables overwrite defaults. To run only a single run of tests, you can use:
```
TESTRUNSCOUNT=1 ./runAllocation.sh 40 10
```


# Scenario Tests

The scenario test allows you to generate a variable number of allocations to
your cluster over time, simulating a game where clients arrive in an unsteady
pattern. The game servers used in the test are configured to shutdown after
being allocated, simulating the GameServer churn that is expected during
normal game play.

## Kubernetes Cluster Setup for Scenario Tests

For the scenario test to achieve high throughput, you can create multiple groups
of nodes in your cluster. During testing (on GKE), we created a node pool for
the Kubernetes system components (such as the metrics server and dns servers), a
node pool for the Agones system components (as recommended in the installation
guide), and a node pool for the game servers.

On GKE, to restrict the Kubernetes system components to their own set of nodes,
you can create a node pool with the taint
`components.gke.io/gke-managed-components=true:NoExecute`.

To prevent the Kubernetes system components from running on the game servers
node pool, that node pool was created with the taint
`scenario-test.io/game-servers=true:NoExecute`
and the Agones system node pool used the normal taint
`agones.dev/agones-system=true:NoExecute`.

In addition, the GKE cluster was configured as a regional cluster to ensure high
availability of the cluster control plane.

The following commands were used to construct a cluster for testing:

```bash
export REGION="us-west1"
export VERSION="1.23"

gcloud container clusters create scenario-test --cluster-version=$VERSION \
  --tags=game-server --scopes=gke-default --num-nodes=2 \
  --no-enable-autoupgrade --machine-type=n2-standard-2 \
  --region=$REGION --enable-ip-alias

gcloud container node-pools create kube-system --cluster=scenario-test \
  --no-enable-autoupgrade \
  --node-taints components.gke.io/gke-managed-components=true:NoExecute \
  --num-nodes=1 --machine-type=n2-standard-16 --region $REGION

gcloud container node-pools create agones-system --cluster=scenario-test \
  --no-enable-autoupgrade --node-taints agones.dev/agones-system=true:NoExecute \
  --node-labels agones.dev/agones-system=true --num-nodes=1 \
  --machine-type=n2-standard-16 --region $REGION

gcloud container node-pools create game-servers --cluster=scenario-test \
  --node-taints scenario-test.io/game-servers=true:NoExecute --num-nodes=1 \
  --machine-type n2-standard-2 --no-enable-autoupgrade \
  --region $REGION --tags=game-server --scopes=gke-default \
  --enable-autoscaling --max-nodes=300 --min-nodes=175
```

## Agones Modifications

For the scenario tests, we modified the Agones installation in a number of ways.

First, we made sure that the Agones pods would _only_ run in the Agones node
pool by changing the node affinity in the deployments for the controller,
allocator service, and ping service to
`requiredDuringSchedulingIgnoredDuringExecution`.

We also increased the resources for the controller and allocator service pods,
and made sure to specify both requests and limits to ensure that the pods were
given the highest quality of service.

These configuration changes are captured in
[scenario-values.yaml](scenario-values.yaml) and can be applied during
installation using helm:

```bash
helm install my-release --namespace agones-system -f scenario-values.yaml agones/agones --create-namespace
```

Alternatively, these changes can be applied to an existing Agones installation
by running [`./configure-agones.sh`](configure-agones.sh).

## Fleet Setting

We used the sample [fleet configuration](./scenario-fleet.yaml) and [fleet autoscaler configuration](./autoscaler.yaml).

To reduce pod churn in the system, the simple game servers are configured to
return themselves to `Ready` after being allocated the first 10 times following
the [Reusing Allocated GameServers for more than one game
session](https://agones.dev/site/docs/integration-patterns/reusing-gameservers/)
integration pattern. After 10 simulated game sessions, the simple game servers
then exit automatically. The fleet configuration above sets each game session to
last for 1 minute, representing a short game.

## Running the test

You can use the provided runScenario.sh script by providing one parameter (a
scenario file). The scenario file is a simple text file where each line
represents a "scenario" that the program will execute before moving to the next
scenario. A scenario is a duration, the number of concurrent clients to use, and
the interval of the allocation requests submitted by each client in milliseconds
separated by a comma. The program will create the desired number of clients and
those clients send allocation requests to the allocator service for the scenario
duration in the defined cadence. At the end of each scenario the program will print
out some statistics for the scenario:

- **Requests Sent / Offered rate**: the load generated by the clients. This is
  not a measure of allocator performance.
- **Throughput**: successful allocations divided by the time from the first
  request to the last response, including requests still in flight when the
  scenario duration ends.
- **Throughput within window**: successful allocations that completed within
  the scenario duration, divided by that duration. When the allocator cannot
  keep up with the offered rate, this is the closest measure of its sustained
  capacity.
- **Allocation latency**: percentiles of the time taken by successful
  allocation requests.

If throughput is well below the offered rate, or latency grows towards the 15
second request timeout, the allocator is saturated.

Two sample scenario files are included in this directory, one which sends a
constant rate of allocations for the duration of the test and another that sends
a variable number of allocations.

Since error counts are gathered per scenario, it's recommended to keep each
scenario short (e.g. 10 minutes) to narrow down the window when errors
occurred even if the allocation rate stays at the same level for longer than
10 minutes at a time.

