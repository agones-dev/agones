---
title: "Rust Game Server Client SDK"
linkTitle: "Rust"
date: 2019-01-02T10:17:57Z
weight: 50
description: "This is the Rust version of the Agones Game Server Client SDK."
---

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.

## SDK Functionality

{{% feature publishVersion="1.52.0" %}}
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
{{% /feature %}}

{{% feature expiryVersion="1.52.0" %}}
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
| Counters        | GetCounterCount     | ❌         |
| Counters        | SetCounterCount     | ❌         |
| Counters        | IncrementCounter    | ❌         |
| Counters        | DecrementCounter    | ❌         |
| Counters        | SetCounterCapacity  | ❌         |
| Counters        | GetCounterCapacity  | ❌         |
| Lists           | AppendListValue     | ❌         |
| Lists           | DeleteListValue     | ❌         |
| Lists           | SetListCapacity     | ❌         |
| Lists           | GetListCapacity     | ❌         |
| Lists           | ListContains        | ❌         |
| Lists           | GetListLength       | ❌         |
| Lists           | GetListValues       | ❌         |
{{% /feature %}}

## Prerequisites

- [Rust >= 1.50](https://www.rust-lang.org/tools/install)

## Usage

Add <a href="https://crates.io/crates/agones" data-proofer-ignore>this crate</a> to `dependencies` section in your Cargo.toml.

Also note that the SDK is [`async`](https://doc.rust-lang.org/std/keyword.async.html) only, so you will need an async runtime to execute the futures exposed by the SDK. It is recommended to use [tokio](https://docs.rs/tokio) as the SDK already depends on tokio due to its choice of gRPC library, [tonic](https://docs.rs/tonic).

```toml
[dependencies]
agones = "1.34.0"
tokio = { version = "1.32.0", features = ["macros", "sync"] }
```

To begin working with the SDK, create an instance of it.

```rust
use std::time::Duration;

#[tokio::main]
async fn main() {
    let mut sdk = agones::Sdk::new(None /* default port */, None /* keep_alive */)
        .await
        .expect("failed to connect to SDK server");
}
```

To send [health checks]({{< relref "_index.md#health" >}}), call `sdk.health_check`, which will return a [`tokio::sync::mpsc::Sender::<()>`](https://docs.rs/tokio/1.7.0/tokio/sync/mpsc/struct.Sender.html) which will send a health check every time a message is posted to the channel.

```rust
let health = sdk.health_check();
if health.send(()).await.is_err() {
    eprintln!("the health receiver was closed");
}
```

To mark the [game session as ready]({{< relref "_index.md#ready" >}}) call `sdk.ready()`.

```rust
sdk.ready().await?;
```

To mark the game server as [reserved]({{< relref "_index.md#reserveseconds" >}}) for a period of time, call `sdk.reserve(duration)`.

```rust
sdk.reserve(Duration::new(5, 0)).await?;
```

To mark that the [game session is completed]({{< relref "_index.md#shutdown" >}}) and the game server should be shut down call `sdk.shutdown()`.

```rust
if let Err(e) = sdk.shutdown().await {
    eprintln!("Could not run Shutdown: {}", e);
}
```

To [set a Label]({{< relref "_index.md#setlabelkey-value" >}}) on the backing `GameServer` call `sdk.set_label(key, value)`.

```rust
sdk.set_label("test-label", "test-value").await?;
```

To [set an Annotation]({{< relref "_index.md#setannotationkey-value" >}}) on the backing `GameServer` call `sdk.set_annotation(key, value)`.

```rust
sdk.set_annotation("test-annotation", "test value").await?;
```

To get [details of the backing `GameServer`]({{< relref "_index.md#gameserver" >}}) call `sdk.get_gameserver()`.

The function will return an instance of `agones::types::GameServer` including `GameServer` configuration info.

```rust
let gameserver = sdk.get_gameserver().await?;
```

To get [updates on the backing `GameServer`]({{< relref "_index.md#watchgameserverfunctiongameserver" >}}) as they happen, call `sdk.watch_gameserver`.

This will stream updates and endlessly until the stream is closed, so it is recommended to push this into its own async task.

```rust
let _watch = {
    // We need to clone the SDK as we are moving it to another task
    let mut watch_client = sdk.clone();
    // We use a simple oneshot to signal to the task when we want it to shutdown
    // and stop watching the gameserver update stream
    let (tx, mut rx) = tokio::sync::oneshot::channel::<()>();

    tokio::task::spawn(async move {
        println!("Starting to watch GameServer updates...");
        match watch_client.watch_gameserver().await {
            Err(e) => eprintln!("Failed to watch for GameServer updates: {}", e),
            Ok(mut stream) => loop {
                tokio::select! {
                    // We've received a new update, or the stream is shutting down
                    gs = stream.message() => {
                        match gs {
                            Ok(Some(gs)) => {
                                println!("GameServer Update, name: {}", gs.object_meta.unwrap().name);
                                println!("GameServer Update, state: {}", gs.status.unwrap().state);
                            }
                            Ok(None) => {
                                println!("Server closed the GameServer watch stream");
                                break;
                            }
                            Err(e) => {
                                eprintln!("GameServer Update stream encountered an error: {}", e);
                            }
                        }

                    }
                    // The watch is being dropped so we're going to shutdown the task
                    // and the watch stream
                    _ = &mut rx => {
                        println!("Shutting down GameServer watch loop");
                        break;
                    }
                }
            },
        }
    });

    tx
};
```

For more information, please read the [SDK Overview]({{< relref "_index.md" >}}), check out {{< ghlink href="sdks/rust/src/sdk.rs" >}}agones sdk implementation{{< /ghlink >}} and also look at the {{< ghlink href="examples/rust-simple" >}}Rust example{{< / >}}.
