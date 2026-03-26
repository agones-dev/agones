---
title: "C++ Game Server Client SDK"
linkTitle: "C++"
date: 2019-01-02T10:17:50Z
weight: 20
description: "This is the C++ version of the Agones Game Server Client SDK. "
---

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.


## SDK Functionality

| Area          | Action             | Implemented |
|---------------|--------------------|-------------|
| Lifecycle     | Ready              | ✔️          |
| Lifecycle     | Health             | ✔️          |
| Lifecycle     | Reserve            | ✔️          |
| Lifecycle     | Allocate           | ✔️          |
| Lifecycle     | Shutdown           | ✔️          |
| Configuration | GameServer         | ✔️          |
| Configuration | WatchGameServer    | ✔️          |
| Metadata      | SetAnnotation      | ✔️          |
| Metadata      | SetLabel           | ✔️          |
| Counters      | GetCounterCount    | ❌           |
| Counters      | SetCounterCount    | ❌           |
| Counters      | IncrementCounter   | ❌           |
| Counters      | DecrementCounter   | ❌           |
| Counters      | SetCounterCapacity | ❌           |
| Counters      | GetCounterCapacity | ❌           |
| Lists         | AppendListValue    | ❌           |
| Lists         | DeleteListValue    | ❌           |
| Lists         | SetListCapacity    | ❌           |
| Lists         | GetListCapacity    | ❌           |
| Lists         | ListContains       | ❌           |
| Lists         | GetListLength      | ❌           |
| Lists         | GetListValues      | ❌           |

## Installation

### Download

Download the source from the [Releases Page](https://github.com/agones-dev/agones/releases) or {{< ghlink href="sdks/cpp" >}}directly from GitHub{{< /ghlink >}}.

### Building the Libraries from source
CMake is used to build SDK for all supported platforms (Linux/Window/macOS).

### Prerequisites
* CMake >= 3.15.0
* Git
* C++17 compiler

### Dependencies

Agones SDK only depends on the [gRPC](https://grpc.io/) library.

{{% alert title="Warning" color="warning" %}}
Prior to Agones release 1.39.0 if the gRPC dependency was not found locally installed, the CMake system would install
the supported gRPC version for you. Unfortunately this process was very brittle and often broke with gRPC updates,
therefore this functionality has been removed, and a manual installation of gRPC is now required.
{{% /alert %}}

This version of the Agones C++ SDK has been tested with gRPC v1.78.1. To install it from source 
[follow the instructions](https://grpc.io/docs/languages/cpp/quickstart/#build-and-install-grpc-and-protocol-buffers).

It may also be available from your system's package manager, but that may not align with the supported gRPC version, so
use at your own risk.



## Linux / MacOS
```bash
mkdir -p .build
cd .build
cmake .. -DCMAKE_BUILD_TYPE=Release -G "Unix Makefiles" -DCMAKE_INSTALL_PREFIX=./install
cmake --build . --target install
```

## Windows
Building with Visual Studio:
```bash
md .build
cd .build
cmake .. -G "Visual Studio 15 2017 Win64" -DCMAKE_INSTALL_PREFIX=./install
cmake --build . --config Release --target install
```
Building with NMake
```bash
md .build
cd .build
cmake .. -G "NMake Makefiles" -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=./install
cmake --build . --target install
```

**CMAKE_INSTALL_PREFIX** may be skipped if it is OK to install Agones SDK to a default location (usually /usr/local or c:/Program Files/Agones).

CMake option `-Wno-dev` is specified to suppress [CMP0048](https://cmake.org/cmake/help/v3.13/policy/CMP0048.html) deprecation warning for gRPC build.


## Usage

### Using SDK
In CMake-based projects it's enough to specify a folder where SDK is installed with `CMAKE_PREFIX_PATH` and use `find_package(agones CONFIG REQUIRED)` command. For example: {{< ghlink href="examples/cpp-simple" >}}cpp-simple{{< / >}}.
It may be useful to disable some [protobuf warnings](https://github.com/protocolbuffers/protobuf/blob/main/cmake/README.md#notes-on-compiler-warnings) in your project.


## Usage

The C++ SDK is specifically designed to be as simple as possible, and deliberately doesn't include any kind
of singleton management, or threading/asynchronous processing to allow developers to manage these aspects as they deem
appropriate for their system.

We may consider these types of features in the future, depending on demand.

To begin working with the SDK, create an instance of it:
```cpp
agones::SDK *sdk = new agones::SDK();
```

To connect to the SDK server, either local or when running on Agones, run the `sdk->Connect()` method.
This will block for up to 30 seconds if the SDK server has not yet started and the connection cannot be made,
and will return `false` if there was an issue connecting.

```cpp
bool ok = sdk->Connect();
```

To send a [health check]({{< relref "_index.md#health" >}}) call `sdk->Health()`. This is a synchronous request that will
return `false` if it has failed in any way. Read [GameServer Health Checking]({{< relref "../health-checking.md" >}}) for more
details on the game server health checking strategy.

```cpp
bool ok = sdk->Health();
```

To mark the game server as [ready to receive player connections]({{< relref "_index.md#ready" >}}), call `sdk->Ready()`.
This will return a grpc::Status object, from which we can call `status.ok()` to determine
if the function completed successfully.

For more information you can also look at the [gRPC Status reference](https://grpc.io/grpc/cpp/classgrpc_1_1_status.html).

```cpp
grpc::Status status = sdk->Ready();
if (!status.ok()) { ... }
```

To mark the game server as [allocated]({{< relref "_index.md#allocate" >}}), call `sdk->Allocate()`.
This will return a grpc::Status object, from which we can call `status.ok()` to determine
if the function completed successfully.

For more information you can also look at the [gRPC Status reference](https://grpc.io/grpc/cpp/classgrpc_1_1_status.html).

```cpp
grpc::Status status = sdk->Allocate();
if (!status.ok()) { ... }
```

To mark the game server as [reserved]({{< relref "_index.md#reserveseconds" >}}), call
`sdk->Reserve(seconds)`. This will return a grpc::Status object, from which we can call `status.ok()` to determine
if the function completed successfully.

For more information you can also look at the [gRPC Status reference](https://grpc.io/grpc/cpp/classgrpc_1_1_status.html).

```cpp
grpc::Status status = sdk->Reserve(std::chrono::seconds(N));
if (!status.ok()) { ... }
```

To mark that the [game session is completed]({{< relref "_index.md#shutdown" >}}) and the game server should be shut down call `sdk->Shutdown()`.
This will return a grpc::Status object, from which we can call `status.ok()` to determine
if the function completed successfully.

For more information you can also look at the [gRPC Status reference](https://grpc.io/grpc/cpp/classgrpc_1_1_status.html).

```cpp
grpc::Status status = sdk->Shutdown();
if (!status.ok()) { ... }
```

To [set a Label]({{< relref "_index.md#setlabelkey-value" >}}) on the backing `GameServer` call
`sdk->SetLabel(key, value)`.
This will return a grpc::Status object, from which we can call `status.ok()` to determine
if the function completed successfully.

For more information you can also look at the [gRPC Status reference](https://grpc.io/grpc/cpp/classgrpc_1_1_status.html).

```cpp
grpc::Status status = sdk->SetLabel("test-label", "test-value");
if (!status.ok()) { ... }
```

To [set an Annotation]({{< relref "_index.md#setannotationkey-value" >}}) on the backing `GameServer` call
`sdk->SetAnnotation(key, value)`.
This will return a grpc::Status object, from which we can call `status.ok()` to determine
if the function completed successfully.

For more information you can also look at the [gRPC Status reference](https://grpc.io/grpc/cpp/classgrpc_1_1_status.html).

```cpp
status = sdk->SetAnnotation("test-annotation", "test value");
if (!status.ok()) { ... }
```

To get the details on the [backing `GameServer`]({{< relref "_index.md#gameserver" >}}) call `sdk->GameServer(&gameserver)`,
passing in a `agones::dev::sdk::GameServer*` to push the results of the `GameServer` configuration into.

This function will return a grpc::Status object, from which we can call `status.ok()` to determine
if the function completed successfully.

```cpp
agones::dev::sdk::GameServer gameserver;
grpc::Status status = sdk->GameServer(&gameserver);
if (!status.ok()) {...}
```

To get [updates on the backing `GameServer`]({{< relref "_index.md#watchgameserverfunctiongameserver" >}}) as they happen,
call `sdk->WatchGameServer([](const agones::dev::sdk::GameServer& gameserver){...})`.

This will call the passed in `std::function`
synchronously (this is a blocking function, so you may want to run it in its own thread) whenever the backing `GameServer`
is updated.

```cpp
sdk->WatchGameServer([](const agones::dev::sdk::GameServer& gameserver){
  std::cout << "GameServer Update:\n"                                 //
            << "\tname: " << gameserver.object_meta().name() << "\n"  //
            << "\tstate: " << gameserver.status().state() << "\n"
            << std::flush;
});
```

## Next Steps

* Read the [SDK Overview]({{< relref "_index.md" >}}) to review all SDK functionality
* Look at the {{< ghlink href="examples/cpp-simple" >}}C++ example for a full build template{{< / >}}.
* Check out {{< ghlink href="sdks/cpp/include/agones/sdk.h" >}}sdk.h{{< /ghlink >}}.
