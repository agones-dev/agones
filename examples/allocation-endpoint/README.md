# Allocation Endpoint

This is a sample solution to enable an Allocation Endpoint proxy in front of GKE Agones clusters.

In this folder you will find:

1. Terraform modules that create resources in GCP
2. The Allocation Endpoint Proxy code that gets deployed to Cloud Run
3. An [ESP container](https://cloud.google.com/endpoints/docs/grpc/get-started-cloud-run) that gets deployed as a sidecar alongside the `agones-allocator`
4. A sample client code to send allocation requests to the proxy
5. Dockerfile that builds the proxy and scripts to push the image to a docker repository
6. Documentation on how to use the solution.

Here is the architecture of GCP resources created:

![architecture](https://github.com/agones-dev/agones/blob/main/examples/allocation-endpoint/architecture.png?raw=true)


## GKE cluster
First and foremost you need to create clusters and install Agones to experiment with this solution.
The clusters can be in the same GCP project as your Allocation Endpoint proxy or they can be in a different project.

---
**NOTE**

The solution has not been tested with non-GKE clusters.

---

When creating GKE Clusters, [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) needs to be enabled. Please follow [GKE cluster setup](https://agones.dev/site/docs/installation/creating-cluster/gke/) and include workload-pool, e.g:

<pre>
gcloud container clusters create [NAME] \
  ...
  <b>--workload-pool=[GKE-PROJECT-ID].svc.id.goog \ </b>
</pre>

Install Agones on your cluster. You need to disable mTLS because the `agones-allocator` container will be the backend for ESP container.

```
helm upgrade my-release --install --namespace agones-system --create-namespace agones/agones \
  --set agones.allocator.disableMTLS=true \
  --set agones.allocator.disableTLS=true \
  --set agones.allocator.service.http.enabled=false
```

After installing Agones, deploy [ESP](https://cloud.google.com/endpoints/docs/grpc/specify-esp-v2-startup-options) which is an envoy based proxy, deployed as a sidecar along side `agones-alloator` container. Run the following to patch the service deployment, change the service port to ESP and add annotation to `agones-allocator` service account to impersonate GCP service account. 

Replace [GKE-PROJECT-ID] in `patch-agones-allocator.yaml` with your project ID before running the scripts.

```
kubectl patch deployment agones-allocator -n agones-system --patch-file patch-agones-allocator.yaml
kubectl patch svc agones-allocator -n agones-system --type merge -p '{"spec":{"ports": [{"port": 443,"name":"https","targetPort":9443}]}}'
kubectl annotate sa -n agones-system agones-allocator iam.gke.io/gcp-service-account=ae-esp-sa@[PROJECT-ID].iam.gserviceaccount.com
```

## Terraform 
The terraform modules create resources in GCP:

```
terraform apply \
  -var "project_id=[PROJECT-ID]" \
  -var "authorized_members=[\"serviceAccount:[SERVICE-ACCOUNT-EMAIL]\"]" \
  -var "clusters_info=[CLUSTERS-INFO]" \
  -var "workload-pool=[GKE-PROJECT-ID].svc.id.goog"
```

`[CLUSTERS-INFO]` is in the form of `[{\"name\":\"cluster1\",\"endpoint\":\"34.83.14.82\",\"namespace\":\"default\",\"allocation_weight\":100},{...}]` deserializing to []ClusterInfo, defined in the `server/clusterselector.go`.

- The `name` is a unique randomly selected name for the cluster.
- The `endpoint` is the `agones-allocator` external IP.
- The `namespace` is the game server namespace.
- The `allocation_weight` is a value between 0 and 100, which sets the relative allocation rate a cluster receives compared to other clusters. By setting weight to zero, a cluster stops receiving allocation requests.

`[SERVICE-ACCOUNT-EMAIL]` is the service account to be granted access the Allocation Endpoint. You need to have [the service account created](https://cloud.google.com/iam/docs/creating-managing-service-accounts) before running terraform.

## Server

The Allocation Endpoint proxy code is in `./server` folder. You can make changes and run the following to build and push the image to your own GCR repository:

```
docker build --tag gcr.io/[PROJECT-ID]/allocation-endpoint-proxy:[VERSION] .
docker push gcr.io/[PROJECT-ID]/allocation-endpoint-proxy:[VERSION]
```

If you are building your own image, you can set `ae_proxy_image` terraform variable to your image.

## Client

The Allocation Endpoint client code is in `./client` folder. Get the Service Account Key for one of the Service Accounts in the list of `authorized_members` and put it under `sa_key.json`. Alternatively, you can leverage [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity) to retrieve the access token using default service account from metadata serer, when deploying your client in a GCP solution, e.g. GKE.

```
go run *.go --url=[CLOUD-RUN-ENDPOINT]

```

`[CLOUD-RUN-ENDPOINT]` is the cloud run endpoint FQDN printed out after running the terraform. Leave out the scheme when setting the value e.g. `allocation-endpoint-proxy-<code>.a.run.app`.

## Future considerations
- Requests using this example goes to public IP. For clusters in the same project you can instead leverage VPC with private IPs and remove dependency to the Service Account and Secret Manager to issue JWT in the proxy.
- The proxy is skipping server cert validation. When you create a valid TLS cert, remove `InsecureSkipVerify: true``.
- The solution should be compatible with non-GKE Agones clusters, but has not been tested.

