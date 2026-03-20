# Performance Testing Guide

This guide covers how to set up and run performance tests for Agones.

## Running Performance Test

To be able to run the performance script located in the following path: `agones/build/performance-test.sh` you need to have the following setup:

### Install a standard GKE cluster 

Follow for more details here: https://agones.dev/site/docs/installation/creating-cluster/gke/

### Install Agones

Follow for more details here: https://agones.dev/site/docs/installation/install-agones/helm/

### Set up node for prometheus 

```
gcloud container node-pools create agones-metrics --cluster={CLUSTER_NAME} --zone={REGION} \
  --node-taints agones.dev/agones-metrics=true:NoExecute \
  --node-labels agones.dev/agones-metrics=true \
  --num-nodes=1 \
  --machine-type=e2-standard-4
```

### Install Prometheus, Grafana and port-forward

cd agones/

https://agones.dev/site/docs/guides/metrics/#installation

### Modify the performance-test.sh

The performance-tests script contains a set of variables that need to be overwritten to work with your cluster and configuration settings. You can also pass these values through command line This is an example:

```
CLUSTER_NAME=agones-standard
CLUSTER_LOCATION=us-central1-c
REGISTRY=us-east1-docker.pkg.dev/user/agones
PROJECT=my-project
REPLICAS=10000
AUTO_SHUTDOWN_DELAY=60
BUFFER_SIZE=9900
MAX_REPLICAS=20000
DURATION=10m
CLIENTS=50
INTERVAL=1000
```

You might also want to comment out the first couple lines that come after the variables are set and also change the cd directoy:

```
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

This script is an entyrpoint to be able to run the allocation performance test which can be found at `agones/test/load/allocation`

You can see the fleet and autoscaler configuration (such as buffer size and min/max replicas, etc) in the following files: 

* [performance-test-fleet-template](https://github.com/agones-dev/agones/blob/main/test/load/allocation/performance-test-fleet-template.yaml)
* [performance-test-autoscaler-template.yaml](https://github.com/agones-dev/agones/blob/main/test/load/allocation/performance-test-autoscaler-template.yaml)

You could also modify the `automatic shutdown delay` parameter where if the value is greater than zero, it will automatically shut down the server this many seconds after the server becomes allocated (cannot be used if `automaticShutdownDelayMin` is set). It's a configuration for the simple game server.

Something to keep in mind with CLIENTS and INTERVAL is the following. Let's say you have client count 50 and interval 500ms, which means every client will submit 2 allocation requests in 1s, so the entire allocation requests that the allocator receives in 1s is 50 * 2 = 100, so the allocation request QPS from the allocator view is 100. 

Finally, you can cd agones/build and run `sh performance-test.sh` if you see timeout issues please re-run the command. 

## Next Steps

- See [Cluster Setup Guide](cluster-setup.md) for setting up GKE clusters for performance testing
- See [Make Reference](make-reference.md) for performance-related make targets