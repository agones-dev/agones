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

"""Beta SDK - Counters and Lists functionality."""

import grpc
from google.protobuf.field_mask_pb2 import FieldMask
from google.protobuf.wrappers_pb2 import Int64Value

from agones._generated.beta import beta_pb2, beta_pb2_grpc


class Beta:
    """Counters and Lists API (Beta feature)."""

    def __init__(self, channel: grpc.Channel):
        self._client = beta_pb2_grpc.SDKStub(channel)

    # --- Counters ---

    def get_counter_count(self, key: str) -> int:
        """Get the current count of a counter."""
        return self._client.GetCounter(beta_pb2.GetCounterRequest(name=key)).count

    def set_counter_count(self, key: str, amount: int) -> None:
        """Set a counter to an exact value."""
        self._client.UpdateCounter(beta_pb2.UpdateCounterRequest(
            counterUpdateRequest=beta_pb2.CounterUpdateRequest(
                name=key, count=Int64Value(value=amount),
            ),
        ))

    def increment_counter(self, key: str, amount: int) -> None:
        """Increment a counter by the given amount."""
        self._client.UpdateCounter(beta_pb2.UpdateCounterRequest(
            counterUpdateRequest=beta_pb2.CounterUpdateRequest(
                name=key, countDiff=amount,
            ),
        ))

    def decrement_counter(self, key: str, amount: int) -> None:
        """Decrement a counter by the given amount."""
        self._client.UpdateCounter(beta_pb2.UpdateCounterRequest(
            counterUpdateRequest=beta_pb2.CounterUpdateRequest(
                name=key, countDiff=-amount,
            ),
        ))

    def get_counter_capacity(self, key: str) -> int:
        """Get the capacity of a counter."""
        return self._client.GetCounter(beta_pb2.GetCounterRequest(name=key)).capacity

    def set_counter_capacity(self, key: str, amount: int) -> None:
        """Set the capacity of a counter."""
        self._client.UpdateCounter(beta_pb2.UpdateCounterRequest(
            counterUpdateRequest=beta_pb2.CounterUpdateRequest(
                name=key, capacity=Int64Value(value=amount),
            ),
        ))

    # --- Lists ---

    def get_list_capacity(self, key: str) -> int:
        """Get the capacity of a list."""
        return self._client.GetList(beta_pb2.GetListRequest(name=key)).capacity

    def set_list_capacity(self, key: str, amount: int) -> None:
        """Set the capacity of a list."""
        self._client.UpdateList(beta_pb2.UpdateListRequest(
            list=beta_pb2.List(name=key, capacity=amount),
            update_mask=FieldMask(paths=["capacity"]),
        ))

    def get_list_length(self, key: str) -> int:
        """Get the number of values in a list."""
        return len(self._client.GetList(beta_pb2.GetListRequest(name=key)).values)

    def get_list_values(self, key: str) -> list[str]:
        """Get all values in a list."""
        return list(self._client.GetList(beta_pb2.GetListRequest(name=key)).values)

    def list_contains(self, key: str, value: str) -> bool:
        """Check if a value exists in a list."""
        return value in self._client.GetList(beta_pb2.GetListRequest(name=key)).values

    def append_list_value(self, key: str, value: str) -> None:
        """Add a value to a list."""
        self._client.AddListValue(beta_pb2.AddListValueRequest(name=key, value=value))

    def delete_list_value(self, key: str, value: str) -> None:
        """Remove a value from a list."""
        self._client.RemoveListValue(beta_pb2.RemoveListValueRequest(name=key, value=value))
