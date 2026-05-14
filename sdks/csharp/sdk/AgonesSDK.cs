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
using Microsoft.Extensions.Logging;
using System;
using System.Threading;
using System.Threading.Tasks;
using Grpc.Core;
using Grpc.Net.Client;

[assembly: System.Runtime.CompilerServices.InternalsVisibleTo("Agones.Test")]
namespace Agones
{
	public sealed class AgonesSDK : IAgonesSDK
	{
		public string Host { get; } = Environment.GetEnvironmentVariable("AGONES_SDK_GRPC_HOST") ?? "localhost";
		public int Port { get; } = Convert.ToInt32(Environment.GetEnvironmentVariable("AGONES_SDK_GRPC_PORT") ?? "9357");

		/// <summary>
		/// The timeout for gRPC calls.
		/// </summary>
		public double RequestTimeoutSec { get; set; }

		internal SDK.SDKClient client;
		internal readonly Alpha alpha;
		internal readonly Beta beta;
		internal readonly GrpcChannel channel;
		internal AsyncClientStreamingCall<Empty,Empty> healthStream;
		internal readonly CancellationTokenSource cts;
		internal readonly bool ownsCts;
		internal CancellationToken ctoken;
		internal volatile bool isWatchingGameServer;

		/// <summary>
		/// Fired every time the GameServer k8s data is updated.
		/// A more efficient way to emulate WatchGameServer behavior
		/// without starting a new task for every subscription request.
		/// </summary>
		private event Action<GameServer> GameServerUpdated;
		internal Delegate[] GameServerUpdatedCallbacks => GameServerUpdated?.GetInvocationList();
		private readonly ILogger _logger;
		private readonly SemaphoreSlim _healthStreamSemaphore = new SemaphoreSlim(1, 1);
		private readonly object _gameServerWatchSyncRoot = new object();
		
		private bool _disposed;

		public AgonesSDK(
			double requestTimeoutSec = 15,
			SDK.SDKClient sdkClient = null,
			CancellationTokenSource cancellationTokenSource = null,
			ILogger logger = null)
		{
			_logger = logger;
			RequestTimeoutSec = requestTimeoutSec;
			
			if (cancellationTokenSource == null)
			{
				cts = new CancellationTokenSource();
				ownsCts = true;
			}
			else
			{
				cts = cancellationTokenSource;
				ownsCts = false;
			}
			
			ctoken = cts.Token;
			channel = GrpcChannel.ForAddress(
				$"http://{Host}:{Port}"
			);

			client = sdkClient ?? new SDK.SDKClient(channel);
			alpha = new Alpha(channel, requestTimeoutSec, cancellationTokenSource, logger);
			beta = new Beta(channel, requestTimeoutSec, cancellationTokenSource, logger);
		}

		/// <summary>
		/// Alpha returns the Alpha SDK
		/// </summary>
		/// <returns>Agones alpha SDK</returns>
		public IAgonesAlphaSDK Alpha()
		{
			return alpha;
		}

		/// <summary>
		/// Beta returns the AlphBeta SDK
		/// </summary>
		/// <returns>Agones beta SDK</returns>
		public IAgonesBetaSDK Beta()
		{
			return beta;
		}

		/// <summary>
		/// Tells Agones that the Game Server is ready to take player connections
		/// </summary>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> ReadyAsync()
		{
			try
			{
				await client.ReadyAsync(new Empty(), deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
				return new Status(StatusCode.OK, "Ready request successful.");
			}
			catch (RpcException ex)
			{
				LogError("Unable to mark GameServer to 'Ready' state.", ex);
				return ex.Status;
			}
		}

		/// <summary>
		/// Marks the game server as Allocated.
		/// </summary>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> AllocateAsync()
		{
			try
			{
				await client.AllocateAsync(new Empty(),
					deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec),
					cancellationToken: ctoken);
				return new Status(StatusCode.OK, "Allocate request successful.");
			}
			catch (RpcException ex)
			{
				LogError("Unable to mark the GameServer to 'Allocated' state.", ex);
				return ex.Status;
			}
		}

		/// <summary>
		/// Reserve(seconds) will move the GameServer into the Reserved state for the specified number of seconds (0 is forever),
		/// and then it will be moved back to Ready state. While in Reserved state,
		/// the GameServer will not be deleted on scale down or Fleet update, and also it could not be Allocated using GameServerAllocation.
		/// </summary>
		/// <param name="seconds">Amount of seconds to reserve.</param>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> ReserveAsync(long seconds)
		{
			try
			{
				await client.ReserveAsync(new Duration { Seconds = seconds},
					deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec),
					cancellationToken: ctoken);
				return new Status(StatusCode.OK, $"Reserve({seconds}) request successful.");
			}
			catch (RpcException ex)
			{
				LogError("Unable to mark the GameServer to 'Reserved' state.", ex);
				return ex.Status;
			}
		}

		/// <summary>
		/// This returns most of the backing GameServer configuration and Status.
		/// This can be useful for instances where you may want to know Health check configuration, or the IP and Port the GameServer is currently allocated to.
		/// </summary>
		/// <returns>A GameServer object containing this GameServer's configuration data</returns>
		public async Task<GameServer> GetGameServerAsync()
		{
			try
			{
				return await client.GetGameServerAsync(new Empty(),
					deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec),
					cancellationToken: ctoken);
			}
			catch (RpcException ex)
			{
				LogError("Unable to get GameServer configuration and status.", ex);
				throw;
			}
		}

		/// <summary>
		/// Starts watching the GameServer updates in the background in it's own task.
		/// On update, it fires the GameServerUpdate event.
		/// </summary>
		private async Task BeginInternalWatchAsync()
		{
			// Begin WatchGameServer in the background for the provided callback(s).
			while (!ctoken.IsCancellationRequested)
			{
				try
				{
					using (var watchStreamingCall = client.WatchGameServer(new Empty(), cancellationToken: ctoken))
					{
						var reader = watchStreamingCall.ResponseStream;
						while (await reader.MoveNext(ctoken))
						{
							try
							{
								GameServerUpdated?.Invoke(reader.Current);
							}
							catch (Exception ex)
							{
								// Swallow any exception thrown here. We don't want a callback's exception to cause
								// our watch to be torn down.
								LogWarning($"A {nameof(WatchGameServer)} callback threw an exception", ex);
							}
						}
					}
				}
				catch (OperationCanceledException) when (ctoken.IsCancellationRequested)
				{
					return;
				}
				catch (RpcException) when (ctoken.IsCancellationRequested)
				{
					return;
				}
				catch (RpcException ex)
				{
					LogError("An error occurred while watching GameServer events, will retry.", ex);
				}
			}
		}

		/// <summary>
		/// This executes the passed in callback with the current GameServer details whenever the underlying GameServer configuration is updated.
		/// This can be useful to track GameServer > Status > State changes, metadata changes, such as labels and annotations, and more.
		/// </summary>
		/// <param name="callback">The action to be called when the underlying GameServer metadata changes.</param>
		public void WatchGameServer(Action<GameServer> callback)
		{
			GameServerUpdated += callback;
			
			lock (_gameServerWatchSyncRoot)
			{
				if (isWatchingGameServer)
				{
					return;
				}
				
				isWatchingGameServer = true;
			}

			// Kick off the watch in a task so the caller doesn't need to handle exceptions that could potentially be
			// thrown before reaching the first yielding async point.
			Task.Run(async () => await BeginInternalWatchAsync(), ctoken);
		}

		/// <summary>
		/// Cancels all running tasks & tells Agones to shut down the currently running game server.
		/// </summary>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> ShutDownAsync()
		{
			try
			{
				await client.ShutdownAsync(new Empty(), deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec));
				return new Status(StatusCode.OK, "Shutdown request successful.");
			}
			catch (RpcException ex)
			{
				LogError("Unable to mark the GameServer to 'Shutdown' state.", ex);
				return ex.Status;
			}
		}

		/// <summary>
		/// Set a Label value on the backing GameServer record that is stored in Kubernetes, with the prefix 'agones.dev/sdk-'.
		/// </summary>
		/// <param name="key">Label key</param>
		/// <param name="value">Label value</param>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> SetLabelAsync(string key, string value)
		{
			try
			{
				await client.SetLabelAsync(new KeyValue()
				{
					Key = key,
					Value = value
				}, deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
				return new Status(StatusCode.OK, $"SetLabel {key}:{value} request successful.");
			}
			catch(RpcException ex)
			{
				LogError($"Unable to set the GameServer label '{key}' to '{value}'.", ex);
				return ex.Status;
			}
		}

		/// <summary>
		/// Set a Annotation value on the backing Gameserver record that is stored in Kubernetes, with the prefix 'agones.dev/sdk-'.
		/// </summary>
		/// <param name="key">Annotation key</param>
		/// <param name="value">Annotation value</param>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> SetAnnotationAsync(string key, string value)
		{
			try
			{
				await client.SetAnnotationAsync(new KeyValue()
				{
					Key = key,
					Value = value
				}, deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
				return new Status(StatusCode.OK, $"SetAnnotation {key}:{value} request successful.");
			}
			catch (RpcException ex)
			{
				LogError($"Unable to set the GameServer annotation '{key}' to '{value}'.", ex);
				return ex.Status;
			}
		}

		/// <summary>
		/// Sends a single ping to designate that the Game Server is alive and healthy.
		/// </summary>
		/// <returns>gRPC Status of the request</returns>
		public async Task<Status> HealthAsync()
		{
			try
			{
				await _healthStreamSemaphore.WaitAsync(ctoken);
			}
			catch (OperationCanceledException)
			{
				return Status.DefaultCancelled;
			}

			try
			{
				if (healthStream == null)
				{
					// Create a new stream if it's the first time we're being called or if the previous stream threw
					// an exception.
					healthStream = client.Health();
				}

				var writer = healthStream.RequestStream;
				await writer.WriteAsync(new Empty());
				return new Status(StatusCode.OK, "Health ping successful.");
			}
			catch (RpcException ex)
			{
				LogError("Unable to invoke the GameServer health check.", ex);

				if (healthStream != null)
				{
					try
					{
						// Best effort to clean up.
						healthStream.Dispose();
					}
					catch (Exception innerEx)
					{
						LogWarning($"Failed to dispose existing {nameof(client.Health)} client stream", innerEx);
					}
				}
				
				// Null out the stream so the subsequent call causes it to be recreated.
				healthStream = null;
				return ex.Status;
			}
			finally
			{
				_healthStreamSemaphore.Release();
			}
		}

		public void Dispose()
		{
			if (_disposed)
			{
				return;
			}

			cts.Cancel();
            
			if (ownsCts)
			{
				cts.Dispose();
			}

			channel.Dispose();

			// Since we don't provide any facility to unregister a WatchGameServer callback, set the event to null to
			// clear its underlying invocation list, so we don't keep holding references to objects that would prevent
			// them to be GC'd in case we don't go out of scope.
			GameServerUpdated = null;
            
			_disposed = true;
			GC.SuppressFinalize(this);
		}

		private void LogDebug(string message, Exception ex = null)
		{
			_logger?.LogDebug(ex, message);
		}

		private void LogInformation(string message, Exception ex = null)
		{
			_logger?.LogInformation(ex, message);
		}

		private void LogWarning(string message, Exception ex = null)
		{
			_logger?.LogWarning(ex, message);
		}

		private void LogError(string message, Exception ex = null)
		{
			_logger?.LogError(ex, message);
		}
	}
}
