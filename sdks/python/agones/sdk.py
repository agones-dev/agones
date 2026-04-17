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

"""Agones Python SDK - Core GameServer lifecycle management."""

import logging
import os
import queue
import threading
from collections.abc import Callable

import grpc

from agones._generated import sdk_pb2, sdk_pb2_grpc
from agones.alpha import Alpha
from agones.beta import Beta

_DEFAULT_HOST = "localhost"
_DEFAULT_PORT = 9357
_DEFAULT_TIMEOUT = 30


class AgonesSDK:
    """SDK for the Agones game server sidecar."""

    def __init__(self, host: str | None = None, port: int | None = None):
        self._host = host or os.environ.get("AGONES_SDK_GRPC_HOST", _DEFAULT_HOST)
        self._port = port or int(os.environ.get("AGONES_SDK_GRPC_PORT", str(_DEFAULT_PORT)))
        self._channel: grpc.Channel | None = None
        self._client: sdk_pb2_grpc.SDKStub | None = None
        self._health_stream = None
        self._health_queue: queue.Queue | None = None
        self._alpha: Alpha | None = None
        self._beta: Beta | None = None

    def connect(self, timeout: float = _DEFAULT_TIMEOUT) -> None:
        """Establish gRPC connection to the Agones sidecar."""
        self._channel = grpc.insecure_channel(f"{self._host}:{self._port}")
        grpc.channel_ready_future(self._channel).result(timeout=timeout)
        self._client = sdk_pb2_grpc.SDKStub(self._channel)
        self._alpha = Alpha(self._channel)
        self._beta = Beta(self._channel)

    def close(self) -> None:
        """Close the gRPC connection and all active streams."""
        if self._health_queue is not None:
            self._health_queue.put(None)
            self._health_queue = None
        if self._channel is not None:
            self._channel.close()
            self._channel = None

    def __enter__(self):
        self.connect()
        return self

    def __exit__(self, *args):
        self.close()

    # --- Lifecycle ---

    def ready(self) -> None:
        """Mark the GameServer as ready to receive connections."""
        self._client.Ready(sdk_pb2.Empty())

    def allocate(self) -> None:
        """Mark the GameServer as allocated."""
        self._client.Allocate(sdk_pb2.Empty())

    def shutdown(self) -> None:
        """Mark the GameServer as ready to shutdown."""
        self._client.Shutdown(sdk_pb2.Empty())

    def reserve(self, seconds: int) -> None:
        """Reserve the GameServer for the given duration."""
        self._client.Reserve(sdk_pb2.Duration(seconds=seconds))

    # --- Health ---

    def health(self) -> None:
        """Send a health ping. Call periodically to keep the server healthy."""
        if self._health_queue is None:
            self._health_queue = queue.Queue()

            def _stream():
                try:
                    self._client.Health(self._health_iter())
                except Exception as e:
                    logging.error("health stream error: %s", e)

            threading.Thread(target=_stream, daemon=True).start()
        self._health_queue.put(sdk_pb2.Empty())

    def _health_iter(self):
        while True:
            msg = self._health_queue.get()
            if msg is None:
                return
            yield msg

    # --- GameServer ---

    def get_game_server(self) -> sdk_pb2.GameServer:
        """Retrieve the current GameServer configuration."""
        return self._client.GetGameServer(sdk_pb2.Empty())

    def watch_game_server(self, callback: Callable[[sdk_pb2.GameServer], None], retry_interval: float = 5.0) -> None:
        """Watch for GameServer updates in a background thread."""
        def _watch():
            while True:
                try:
                    for gs in self._client.WatchGameServer(sdk_pb2.Empty()):
                        callback(gs)
                except Exception as e:
                    logging.error("watch_game_server error: %s, retrying in %ss", e, retry_interval)
                    threading.Event().wait(retry_interval)

        t = threading.Thread(target=_watch, daemon=True)
        t.start()

    # --- Metadata ---

    def set_label(self, key: str, value: str) -> None:
        """Set a label on the backing GameServer metadata."""
        self._client.SetLabel(sdk_pb2.KeyValue(key=key, value=value))

    def set_annotation(self, key: str, value: str) -> None:
        """Set an annotation on the backing GameServer metadata."""
        self._client.SetAnnotation(sdk_pb2.KeyValue(key=key, value=value))

    # --- Sub-SDKs ---

    @property
    def alpha(self) -> Alpha:
        """Access the Alpha SDK (player tracking)."""
        if self._alpha is None:
            raise RuntimeError("SDK not connected. Call connect() first.")
        return self._alpha

    @property
    def beta(self) -> Beta:
        """Access the Beta SDK (counters and lists)."""
        if self._beta is None:
            raise RuntimeError("SDK not connected. Call connect() first.")
        return self._beta
