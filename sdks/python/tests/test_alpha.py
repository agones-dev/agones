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

import unittest
from unittest.mock import MagicMock

from agones._generated.alpha import alpha_pb2
from agones.alpha import Alpha


class TestAlpha(unittest.TestCase):

    def setUp(self):
        self.channel = MagicMock()
        self.alpha = Alpha(self.channel)
        self.alpha._client = MagicMock()

    def test_player_connect(self):
        self.alpha._client.PlayerConnect.return_value = alpha_pb2.Bool(bool=True)
        result = self.alpha.player_connect("player-1")
        self.assertTrue(result)
        call_args = self.alpha._client.PlayerConnect.call_args[0][0]
        self.assertEqual(call_args.playerID, "player-1")

    def test_player_disconnect(self):
        self.alpha._client.PlayerDisconnect.return_value = alpha_pb2.Bool(bool=True)
        result = self.alpha.player_disconnect("player-1")
        self.assertTrue(result)

    def test_set_player_capacity(self):
        self.alpha.set_player_capacity(100)
        call_args = self.alpha._client.SetPlayerCapacity.call_args[0][0]
        self.assertEqual(call_args.count, 100)

    def test_get_player_capacity(self):
        self.alpha._client.GetPlayerCapacity.return_value = alpha_pb2.Count(count=64)
        self.assertEqual(self.alpha.get_player_capacity(), 64)

    def test_get_player_count(self):
        self.alpha._client.GetPlayerCount.return_value = alpha_pb2.Count(count=10)
        self.assertEqual(self.alpha.get_player_count(), 10)

    def test_is_player_connected(self):
        self.alpha._client.IsPlayerConnected.return_value = alpha_pb2.Bool(bool=False)
        self.assertFalse(self.alpha.is_player_connected("unknown"))

    def test_get_connected_players(self):
        self.alpha._client.GetConnectedPlayers.return_value = alpha_pb2.PlayerIDList(list=["a", "b"])
        result = self.alpha.get_connected_players()
        self.assertEqual(result, ["a", "b"])


if __name__ == "__main__":
    unittest.main()
