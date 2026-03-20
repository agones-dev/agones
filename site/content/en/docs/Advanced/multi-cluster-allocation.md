---
title: "Multi-cluster Allocation"
date: 2019-10-25T05:45:05Z
description: >
  In order to allow allocation from multiple clusters, Agones provides a mechanism to set redirect rules for allocation requests to the right cluster.
---

{{% pageinfo color="info" %}}
This implementation of multi-cluster allocation was written before managed and open source multi-cluster Service Meshes 
such as [Istio](https://istio.io/latest/docs/setup/install/multicluster/)
and [Linkerd](https://linkerd.io/2.15/features/multicluster/), were available and so widely utilised.

We now recommend implementing a Service Mesh in each of your Agones clusters and backend services cluster to provide 
a multi-cluster allocation endpoint that points to each Agones cluster's 
[Allocation Service]({{< relref "allocator-service.md">}}).

Service Mesh specific projects provide far more powerful features, easier configuration and maintenance, and we 
expect that they will be something that you will likely be installing in your multi-cluster architecture anyway.

Further documentation on setting up Agones with a Service Mesh is incoming, but to see an example utilising
[Google Cloud Service Mesh](https://cloud.google.com/service-mesh), which is backed by Istio, see the 
[Global Scale Game](https://github.com/googleforgames/global-multiplayer-demo) demo project.
{{% /pageinfo %}}

There may be different types of clusters, such as on-premise, and Google Kubernetes Engine (GKE), used by a game to help with the cost-saving and availability.
For this purpose, Agones provides a mechanism to define priorities on the clusters. Priorities are defined on {{< ghlink href="pkg/apis/multicluster/v1/gameserverallocationpolicy.go" >}}GameServerAllocationPolicy{{< /ghlink >}} agones CRD. A matchmaker can enable the multi-cluster rules on a request and target [agones-allocator]({{< relref "allocator-service.md">}}) endpoint in any of the clusters and get resources allocated on the cluster with the highest priority. If the cluster with the highest priority is overloaded, the allocation request is redirected to the cluster with the next highest priority.

The remainder of this article describes how to enable multi-cluster allocation.

## Define Cluster Priority

{{< ghlink href="pkg/apis/multicluster/v1/gameserverallocationpolicy.go" >}}GameServerAllocationPolicy{{< /ghlink >}} is the CRD defined by Agones for setting multi-cluster allocation rules. In addition to cluster priority, it describes the connection information for the target cluster, including the game server namespace, agones-allocator endpoint and client K8s secrets name for redirecting the allocation request. Game servers will be allocated from clusters with the lowest `priority` number. If there are no available game servers available in clusters with the lowest `priority` number, they will be allocated from clusters with the next lowest `priority` number. For clusters with the same priority, the cluster is chosen with a probability relative to its weight.

Here is an example of setting the priority for a cluster and it's connection rules. One such resource should be defined per cluster.

In the following example the policy is defined for cluster B in cluster A.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: multicluster.agones.dev/v1
kind: GameServerAllocationPolicy
metadata:
  name: allocator-cluster-b
  namespace: cluster-a-ns
spec:
  connectionInfo:
    allocationEndpoints:
    - 34.82.195.204
    clusterName: "clusterB"
    namespace: cluster-b-ns
    secretName: allocator-client-to-cluster-b
    serverCa: c2VydmVyQ0E=
  priority: 1
  weight: 100
EOF
```

To define the local cluster priority a GameServerAllocationPolicy should be defined _without_ an `allocationEndpoints` field. If the local cluster priority is not defined, the allocation from the local cluster happens only if allocation from other clusters with the existing allocation rules is unsuccessful.

Allocation requests with multi-cluster allocation enabled but with only the local cluster available (e.g. in development) _must_ have a local cluster priority defined, or the request fails with the error "no multi-cluster allocation policy is specified".

The `namespace` field in `connectionInfo` is the namespace that the game servers will be allocated in, and must be a namespace in the target cluster that has been previously defined as allowed to host game servers. The `Namespace` specified in the allocation request (below) is used to refer to the namespace that the GameServerAllocationPolicy itself is located in.

`serverCa` is the server TLS CA public certificate, set only if the remote server certificate is not signed by a public CA (e.g. self-signed). If this field is not specified, the certificate can also be specified in a field named `ca.crt` of the client secret (the secret referred to in the `secretName` field).

## Establish trust

To accept allocation requests from other clusters, agones-allocator for cluster B should be configured to accept the client's certificate from cluster A and the cluster A's client should be configured to accept the server TLS certificate, if it is not signed by a public Certificate Authority (CA).

Follow the steps to configure the [agones allocator gRPC service]({{< relref "allocator-service.md">}}). The client certificate pair in the mentioned document is stored as a K8s secret. Here are the secrets to set:

1.Client certificate to talk to other clusters:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: allocator-client-to-cluster-b
  namespace: cluster-a-ns
type: Opaque
data:
  tls.crt: <REDACTED>
  tls.key: <REDACTED>
  ca.crt: <REDACTED>
EOF
```

The certificates are base 64 string of the certificate file e.g. `cat ${CERT_FILE} | base64 -w 0`

Agones recommends using [cert-manager.io](https://cert-manager.io/) solution for generating client certificates.

2.Add client CA to the list of authorized client certificates by agones-allocator in the targeted cluster.

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: allocator-client-ca
  namespace: agones-system
type: Opaque
data:
  client1.crt: <REDACTED>
  client2.crt: <REDACTED>
  …
  clientN.crt: <REDACTED>
EOF
```

## Allocate multi-cluster

To enable multi-cluster allocation, set `multiClusterSetting.enabled` to `true` in {{< ghlink href="proto/allocation/allocation.proto" >}}allocation.proto{{< /ghlink >}} and send allocation requests. For more information visit [agones-allocator]({{< relref "allocator-service.md">}}). In the following, using {{< ghlink href="examples/allocator-client/main.go" >}}allocator-client sample{{< /ghlink >}}, a multi-cluster allocation request is sent to the agones-allocator service. If the allocation succeeds, the AllocationResponse will contain a {{< ghlink href="proto/allocation/allocation.proto" >}}Source{{< /ghlink >}} field which indicates the endpoint of the remote agones-allocator.

Set the environment variables and store the client secrets before allocating using gRPC or REST APIs

```bash
#!/bin/bash

NAMESPACE=default # replace with any namespace
EXTERNAL_IP=$(kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
KEY_FILE=client.key
CERT_FILE=client.crt
TLS_CA_FILE=ca.crt

# allocator-client.default secret is created only when using helm installation. Otherwise generate the client certificate and replace the following.
# In case of MacOS replace "base64 -d" with "base64 -D"
kubectl get secret allocator-client.default -n "${NAMESPACE}" -ojsonpath="{.data.tls\.crt}" | base64 -d > "${CERT_FILE}"
kubectl get secret allocator-client.default -n "${NAMESPACE}" -ojsonpath="{.data.tls\.key}" | base64 -d > "${KEY_FILE}"
kubectl get secret allocator-tls-ca -n agones-system -ojsonpath="{.data.tls-ca\.crt}" | base64 -d > "${TLS_CA_FILE}"
```

```bash
#!/bin/bash

go run examples/allocator-client/main.go --ip ${EXTERNAL_IP} \
    --namespace ${NAMESPACE} \
    --key ${KEY_FILE} \
    --cert ${CERT_FILE} \
    --cacert ${TLS_CA_FILE} \
    --multicluster true
```

If using REST use

```bash
#!/bin/bash

curl --key ${KEY_FILE} \
     --cert ${CERT_FILE} \
     --cacert ${TLS_CA_FILE} \
     -H "Content-Type: application/json" \
     --data '{"namespace":"'${NAMESPACE}'", "multiClusterSetting":{"enabled":true}}' \
     https://${EXTERNAL_IP}/gameserverallocation \
     -X POST
     
```

## Troubleshooting

If you encounter problems, explore the following potential root causes:

1. Make sure single cluster allocation works for each cluster using [this troubleshooting]({{< relref "allocator-service.md#troubleshooting">}}).

2. For each cluster, make sure there is a `GameServerAllocationPolicy` resource defined in the game server cluster.

3. Inspect the `.spec.connectionInfo` for `GameServerAllocationPolicy` for each cluster. Use the cluster connection information in that field to verify that single cluster allocation works. Use the information to verify the connection:

```none
POLICY_NAME=<policy-name>
POLICY_NAMESPACE=<policy-namespace>

NAMESPACE=$(kubectl get gameserverallocationpolicy ${POLICY_NAME} -n ${POLICY_NAMESPACE} -ojsonpath={.spec.connectionInfo.namespace})
EXTERNAL_IP=$(kubectl get gameserverallocationpolicy ${POLICY_NAME} -n ${POLICY_NAMESPACE} -ojsonpath={.spec.connectionInfo.allocationEndpoints\[0\]})
CLIENT_SECRET_NAME=$(kubectl get gameserverallocationpolicy ${POLICY_NAME} -n ${POLICY_NAMESPACE} -ojsonpath={.spec.connectionInfo.secretName})

KEY_FILE=client.key
CERT_FILE=client.crt
TLS_CA_FILE=ca.crt

# In case of MacOS replace "base64 -d" with "base64 -D"
kubectl get secret "${CLIENT_SECRET_NAME}" -n "${POLICY_NAMESPACE}" -ojsonpath="{.data.tls\.crt}" | base64 -d > "${CERT_FILE}"
kubectl get secret "${CLIENT_SECRET_NAME}" -n "${POLICY_NAMESPACE}" -ojsonpath="{.data.tls\.key}" | base64 -d > "${KEY_FILE}"
kubectl get secret "${CLIENT_SECRET_NAME}" -n "${POLICY_NAMESPACE}" -ojsonpath="{.data.ca\.crt}" | base64 -d > "${TLS_CA_FILE}"
```

```bash
#!/bin/bash

go run examples/allocator-client/main.go --ip ${EXTERNAL_IP} \
    --port 443 \
    --namespace ${NAMESPACE} \
    --key ${KEY_FILE} \
    --cert ${CERT_FILE} \
    --cacert ${TLS_CA_FILE}
```
