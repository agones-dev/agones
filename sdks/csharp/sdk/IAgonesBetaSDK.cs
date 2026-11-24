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

using System;
using System.Collections.Generic;
using System.Threading.Tasks;
using Grpc.Core;

namespace Agones
{
    /// <summary>
    /// The Counters and Lists methods on IAgonesBetaSDK are deprecated, as Counters and Lists are now Stable.
    /// Use the equivalent methods on <see cref="IAgonesSDK"/> instead.
    /// </summary>
    public interface IAgonesBetaSDK : IDisposable
    {
        [Obsolete("Counters and Lists are now Stable. Use IAgonesSDK.GetCounterCountAsync instead. This Beta method will be removed in a future release.")]
        Task<long> GetCounterCountAsync(string key);
        [Obsolete("Counters and Lists are now Stable. Use IAgonesSDK.IncrementCounterAsync instead. This Beta method will be removed in a future release.")]
        Task IncrementCounterAsync(string key, long amount);
        [Obsolete("Counters and Lists are now Stable. Use IAgonesSDK.DecrementCounterAsync instead. This Beta method will be removed in a future release.")]
        Task DecrementCounterAsync(string key, long amount);
        [Obsolete("Counters and Lists are now Stable. Use IAgonesSDK.SetCounterCountAsync instead. This Beta method will be removed in a future release.")]
        Task SetCounterCountAsync(string key, long amount);
        [Obsolete("Counters and Lists are now Stable. Use IAgonesSDK.GetCounterCapacityAsync instead. This Beta method will be removed in a future release.")]
        Task<long> GetCounterCapacityAsync(string key);
        [Obsolete("Counters and Lists are now Stable. Use IAgonesSDK.SetCounterCapacityAsync instead. This Beta method will be removed in a future release.")]
        Task SetCounterCapacityAsync(string key, long amount);
        [Obsolete("Counters and Lists are now Stable. Use IAgonesSDK.GetListCapacityAsync instead. This Beta method will be removed in a future release.")]
        Task<long> GetListCapacityAsync(string key);
        [Obsolete("Counters and Lists are now Stable. Use IAgonesSDK.SetListCapacityAsync instead. This Beta method will be removed in a future release.")]
        Task SetListCapacityAsync(string key, long amount);
        [Obsolete("Counters and Lists are now Stable. Use IAgonesSDK.ListContainsAsync instead. This Beta method will be removed in a future release.")]
        Task<bool> ListContainsAsync(string key, string value);
        [Obsolete("Counters and Lists are now Stable. Use IAgonesSDK.GetListLengthAsync instead. This Beta method will be removed in a future release.")]
        Task<int> GetListLengthAsync(string key);
        [Obsolete("Counters and Lists are now Stable. Use IAgonesSDK.GetListValuesAsync instead. This Beta method will be removed in a future release.")]
        Task<IList<string>> GetListValuesAsync(string key);
        [Obsolete("Counters and Lists are now Stable. Use IAgonesSDK.AppendListValueAsync instead. This Beta method will be removed in a future release.")]
        Task AppendListValueAsync(string key, string value);
        [Obsolete("Counters and Lists are now Stable. Use IAgonesSDK.DeleteListValueAsync instead. This Beta method will be removed in a future release.")]
        Task DeleteListValueAsync(string key, string value);
    }
}
