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
using System.Threading.Tasks;
using Agones;
using NUnit.Framework;
using Tests.TestingEnvironment;
using UnityEngine;
using UnityEngine.Networking;
using UnityEngine.TestTools;

namespace Tests.Runtime.Agones
{
    public class AgonesSdkComplianceTests
    {
        [UnityTest]
        public IEnumerator AgonesSdk_Ready_ShouldInteractWithReadyApiEndpoint()
        {
            var sut = new GameObject().AddComponent<AgonesSdk>();
            var spy = new SpyRequestSender();
            sut.requestSender = spy;
            yield return null;
            var task = sut.Ready();
            yield return AwaitTask(task);
            Assert.IsTrue(spy.LastApi.Contains("/ready"));
            Assert.IsTrue(spy.LastJson.Equals("{}"));
            Assert.AreEqual(spy.LastMethod, UnityWebRequest.kHttpVerbPOST);
        }
        private IEnumerator AwaitTask(Task task)
        {
            while (!task.IsCompleted)
                yield return null;
            if (task.Exception != null)
                throw task.Exception;
        }
    }
}
