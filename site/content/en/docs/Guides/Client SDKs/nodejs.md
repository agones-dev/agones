---
title: "Node.js Game Server Client SDK"
linkTitle: "Node.js"
date: 2019-02-24T15:56:57Z
publishDate: 2019-04-01
weight: 50
description: "This is the Node.js version of the Agones Game Server Client SDK."
---

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.


## SDK Functionality

| Area            | Action              | Implemented |
|-----------------|---------------------|-------------|
| Lifecycle       | Ready               | ✔️          |
| Lifecycle       | Health              | ✔️          |
| Lifecycle       | Reserve             | ✔️          |
| Lifecycle       | Allocate            | ✔️          |
| Lifecycle       | Shutdown            | ✔️          |
| Configuration   | GetGameServer       | ✔️          |
| Configuration   | WatchGameServer     | ✔️          |
| Metadata        | SetAnnotation       | ✔️          |
| Metadata        | SetLabel            | ✔️          |
| Counters        | GetCounterCount     | ✔️          |
| Counters        | SetCounterCount     | ✔️          |
| Counters        | IncrementCounter    | ✔️          |
| Counters        | DecrementCounter    | ✔️          |
| Counters        | SetCounterCapacity  | ✔️          |
| Counters        | GetCounterCapacity  | ✔️          |
| Lists           | AppendListValue     | ✔️          |
| Lists           | DeleteListValue     | ✔️          |
| Lists           | SetListCapacity     | ✔️          |
| Lists           | GetListCapacity     | ✔️          |
| Lists           | ListContains        | ✔️          |
| Lists           | GetListLength       | ✔️          |
| Lists           | GetListValues       | ✔️          |

## Prerequisites

- Node.js >= 10.13.0

## Usage

Add the agones dependency to your project:

```sh
npm install @google-cloud/agones-sdk
```

If you need to download the source, rather than install from NPM, you can find it on 
{{< ghlink href="sdks/nodejs" >}}GitHub{{< /ghlink >}}.

To begin working with the SDK, create an instance of it.

```javascript
const AgonesSDK = require('@google-cloud/agones-sdk');

let agonesSDK = new AgonesSDK();
```

To connect to the SDK server, either local or when running on Agones, run the `async` method `sdk.connect()`, which will
`resolve` once connected or `reject` on error or if no connection can be made after 30 seconds.

```javascript
await agonesSDK.connect();
```

To send a [health check]({{< relref "_index.md#health" >}}) ping call `health(errorCallback)`. The error callback is optional and if provided will receive an error whenever emitted from the health check stream.

```javascript
agonesSDK.health((error) => {
	console.error('error', error);
});
```

To mark the game server as [ready to receive player connections]({{< relref "_index.md#ready" >}}), call the async method `ready()`. The result will be an empty object in this case.

```javascript
let result = await agonesSDK.ready();
```

Similarly `shutdown()`, `allocate()`, `setAnnotation(key, value)` and `setLabel(key, value)` are async methods that perform an action and return an empty result.

To get [details of the backing GameServer]({{< relref "_index.md#gameserver" >}}) call the async method
`getGameServer()`. The result will be an object representing `GameServer` defined
in {{< ghlink href="proto/sdk/sdk.proto" >}}`sdk.proto`{{< /ghlink >}}.

```javascript
let result = await agonesSDK.getGameServer();
```

To get [updates on the backing GameServer]({{< relref "_index.md#watchgameserverfunctiongameserver" >}}) as they happen, call `watchGameServer(callback, errorCallback)`. The callback will be called with a parameter matching the result of `getGameServer()`. The error callback is optional and if provided will receive an error whenever emitted from the watch stream.

```javascript
agonesSDK.watchGameServer((result) => {
	console.log('watch', result);
}, (error) => {
	console.error('error', error);
});
```

To mark the game server as [reserved]({{< relref "_index.md#reserveseconds" >}}) for a period of time, call the async method `reserve(seconds)`. The result will be an empty object.

For more information, please read the [SDK Overview]({{< relref "_index.md" >}}), check out {{< ghlink href="sdks/nodejs/src/agonesSDK.js" >}}agonesSDK.js{{< /ghlink >}} and also look at the {{< ghlink href="examples/nodejs-simple" >}}Node.js example{{< / >}}.
