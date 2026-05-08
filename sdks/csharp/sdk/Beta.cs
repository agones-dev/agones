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
    public sealed class Beta : IAgonesBetaSDK
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

        public Beta(
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
        /// GetCounterCountAsync returns the Count for a Counter, given the Counter's key (name).
        /// Throws error if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>The Counter's Count</returns>
        public async Task<long> GetCounterCountAsync(string key)
        {
          try
          {
                var request = new GetCounterRequest()
                {
                    Name = key,
                };
                var counter = await client.GetCounterAsync(request,
              deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
            return counter.Count;
          }
          catch (RpcException ex)
          {
                LogError(ex, $"Unable to invoke GetCounterCount({key}).");
                throw;
            }
        }

        /// <summary>
        /// IncrementCounterAsync increases a counter by the given nonnegative integer amount.
        /// Will execute the increment operation against the current CRD value. Will max at max(int64).
        /// Throws error if the key was not predefined in the GameServer resource on creation.
        /// Throws error if the count is at the current capacity (to the latest knowledge of the SDK),
        /// and no increment will occur.
        ///
        /// Note: A potential race condition here is that if count values are set from both the SDK and
        /// through the K8s API (Allocation or otherwise), since the SDK append operation back to the CRD
        /// value is batched asynchronous any value incremented past the capacity will be silently truncated.
        /// </summary>
        public async Task IncrementCounterAsync(string key, long amount)
        {
            if (amount < 0)
            {
                throw new ArgumentOutOfRangeException($"CountIncrement amount must be a positive number, found {amount}");
            }
            try
            {
                var request = new CounterUpdateRequest()
                {
                    Name = key,
                    CountDiff = amount,
                };
                var updateRequest = new UpdateCounterRequest()
                {
                    CounterUpdateRequest = request,
                };
                await client.UpdateCounterAsync(updateRequest,
                  deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
                // If there is no error, then the request was successful.
            }
            catch (RpcException ex)
            {
                LogError(ex, $"Unable to invoke IncrementCounter({key}, {amount}).");
                throw;
            }
        }

        /// <summary>
        /// DecrementCounterAsync decreases the current count by the given nonnegative integer amount.
        /// The Counter will not go below 0. Will execute the decrement operation against the current CRD value.
        /// Throws error if the count is at 0 (to the latest knowledge of the SDK), and no decrement will occur.
        /// </summary>
        public async Task DecrementCounterAsync(string key, long amount)
        {
            if (amount < 0)
            {
                throw new ArgumentOutOfRangeException($"DecrementCounter amount must be a positive number, found {amount}");
            }
            try
            {
                var request = new CounterUpdateRequest()
                {
                    Name = key,
                    CountDiff = amount * -1,
                };
                var updateRequest = new UpdateCounterRequest()
                {
                    CounterUpdateRequest = request,
                };
                await client.UpdateCounterAsync(updateRequest,
                  deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
            }
            catch (RpcException ex)
            {
                LogError(ex, $"Unable to invoke DecrementCounter({key}, {amount}).");
                throw;
            }
        }

        /// <summary>
        /// SetCounterCountAsync sets a count to the given value. Use with care, as this will
        /// overwrite any previous invocations’ value. Cannot be greater than Capacity.
        /// </summary>
        public async Task SetCounterCountAsync(string key, long amount)
        {
            try
            {
                var request = new CounterUpdateRequest()
                {
                    Name = key,
                    Count = amount,
                };
                var updateRequest = new UpdateCounterRequest()
                {
                    CounterUpdateRequest = request,
                };
                await client.UpdateCounterAsync(updateRequest,
                  deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
            }
            catch (RpcException ex)
            {
                LogError(ex, $"Unable to invoke SetCounterCount({key}, {amount}).");
                throw;
            }
        }

        /// <summary>
        /// GetCounterCapacityAsync returns the Capacity for a Counter, given the Counter's key (name).
        /// Throws error if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>The Counter's capacity</returns>
        public async Task<long> GetCounterCapacityAsync(string key)
        {
            try
            {
                var request = new GetCounterRequest()
                {
                    Name = key,
                };
                var counter = await client.GetCounterAsync(request,
                  deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
                return counter.Capacity;
            }
            catch (RpcException ex)
            {
                LogError(ex, $"Unable to invoke GetCounterCapacity({key}).");
                throw;
            }
        }

        /// <summary>
        /// SetCounterCapacityAsync sets the capacity for the given Counter.
        /// A capacity of 0 is no capacity.
        /// </summary>
        public async Task SetCounterCapacityAsync(string key, long amount)
        {
            try
            {
                var request = new CounterUpdateRequest()
                {
                    Name = key,
                    Capacity = amount,
                };
                var updateRequest = new UpdateCounterRequest()
                {
                    CounterUpdateRequest = request,
                };
                await client.UpdateCounterAsync(updateRequest,
                  deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
            }
            catch (RpcException ex)
            {
                LogError(ex, $"Unable to invoke SetCounterCapacity({key}, {amount}).");
                throw;
            }
        }

        /// <summary>
        /// GetListCapacityAsync returns the Capacity for a List, given the List's key (name).
        /// Throws error if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>The List's capacity</returns>
        public async Task<long> GetListCapacityAsync(string key)
        {
            try
            {
                var request = new GetListRequest()
                {
                    Name = key,
                };
                var list = await client.GetListAsync(request,
                  deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
                return list.Capacity;
            }
            catch (RpcException ex)
            {
                LogError(ex, $"Unable to invoke GetListCapacity({key}).");
                throw;
            }
        }

        /// <summary>
        /// SetListCapacityAsync sets the capacity for a given list. Capacity must be between 0 and 1000.
        /// Throws error if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        public async Task SetListCapacityAsync(string key, long amount)
        {
            try
            {
                var list = new List()
                {
                    Name = key,
                    Capacity = amount,
                };
                // FieldMask to update the capacity field only
                var updateMask = new gProto.FieldMask()
                {
                    Paths = { "capacity" },
                };
                var request = new UpdateListRequest()
                {
                    List = list,
                    UpdateMask = updateMask,
                };
                await client.UpdateListAsync(request,
                  deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
            }
            catch (RpcException ex)
            {
                LogError(ex, $"Unable to invoke SetListCapacity({key}, {amount}).");
                throw;
            }
        }

        /// <summary>
        /// ListContainsAsync returns if a string exists in a List's values list, given the List's key
        /// and the string value. Search is case-sensitive.
        /// Throws error if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>True if the value is found in the List</returns>
        public async Task<bool> ListContainsAsync(string key, string value)
        {
            try
            {
                var request = new GetListRequest()
                {
                    Name = key,
                };
                var list = await client.GetListAsync(request,
                  deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
                if (list.Values.Contains(value))
                {
                    return true;
                };
                return false;
            }
            catch (RpcException ex)
            {
                LogError(ex, $"Unable to invoke ListContains({key}, {value}).");
                throw;
            }
        }

        /// <summary>
        /// GetListLengthAsync returns the length of the Values list for a List, given the List's key.
        /// Throws error if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>The length of List's values array</returns>
        public async Task<int> GetListLengthAsync(string key)
        {
            try
            {
                var request = new GetListRequest()
                {
                    Name = key,
                };
                var list = await client.GetListAsync(request,
                  deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
                return list.Values.Count;
            }
            catch (RpcException ex)
            {
                LogError(ex, $"Unable to invoke GetListLength({key}).");
                throw;
            }
        }

        /// <summary>
        /// GetListValuesAsync returns the Values for a List, given the List's key (name).
        /// Throws error if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>The List's values array</returns>
        public async Task<IList<string>> GetListValuesAsync(string key)
        {
            try
            {
                var request = new GetListRequest()
                {
                    Name = key,
                };
                var list = await client.GetListAsync(request,
                  deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
                return list.Values;
            }
            catch (RpcException ex)
            {
                LogError(ex, $"Unable to invoke GetListValues({key}).");
                throw;
            }
        }

        /// <summary>
        /// AppendListValueAsync appends a string to a List's values list, given the List's key (name)
        /// and the string value. Throws error if the string already exists in the list.
        /// Throws error if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        public async Task AppendListValueAsync(string key, string value)
        {
            try
            {
                var request = new AddListValueRequest()
                {
                    Name = key,
                    Value = value,
                };
                await client.AddListValueAsync(request,
                  deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
            }
            catch (RpcException ex)
            {
                LogError(ex, $"Unable to invoke AppendListValue({key}, {value}).");
                throw;
            }
        }

        /// <summary>
        /// DeleteListValueAsync removes a string from a List's values list, given the List's key
        /// and the string value. Throws error if the string does not exist in the list.
        /// Throws error if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        public async Task DeleteListValueAsync(string key, string value)
        {
            try
            {
                var request = new RemoveListValueRequest()
                {
                    Name = key,
                    Value = value,
                };
                await client.RemoveListValueAsync(request,
                  deadline: DateTime.UtcNow.AddSeconds(RequestTimeoutSec), cancellationToken: ctoken);
            }
            catch (RpcException ex)
            {
                LogError(ex, $"Unable to invoke DeleteListValue({key}, {value}).");
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
