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


using Agones.Dev.Sdk;
using Grpc.Core;
using System.Collections.Generic;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Moq;
using System;
using gProto = Google.Protobuf.WellKnownTypes;

namespace Agones.Tests
{
    [TestClass]
    public class AgonesSDKCountersAndListsTests
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
            mockClient.Setup(m => m.GetCounterAsync(expected, It.IsAny<Metadata>(),
            It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (GetCounterRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                new AsyncUnaryCall<Counter>(Task.FromResult(counter), Task.FromResult(new Metadata()),
                () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.client = mockClient.Object;

            var response = await mockSdk.GetCounterCountAsync(key);
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
            mockSdk.client = mockClient.Object;

            await mockSdk.IncrementCounterAsync(key, amount);
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
            mockSdk.client = mockClient.Object;

            await mockSdk.DecrementCounterAsync(key, 9);
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
            mockSdk.client = mockClient.Object;

            await mockSdk.SetCounterCountAsync(key, amount);
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
            mockSdk.client = mockClient.Object;

            var response = await mockSdk.GetCounterCapacityAsync(key);
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
            mockSdk.client = mockClient.Object;

            await mockSdk.SetCounterCapacityAsync(key, amount);
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
            mockSdk.client = mockClient.Object;

            var response = await mockSdk.GetListCapacityAsync(key);
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
            mockSdk.client = mockClient.Object;

            await mockSdk.SetListCapacityAsync(key, amount);
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
            mockSdk.client = mockClient.Object;

            var response = await mockSdk.ListContainsAsync(key, value);
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
            mockSdk.client = mockClient.Object;

            await mockSdk.AppendListValueAsync(key, value);
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
            mockSdk.client = mockClient.Object;

            await mockSdk.DeleteListValueAsync(key, value);
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
            mockSdk.client = mockClient.Object;

            var response = await mockSdk.GetListLengthAsync(key);
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
            mockSdk.client = mockClient.Object;


            var response = await mockSdk.GetListValuesAsync(key);
            Assert.IsTrue(Enumerable.SequenceEqual(wantValues, response));
        }

        [TestMethod]
        public async Task IncrementCounterAsync_ZeroAmount_Only_Gets_Counter()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "counterKey";
            var expected = new GetCounterRequest()
            {
                Name = key,
            };

            mockClient.Setup(m => m.GetCounterAsync(expected, It.IsAny<Metadata>(),
            It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
                (GetCounterRequest _, Metadata _, DateTime? _, CancellationToken _) =>
                new AsyncUnaryCall<Counter>(Task.FromResult(new Counter() { Name = key }), Task.FromResult(new Metadata()),
                () => Status.DefaultSuccess, () => new Metadata(), () => { }));
            mockSdk.client = mockClient.Object;

            await mockSdk.IncrementCounterAsync(key, 0);
            await mockSdk.DecrementCounterAsync(key, 0);

            mockClient.Verify(m => m.GetCounterAsync(expected, It.IsAny<Metadata>(),
                It.IsAny<DateTime?>(), It.IsAny<CancellationToken>()), Times.Exactly(2));
            mockClient.Verify(m => m.UpdateCounterAsync(It.IsAny<UpdateCounterRequest>(), It.IsAny<Metadata>(),
                It.IsAny<DateTime?>(), It.IsAny<CancellationToken>()), Times.Never());
        }

        [TestMethod]
        public async Task IncrementCounterAsync_NegativeAmount_Throws()
        {
            var mockSdk = new AgonesSDK();
            await Assert.ThrowsExceptionAsync<ArgumentOutOfRangeException>(
                () => mockSdk.IncrementCounterAsync("counterKey", -1));
        }

        [TestMethod]
        public async Task DecrementCounterAsync_NegativeAmount_Throws()
        {
            var mockSdk = new AgonesSDK();
            await Assert.ThrowsExceptionAsync<ArgumentOutOfRangeException>(
                () => mockSdk.DecrementCounterAsync("counterKey", -1));
        }

        [TestMethod]
        public async Task GetCounterCountAsync_Error_Throws()
        {
            var mockClient = new Mock<SDK.SDKClient>();
            var mockSdk = new AgonesSDK();
            var key = "counterKey";

            mockClient.Setup(m => m.GetCounterAsync(It.IsAny<GetCounterRequest>(), It.IsAny<Metadata>(),
            It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Throws(
                new RpcException(new Status(StatusCode.NotFound, $"counter not found: {key}")));
            mockSdk.client = mockClient.Object;

            var ex = await Assert.ThrowsExceptionAsync<RpcException>(() => mockSdk.GetCounterCountAsync(key));
            Assert.AreEqual(StatusCode.NotFound, ex.StatusCode);
        }
    }
}
