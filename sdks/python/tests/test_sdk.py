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
import threading
import unittest
from unittest.mock import MagicMock, patch

from agones._generated import sdk_pb2
from agones.sdk import AgonesSDK


class TestAgonesSDK(unittest.TestCase):

    def setUp(self):
        self.sdk = AgonesSDK()
        self.sdk._client = MagicMock()
        self.sdk._alpha = MagicMock()
        self.sdk._beta = MagicMock()

    def test_default_host_and_port(self):
        sdk = AgonesSDK()
        self.assertEqual(sdk._host, "localhost")
        self.assertEqual(sdk._port, 9357)

    def test_custom_host_and_port(self):
        sdk = AgonesSDK(host="10.0.0.1", port=8080)
        self.assertEqual(sdk._host, "10.0.0.1")
        self.assertEqual(sdk._port, 8080)

    def test_env_var_config(self):
        with patch.dict(os.environ, {"AGONES_SDK_GRPC_HOST": "sidecar", "AGONES_SDK_GRPC_PORT": "1234"}):
            sdk = AgonesSDK()
            self.assertEqual(sdk._host, "sidecar")
            self.assertEqual(sdk._port, 1234)

    def test_ready(self):
        self.sdk.ready()
        self.sdk._client.Ready.assert_called_once()

    def test_allocate(self):
        self.sdk.allocate()
        self.sdk._client.Allocate.assert_called_once()

    def test_shutdown(self):
        self.sdk.shutdown()
        self.sdk._client.Shutdown.assert_called_once()

    def test_reserve(self):
        self.sdk.reserve(10)
        call_args = self.sdk._client.Reserve.call_args[0][0]
        self.assertEqual(call_args.seconds, 10)

    def test_get_game_server(self):
        gs = sdk_pb2.GameServer()
        gs.status.state = "Ready"
        self.sdk._client.GetGameServer.return_value = gs
        result = self.sdk.get_game_server()
        self.assertEqual(result.status.state, "Ready")

    def test_set_label(self):
        self.sdk.set_label("map", "dungeon")
        call_args = self.sdk._client.SetLabel.call_args[0][0]
        self.assertEqual(call_args.key, "map")
        self.assertEqual(call_args.value, "dungeon")

    def test_set_annotation(self):
        self.sdk.set_annotation("score", "100")
        call_args = self.sdk._client.SetAnnotation.call_args[0][0]
        self.assertEqual(call_args.key, "score")
        self.assertEqual(call_args.value, "100")

    def test_health(self):
        self.sdk._client.Health.return_value = MagicMock()
        self.sdk.health()
        self.assertIsNotNone(self.sdk._health_queue)
        self.sdk._client.Health.assert_called_once()
        # Second call reuses the stream
        self.sdk.health()
        self.sdk._client.Health.assert_called_once()

    def test_watch_game_server(self):
        gs1 = sdk_pb2.GameServer()
        gs1.status.state = "Ready"

        def blocking_stream(empty):
            yield gs1
            threading.Event().wait()

        self.sdk._client.WatchGameServer = blocking_stream
        received = []
        self.sdk.watch_game_server(lambda gs: received.append(gs))
        import time
        time.sleep(0.1)
        self.assertEqual(len(received), 1)
        self.assertEqual(received[0].status.state, "Ready")

    def test_context_manager(self):
        with patch("agones.sdk.grpc") as mock_grpc:
            mock_channel = MagicMock()
            mock_grpc.insecure_channel.return_value = mock_channel
            mock_future = MagicMock()
            mock_grpc.channel_ready_future.return_value = mock_future
            with AgonesSDK() as sdk:
                self.assertIsNotNone(sdk._client)
            mock_channel.close.assert_called_once()

    def test_close_stops_health(self):
        self.sdk._client.Health.return_value = MagicMock()
        self.sdk.health()
        self.assertIsNotNone(self.sdk._health_queue)
        self.sdk.close()
        self.assertIsNone(self.sdk._health_queue)

    def test_alpha_not_connected_raises(self):
        sdk = AgonesSDK()
        with self.assertRaises(RuntimeError):
            _ = sdk.alpha

    def test_beta_not_connected_raises(self):
        sdk = AgonesSDK()
        with self.assertRaises(RuntimeError):
            _ = sdk.beta


if __name__ == "__main__":
    unittest.main()
