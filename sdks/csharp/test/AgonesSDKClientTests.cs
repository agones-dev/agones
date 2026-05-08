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
using System.Threading;
using System.Threading.Tasks;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Moq;
using System;
using Microsoft.Extensions.Logging;

namespace Agones.Tests
{
	[TestClass]
	public class AgonesSDKClientTests
	{
		[TestMethod]
		public async Task Ready_Mock_Returns_OK()
		{
			var mockClient = new Mock<SDK.SDKClient>();
			var mockSdk = new AgonesSDK();
			var expected = StatusCode.OK;
			mockClient.Setup(m => m.ReadyAsync(It.IsAny<Empty>(), It.IsAny<Metadata>() , It.IsAny<DateTime?>(),  It.IsAny<CancellationToken>())).Returns(
				(Empty _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Empty>(Task.FromResult(new Empty()), Task.FromResult(new Metadata()), () => new Status(expected, ""), () => new Metadata(), () => { }));
			mockSdk.client = mockClient.Object;

			var result = await mockSdk.ReadyAsync();
			Assert.AreEqual(expected, result.StatusCode);
		}

		[TestMethod]
		public async Task Allocate_Mock_Returns_OK()
		{
			var mockClient = new Mock<SDK.SDKClient>();
			var mockSdk = new AgonesSDK();
			var expected = StatusCode.OK;
			mockClient.Setup(m => m.AllocateAsync(It.IsAny<Empty>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
				(Empty _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Empty>(Task.FromResult(new Empty()), Task.FromResult(new Metadata()), () => new Status(expected, ""), () => new Metadata(), () => { }));
			mockSdk.client = mockClient.Object;

			var result = await mockSdk.AllocateAsync();
			Assert.AreEqual(expected, result.StatusCode);
		}

		[TestMethod]
		public async Task Reserve_Returns_OK()
		{
			var mockClient = new Mock<SDK.SDKClient>();
			var mockSdk = new AgonesSDK();
			var expected = StatusCode.OK;
			mockClient.Setup(m => m.ReserveAsync(It.IsAny<Duration>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
				(Duration _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Empty>(Task.FromResult(new Empty()), Task.FromResult(new Metadata()), () => new Status(expected, ""), () => new Metadata(), () => { }));
			mockSdk.client = mockClient.Object;

			var result = await mockSdk.ReserveAsync(30);
			Assert.AreEqual(expected, result.StatusCode);
		}

		[TestMethod]
		public async Task Reserve_Sends_OK()
		{
			var mockClient = new Mock<SDK.SDKClient>();
			var mockSdk = new AgonesSDK();
			var expectedDuration = new Duration();
			expectedDuration.Seconds = 30;
			Duration actualDuration = null;

			mockClient.Setup(m => m.ReserveAsync(It.IsAny<Duration>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
					(Duration _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Empty>(Task.FromResult(new Empty()), Task.FromResult(new Metadata()), () => Status.DefaultSuccess, () => new Metadata(), () => { }))
				.Callback(
				(Duration dur, Metadata md, DateTime? dt, CancellationToken ct) => {
					actualDuration = dur;
				});
			mockSdk.client = mockClient.Object;

			var result = await mockSdk.ReserveAsync(30);
			Assert.AreEqual(expectedDuration, actualDuration);
		}

		[TestMethod]
		public async Task GetGameServer_Returns_OK()
		{
			var mockClient = new Mock<SDK.SDKClient>();
			var mockSdk = new AgonesSDK();
			var expected = new GameServer();

			mockClient.Setup(m => m.GetGameServerAsync(It.IsAny<Empty>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
				(Empty _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<GameServer>(Task.FromResult(expected), Task.FromResult(new Metadata()), () => Status.DefaultSuccess, () => new Metadata(), () => { }));
			mockSdk.client = mockClient.Object;

			var result = await mockSdk.GetGameServerAsync();
			Assert.AreSame(expected, result);
		}

		[TestMethod]
		public void WatchGameServer_Event_Subscribes()
		{
			var mockSdk = new AgonesSDK();
			Action<GameServer> expected = (gs) => { };

			mockSdk.isWatchingGameServer = true;
			mockSdk.WatchGameServer(expected);

			var result = mockSdk.GameServerUpdatedCallbacks[0];
			Assert.AreSame(expected, result);
		}

		[TestMethod]
		public async Task WatchGameServer_Returns_OK()
		{
			var mockClient = new Mock<SDK.SDKClient>();
			var mockResponseStream = new Moq.Mock<IAsyncStreamReader<GameServer>>();
			var mockSdk = new AgonesSDK();
			var expectedWatchReturn = new GameServer();
			GameServer actualWatchReturn = null;

			mockClient.Setup(m => m.WatchGameServer(It.IsAny<Empty>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
				(Empty _, Metadata _, DateTime? _, CancellationToken _) => new AsyncServerStreamingCall<GameServer>(mockResponseStream.Object, Task.FromResult(new Metadata()), () => Status.DefaultSuccess, () => new Metadata(), () => { }));
			mockResponseStream.Setup(m => m.Current).Returns(
				() => expectedWatchReturn);
			mockResponseStream.SetupSequence(m => m.MoveNext(It.IsAny<CancellationToken>()))
				.Returns(Task.FromResult(true))
				.Returns(Task.FromResult(false));
			mockSdk.client = mockClient.Object;

			mockSdk.WatchGameServer((gs) => { actualWatchReturn = gs; });

			// Asynchronously wait for our callback to be invoked and actualWatchReturn to be set.
			// This is because the underlying watch is started through a task so we can't expect the callback to
			// run synchronously.
			using (var cts = new CancellationTokenSource(TimeSpan.FromSeconds(2)))
			{
				while (actualWatchReturn == null)
				{
					await Task.Delay(15, cts.Token);
				}
			}
			
			Assert.AreSame(expectedWatchReturn, actualWatchReturn);
		}

		[TestMethod]
		public async Task Shutdown_Returns_OK()
		{
			var mockClient = new Mock<SDK.SDKClient>();
			var mockSdk = new AgonesSDK();
			var expected = StatusCode.OK;
			mockClient.Setup(m => m.ShutdownAsync(It.IsAny<Empty>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
				(Empty _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Empty>(Task.FromResult(new Empty()), Task.FromResult(new Metadata()), () => Status.DefaultSuccess, () => new Metadata(), () => { }));
			mockSdk.client = mockClient.Object;

			var result = await mockSdk.ShutDownAsync();
			Assert.AreEqual(expected, result.StatusCode);
		}

		[TestMethod]
		public async Task SetLabel_Returns_OK()
		{
			var mockClient = new Mock<SDK.SDKClient>();
			var mockSdk = new AgonesSDK();
			var expected = StatusCode.OK;
			mockClient.Setup(m => m.SetLabelAsync(It.IsAny<KeyValue>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
				(KeyValue _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Empty>(Task.FromResult(new Empty()), Task.FromResult(new Metadata()), () => Status.DefaultSuccess, () => new Metadata(), () => { }));
			mockSdk.client = mockClient.Object;

			var result = await mockSdk.SetLabelAsync("","");
			Assert.AreEqual(expected, result.StatusCode);
		}

		[TestMethod]
		public async Task SetLabel_Sends_OK()
		{
			var mockClient = new Mock<SDK.SDKClient>();
			var mockSdk = new AgonesSDK();
			KeyValue expectedKeyValue = new KeyValue();
			expectedKeyValue.Key = "Test";
			expectedKeyValue.Value = "Test";
			KeyValue actualKeyValue = null;
			mockClient.Setup(m => m.SetLabelAsync(It.IsAny<KeyValue>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
					(KeyValue _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Empty>(Task.FromResult(new Empty()), Task.FromResult(new Metadata()), () => Status.DefaultSuccess, () => new Metadata(), () => { }))
				.Callback(
				(KeyValue kv, Metadata md,DateTime? dt, CancellationToken ct) => { actualKeyValue = kv;
				});
			mockSdk.client = mockClient.Object;

			var result = await mockSdk.SetLabelAsync(expectedKeyValue.Key, expectedKeyValue.Value);
			Assert.AreEqual(expectedKeyValue, actualKeyValue);
		}

		[TestMethod]
		public async Task SetAnnotation_Returns_OK()
		{
			var mockClient = new Mock<SDK.SDKClient>();
			var mockSdk = new AgonesSDK();
			var expected = StatusCode.OK;
			mockClient.Setup(m => m.SetAnnotationAsync(It.IsAny<KeyValue>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
				(KeyValue _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Empty>(Task.FromResult(new Empty()), Task.FromResult(new Metadata()), () => Status.DefaultSuccess, () => new Metadata(), () => { }));
			mockSdk.client = mockClient.Object;

			var result = await mockSdk.SetAnnotationAsync("", "");
			Assert.AreEqual(expected, result.StatusCode);
		}

		[TestMethod]
		public async Task SetAnnotation_Sends_OK()
		{
			var mockClient = new Mock<SDK.SDKClient>();
			var mockSdk = new AgonesSDK();
			KeyValue expectedKeyValue = new KeyValue();
			expectedKeyValue.Key = "Test";
			expectedKeyValue.Value = "Test";
			KeyValue actualKeyValue = null;

			mockClient.Setup(m => m.SetAnnotationAsync(It.IsAny<KeyValue>(), It.IsAny<Metadata>(), It.IsAny<DateTime?>(), It.IsAny<CancellationToken>())).Returns(
					(KeyValue _, Metadata _, DateTime? _, CancellationToken _) => new AsyncUnaryCall<Empty>(Task.FromResult(new Empty()), Task.FromResult(new Metadata()), () => Status.DefaultSuccess, () => new Metadata(), () => { }))
				.Callback(
				(KeyValue kv, Metadata md, DateTime? dt, CancellationToken ct) => {
					actualKeyValue = kv;
				});
			mockSdk.client = mockClient.Object;

			var result = await mockSdk.SetAnnotationAsync(expectedKeyValue.Key, expectedKeyValue.Value);
			Assert.AreEqual(expectedKeyValue, actualKeyValue);
		}

		[TestMethod]
		public void InstantiateWithParameters_OK()
		{
			var mockClient = new Mock<SDK.SDKClient>().Object;
			ILogger mockLogger = new Mock<ILogger>().Object;
			CancellationTokenSource mockCancellationTokenSource = new Mock<CancellationTokenSource>().Object;
			bool exceptionOccured = false;
			try
			{
				new AgonesSDK(
					requestTimeoutSec: 15,
					sdkClient: mockClient,
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