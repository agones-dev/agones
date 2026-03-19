---
title: "Unity Game Server Client SDK"
linkTitle: "Unity"
date: 2019-06-13T10:17:50Z
weight: 15
description: "This is the Unity version of the Agones Game Server Client SDK."
---

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.

## SDK Functionality

| Area                 | Action                   | Implemented                   |
|----------------------|--------------------------|-------------------------------|
| Lifecycle            | Ready                    | ✔️                            |
| Lifecycle            | Health                   | ✔️                            | 
| Lifecycle            | Reserve                  | ✔️                            | 
| Lifecycle            | Allocate                 | ✔️                            | 
| Lifecycle            | Shutdown                 | ✔️                            | 
| Configuration        | GameServer               | ✔️                            | 
| Configuration        | Watch                    | ✔️                            | 
| Metadata             | SetAnnotation            | ✔️                            | 
| Metadata             | SetLabel                 | ✔️                            | 
| Counters             | GetCounterCount          | ❌                           |
| Counters             | SetCounterCount          | ❌                           |
| Counters             | IncrementCounter         | ❌                           |
| Counters             | DecrementCounter         | ❌                           |
| Counters             | SetCounterCapacity       | ❌                           |
| Counters             | GetCounterCapacity       | ❌                           |
| Lists                | AppendListValue          | ❌                           |
| Lists                | DeleteListValue          | ❌                           |
| Lists                | SetListCapacity          | ❌                           |
| Lists                | GetListCapacity          | ❌                           |
| Lists                | ListContains             | ❌                           |
| Lists                | GetListLength            | ❌                           |
| Lists                | GetListValues            | ❌                           |
| Player Tracking      | GetConnectedPlayers      | ✔️                            | 
| Player Tracking      | GetPlayerCapacity        | ✔️                            | 
| Player Tracking      | GetPlayerCount           | ✔️                            | 
| Player Tracking      | IsPlayerConnected        | ✔️                            | 
| Player Tracking      | PlayerConnect            | ✔️                            | 
| Player Tracking      | PlayerDisconnect         | ✔️                            | 
| Player Tracking      | SetPlayerCapacity        | ✔️                            | 

Additional methods have been added for ease of use:

- Connect

## Installation

The client SDK code can be manually downloaded and added to your project hierarchy.

It can also be imported into your project via the Unity Package Manager (UPM). To do that, open your project's `manifest.json` file, and add the following line to the dependencies section:

```
{
  "dependencies": {
        "com.googleforgames.agones": "https://github.com/agones-dev/agones.git?path=/sdks/unity",
...
```

If you want a specific release, the dependency can be pinned to that version. For example: 

`"com.googleforgames.agones": "https://github.com/agones-dev/agones.git?path=/sdks/unity#v{{< release-version >}}",`

## Download

Download the source {{< ghlink href="sdks/unity" >}}directly from GitHub{{< /ghlink >}}.

## Prerequisites

- Unity >= 2018.x (.NET 4.x)

## Usage

Import this script to your unity project and attach it to GameObject.

To begin working with the SDK, get an instance of it.

```csharp
var agones = agonesGameObject.GetComponent<Agones.AgonesSdk>();
```

To connect to the SDK server, either local or when running on Agones, run the async `Connect()` method.
This will wait for up to 30 seconds if the SDK server has not yet started and the connection cannot be made,
and will return `false` if there was an issue connecting.

```csharp
bool ok = await agones.Connect();
```

To mark the game server as [ready to receive player connections]({{< relref "_index.md#ready" >}}), call the async method `Ready()`.

```csharp
async void SomeMethod()
{
    bool ok = await agones.Ready();
}
```

To get the details on the [backing `GameServer`]({{< relref "_index.md#gameserver" >}}) call `GameServer()`.

Will return `null` if there is an error in retrieving the `GameServer` record.

```csharp
var gameserver = await agones.GameServer();
```

To mark the GameServer as [Reserved]({{< relref "_index.md#reserveseconds" >}}) for a duration call 
`Reserve(TimeSpan duration)`.

```csharp
ok = await agones.Reserve(duration);
```

To mark that the [game session is completed]({{< relref "_index.md#shutdown" >}}) and the game server should be shut down call `Shutdown()`.

```csharp
bool ok = await agones.Shutdown();
```

Similarly `SetAnnotation(string key, string value)` and `SetLabel(string key, string value)` are async methods that perform an action.

And there is no need to call `Health()`, it is automatically called.

To watch when 
[the backing `GameServer` configuration changes]({{< relref "_index.md#watchgameserverfunctiongameserver" >}})
call `WatchGameServer(callback)`, where the delegate function `callback` will be executed every time the `GameServer` 
configuration changes.

```csharp
agones.WatchGameServer(gameServer => Debug.Log($"Server - Watch {gameServer}"));
```


## Player Tracking

{{< alpha title="Player Tracking" gate="PlayerTracking" >}}

To use alpha features use the AgonesAlphaSDK class.

```csharp
var agones = agonesGameObject.GetComponent<Agones.AgonesAlphaSdk>();
```

### Alpha: PlayerConnect

This method increases the SDK’s stored player count by one, and appends this playerID to GameServer.Status.Players.IDs.
Returns true and adds the playerID to the list of playerIDs if the playerIDs was not already in the list of connected playerIDs.

```csharp
bool ok = await agones.PlayerConnect(playerId);
```

### Alpha: PlayerDisconnect

This function decreases the SDK’s stored player count by one, and removes the playerID from GameServer.Status.Players.IDs.
Will return true and remove the supplied playerID from the list of connected playerIDs if the playerID value exists within the list.

```csharp
bool ok = await agones.PlayerDisconnect(playerId);
```

### Alpha: SetPlayerCapacity

Update the `GameServer.Status.Players.Capacity` value with a new capacity.

```csharp
var capacity = 100;
bool ok = await agones.SetPlayerCapacity(capacity);
```

### Alpha: GetPlayerCapacity

This function retrieves the current player capacity `GameServer.Status.Players.Capacity`. 
This is always accurate from what has been set through this SDK, even if the value has yet to be updated on the GameServer status resource.

```csharp
long capacity = await agones.GetPlayerCapacity();
```

### Alpha: GetPlayerCount

Returns the current player count

```csharp
long count = await agones.GetPlayerCount();
```

### Alpha: IsPlayerConnected

This returns if the playerID is currently connected to the GameServer.
This is always accurate, even if the value hasn’t been updated to the GameServer status yet.

```csharp
bool isConnected = await agones.IsPlayerConnected(playerId);
```


### Alpha: GetConnectedPlayers

This returns a list of the playerIDs that are currently connected to the GameServer.

```csharp
List<string> players = await agones.GetConnectedPlayers();
```

{{% alert title="Warning" color="warning" %}}
The following code causes deadlock. Do not use a `Wait` method with the returned Task.
```csharp
void Deadlock()
{
    Task<bool> t = agones.Shutdown();
    t.Wait(); // deadlock!!!
}
```
{{% /alert %}}

## Settings

The properties for the Unity Agones SDK can be found in the Inspector.

- Health Interval Second
  - Interval of the server sending a health ping to the Agones sidecar. (default: `5`)
- Health Enabled
  - Whether the server sends a health ping to the Agones sidecar. (default: `true`)
- Log Enabled
  - Debug Logging Enabled. Debug logging for development of this Plugin. (default: `false`)
