// Copyright Contributors to Agones a Series of LF Projects, LLC.
// All Rights Reserved.
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

using System.Collections;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.TestTools;
using Agones;
using System.Threading.Tasks;
using System;

namespace Tests.Runtime.Agones
{
    public class AgonesSdkIntegrationTests
    {

        private AgonesSdk sdk;
        private AgonesBetaSdk betaSdk;
        private AgonesAlphaSdk alphaSdk;

        [UnitySetUp]
        public IEnumerator UnitySetUp()
        {
            GameObject gameObject = new GameObject();
            yield return null;

            this.sdk = gameObject.AddComponent<AgonesSdk>();
            this.betaSdk = gameObject.AddComponent<AgonesBetaSdk>();
            this.alphaSdk = gameObject.AddComponent<AgonesAlphaSdk>();

            Assert.IsNotNull(this.sdk);
            Assert.IsNotNull(this.betaSdk);
            Assert.IsNotNull(this.alphaSdk);
        }

        [UnityTest]
        public IEnumerator TestSdk()
        {
            var task = RunSdkTests();
            yield return new WaitUntil(() => task.IsCompleted);

            if (task.Exception != null)
            {
                Debug.LogError(task.Exception);
                Assert.Fail();
            }
        }

        private async Task RunSdkTests()
        {
            var connected = await this.sdk.Connect();
            Assert.IsTrue(connected);

            bool hasReceivedUpdates = false;
            this.sdk.WatchGameServer((gs) =>
            {
                hasReceivedUpdates = true;
                Assert.NotNull(gs.ObjectMeta);
                Assert.NotNull(gs.Status);
                Assert.NotNull(gs.Spec);
            });

            // Run tests
            var ready = await this.sdk.Ready();
            Assert.IsTrue(ready);

            var setLabel = await this.sdk.SetLabel("label", "test_label");
            Assert.IsTrue(setLabel);

            var setAnnotation = await this.sdk.SetAnnotation("annotation", "test_annotation");
            Assert.IsTrue(setAnnotation);

            var reserved = await this.sdk.Reserve(TimeSpan.FromSeconds(5));
            Assert.IsTrue(reserved);

            await Task.Delay(1000);

            var allocated = await this.sdk.Allocate();
            Assert.IsTrue(allocated);

            // Run beta tests
            await this.RunBetaSdkTests();

            Assert.IsTrue(hasReceivedUpdates);

            // Shutdown
            var shutdown = await this.sdk.Shutdown();
            Assert.IsTrue(shutdown);
        }

        private async Task RunBetaSdkTests()
        {
            // LocalSDKServer starting "rooms": {Count: 1, Capacity: 10}
            // Counters
            string counter = "rooms";

            var countSet = await this.betaSdk.SetCounterCount(counter, 4);
            Assert.IsTrue(countSet);

            var counterValue = await this.betaSdk.GetCounterCount(counter);
            Assert.AreEqual(4, counterValue);

            var incremented = await this.betaSdk.IncrementCounter(counter, 2);
            Assert.IsTrue(incremented);

            var incrementedValue = await this.betaSdk.GetCounterCount(counter);
            Assert.AreEqual(6, incrementedValue);

            var decremented = await this.betaSdk.DecrementCounter(counter, 1);
            Assert.IsTrue(decremented);

            var decrementedValue = await this.betaSdk.GetCounterCount(counter);
            Assert.AreEqual(5, decrementedValue);

            var setCounterCapacity = await this.betaSdk.SetCounterCapacity(counter, 123);
            Assert.IsTrue(setCounterCapacity);

            var counterCapacity = await this.betaSdk.GetCounterCapacity(counter);
            Assert.AreEqual(123, counterCapacity);

            // LocalSDKServer starting "players": {Values: []string{"test0", "test1", "test2"}, Capacity: 100}}
            // Lists
            string list = "players";

            var listSet = await this.betaSdk.AppendListValue(list, "test123");
            Assert.IsTrue(listSet);

            var listValues = await this.betaSdk.GetListValues(list);
            Assert.NotNull(listValues);
            Assert.AreEqual(4, listValues.Count);
            Assert.AreEqual("test123", listValues[3]);

            var listSize = await this.betaSdk.GetListLength(list);
            Assert.AreEqual(4, listSize);

            var setCapacity = await this.betaSdk.SetListCapacity(list, 25);
            Assert.IsTrue(setCapacity);

            var capacity = await this.betaSdk.GetListCapacity(list);
            Assert.AreEqual(25, capacity);

            var removedValue = await this.betaSdk.DeleteListValue(list, "test123");
            Assert.IsTrue(removedValue);

            var removedValue2 = await this.betaSdk.DeleteListValue(list, "test0");
            Assert.IsTrue(removedValue2);

            var newSize = await this.betaSdk.GetListLength(list);
            Assert.AreEqual(2, newSize);
        }
    }
}
