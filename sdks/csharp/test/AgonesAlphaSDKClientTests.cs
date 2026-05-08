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

using Agones.Dev.Sdk.Alpha;
using Grpc.Core;
using System.Collections.Generic;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Moq;
using System;
using Grpc.Net.Client;
using Microsoft.Extensions.Logging;
using gProto = Google.Protobuf.WellKnownTypes;

namespace Agones.Tests
{
    [TestClass]
    public class AgonesAlphaSDKClientTests
    {
        [TestMethod]
        public async Task GetPlayerCapacity_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var expected = new Count() { Count_ = 1 };
            mockClient.Setup(m => m.GetPlayerCapacityAsync(It.IsAny<Empty>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (Empty _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Count>(Task.FromResult(expected), Task.FromResult(new Metadata()), () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.alpha.client = mockClient.Object;

            var result = await mockSdk.Alpha().GetPlayerCapacityAsync();
            Assert.AreEqual(expected.Count_, result);
        }

        [TestMethod]
        public async Task SetPlayerCapacity_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var expected = StatusCode.OK;
            mockClient.Setup(m => m.SetPlayerCapacityAsync(It.IsAny<Count>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (Count _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Empty>(Task.FromResult(new Empty()), Task.FromResult(new Metadata()), () => new Status(expected, ""), () => new Metadata(), () => { }));
            mockSdk.alpha.client = mockClient.Object;

            var result = await mockSdk.Alpha().SetPlayerCapacityAsync(1);
            Assert.AreEqual(expected, result.StatusCode);
        }

        [TestMethod]
        public async Task PlayerConnect_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var expected = new Bool() { Bool_ = true };

            mockClient.Setup(m => m.PlayerConnectAsync(It.IsAny<PlayerID>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (PlayerID _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Bool>(Task.FromResult(new Bool
                {
                    Bool_ = true
                }), Task.FromResult(new Metadata()), () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.alpha.client = mockClient.Object;

            var result = await mockSdk.Alpha().PlayerConnectAsync("test");
            Assert.AreEqual(expected.Bool_, result);
        }

        [TestMethod]
        public async Task PlayerDisconnect_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var expected = new Bool() { Bool_ = true };

            mockClient.Setup(m => m.PlayerDisconnectAsync(It.IsAny<PlayerID>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (PlayerID _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Bool>(Task.FromResult(new Bool
                {
                    Bool_ = true
                }), Task.FromResult(new Metadata()), () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.alpha.client = mockClient.Object;

            var result = await mockSdk.Alpha().PlayerDisconnectAsync("test");
            Assert.AreEqual(expected.Bool_, result);
        }

        [TestMethod]
        public async Task GetPlayerCount_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var expected = new Count() { Count_ = 1 };
            mockClient.Setup(m => m.GetPlayerCountAsync(It.IsAny<Empty>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (Empty _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Count>(Task.FromResult(expected), Task.FromResult(new Metadata()), () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.alpha.client = mockClient.Object;

            var result = await mockSdk.Alpha().GetPlayerCountAsync();
            Assert.AreEqual(expected.Count_, result);
        }

        [TestMethod]
        public async Task IsPlayerConnected_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var expected = new Bool() { Bool_ = true };
            mockClient.Setup(m => m.IsPlayerConnectedAsync(It.IsAny<PlayerID>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (PlayerID _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Bool>(Task.FromResult(expected), Task.FromResult(new Metadata()), () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.alpha.client = mockClient.Object;

            var result = await mockSdk.Alpha().IsPlayerConnectedAsync("test");
            Assert.AreEqual(expected.Bool_, result);
        }

        [TestMethod]
        public async Task GetConnectedPlayers_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var expected = new List<string> { "player1", "player2" };
            var playerList = new PlayerIDList() { List = { expected } };
            mockClient.Setup(m => m.GetConnectedPlayersAsync(It.IsAny<Empty>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (Empty _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<PlayerIDList>(Task.FromResult(playerList), Task.FromResult(new Metadata()), () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.alpha.client = mockClient.Object;

            var result = await mockSdk.Alpha().GetConnectedPlayersAsync();
            CollectionAssert.AreEquivalent(expected, result);
        }

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
