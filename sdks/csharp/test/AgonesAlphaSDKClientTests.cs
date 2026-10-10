// Copyright Contributors to Agones a Series of LF Projects, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

using System.Threading;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Moq;
using Grpc.Net.Client;
using Microsoft.Extensions.Logging;

namespace Agones.Tests
{
    [TestClass]
    public class AgonesAlphaSDKClientTests
    {
        [TestMethod]
        public void InstantiateWithParameters_OK()
        {
            var mockSdk = new AgonesSDK();
            //var mockChannel = new Channel(mockSdk.Host, mockSdk.Port, ChannelCredentials.Insecure);
            var mockChannel = GrpcChannel.ForAddress($"http://{mockSdk.Host}:{mockSdk.Port}");
            ILogger mockLogger = new Mock<ILogger>().Object;
            CancellationTokenSource mockCancellationTokenSource = new Mock<CancellationTokenSource>().Object;
            bool exceptionOccured = false;
            try
            {
                new Alpha(
                    channel: mockChannel,
                    requestTimeoutSec: 15,
                    cancellationTokenSource: mockCancellationTokenSource,
                    logger: mockLogger
                );
            }
            catch
            {
                exceptionOccured = true;
            }

            Assert.IsFalse(exceptionOccured);
        }
    }
}
