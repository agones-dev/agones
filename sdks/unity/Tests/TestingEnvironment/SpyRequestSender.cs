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

using System.Threading.Tasks;
using Agones;
using UnityEngine.Networking;

namespace Tests.TestingEnvironment
{
    public class SpyRequestSender: IRequestSender
    {
        public string LastApi { get; private set; }
        public string LastJson { get; private set; }
        public string LastMethod { get; private set; }
        public async Task<AgonesSdk.AsyncResult> SendRequestAsync(string api, string json,
            string method = UnityWebRequest.kHttpVerbPOST)
        {
            LastApi = api;
            LastJson = json;
            LastMethod = method;
            return new AgonesSdk.AsyncResult
            {
                ok = true,
                json = "{}"
            };
        }
    }
}
