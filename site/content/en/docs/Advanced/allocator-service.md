---
title: "Allocator Service"
date: 2020-05-19T05:45:05Z
publishDate: 2019-10-25T05:45:05Z
description: >
  Agones provides an mTLS based allocator service that is accessible from outside the cluster using a load balancer. The service is deployed and scales independent to Agones controller.
---

To allocate a game server, Agones provides a gRPC and REST service with mTLS authentication, called `agones-allocator` that can be used instead of {{< ghlink href="pkg/apis/allocation/v1/gameserverallocation.go" >}}GameServerAllocations{{< /ghlink >}}.

Both gRPC and REST are accessible through a Kubernetes service that can be externalized using a load balancer. By default, gRPC and REST are served from the same port. However, either service can be disabled or the services can be served from separate ports using the [helm configuration]({{< relref "/docs/Installation/Install Agones/helm.md" >}}).

{{% alert title="Warning" color="warning" %}}
If gRPC and REST are served using the same port, then an http multi-plexer is used along with an [experimental gRPC server](https://github.com/grpc/grpc-go/blob/2608e38e6386be7400720fecf2ece176c4cbc1b2/server.go#L933-L960) which has [noticeably worse performance](https://github.com/grpc/grpc-go/issues/586#issuecomment-286257439) than using the standard gRPC server.

If you require a fully compatible or feature compatible gRPC server implementation, you must separate the gRPC port from the REST port or disable the REST service.
{{% /alert %}}

For requests to either service to succeed, a client certificate must be provided that is in the authorization list of the allocator service.
The remainder of this article describes how to manually make a successful allocation request using the API.

The guide assumes you have command line tools installed for [jq](https://stedolan.github.io/jq/), [go](https://golang.org/) and [openssl](https://www.openssl.org/).

## `GameServerAllocation` vs Allocator Service

There are several reasons you may prefer to use the Allocator Service over the `GameServerAllocation` custom resource
definition, depending on your architecture and requirements:

* A requirement to do [multi-cluster allocation]({{% relref "multi-cluster-allocation.md" %}}).
* Want to create Allocations from outside the Agones Kubernetes cluster.
* Prefer SSL based authentication over Kubernetes [RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/).
* Prefer a [gRPC](https://grpc.github.io/) or REST based API over an integration with the
  [Kubernetes API]({{% ref "/docs/Guides/access-api.md" %}}).

## Find the external IP

The service is hosted under the same namespace as the Agones controller. To find the external IP of your allocator service, replace agones-system namespace with the namespace to which Agones is deployed and execute the following command:

```bash
kubectl get service agones-allocator -n agones-system
```

The output of the command should look like:

<pre>
NAME                        TYPE           CLUSTER-IP      <b>EXTERNAL-IP</b>     PORT(S)            AGE
agones-allocator            LoadBalancer   10.55.251.73    <b>34.82.195.204</b>   443:30250/TCP      7d22h
</pre>

## Server TLS certificate

If the `agones-allocator` service is installed as a `LoadBalancer` [using a reserved IP]({{< relref "/docs/Installation/Install Agones/helm.md#reserved-allocator-load-balancer-ip" >}}), a valid self-signed server TLS certificate is generated using the IP provided. Otherwise, the server TLS certificate should be replaced. If you installed Agones using [helm]({{< relref "/docs/Installation/Install Agones/helm.md" >}}), you can easily reconfigure the allocator service with a preset IP address by setting the `agones.allocator.service.loadBalancerIP` parameter to the address that was automatically assigned to the service and `helm upgrade`:

```bash
EXTERNAL_IP=$(kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
helm upgrade my-release agones/agones -n agones-system --wait \
   --set agones.allocator.service.loadBalancerIP=${EXTERNAL_IP} \
   ...
```

{{% alert title="Warning" color="warning" %}} The parameter used to automatically
replace the certificate changed in Agones 1.18.0. If you are using an older
version of Agones you should pass the parameter
`agones.allocator.http.loadBalancerIP` instead. If you need your script to work
with both older and newer versions of Agones, you can pass both parameters as
only one of them will effect the helm chart templates.
{{% /alert %}}

Another approach is to replace the default server TLS certificate with a certificate with CN and subjectAltName. There are multiple approaches to generate a certificate. Agones recommends using [cert-manager.io](https://cert-manager.io/) solution for cluster level certificate management.

In order to use the cert-manager solution, first [install cert-manager](https://cert-manager.io/docs/installation/kubernetes/) on the cluster.
Then, [configure](https://cert-manager.io/docs/configuration/) an `Issuer`/`ClusterIssuer` resource and
last [configure](https://cert-manager.io/docs/usage/certificate/) a `Certificate` resource to manage allocator-tls `Secret`.
Make sure to configure the `Certificate` based on your system's requirements, including the validity `duration`.

Here is an example of using a self-signed `ClusterIssuer` for configuring allocator-tls `Secret`:

```bash
#!/bin/bash
# Create a self-signed ClusterIssuer
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: selfsigned
spec:
  selfSigned: {}
EOF

EXTERNAL_IP=$(kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# for EKS use hostname
# HOST_NAME=$(kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')

# Create a Certificate with IP for the allocator-tls secret
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: allocator-tls
  namespace: agones-system
spec:
  commonName: ${EXTERNAL_IP}
  ipAddresses:
    - ${EXTERNAL_IP}
  secretName: allocator-tls
  issuerRef:
    name: selfsigned
    kind: ClusterIssuer
EOF

# Wait for the allocator-tls Secret
sleep 1
TLS_CA_VALUE=$(kubectl get secret allocator-tls -n agones-system -ojsonpath='{.data.ca\.crt}')

# Add ca.crt to the allocator-tls-ca Secret
kubectl get secret allocator-tls-ca -o json -n agones-system | jq '.data["tls-ca.crt"]="'${TLS_CA_VALUE}'"' | kubectl apply -f -
echo $TLS_CA_VALUE | base64 -d > ca.crt
# In case of MacOS
# echo $TLS_CA_VALUE | base64 -D > ca.crt
```

### Bring Your Own Certificates (advanced)

If you would like to completely manage the tls secrets outside of helm, you can create them in the namespace where agones is going to be installed, and then set the helm value `agones.allocator.disableSecretCreation` to `true`. This method will also work with the cert-manager method, as long as your certificate and secret are created ahead of time, and you populate the `allocator-tls-ca` and `allocator-client-ca` yourself.

## Client Certificate

Because agones-allocator uses an mTLS authentication mechanism, a client must provide a certificate that is accepted by the server.

If Agones is installed using Helm, you can leverage a default client secret, `allocator-client.default`, created in the game server namespace and allowlisted in `allocator-client-ca` Kubernetes secret. You can extract and use that secret for client side authentication, by following [the allocation example]({{< relref "#send-allocation-request" >}}).

Otherwise, here is an example of generating a client certificate using openssl.

```bash
#!/bin/bash

EXTERNAL_IP=$(kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout client.key -out client.crt -addext 'subjectAltName=IP:'${EXTERNAL_IP}''

CERT_FILE_VALUE=$(cat client.crt | base64 -w 0)

# In case of MacOS
# CERT_FILE_VALUE=$(cat client.crt | base64)

# allowlist client certificate
kubectl get secret allocator-client-ca -o json -n agones-system | jq '.data["client_trial.crt"]="'${CERT_FILE_VALUE}'"' | kubectl apply -f -
```

The last command creates a new entry in the secret data map for `allocator-client-ca` for the client CA. This is for the `agones-allocator` service to accept the newly generated client certificate.

## Send allocation request

After setting up `agones-allocator` with server certificate and allowlisting the client certificate, the service can be used to allocate game servers. Make sure you have a [fleet]({{< ref "/docs/Getting Started/create-fleet.md" >}}) with ready game servers in the game server namespace.

Set the environment variables and store the client secrets before allocating using gRPC or REST APIs:

```bash
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

### Using gRPC

{{% feature expiryVersion="1.63.0" %}}
To start, take a look at the allocation gRPC client examples in {{< ghlink href="examples/allocator-client/main.go" >}}golang{{< /ghlink >}} and {{< ghlink href="examples/allocator-client-csharp/Program.cs" >}}C#{{< /ghlink >}} languages.
{{% /feature %}}
{{% feature publishVersion="1.63.0" %}}
To start, take a look at the allocation gRPC client example in {{< ghlink href="examples/allocator-client/main.go" >}}golang{{< /ghlink >}}.
{{% /feature %}}

In the following, the {{< ghlink href="examples/allocator-client/main.go" >}}golang gRPC client example{{< /ghlink >}} is used to allocate a Game Server in the `default` namespace.

```bash
#!/bin/bash

go run examples/allocator-client/main.go --ip ${EXTERNAL_IP} \
    --port 443 \
    --namespace ${NAMESPACE} \
    --key ${KEY_FILE} \
    --cert ${CERT_FILE} \
    --cacert ${TLS_CA_FILE}
```

### Using REST

```bash
#!/bin/bash

curl --key ${KEY_FILE} \
     --cert ${CERT_FILE} \
     --cacert ${TLS_CA_FILE} \
     -H "Content-Type: application/json" \
     --data '{"namespace":"'${NAMESPACE}'"}' \
     https://${EXTERNAL_IP}/gameserverallocation \
     -X POST

```

You should expect to see the following output:

```json
{"gameServerName":"game-server-name","ports":[{"name":"default","port":7463}],"address":"1.2.3.4","nodeName":"node-name"}
```

### Sending Data to the Game Server

The service accepts a `metadata` field, which can be used to apply `labels` and `annotations` to the allocated `GameServer`. The old `metaPatch` fields is now deprecated, but can still be used for compatibility. If both `metadata` and `metaPatch` fields are set, `metaPatch` is ignored.

## Secrets Explained

`agones-allocator` has a dependency on three Kubernetes secrets:

1. `allocator-tls` - stores the server certificate.
2. `allocator-client-ca` - stores the allocation authorized client CA for mTLS to allowlist client certificates.
3. `allocator-tls-ca` (optional) - stores `allocator-tls` CA.

The separation of CA secret from the private secret is for the security reason to avoid reading the private secret, while retrieving the allocator CA that is used by the allocation client to validate the server. It is optional to set or maintain the `allocator-tls-ca` secret.

## Troubleshooting

If you encounter problems, explore the following potential root causes:

1. Check server certificate - Using openssl you can get the certificate chain for the server.

    ```bash
    EXTERNAL_IP=$(kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
    openssl s_client -connect ${EXTERNAL_IP}:443
    ```

    - Inspect the server certificate by storing the certificate returned, under `Server certificate` and validating using `openssl x509 -in tls.crt -text -noout`.
    - Make sure the certificate is not expired and the Subject Alternative Name is set.
    - If the issuer is `CN = allocation-ca`, the certificate is generated using Agones helm installation.

2. Check client certificate

    - You may get an error such as `rpc error: code = Unavailable desc = all SubConns are in TransientFailure, latest connection error: connection closed`, make sure your client certificate is allowlisted by being added to `allocator-client-ca`.

    ```bash
    kubectl get secret allocator-client-ca -o json -n agones-system
    ```

    - If the server certificate is not accepted by the client, you may get an error such as `rpc error: code = Unavailable desc = all SubConns are in TransientFailure, latest connection error: connection error: desc = "transport: authentication handshake failed: x509: certificate signed by unknown authority"`, depending on the client. In this case, verify that the TLS CA file matches the server certificate.

    ```bash
    kubectl get secret allocator-tls -n agones-system -ojsonpath="{.data.tls\.crt}" | base64 -d > tls.crt
    openssl verify -verbose -CAfile ca.crt tls.crt
    tls.crt: OK
    ```

3. Make sure the service is up and running.

    ```bash
    kubectl get pod -n agones-system | grep agones-allocator
    agones-allocator-59b4f6b5c6-86j62      1/1     Running     0          6m36s
    agones-allocator-59b4f6b5c6-kbqrq      1/1     Running     0          6m45s
    agones-allocator-59b4f6b5c6-trbkl      1/1     Running     0          6m28s
    ```

    ```bash
    kubectl get service agones-allocator -n agones-system
    agones-allocator   LoadBalancer   10.55.248.14   34.82.195.204    443:32468/TCP   6d23h
    ```

## API Reference

The AllocationService API is located as a gRPC service {{< ghlink href="proto/allocation/allocation.proto" >}}here{{< /ghlink >}}. Additionally, the REST API is available as a {{< ghlink href="pkg/allocation/go/allocation.swagger.json" >}}Swagger API{{< /ghlink >}}.
