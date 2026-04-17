---
title: "Python Game Server Client SDK"
linkTitle: "Python"
date: 2026-04-01T00:00:00Z
publishDate: 2026-05-19T00:00:00Z
weight: 55
description: "This is the Python version of the Agones Game Server Client SDK."
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
| Configuration   | GameServer          | ✔️          |
| Configuration   | Watch               | ✔️          |
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
| Player Tracking | GetConnectedPlayers | ✔️          |
| Player Tracking | GetPlayerCapacity   | ✔️          |
| Player Tracking | GetPlayerCount      | ✔️          |
| Player Tracking | IsPlayerConnected   | ✔️          |
| Player Tracking | PlayerConnect       | ✔️          |
| Player Tracking | PlayerDisconnect    | ✔️          |
| Player Tracking | SetPlayerCapacity   | ✔️          |

## Prerequisites

- [Python >= 3.10](https://www.python.org/downloads/)

## Installation

Install from source, {{< ghlink href="sdks/python" >}}directly from GitHub{{< /ghlink >}}

## Usage

To begin working with the SDK, create an instance and connect to the sidecar.

```python
from agones import AgonesSDK

sdk = AgonesSDK()
sdk.connect()
```

A context manager is also supported:

```python
with AgonesSDK() as sdk:
    sdk.ready()
    # game logic
```

To send a [health check]({{< relref "_index.md#health" >}}) ping, call `sdk.health()`. This should be called periodically in a background thread.

```python
import threading
import time

def health_loop():
    while True:
        sdk.health()
        time.sleep(2)

threading.Thread(target=health_loop, daemon=True).start()
```

To mark the [game session as ready]({{< relref "_index.md#ready" >}}) call `sdk.ready()`.

```python
sdk.ready()
```

To mark that the [game session is completed]({{< relref "_index.md#shutdown" >}}) and the game server should be shut down call `sdk.shutdown()`.

```python
sdk.shutdown()
```

To mark the game server as [reserved]({{< relref "_index.md#reserveseconds" >}}) for a period of time, call `sdk.reserve(seconds)`.

```python
sdk.reserve(30)
```

To [set a Label]({{< relref "_index.md#setlabelkey-value" >}}) on the backing `GameServer` call `sdk.set_label(key, value)`.

```python
sdk.set_label("test-label", "test-value")
```

To [set an Annotation]({{< relref "_index.md#setannotationkey-value" >}}) on the backing `GameServer` call `sdk.set_annotation(key, value)`.

```python
sdk.set_annotation("test-annotation", "test value")
```

To get [details of the backing `GameServer`]({{< relref "_index.md#gameserver" >}}) call `sdk.get_game_server()`.

```python
gameserver = sdk.get_game_server()
print(f"Name: {gameserver.object_meta.name}")
print(f"State: {gameserver.status.state}")
```

To get [updates on the backing `GameServer`]({{< relref "_index.md#watchgameserverfunctiongameserver" >}}) as they happen, call `sdk.watch_game_server(callback)`.

This will watch for updates in a background daemon thread and call the provided callback on each update.

```python
def on_update(gameserver):
    print(f"GameServer update, state: {gameserver.status.state}")

sdk.watch_game_server(on_update)
```

For more information, please read the [SDK Overview]({{< relref "_index.md" >}}), check out {{< ghlink href="sdks/python/agones/sdk.py" >}}the Python SDK implementation{{< /ghlink >}}.
