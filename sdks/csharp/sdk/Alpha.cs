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
using Microsoft.Extensions.Logging;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using Grpc.Net.Client;
using gProto = Google.Protobuf.WellKnownTypes;

[assembly: System.Runtime.CompilerServices.InternalsVisibleTo("Agones.Test")]
namespace Agones
{
    public sealed class Alpha : IAgonesAlphaSDK
    {

        /// <summary>
        /// The timeout for gRPC calls.
        /// </summary>
        public double RequestTimeoutSec { get; set; }

        internal SDK.SDKClient client;
        internal readonly IClientStreamWriter<Empty> healthStream;
        internal readonly CancellationTokenSource cts;
        internal readonly bool ownsCts;
        internal CancellationToken ctoken;

        private readonly ILogger _logger;
        private bool _disposed;

        public Alpha(
            GrpcChannel channel,
            double requestTimeoutSec = 15,
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
            client = new SDK.SDKClient(channel);
        }


        /// <summary>
        /// This returns the last player capacity that was set through the SDK.
        /// If the player capacity is set from outside the SDK, use SDK.GameServer() instead.
        /// </summary>
        /// <returns>Player capacity</returns>
        public async Task<long> GetPlayerCapacityAsync()
        {
            try
            {
                var count = await client.GetPlayerCapacityAsync(new Empty(), deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
                return count.Count_;
            }
            catch (RpcException ex)
            {
                LogError(ex, "Unable to invoke the GetPlayerCapacity.");
                throw;
            }
        }

        /// <summary>
        /// This changes the player capacity to a new value.
        /// </summary>
        /// <returns>gRPC Status of the request</returns>
        public async Task<Status> SetPlayerCapacityAsync(long count)
        {
            try
            {
                await client.SetPlayerCapacityAsync(new Count()
                {
                    Count_ = count
                }, deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
                return new Status(StatusCode.OK, "SetPlayerCapacity request successful.");
            }
            catch (RpcException ex)
            {
                LogError(ex, "Unable to invoke the SetPlayerCapacity.");
                return ex.Status;
            }

        }

        /// <summary>
        /// This function increases the SDK’s stored player count by one, and appends this playerID to GameServer.Status.Players.IDs.
        /// Returns true and adds the playerID to the list of playerIDs if the playerIDs was not already in the list of connected playerIDs.
        /// </summary>
        /// <returns>True if the playerID was added to the list of playerIDs</returns>
		public async Task<bool> PlayerConnectAsync(string id)
        {
            try
            {
                var result = await client.PlayerConnectAsync(new PlayerID()
                {
                    PlayerID_ = id
                }, deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
                return result.Bool_;
            }
            catch (RpcException ex)
            {
                LogError(ex, "Unable to invoke the PlayerConnect.");
                throw;
            }
        }

        /// <summary>
        /// This function decreases the SDK’s stored player count by one, and removes the playerID from GameServer.Status.Players.IDs.
        /// Will return true and remove the supplied playerID from the list of connected playerIDs if the playerID value exists within the list.
        /// </summary>
        /// <returns>True if the playerID was removed from the list of playerIDs</returns>
        public async Task<bool> PlayerDisconnectAsync(string id)
        {
            try
            {
                var result = await client.PlayerDisconnectAsync(new PlayerID()
                {
                    PlayerID_ = id
                }, deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
                return result.Bool_;
            }
            catch (RpcException ex)
            {
                LogError(ex, "Unable to invoke the PlayerDisconnect.");
                throw;
            }
        }

        /// <summary>
        /// Returns the current player count.
        /// </summary>
        /// <returns>Player count</returns>
        public async Task<long> GetPlayerCountAsync()
        {
            try
            {
                var count = await client.GetPlayerCountAsync(new Empty(), deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
                return count.Count_;
            }
            catch (RpcException ex)
            {
                LogError(ex, "Unable to invoke the GetPlayerCount.");
                throw;
            }
        }

        /// <summary>
        /// This returns if the playerID is currently connected to the GameServer.
        /// This is always accurate, even if the value hasn’t been updated to the GameServer status yet.
        /// </summary>
        /// <returns>True if the playerID is currently connected</returns>
        public async Task<bool> IsPlayerConnectedAsync(string id)
        {
            try
            {
                var result = await client.IsPlayerConnectedAsync(new PlayerID()
                {
                    PlayerID_ = id
                }, deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
                return result.Bool_;
            }
            catch (RpcException ex)
            {
                LogError(ex, "Unable to invoke the IsPlayerConnected.");
                throw;
            }
        }

        /// <summary>
        /// This returns the list of the currently connected player ids.
        /// This is always accurate, even if the value has not been updated to the Game Server status yet.
        /// </summary>
        /// <returns>The list of the currently connected player ids</returns>
        public async Task<List<string>> GetConnectedPlayersAsync()
        {
            try
            {
                var playerIDList = await client.GetConnectedPlayersAsync(new Empty(), deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
                return playerIDList.List.ToList();
            }
            catch (RpcException ex)
            {
                LogError(ex, "Unable to invoke the GetConnectedPlayers.");
                throw;
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

            _disposed = true;
            GC.SuppressFinalize(this);
        }

        private void LogError(Exception ex, string message)
        {
            _logger?.LogError(ex, message);
        }
    }
}
