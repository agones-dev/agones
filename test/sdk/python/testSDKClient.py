#!/usr/bin/env python3

# Copyright Contributors to Agones a Series of LF Projects, LLC.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import sys
import threading
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../../..', 'sdks/python'))

from agones import AgonesSDK


def run_player_tracking(alpha):
    print("python: Setting player capacity...")
    alpha.set_player_capacity(10)

    capacity = alpha.get_player_capacity()
    print(f"python: Player capacity: {capacity}")

    player_id = "1234"
    print("python: Increasing the player count...")
    added = alpha.player_connect(player_id)
    if not added:
        raise RuntimeError("Failed to add player")
    print("python: Added player")

    connected = alpha.is_player_connected(player_id)
    if not connected:
        raise RuntimeError(f"{player_id} is not connected")
    print(f"python: {player_id} is connected")

    players = alpha.get_connected_players()
    print(f"python: Connected players: {players}")

    count = alpha.get_player_count()
    print(f"python: Current player count: {count}")

    print("python: Decreasing the player count...")
    removed = alpha.player_disconnect(player_id)
    if not removed:
        raise RuntimeError("Failed to remove player")
    print("python: Removed player")

    count = alpha.get_player_count()
    print(f"python: Current player count: {count}")


def run_counts_and_lists(beta):
    counter = "rooms"
    print("python: Getting Counter count...")
    count = beta.get_counter_count(counter)
    if count != 1:
        raise RuntimeError(f"Counter count should be 1, but is {count}")

    print("python: Incrementing Counter...")
    beta.increment_counter(counter, 9)

    print("python: Decrementing Counter...")
    beta.decrement_counter(counter, 10)

    print("python: Setting Counter count...")
    beta.set_counter_count(counter, 10)

    print("python: Getting Counter capacity...")
    capacity = beta.get_counter_capacity(counter)
    if capacity != 10:
        raise RuntimeError(f"Counter capacity should be 10, but is {capacity}")

    print("python: Setting Counter capacity...")
    beta.set_counter_capacity(counter, 1)

    list_name = "players"
    print("python: Checking List contains...")
    if not beta.list_contains(list_name, "test1"):
        raise RuntimeError('List should contain "test1"')

    print("python: Getting List length...")
    length = beta.get_list_length(list_name)
    if length != 3:
        raise RuntimeError(f"List length should be 3, but is {length}")

    print("python: Getting List values...")
    values = beta.get_list_values(list_name)
    expected = ["test0", "test1", "test2"]
    if values != expected:
        raise RuntimeError(f"List values should be {expected}, but is {values}")

    print("python: Appending List value...")
    beta.append_list_value(list_name, "test3")

    print("python: Deleting List value...")
    beta.delete_list_value(list_name, "test2")

    print("python: Getting List capacity...")
    capacity = beta.get_list_capacity(list_name)
    if capacity != 100:
        raise RuntimeError(f"List capacity should be 100, but is {capacity}")

    print("python: Setting List capacity...")
    beta.set_list_capacity(list_name, 2)


def main():
    print("Python Game Server has started!")

    sdk = AgonesSDK()
    sdk.connect()
    print("python: Connected!")

    uid = {"value": ""}
    got_uid = threading.Event()

    def on_game_server(gs):
        uid["value"] = gs.object_meta.uid
        got_uid.set()

    sdk.watch_game_server(on_game_server)

    got_uid.wait(timeout=10)
    sdk.set_annotation("annotation", uid["value"])

    print("python: Marking server as ready...")
    sdk.ready()
    print("python: ...marked Ready")

    gs = sdk.get_game_server()
    print(f"python: GameServer name: {gs.object_meta.name}")
    sdk.set_label("label", str(gs.object_meta.creation_timestamp))

    sdk.health()

    print("python: Reserving for 5 seconds...")
    sdk.reserve(5)
    print("python: ...Reserved")

    print("python: Allocating...")
    sdk.allocate()
    print("python: ...Allocated")

    feature_gates = os.environ.get("FEATURE_GATES", "")
    if "PlayerTracking=true" in feature_gates:
        run_player_tracking(sdk.alpha)
    if "CountsAndLists=true" in feature_gates:
        run_counts_and_lists(sdk.beta)

    print("python: Shutting down...")
    sdk.shutdown()
    print("python: ...marked for Shutdown")
    print("Python Game Server finished.")


if __name__ == "__main__":
    main()
