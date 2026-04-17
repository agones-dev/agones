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

from agones._generated.beta import beta_pb2
from agones.beta import Beta


class TestBeta(unittest.TestCase):

    def setUp(self):
        self.channel = MagicMock()
        self.beta = Beta(self.channel)
        self.beta._client = MagicMock()

    # --- Counter tests ---

    def test_get_counter_count(self):
        self.beta._client.GetCounter.return_value = beta_pb2.Counter(name="rounds", count=5, capacity=100)
        self.assertEqual(self.beta.get_counter_count("rounds"), 5)

    def test_set_counter_count(self):
        self.beta._client.UpdateCounter.return_value = beta_pb2.Counter()
        self.beta.set_counter_count("rounds", 10)
        call_args = self.beta._client.UpdateCounter.call_args[0][0]
        self.assertEqual(call_args.counterUpdateRequest.name, "rounds")
        self.assertEqual(call_args.counterUpdateRequest.count.value, 10)

    def test_increment_counter(self):
        self.beta._client.UpdateCounter.return_value = beta_pb2.Counter()
        self.beta.increment_counter("rounds", 3)
        call_args = self.beta._client.UpdateCounter.call_args[0][0]
        self.assertEqual(call_args.counterUpdateRequest.name, "rounds")
        self.assertEqual(call_args.counterUpdateRequest.countDiff, 3)

    def test_decrement_counter(self):
        self.beta._client.UpdateCounter.return_value = beta_pb2.Counter()
        self.beta.decrement_counter("rounds", 2)
        call_args = self.beta._client.UpdateCounter.call_args[0][0]
        self.assertEqual(call_args.counterUpdateRequest.countDiff, -2)

    def test_get_counter_capacity(self):
        self.beta._client.GetCounter.return_value = beta_pb2.Counter(name="rounds", count=5, capacity=100)
        self.assertEqual(self.beta.get_counter_capacity("rounds"), 100)

    def test_set_counter_capacity(self):
        self.beta._client.UpdateCounter.return_value = beta_pb2.Counter()
        self.beta.set_counter_capacity("rounds", 200)
        call_args = self.beta._client.UpdateCounter.call_args[0][0]
        self.assertEqual(call_args.counterUpdateRequest.capacity.value, 200)

    # --- List tests ---

    def test_get_list_capacity(self):
        self.beta._client.GetList.return_value = beta_pb2.List(name="items", capacity=50)
        self.assertEqual(self.beta.get_list_capacity("items"), 50)

    def test_set_list_capacity(self):
        self.beta._client.UpdateList.return_value = beta_pb2.List()
        self.beta.set_list_capacity("items", 100)
        call_args = self.beta._client.UpdateList.call_args[0][0]
        self.assertEqual(call_args.list.name, "items")
        self.assertEqual(call_args.list.capacity, 100)
        self.assertIn("capacity", call_args.update_mask.paths)

    def test_get_list_length(self):
        self.beta._client.GetList.return_value = beta_pb2.List(name="items", values=["a", "b", "c"])
        self.assertEqual(self.beta.get_list_length("items"), 3)

    def test_get_list_values(self):
        self.beta._client.GetList.return_value = beta_pb2.List(name="items", values=["x", "y"])
        self.assertEqual(self.beta.get_list_values("items"), ["x", "y"])

    def test_list_contains_true(self):
        self.beta._client.GetList.return_value = beta_pb2.List(name="items", values=["sword", "shield"])
        self.assertTrue(self.beta.list_contains("items", "sword"))

    def test_list_contains_false(self):
        self.beta._client.GetList.return_value = beta_pb2.List(name="items", values=["sword", "shield"])
        self.assertFalse(self.beta.list_contains("items", "axe"))

    def test_append_list_value(self):
        self.beta._client.AddListValue.return_value = beta_pb2.List()
        self.beta.append_list_value("items", "bow")
        call_args = self.beta._client.AddListValue.call_args[0][0]
        self.assertEqual(call_args.name, "items")
        self.assertEqual(call_args.value, "bow")

    def test_delete_list_value(self):
        self.beta._client.RemoveListValue.return_value = beta_pb2.List()
        self.beta.delete_list_value("items", "sword")
        call_args = self.beta._client.RemoveListValue.call_args[0][0]
        self.assertEqual(call_args.name, "items")
        self.assertEqual(call_args.value, "sword")


if __name__ == "__main__":
    unittest.main()
