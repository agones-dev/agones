---
title: "REST Game Server Client API"
linkTitle: "Rest"
date: 2019-01-02T10:18:08Z
weight: 100
description: "This is the REST version of the Agones Game Server Client SDK. "
---

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.


## SDK Functionality

| Area                | Action                | Implemented |
|---------------------|-----------------------|-------------|
| Lifecycle           | Ready                 | ✔️          |
| Lifecycle           | Health                | ✔️          |
| Lifecycle           | Reserve               | ✔️          |
| Lifecycle           | Allocate              | ✔️          |
| Lifecycle           | Shutdown              | ✔️          |
| Configuration       | GetGameServer         | ✔️          |
| Configuration       | WatchGameServer       | ✔️          |
| Metadata            | SetAnnotation         | ✔️          |
| Metadata            | SetLabel              | ✔️          |
| Counters            | GetCounter            | ✔️          |
| Counters            | UpdateCounter         | ✔️          |
| Lists               | GetList               | ✔️          |
| Lists               | UpdateList            | ✔️          |
| Lists               | AddListValue          | ✔️          |
| Lists               | RemoveListValue       | ✔️          |


The REST API can be accessed from `http://localhost:${AGONES_SDK_HTTP_PORT}/` from the game server process.
`AGONES_SDK_HTTP_PORT` is an environment variable automatically set for the game server process by Agones to
support binding the REST API to a dynamic port. It is advised to use the environment variable rather than a
hard coded port; otherwise your game server will not be able to contact the SDK server if it is configured to
use a non-default port.

Generally the REST interface gets used if gRPC isn't well supported for a given language or platform.

{{% alert title="Warning" color="warning" %}}
The SDK Server sidecar process may startup after your game server binary. So your REST SDK API calls should
contain some retry logic to take this into account.
{{% /alert %}}

## Generating clients

While you can hand write REST integrations, we also have a set
of {{< ghlink href="sdks/swagger" >}}generated OpenAPI/Swagger definitions{{< /ghlink >}} available.
This means you can use OpenAPI/Swagger tooling to generate clients as well, if you need them.

For example, to create a cpp client for the stable sdk endpoints (to be run in the `agones` home directory):
```bash
docker run --rm -v ${PWD}:/local swaggerapi/swagger-codegen-cli generate -i /local/sdks/swagger/sdk.swagger.json  -l cpprest -o /local/out/cpp
```

The same could be run for `alpha.swagger.json` and `beta.swagger.json` as required.

You can read more about OpenAPI/Swagger code generation in their [Command Line Tool Documentation](https://swagger.io/docs/open-source-tools/swagger-codegen/codegen-v3/about/)

## Reference

### Lifecycle Management

#### Ready

Call when the GameServer is ready to accept connections

- Path: `/ready`
- Method: `POST`
- Body: `{}`

##### Example

```bash
curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:${AGONES_SDK_HTTP_PORT}/ready
```

#### Health
Send a Empty every d Duration to declare that this GameServer is healthy

- Path: `/health`
- Method: `POST`
- Body: `{}`

##### Example

```bash
curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:${AGONES_SDK_HTTP_PORT}/health
```

#### Reserve

Move Gameserver into a Reserved state for a certain amount of seconds for the future allocation.

- Path: `/reserve`
- Method: `POST`
- Body: `{"seconds": "5"}`

##### Example

```bash
curl -d '{"seconds": "5"}' -H "Content-Type: application/json" -X POST http://localhost:${AGONES_SDK_HTTP_PORT}/reserve
```

#### Allocate

With some matchmakers and game matching strategies, it can be important for game servers to mark themselves as `Allocated`.
For those scenarios, this SDK functionality exists.

{{% alert title="Note" color="info" %}}
Using a [GameServerAllocation]({{< ref "/docs/Reference/gameserverallocation.md" >}}) is preferred in all other scenarios,
as it gives Agones control over how packed `GameServers` are scheduled within a cluster, whereas with `Allocate()` you
relinquish control to an external service which likely doesn't have as much information as Agones.
{{% /alert %}}

##### Example

```bash
curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:${AGONES_SDK_HTTP_PORT}/allocate
```

#### Shutdown

Call when the GameServer session is over and it's time to shut down

- Path: `/shutdown`
- Method: `POST`
- Body: `{}`

##### Example

```bash
curl -d "{}" -H "Content-Type: application/json" -X POST http://localhost:${AGONES_SDK_HTTP_PORT}/shutdown
```

### Configuration Retrieval


#### GameServer

Call when you want to retrieve the backing `GameServer` configuration details

- Path: `/gameserver`
- Method: `GET`

```bash
curl -H "Content-Type: application/json" -X GET http://localhost:${AGONES_SDK_HTTP_PORT}/gameserver
```

Response:
```json
{
    "object_meta": {
        "name": "local",
        "namespace": "default",
        "uid": "1234",
        "resource_version": "v1",
        "generation": "1",
        "creation_timestamp": "1531795395",
        "annotations": {
            "annotation": "true"
        },
        "labels": {
            "islocal": "true"
        }
    },
    "status": {
        "state": "Ready",
        "address": "127.0.0.1",
        "ports": [
            {
                "name": "default",
                "port": 7777
            }
        ]
    }
}
```

#### Watch GameServer

Call this when you want to get updates of when the backing `GameServer` configuration is updated.

These updates will come as newline delimited JSON, send on each update. To that end, you will
want to keep the http connection open, and read lines from the result stream and and process as they
come in.

```bash
curl -H "Content-Type: application/json" -X GET http://localhost:${AGONES_SDK_HTTP_PORT}/watch/gameserver
```

Response:
```json
{"result":{"object_meta":{"name":"local","namespace":"default","uid":"1234","resource_version":"v1","generation":"1","creation_timestamp":"1533766607","annotations":{"annotation":"true"},"labels":{"islocal":"true"}},"status":{"state":"Ready","address":"127.0.0.1","ports":[{"name":"default","port":7777}]}}}
{"result":{"object_meta":{"name":"local","namespace":"default","uid":"1234","resource_version":"v1","generation":"1","creation_timestamp":"1533766607","annotations":{"annotation":"true"},"labels":{"islocal":"true"}},"status":{"state":"Ready","address":"127.0.0.1","ports":[{"name":"default","port":7777}]}}}
{"result":{"object_meta":{"name":"local","namespace":"default","uid":"1234","resource_version":"v1","generation":"1","creation_timestamp":"1533766607","annotations":{"annotation":"true"},"labels":{"islocal":"true"}},"status":{"state":"Ready","address":"127.0.0.1","ports":[{"name":"default","port":7777}]}}}
```

The Watch GameServer stream is also exposed as a WebSocket endpoint on the same URL and port as the HTTP `watch/gameserver` API. This endpoint is provided as a convienence for streaming data to clients such as Unreal that support WebSocket but not HTTP streaming, and HTTP streaming should be used instead if possible.

An example command that uses the WebSocket endpoint instead of streaming over HTTP is:


```bash
curl -N -H "Connection: Upgrade" -H "Upgrade: websocket" -H "Sec-WebSocket-Key: ExampleKey1234567890===" -H "Sec-WebSocket-Version: 13" -X GET http://localhost:${AGONES_SDK_HTTP_PORT}/watch/gameserver
```

The data returned from this endpoint is delimited by the boundaries of a WebSocket payload as defined by [RFC 6455, section 5.2](https://tools.ietf.org/html/rfc6455#section-5.2). When reading from this endpoint, if your WebSocket client does not automatically handle frame reassembly (e.g. Unreal), make sure to read to the end of the WebSocket payload (as defined by the FIN bit) before attempting to parse the data returned. This is transparent in most clients.

### Metadata Management

#### Set Label

Apply a Label with the prefix "agones.dev/sdk-" to the backing `GameServer` metadata.

See the SDK [SetLabel]({{< ref "/docs/Guides/Client SDKs/_index.md#setlabelkey-value" >}}) documentation for restrictions.

##### Example

```bash
curl -d '{"key": "foo", "value": "bar"}' -H "Content-Type: application/json" -X PUT http://localhost:${AGONES_SDK_HTTP_PORT}/metadata/label
```

#### Set Annotation

Apply an Annotation with the prefix "agones.dev/sdk-" to the backing `GameServer` metadata

##### Example

```bash
curl -d '{"key": "foo", "value": "bar"}' -H "Content-Type: application/json" -X PUT http://localhost:${AGONES_SDK_HTTP_PORT}/metadata/annotation
```

### Counters and Lists

{{< beta title="Counters and Lists" gate="CountsAndLists" >}}

#### Counters

In all the Counter examples, we retrieve the counter under the key `rooms` as if it was previously defined in [`GameServer.Spec.counters[room]`]({{< ref "/docs/Reference/agones_crd_api_reference.html#agones.dev/v1.GameServerSpec" >}}).

For your own Counter REST requests, replace the value `rooms` with your own key in the path.

##### Beta: GetCounter
This function retrieves a specified counter by its key, `rooms`, and returns its information.

###### Example

```bash
curl -H "Content-Type: application/json" -X GET http://localhost:${AGONES_SDK_HTTP_PORT}/v1beta1/counters/rooms
```

Response:
```json
{"name":"rooms", "count":"1", "capacity":"10"}
```

##### Beta: UpdateCounter
This function updates the properties of the counter with the key `rooms`, such as its count and capacity, and returns the updated counter details.

###### Example

```bash
curl -d '{"count": "5", "capacity": "11"}' -H "Content-Type: application/json" -X PATCH http://localhost:${AGONES_SDK_HTTP_PORT}/v1beta1/counters/rooms
```

Response:
```json
{"name":"rooms", "count":"5", "capacity":"11"}
```

#### Lists

In all the List examples, we retrieve the list under the key `players` as if it was previously defined in [`GameServer.Spec.lists[players]`]({{< ref "/docs/Reference/agones_crd_api_reference.html#agones.dev/v1.GameServerSpec" >}}).

For your own List REST based requests, replace the value `players` with your own key in the path.

##### Beta: GetList
This function retrieves the list's properties with the key `players`, returns the list's information.

###### Example
```bash
curl -H "Content-Type: application/json" -X GET http://localhost:${AGONES_SDK_HTTP_PORT}/v1beta1/lists/players
```

Response:
```json
{"name":"players", "capacity":"100", "values":["player0", "player1", "player2"]}
```

##### Beta: UpdateList
This function updates the list's properties with the key `players`, such as its capacity and values, and returns the updated list details. Use `AddListValue` or `RemoveListValue` for modifying the `List.Values` field.


###### Example
```bash
curl -d '{"capacity": "120", "values": ["player3", "player4"]}' -H "Content-Type: application/json" -X PATCH http://localhost:${AGONES_SDK_HTTP_PORT}/v1beta1/lists/players
```
Response:
```json
{"name":"players", "capacity":"120", "values":["player3", "player4"]}
```
##### Beta: AddListValue
This function adds a new value to a list with the key `players` and returns the list with this addition.

###### Example

```bash
curl -d '{"value": "player9"}' -H "Content-Type: application/json" -X POST http://localhost:${AGONES_SDK_HTTP_PORT}/v1beta1/lists/players:addValue
```

Response:
```json
{"name":"players", "capacity":"120", "values":["player3", "player4", "player9"]}
```

##### Beta: RemoveListValue
This function removes a value from the list with the key `players` and returns updated list.

###### Example

```bash
curl -d '{"value": "player3"}' -H "Content-Type: application/json" -X POST http://localhost:${AGONES_SDK_HTTP_PORT}/v1beta1/lists/players:removeValue
```

Response:
```json
{"name":"players", "capacity":"120", "values":["player4", "player9"]}
```