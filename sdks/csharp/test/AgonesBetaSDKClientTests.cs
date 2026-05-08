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

using Agones.Dev.Sdk.Beta;
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
    public class AgonesBetaSDKClientTests
    {
        [TestMethod]
        public async Task GetCounterCountAsync_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "counterKey";
            long wantCount = 1;
            var counter = new Counter()
            {
                Name = key,
                Count = wantCount,
            };
            var expected = new GetCounterRequest()
            {
                Name = key,
            };
            // TODO: Remove comments
            // https://github.com/devlooped/moq/wiki/Quickstart#async-methods
            // Task<bool> DoSomethingAsync();
            // mock.Setup(foo => foo.DoSomethingAsync().Result).Returns(true);
            // https://grpc.github.io/grpc/csharp/api/Grpc.Core.AsyncUnaryCall-1.html
            // public AsyncUnaryCall(Task<TResponse> responseAsync, Task<Metadata> responseHeadersAsync, Func<Status> getStatusFunc, Func<Metadata> getTrailersFunc, Action disposeAction)
            mockClient.Setup(m => m.GetCounterAsync(expected, It.IsAny<Metadata>(),
            It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (GetCounterRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                new AsyncUnaryCall<Counter>(Task.FromResult(counter), Task.FromResult(new Metadata()),
                () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.beta.client = mockClient.Object;

            var response = await mockSdk.Beta().GetCounterCountAsync(key);
            Assert.AreEqual(wantCount, response);
        }

        [TestMethod]
        public async Task IncrementCounterAsync_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "counterKey";
            var amount = 9;
            var counter = new Counter();
            var updateReq = new CounterUpdateRequest()
            {
                Name = key,
                CountDiff = amount,
            };
            var expected = new UpdateCounterRequest()
            {
                CounterUpdateRequest = updateReq,
            };

            mockClient.Setup(m => m.UpdateCounterAsync(expected, It.IsAny<Metadata>(),
                It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (UpdateCounterRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                new AsyncUnaryCall<Counter>(Task.FromResult(counter), Task.FromResult(new Metadata()),
              () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.beta.client = mockClient.Object;

            await mockSdk.Beta().IncrementCounterAsync(key, amount);
        }

        [TestMethod]
        public async Task DecrementCounterAsync_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "counterKey";
            var counter = new Counter();
            var updateReq = new CounterUpdateRequest()
            {
                Name = key,
                CountDiff = -9,
            };
            var expected = new UpdateCounterRequest()
            {
                CounterUpdateRequest = updateReq,
            };

            mockClient.Setup(m => m.UpdateCounterAsync(expected, It.IsAny<Metadata>(),
                It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (UpdateCounterRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                new AsyncUnaryCall<Counter>(Task.FromResult(counter), Task.FromResult(new Metadata()),
              () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.beta.client = mockClient.Object;

            await mockSdk.Beta().DecrementCounterAsync(key, 9);
        }

        [TestMethod]
        public async Task SetCounterCountAsync_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "counterKey";
            var amount = 99;
            var counter = new Counter();
            var updateReq = new CounterUpdateRequest()
            {
                Name = key,
                Count = amount,
            };
            var expected = new UpdateCounterRequest()
            {
                CounterUpdateRequest = updateReq,
            };

            mockClient.Setup(m => m.UpdateCounterAsync(expected, It.IsAny<Metadata>(),
                It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (UpdateCounterRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                new AsyncUnaryCall<Counter>(Task.FromResult(counter), Task.FromResult(new Metadata()),
              () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.beta.client = mockClient.Object;

            await mockSdk.Beta().SetCounterCountAsync(key, amount);
        }

        [TestMethod]
        public async Task GetCounterCapacityAsync_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "counterKey";
            long wantCapacity = 11;
            var counter = new Counter()
            {
                Name = key,
                Capacity = wantCapacity,
            };
            var expected = new GetCounterRequest()
            {
                Name = key,
            };

            mockClient.Setup(m => m.GetCounterAsync(expected, It.IsAny<Metadata>(),
            It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (GetCounterRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                new AsyncUnaryCall<Counter>(Task.FromResult(counter), Task.FromResult(new Metadata()),
                () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.beta.client = mockClient.Object;

            var response = await mockSdk.Beta().GetCounterCapacityAsync(key);
            Assert.AreEqual(wantCapacity, response);
        }

        [TestMethod]
        public async Task SetCounterCapacityAsync_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "counterKey";
            var amount = 99;
            var counter = new Counter();
            var updateReq = new CounterUpdateRequest()
            {
                Name = key,
                Capacity = amount,
            };
            var expected = new UpdateCounterRequest()
            {
                CounterUpdateRequest = updateReq,
            };

            mockClient.Setup(m => m.UpdateCounterAsync(expected, It.IsAny<Metadata>(),
                It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (UpdateCounterRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                new AsyncUnaryCall<Counter>(Task.FromResult(counter), Task.FromResult(new Metadata()),
              () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.beta.client = mockClient.Object;

            await mockSdk.Beta().SetCounterCapacityAsync(key, amount);
        }

        [TestMethod]
        public async Task GetListCapacityAsync_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "listKey";
            long wantCapacity = 999;
            var list = new List()
            {
                Name = key,
                Capacity = wantCapacity,
            };
            var expected = new GetListRequest()
            {
                Name = key,
            };

            mockClient.Setup(m => m.GetListAsync(expected, It.IsAny<Metadata>(),
            It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (GetListRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                new AsyncUnaryCall<List>(Task.FromResult(list), Task.FromResult(new Metadata()),
                () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.beta.client = mockClient.Object;

            var response = await mockSdk.Beta().GetListCapacityAsync(key);
            Assert.AreEqual(wantCapacity, response);
        }

        [TestMethod]
        public async Task SetListCapacityAsync_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "listKey";
            var amount = 99;
            var list = new List()
            {
                Name = key,
                Capacity = amount,
            };
            var updateMask = new gProto.FieldMask()
            {
                Paths = { "capacity" },
            };
            var expected = new UpdateListRequest()
            {
                List = list,
                UpdateMask = updateMask,
            };

            mockClient.Setup(m => m.UpdateListAsync(expected, It.IsAny<Metadata>(),
                    It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                    (UpdateListRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                    new AsyncUnaryCall<List>(Task.FromResult(list), Task.FromResult(new Metadata()),
                  () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.beta.client = mockClient.Object;

            await mockSdk.Beta().SetListCapacityAsync(key, amount);
        }

        [TestMethod]
        public async Task ListContainsAsync_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "listKey";
            var value = "foo";
            var list = new List()
            {
                Name = key,
            };
            list.Values.Add(value);
            var expected = new GetListRequest()
            {
                Name = key,
            };

            mockClient.Setup(m => m.GetListAsync(expected, It.IsAny<Metadata>(),
            It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (GetListRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                new AsyncUnaryCall<List>(Task.FromResult(list), Task.FromResult(new Metadata()),
                () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.beta.client = mockClient.Object;

            var response = await mockSdk.Beta().ListContainsAsync(key, value);
            Assert.AreEqual(true, response);
        }

        [TestMethod]
        public async Task AppendListValueAsync_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "listKey";
            var value = "foo";
            var list = new List()
            {
                Name = key,
            };
            var expected = new AddListValueRequest()
            {
                Name = key,
                Value = value,
            };

            mockClient.Setup(m => m.AddListValueAsync(expected, It.IsAny<Metadata>(),
                    It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                    (AddListValueRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                    new AsyncUnaryCall<List>(Task.FromResult(list), Task.FromResult(new Metadata()),
                  () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.beta.client = mockClient.Object;

            await mockSdk.Beta().AppendListValueAsync(key, value);
        }

        [TestMethod]
        public async Task DeleteListValueAsync_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "listKey";
            var value = "foo";
            var list = new List()
            {
                Name = key,
            };
            var expected = new RemoveListValueRequest()
            {
                Name = key,
                Value = value,
            };

            mockClient.Setup(m => m.RemoveListValueAsync(expected, It.IsAny<Metadata>(),
                    It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                    (RemoveListValueRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                    new AsyncUnaryCall<List>(Task.FromResult(list), Task.FromResult(new Metadata()),
                  () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.beta.client = mockClient.Object;

            await mockSdk.Beta().DeleteListValueAsync(key, value);
        }

        [TestMethod]
        public async Task GetListLengthAsync_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "listKey";
            long wantLength = 3;
            var values = new string[] { "foo", "bar", "baz" };
            var list = new List()
            {
                Name = key,
            };
            list.Values.Add(values);
            var expected = new GetListRequest()
            {
                Name = key,
            };

            mockClient.Setup(m => m.GetListAsync(expected, It.IsAny<Metadata>(),
            It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (GetListRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                new AsyncUnaryCall<List>(Task.FromResult(list), Task.FromResult(new Metadata()),
                () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.beta.client = mockClient.Object;

            var response = await mockSdk.Beta().GetListLengthAsync(key);
            Assert.AreEqual(wantLength, response);
        }

        [TestMethod]
        public async Task GetListValuesAsync_Sends_OK()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "listKey";
            IList<string> wantValues = new List<string> { "foo", "bar", "baz" };
            var list = new List()
            {
                Name = key,
            };
            list.Values.Add(wantValues);
            var expected = new GetListRequest()
            {
                Name = key,
            };

            mockClient.Setup(m => m.GetListAsync(expected, It.IsAny<Metadata>(),
            It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (GetListRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                new AsyncUnaryCall<List>(Task.FromResult(list), Task.FromResult(new Metadata()),
                () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.beta.client = mockClient.Object;


            var response = await mockSdk.Beta().GetListValuesAsync(key);
            Assert.IsTrue(Enumerable.SequenceEqual(wantValues, response));
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
                new Beta(
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
