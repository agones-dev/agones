---
title: "Unreal Engine Game Server Client Plugin"
linkTitle: "Unreal Engine"
date: 2019-06-13T10:17:50Z
publishDate: 2019-05-13
weight: 10
description: "This is the Unreal Engine Agones Game Server Client Plugin. "
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
| Counters             | GetCounterCount          | ✔️                            |
| Counters             | SetCounterCount          | ✔️                            |
| Counters             | IncrementCounter         | ✔️                            |
| Counters             | DecrementCounter         | ✔️                            |
| Counters             | SetCounterCapacity       | ✔️                            |
| Counters             | GetCounterCapacity       | ✔️                            |
| Lists                | GetList                  | ✔️                            |
| Lists                | UpdateList               | ✔️                            |
| Lists                | AddListValue             | ✔️                            |
| Lists                | RemoveListValue          | ✔️                            |
| Player Tracking      | GetConnectedPlayers      | ✔️                            |
| Player Tracking      | GetPlayerCapacity        | ✔️                            |
| Player Tracking      | GetPlayerCount           | ✔️                            |
| Player Tracking      | IsPlayerConnected        | ✔️                            |
| Player Tracking      | PlayerConnect            | ✔️                            |
| Player Tracking      | PlayerDisconnect         | ✔️                            |
| Player Tracking      | SetPlayerCapacity        | ✔️                            |

Additional methods have been added for ease of use (both of which are enabled by default):

- Connect
  - will call `/gameserver` till a succesful response is returned and then call `/ready`.
  - disabled by setting `bDisableAutoConnect` to `true`.
  - An event is broadcast with the `GameServer` data once the `/gameserver` call succeeds.
- Health
  - calls `/health` endpoint on supplied rate
  - enabled by default with 10 second rate
  - disabled by default by setting `HealthRateSeconds` to `0`.

Both of the above are automatically kicked off in the `BeginPlay` of the component.

## Download

Download the source from the [Releases Page](https://github.com/agones-dev/agones/releases)
or {{< ghlink href="sdks/unreal" >}}directly from GitHub{{< /ghlink >}}.

## Resources

<a href="https://www.unrealengine.com/" data-proofer-ignore>Unreal</a>
is a [game engine](https://en.wikipedia.org/wiki/Game_engine) that is used by
anyone from hobbyists all the way through to huge AAA Game Studios.

With this in mind there is a vast amount to learn to run a production game using Unreal, even before you get to learning
how it integrates with Agones. If you want to kick the tires with a starter project you will probably be fine with one
of the starter projects out of the box.

However, as your Unreal/Agones project gets more advanced you will want to understand more about the engine itself and
how it can be used to integrate with this project. There will be different ways of interacting via in Play In Editor (
PIE) versus running as an actual dedicated game server packaged into a container.

There are few helpful links for latest Unreal Engine 5:
- [UE5 Documentation Site](https://docs.unrealengine.com/en-US/)
- [UE5 Dedicated Servers](https://docs.unrealengine.com/en-US/setting-up-dedicated-servers-in-unreal-engine/)
  - useful guide to getting started with dedicated servers in Unreal
- [UE5 Game Mode and Game State](https://docs.unrealengine.com/en-US/game-mode-and-game-state-in-unreal-engine/)
- [UE5 Game Mode API Reference](https://docs.unrealengine.com/en-US/API/Runtime/Engine/GameFramework/AGameMode/)
  - useful for hooking up calls to Agones
- [UE5 Game Session API Reference](https://docs.unrealengine.com/en-US/API/Runtime/Engine/GameFramework/AGameSession/)
  - as above there are hooks in Game Session that can be used to call into Agones
- [UE5 Building & Packaging Games](https://docs.unrealengine.com/en-US/build-operations-cooking-packaging-deploying-and-running-projects-in-unreal-engine/)
  - only building out Unreal game servers / clients, will also need to package into a container

If you use Unreal Engine 4, There are few helpful links for it:
- [UE4 Documentation Site](https://docs.unrealengine.com/4.27/en-US/index.html)
- [UE4 Dedicated Servers](https://docs.unrealengine.com/4.27/en-US/Gameplay/Networking/HowTo/DedicatedServers/index.html)
- [UE4 Game Flow](https://docs.unrealengine.com/4.27/en-US/InteractiveExperiences/Framework/GameFlow/)
- [UE4 Game Mode](https://docs.unrealengine.com/4.27/en-US/API/Runtime/Engine/GameFramework/AGameMode/index.html)
- [UE4 Game Session](https://docs.unrealengine.com/4.27/en-US/API/Runtime/Engine/GameFramework/AGameSession/index.html)
- [UE4 Building & Packaging Games](https://docs.unrealengine.com/4.27/en-US/Engine/Deployment/BuildOperations/index.html)

## Getting Started

This is a SDK inspired by the REST API to the Agones sidecars that allows engineers to communicate with the sidecar from either C++ or Blueprints.

### Getting the Code

Easiest way to get this code is to clone the repository and drop the entire plugin folder into your own `Plugins` folder. This runs the plugin as a Project plugin rather than an engine plugin.

We could however turn this into a marketplace plugin that can be retrived from the marketplace directly into the UE editor.

#### Using C++ (UE5/UE4)
- Add Plugin (in your own `.uproject` file)
```json
  "Plugins": [
    {
      "Enabled": true,
      "Name": "Agones"
    }
  ],
```
- Add Plugin (in your own `*.Build.cs`)
```json
PublicDependencyModuleNames.AddRange(
    new[]
    {
        "Agones",
    });
```
- Add subsystem in header
```c++
#include "AgonesSubsystem.h"
```

- Use the Agones subsystem to call PlayerReady
```c++
void APlatformGameSession::PostLogin(APlayerController* NewPlayer)
{
  UAgonesSubsystem* AgonesSDK = UAgonesSubsystem::Get(this);
  if (AgonesSDK) // Check for nullptr is a must.
  {
    // Empty brances are for callbacks on success and errror.
    AgonesSDK->PlayerConnect("netspeak-player", {}, {});
  }
}
```

#### Using Blueprints
- Accessing Agones functionality can be done via Blueprints.
![usage](../../../../images/unreal_bp_usage.png)

## Configuration Options

A number of options can be altered via config files in Unreal these are supplied via `Game` configuration eg. `DefaultGame.ini`.

```
[/Script/Agones.AgonesComponent]
HttpPort=1337
HealthRateSeconds=5.0
bDisableAutoHealthPing=false
bDisableAutoConnect=true
```

## Unreal Hooks

Within the Unreal [GameMode](https://docs.unrealengine.com/en-US/API/Runtime/Engine/GameFramework/AGameMode/index.html) and [GameSession](https://docs.unrealengine.com/en-US/API/Runtime/Engine/GameFramework/AGameSession/index.html) exist a number of useful existing
funtions that can be used to fit in with making calls out to Agones.

A few examples are:
- `RegisterServer` to call `SetLabel`, `SetPlayerCapacity`
- `PostLogin` to call `PlayerConnect`
- `NotifyLogout` to call `PlayerDisconnect`
