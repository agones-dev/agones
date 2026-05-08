// Copyright Contributors to Agones a Series of LF Projects, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

use std::{env, thread, time::Duration};

#[tokio::main(flavor = "multi_thread", worker_threads = 4)]
async fn main() {
    println!("Rust Game Server has started!");

    ::std::process::exit(match run().await {
        Ok(_) => {
            println!("Rust Game Server finished.");
            0
        }
        Err(msg) => {
            println!("rust: {}", msg);
            1
        }
    });
}

async fn run() -> Result<(), String> {
    let env_run_async = env::var("RUN_ASYNC").unwrap_or_default();
    if env_run_async.contains("true") {
        println!("rust: RUN_ASYNC is set to true, so run test for async functions");
        run_async().await
    } else {
        tokio::task::block_in_place(run_sync)
    }
}

fn run_sync() -> Result<(), String> {
    use tokio::runtime::Handle;

    println!("rust: Creating SDK instance");
    let mut sdk = Handle::current().block_on(async move {
        agones::Sdk::new(None /* default port */, None /* keep_alive */)
            .await
            .map_err(|e| format!("unable to create sdk client: {}", e))
    })?;

    // Spawn a task that will send health checks every 2 seconds. If this current
    // thread/task panics or dropped, the health check will also be stopped
    let _health = {
        let health_tx = sdk.health_check();
        let (tx, mut rx) = tokio::sync::oneshot::channel::<()>();

        Handle::current().spawn(async move {
            let mut interval = tokio::time::interval(Duration::from_secs(2));

            loop {
                tokio::select! {
                    _ = interval.tick() => {
                        if health_tx
                            .send(())
                            .await.is_err() {
                            eprintln!("Health check receiver was dropped");
                            break;
                        }
                    }
                    _ = &mut rx => {
                        println!("Health check task canceled");
                        break;
                    }
                }
            }
        });

        tx
    };

    let _watch = {
        let mut watch_client = sdk.clone();
        let (tx, mut rx) = tokio::sync::oneshot::channel::<()>();

        tokio::task::spawn(async move {
            println!("rust: Starting to watch GameServer updates...");
            let mut once = true;
            match watch_client.watch_gameserver().await {
                Err(e) => eprintln!("rust: Failed to watch for GameServer updates: {}", e),
                Ok(mut stream) => loop {
                    tokio::select! {
                        gs = stream.message() => {
                            match gs {
                                Ok(Some(gs)) => {
                                    let om = gs.object_meta.unwrap();
                                    println!("rust: GameServer Update, name: {}", om.name);
                                    println!("rust: GameServer Update, state: {}", gs.status.unwrap().state);

                                    if once {
                                        println!("rust: Setting an annotation");
                                        let uid = om.uid.clone();

                                        if let Err(e) = watch_client.set_annotation("test-annotation", uid).await {
                                            eprintln!("rust: Failed to set annotation from watch task: {}", e);
                                        }

                                        once = false;
                                    }
                                }
                                Ok(None) => {
                                    println!("rust: Server closed the GameServer watch stream");
                                    break;
                                }
                                Err(e) => {
                                    eprintln!("rust: GameServer Update stream encountered an error: {}", e);
                                }
                            }

                        }
                        _ = &mut rx => {
                            println!("rust: Shutting down GameServer watch loop");
                            break;
                        }
                    }
                },
            }
        });

        tx
    };

    // Waiting for a thread to spawn
    thread::sleep(Duration::from_secs(2));

    println!("rust: Marking server as ready...");
    Handle::current().block_on(async {
        sdk.ready()
            .await
            .map_err(|e| format!("Could not run Ready(): {}. Exiting!", e))
    })?;

    println!("rust: ...marked Ready");

    println!("rust: Reserving for 5 seconds");
    Handle::current().block_on(async {
        sdk.reserve(Duration::from_secs(5))
            .await
            .map_err(|e| format!("Could not run Reserve(): {}. Exiting!", e))
    })?;
    println!("rust: ...Reserved");

    println!("rust: Allocate game server ...");
    Handle::current().block_on(async {
        sdk.allocate()
            .await
            .map_err(|e| format!("Could not run Allocate(): {}. Exiting!", e))
    })?;

    println!("rust: ...marked Allocated");

    println!("rust: Getting GameServer details...");
    let gameserver = Handle::current().block_on(async {
        sdk.get_gameserver()
            .await
            .map_err(|e| format!("Could not run GameServer(): {}. Exiting!", e))
    })?;

    println!(
        "rust: GameServer name: {}",
        gameserver.object_meta.clone().unwrap().name
    );

    println!("rust: Setting a label");
    let creation_ts = gameserver.object_meta.unwrap().creation_timestamp;
    Handle::current().block_on(async {
        sdk.set_label("test-label", &creation_ts.to_string())
            .await
            .map_err(|e| format!("Could not run SetLabel(): {}. Exiting!", e))
    })?;

    let feature_gates = env::var("FEATURE_GATES").unwrap_or_default();
    if feature_gates.contains("PlayerTracking=true") {
        run_player_tracking_features(sdk.alpha().clone())?;
    }
    if feature_gates.contains("CountsAndLists=true") {
        run_counts_and_lists_features(sdk.beta().clone())?;
    }

    for i in 0..1 {
        let time = i * 5;
        println!("rust: Running for {} seconds", time);

        thread::sleep(Duration::from_secs(5));
    }

    println!("rust: Shutting down...");
    Handle::current().block_on(async {
        sdk.shutdown()
            .await
            .map_err(|e| format!("Could not run Shutdown: {}. Exiting!", e))
    })?;
    println!("rust: ...marked for Shutdown");

    Ok(())
}

fn run_player_tracking_features(mut alpha: agones::alpha::Alpha) -> Result<(), String> {
    use tokio::runtime::Handle;

    println!("rust: Setting player capacity...");
    Handle::current().block_on(async {
        alpha
            .set_player_capacity(10)
            .await
            .map_err(|e| format!("Could not run SetPlayerCapacity(): {:#?}. Exiting!", e))
    })?;

    println!("rust: Getting player capacity...");
    let capacity = Handle::current().block_on(async {
        alpha
            .get_player_capacity()
            .await
            .map_err(|e| format!("Could not run GetPlayerCapacity(): {}. Exiting!", e))
    })?;
    println!("rust: Player capacity: {}", capacity);

    println!("rust: Increasing the player count...");
    let player_id = "1234".to_string();

    let added = Handle::current().block_on(async {
        alpha
            .player_connect(&player_id)
            .await
            .map_err(|e| format!("Could not run PlayerConnect(): {}. Exiting!", e))
    })?;
    if added {
        println!("rust: Added player");
    } else {
        panic!("rust: Failed to add player. Exiting!");
    }

    let connected = Handle::current().block_on(async {
        alpha
            .is_player_connected(&player_id)
            .await
            .map_err(|e| format!("Could not run IsPlayerConnected(): {}. Exiting!", e))
    })?;
    if connected {
        println!("rust: {} is connected", player_id);
    } else {
        panic!("rust: {} is not connected. Exiting!", player_id);
    }

    let player_ids = Handle::current().block_on(async {
        alpha
            .get_connected_players()
            .await
            .map_err(|e| format!("Could not run GetConnectedPlayers(): {}. Exiting!", e))
    })?;
    println!("rust: Connected players: {:?}", player_ids);

    let player_count = Handle::current().block_on(async {
        alpha
            .get_player_count()
            .await
            .map_err(|e| format!("Could not run GetConnectedPlayers(): {}. Exiting!", e))
    })?;
    println!("rust: Current player count: {}", player_count);

    println!("rust: Decreasing the player count...");
    let removed = Handle::current().block_on(async {
        alpha
            .player_disconnect(&player_id)
            .await
            .map_err(|e| format!("Could not run PlayerDisconnect(): {}. Exiting!", e))
    })?;
    if removed {
        println!("rust: Removed player");
    } else {
        panic!("rust: Failed to remove player. Exiting!");
    }

    let player_count = Handle::current().block_on(async {
        alpha
            .get_player_count()
            .await
            .map_err(|e| format!("Could not GetPlayerCount(): {}. Exiting!", e))
    })?;
    println!("rust: Current player count: {}", player_count);

    Ok(())
}

fn run_counts_and_lists_features(mut beta: agones::beta::Beta) -> Result<(), String> {
    use tokio::runtime::Handle;

    // Counter tests
    let counter = "rooms";
    println!("rust: Getting Counter count...");
    let count = Handle::current().block_on(async {
        beta.get_counter_count(counter)
            .await
            .map_err(|e| format!("Could not run GetCounterCount(): {}. Exiting!", e))
    })?;
    if count != 1 {
        return Err(format!("Counter count should be 1, but is {}", count));
    }

    println!("rust: Incrementing Counter...");
    Handle::current().block_on(async {
        beta.increment_counter(counter, 9)
            .await
            .map_err(|e| format!("Could not run IncrementCounter(): {}. Exiting!", e))
    })?;

    println!("rust: Decrementing Counter...");
    Handle::current().block_on(async {
        beta.decrement_counter(counter, 10)
            .await
            .map_err(|e| format!("Could not run DecrementCounter(): {}. Exiting!", e))
    })?;

    println!("rust: Setting Counter count...");
    Handle::current().block_on(async {
        beta.set_counter_count(counter, 10)
            .await
            .map_err(|e| format!("Could not run SetCounterCount(): {}. Exiting!", e))
    })?;

    println!("rust: Getting Counter capacity...");
    let capacity = Handle::current().block_on(async {
        beta.get_counter_capacity(counter)
            .await
            .map_err(|e| format!("Could not run GetCounterCapacity(): {}. Exiting!", e))
    })?;
    if capacity != 10 {
        return Err(format!("Counter capacity should be 10, but is {}", capacity));
    }

    println!("rust: Setting Counter capacity...");
    Handle::current().block_on(async {
        beta.set_counter_capacity(counter, 1)
            .await
            .map_err(|e| format!("Could not run SetCounterCapacity(): {}. Exiting!", e))
    })?;

    // List tests
    let list = "players";
    let vals = vec!["test0".to_string(), "test1".to_string(), "test2".to_string()];

    println!("rust: Checking if List contains 'test1'...");
    let contains = Handle::current().block_on(async {
        beta.list_contains(list, "test1")
            .await
            .map_err(|e| format!("Could not run ListContains(): {}. Exiting!", e))
    })?;
    if !contains {
        return Err("List should contain value \"test1\"".to_string());
    }

    println!("rust: Getting List length...");
    let length = Handle::current().block_on(async {
        beta.get_list_length(list)
            .await
            .map_err(|e| format!("Could not run GetListLength(): {}. Exiting!", e))
    })?;
    if length != 3 {
        return Err(format!("List length should be 3, but is {}", length));
    }

    println!("rust: Getting List values...");
    let values = Handle::current().block_on(async {
        beta.get_list_values(list)
            .await
            .map_err(|e| format!("Could not run GetListValues(): {}. Exiting!", e))
    })?;
    if values != vals {
        return Err(format!("List values should be {:?}, but is {:?}", vals, values));
    }

    println!("rust: Appending value 'test3' to List...");
    Handle::current().block_on(async {
        beta.append_list_value(list, "test3")
            .await
            .map_err(|e| format!("Could not run AppendListValue(): {}. Exiting!", e))
    })?;

    println!("rust: Deleting value 'test2' from List...");
    Handle::current().block_on(async {
        beta.delete_list_value(list, "test2")
            .await
            .map_err(|e| format!("Could not run DeleteListValue(): {}. Exiting!", e))
    })?;

    println!("rust: Getting List capacity...");

    Ok(())
}


async fn run_async() -> Result<(), String> {
    let mut sdk = match tokio::time::timeout(
        Duration::from_secs(30),
        agones::Sdk::new(None /* default port */, None /* keep_alive */),
    )
    .await
    {
        Ok(sdk) => sdk.map_err(|e| format!("unable to create sdk client: {}", e))?,
        Err(_) => return Err("timed out attempting to connect to the sidecar".to_owned()),
    };

    let _health = {
        let health_tx = sdk.health_check();
        let (tx, mut rx) = tokio::sync::oneshot::channel::<()>();

        tokio::task::spawn(async move {
            let mut interval = tokio::time::interval(Duration::from_secs(2));

            loop {
                tokio::select! {
                    _ = interval.tick() => {
                        if health_tx
                            .send(())
                            .await.is_err() {
                            eprintln!("Health check receiver was dropped");
                            break;
                        }
                    }
                    _ = &mut rx => {
                        println!("Health check task canceled");
                        break;
                    }
                }
            }
        });

        tx
    };

    let _watch = {
        let mut watch_client = sdk.clone();
        let (tx, mut rx) = tokio::sync::oneshot::channel::<()>();

        tokio::task::spawn(async move {
            println!("rust_async: Starting to watch GameServer updates...");
            let mut once = true;
            match watch_client.watch_gameserver().await {
                Err(e) => eprintln!("rust_async: Failed to watch for GameServer updates: {}", e),
                Ok(mut stream) => loop {
                    tokio::select! {
                        gs = stream.message() => {
                            match gs {
                                Ok(Some(gs)) => {
                                    let om = gs.object_meta.unwrap();
                                    println!("rust_async: GameServer Update, name: {}", om.name);
                                    println!("rust_async: GameServer Update, state: {}", gs.status.unwrap().state);

                                    if once {
                                        println!("rust_async: Setting an annotation");
                                        let uid = om.uid.clone();

                                        if let Err(e) = watch_client.set_annotation("test-annotation", uid).await {
                                            eprintln!("rust_async: Failed to set annotation from watch task: {}", e);
                                        }

                                        once = false;
                                    }
                                }
                                Ok(None) => {
                                    println!("rust_async: Server closed the GameServer watch stream");
                                    break;
                                }
                                Err(e) => {
                                    eprintln!("rust_async: GameServer Update stream encountered an error: {}", e);
                                }
                            }

                        }
                        _ = &mut rx => {
                            println!("rust_async: Shutting down GameServer watch loop");
                            break;
                        }
                    }
                },
            }
        });

        tx
    };

    tokio::time::sleep(Duration::from_secs(2)).await;

    println!("rust_async: Marking server as ready...");
    sdk.ready()
        .await
        .map_err(|e| format!("Could not run Ready(): {}. Exiting!", e))?;
    println!("rust_async: ...marked Ready");

    println!("rust_async: Reserving for 5 seconds");
    sdk.reserve(Duration::new(5, 0))
        .await
        .map_err(|e| format!("Could not run Reserve(): {}. Exiting!", e))?;
    println!("rust_async: ...Reserved");

    println!("rust_async: Allocate game server ...");
    sdk.allocate()
        .await
        .map_err(|e| format!("Could not run Allocate(): {}. Exiting!", e))?;

    println!("rust_async: ...marked Allocated");

    println!("rust_async: Getting GameServer details...");
    let gameserver = sdk
        .get_gameserver()
        .await
        .map_err(|e| format!("Could not run GameServer(): {}. Exiting!", e))?;

    println!(
        "rust_async: GameServer name: {}",
        gameserver.object_meta.clone().unwrap().name
    );

    println!("rust_async: Setting a label");
    let creation_ts = gameserver.object_meta.clone().unwrap().creation_timestamp;
    sdk.set_label("test-label", &creation_ts.to_string())
        .await
        .map_err(|e| format!("Could not run SetLabel(): {}. Exiting!", e))?;

    let feature_gates = env::var("FEATURE_GATES").unwrap_or_default();
    if feature_gates.contains("PlayerTracking=true") {
        run_player_tracking_features_async(sdk.alpha().clone()).await?;
    }
    if feature_gates.contains("CountsAndLists=true") {
        run_counts_and_lists_features_async(sdk.beta().clone()).await?;
    }

    for i in 0..1 {
        let time = i * 5;
        println!("rust_async: Running for {} seconds", time);

        tokio::time::sleep(Duration::from_secs(5)).await;
    }

    println!("rust_async: Shutting down...");
    sdk.shutdown()
        .await
        .map_err(|e| format!("Could not run Shutdown: {}. Exiting!", e))?;
    println!("rust_async: ...marked for Shutdown");

    Ok(())
}

async fn run_player_tracking_features_async(mut alpha: agones::alpha::Alpha) -> Result<(), String> {
    println!("rust_async: Setting player capacity...");
    alpha
        .set_player_capacity(10)
        .await
        .map_err(|e| format!("Could not run SetPlayerCapacity(): {}. Exiting!", e))?;

    println!("rust_async: Getting player capacity...");
    let capacity = alpha
        .get_player_capacity()
        .await
        .map_err(|e| format!("Could not run GetPlayerCapacity(): {}. Exiting!", e))?;
    println!("rust_async: Player capacity: {}", capacity);

    println!("rust_async: Increasing the player count...");
    let player_id = "1234".to_string();
    let added = alpha
        .player_connect(&player_id)
        .await
        .map_err(|e| format!("Could not run PlayerConnect(): {}. Exiting!", e))?;
    if added {
        println!("Added player");
    } else {
        panic!("rust_async: Failed to add player. Exiting!");
    }

    let connected = alpha
        .is_player_connected(&player_id)
        .await
        .map_err(|e| format!("Could not run IsPlayerConnected(): {}. Exiting!", e))?;
    if connected {
        println!("rust_async: {} is connected", player_id);
    } else {
        panic!("rust_async: {} is not connected. Exiting!", player_id);
    }

    let player_ids = alpha
        .get_connected_players()
        .await
        .map_err(|e| format!("Could not run GetConnectedPlayers(): {}. Exiting!", e))?;
    println!("rust_async: Connected players: {:?}", player_ids);

    let player_count = alpha
        .get_player_count()
        .await
        .map_err(|e| format!("Could not run GetConnectedPlayers(): {}. Exiting!", e))?;
    println!("rust_async: Current player count: {}", player_count);

    println!("rust_async: Decreasing the player count...");
    let removed = alpha
        .player_disconnect(&player_id)
        .await
        .map_err(|e| format!("Could not run PlayerDisconnect(): {}. Exiting!", e))?;
    if removed {
        println!("rust_async: Removed player");
    } else {
        panic!("rust_async: Failed to remove player. Exiting!");
    }

    let player_count = alpha
        .get_player_count()
        .await
        .map_err(|e| format!("Could not GetPlayerCount(): {}. Exiting!", e))?;
    println!("rust_async: Current player count: {}", player_count);

    Ok(())
}

async fn run_counts_and_lists_features_async(mut beta: agones::beta::Beta) -> Result<(), String> {
    // Counter tests
    let counter = "rooms";
    println!("rust_async: Getting Counter count...");
    let count = beta.get_counter_count(counter)
        .await
        .map_err(|e| format!("Could not run GetCounterCount(): {}. Exiting!", e))?;
    if count != 1 {
        return Err(format!("Counter count should be 1, but is {}", count));
    }

    println!("rust_async: Incrementing Counter...");
    beta.increment_counter(counter, 9)
        .await
        .map_err(|e| format!("Could not run IncrementCounter(): {}. Exiting!", e))?;

    println!("rust_async: Decrementing Counter...");
    beta.decrement_counter(counter, 10)
        .await
        .map_err(|e| format!("Could not run DecrementCounter(): {}. Exiting!", e))?;

    println!("rust_async: Setting Counter count...");
    beta.set_counter_count(counter, 10)
        .await
        .map_err(|e| format!("Could not run SetCounterCount(): {}. Exiting!", e))?;

    println!("rust_async: Getting Counter capacity...");
    let capacity = beta.get_counter_capacity(counter)
        .await
        .map_err(|e| format!("Could not run GetCounterCapacity(): {}. Exiting!", e))?;
    if capacity != 10 {
        return Err(format!("Counter capacity should be 10, but is {}", capacity));
    }

    println!("rust_async: Setting Counter capacity...");
    beta.set_counter_capacity(counter, 1)
        .await
        .map_err(|e| format!("Could not run SetCounterCapacity(): {}. Exiting!", e))?;

    // List tests
    let list = "players";
    let vals = vec!["test0".to_string(), "test1".to_string(), "test2".to_string()];

    println!("rust_async: Checking if List contains 'test1'...");
    let contains = beta.list_contains(list, "test1")
        .await
        .map_err(|e| format!("Could not run ListContains(): {}. Exiting!", e))?;
    if !contains {
        return Err("List should contain value \"test1\"".to_string());
    }

    println!("rust_async: Getting List length...");
    let length = beta.get_list_length(list)
        .await
        .map_err(|e| format!("Could not run GetListLength(): {}. Exiting!", e))?;
    if length != 3 {
        return Err(format!("List length should be 3, but is {}", length));
    }

    println!("rust_async: Getting List values...");
    let values = beta.get_list_values(list)
        .await
        .map_err(|e| format!("Could not run GetListValues(): {}. Exiting!", e))?;
    if values != vals {
        return Err(format!("List values should be {:?}, but is {:?}", vals, values));
    }

    println!("rust_async: Appending value 'test3' to List...");
    beta.append_list_value(list, "test3")
        .await
        .map_err(|e| format!("Could not run AppendListValue(): {}. Exiting!", e))?;

    println!("rust_async: Deleting value 'test2' from List...");
    beta.delete_list_value(list, "test2")
        .await
        .map_err(|e| format!("Could not run DeleteListValue(): {}. Exiting!", e))?;

    println!("rust_async: Getting List capacity...");
    let list_capacity = beta.get_list_capacity(list)
        .await
        .map_err(|e| format!("Could not run GetListCapacity(): {}. Exiting!", e))?;
    if list_capacity != 100 {
        return Err(format!("List capacity should be 100, but is {}", list_capacity));
    }

    println!("rust_async: Setting List capacity to 2...");
    beta.set_list_capacity(list, 2)
        .await
        .map_err(|e| format!("Could not run SetListCapacity(): {}. Exiting!", e))?;

    Ok(())
}