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

"""Alpha SDK - Player tracking functionality."""

import grpc

from agones._generated.alpha import alpha_pb2, alpha_pb2_grpc


class Alpha:
    """Player tracking API (Alpha feature)."""

    def __init__(self, channel: grpc.Channel):
        self._client = alpha_pb2_grpc.SDKStub(channel)

    def player_connect(self, player_id: str) -> bool:
        """Register a player connection. Returns True if newly added."""
        return self._client.PlayerConnect(alpha_pb2.PlayerID(playerID=player_id)).bool

    def player_disconnect(self, player_id: str) -> bool:
        """Register a player disconnection. Returns True if removed."""
        return self._client.PlayerDisconnect(alpha_pb2.PlayerID(playerID=player_id)).bool

    def set_player_capacity(self, capacity: int) -> None:
        """Set the max player capacity."""
        self._client.SetPlayerCapacity(alpha_pb2.Count(count=capacity))

    def get_player_capacity(self) -> int:
        """Get the max player capacity."""
        return self._client.GetPlayerCapacity(alpha_pb2.Empty()).count

    def get_player_count(self) -> int:
        """Get the current player count."""
        return self._client.GetPlayerCount(alpha_pb2.Empty()).count

    def is_player_connected(self, player_id: str) -> bool:
        """Check if a player is currently connected."""
        return self._client.IsPlayerConnected(alpha_pb2.PlayerID(playerID=player_id)).bool

    def get_connected_players(self) -> list[str]:
        """Get the list of connected player IDs."""
        return list(self._client.GetConnectedPlayers(alpha_pb2.Empty()).list)
