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

using System;
using System.Collections.Generic;
using System.Linq;
using System.Net;
using System.Runtime.CompilerServices;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using Agones.Model;
using JetBrains.Annotations;
using MiniJSON;
using UnityEngine;
using UnityEngine.Networking;

namespace Agones
{
    /// <summary>
    /// Agones Beta SDK for Unity.
    /// </summary>
    public class AgonesBetaSdk : AgonesSdk
    {
        #region AgonesRestClient Public Methods
        
        /// <summary>
        /// GetCounterCountAsync returns the Count for a Counter, given the Counter's key (name).
        /// Always returns 0 if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>The counter's count</returns>
        public async Task<long> GetCounterCount(string key)
        {
            var result = await SendRequestAsync($"/v1beta1/counters/{key}", "{}", UnityWebRequest.kHttpVerbGET);
            if (!result.ok)
            {
                return 0;
            }

            if (Json.Deserialize(result.json) is not Dictionary<string, object> data
                || !data.TryGetValue("count", out object countObject)
                || countObject is not string countString
                || !long.TryParse(countString, out long count))
            {
                return 0;
            }

            return count;
        }
        
        private struct CounterUpdateRequest
        {
            public long countDiff;
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
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> IncrementCounter(string key, long amount)
        {
            if (amount < 0)
            {
                throw new ArgumentOutOfRangeException($"CountIncrement amount must be a positive number, found {amount}");
            }
            
            string json = JsonUtility.ToJson(new CounterUpdateRequest {countDiff = amount });
            return await SendRequestAsync($"/v1beta1/counters/{key}", json, "PATCH").ContinueWith(task => task.Result.ok);
        }

        /// <summary>
        /// DecrementCounterAsync decreases the current count by the given nonnegative integer amount.
        /// The Counter will not go below 0. Will execute the decrement operation against the current CRD value.
        /// Throws error if the count is at 0 (to the latest knowledge of the SDK), and no decrement will occur.
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> DecrementCounter(string key, long amount)
        {
            if (amount < 0)
            {
                throw new ArgumentOutOfRangeException($"CountIncrement amount must be a positive number, found {amount}");
            }
            
            string json = JsonUtility.ToJson(new CounterUpdateRequest {countDiff = amount * -1});
            return await SendRequestAsync($"/v1beta1/counters/{key}", json, "PATCH").ContinueWith(task => task.Result.ok);
        }

        private struct CounterSetRequest {
            public long count;
        }

        /// <summary>
        /// SetCounterCountAsync sets a count to the given value. Use with care, as this will
        /// overwrite any previous invocations’ value. Cannot be greater than Capacity.
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> SetCounterCount(string key, long amount)
        {
            string json = JsonUtility.ToJson(new CounterSetRequest {count = amount});
            return await SendRequestAsync($"/v1beta1/counters/{key}", json, "PATCH").ContinueWith(task => task.Result.ok);
        }

        /// <summary>
        /// GetCounterCapacityAsync returns the Capacity for a Counter, given the Counter's key (name).
        /// Always returns 0 if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>The Counter's capacity</returns>
        public async Task<long> GetCounterCapacity(string key)
        {
            var result =  await SendRequestAsync($"/v1beta1/counters/{key}", "{}", UnityWebRequest.kHttpVerbGET);
            if (!result.ok)
            {
                return 0;
            }

            if (Json.Deserialize(result.json) is not Dictionary<string, object> data
                || !data.TryGetValue("capacity", out object capacityObject)
                || capacityObject is not string capacityString
                || !long.TryParse(capacityString, out long capacity))
            {
                return 0;
            }

            return capacity;
        }

        private struct CounterSetCapacityRequest {
            public long capacity;
        }

        /// <summary>
        /// SetCounterCapacityAsync sets the capacity for the given Counter.
        /// A capacity of 0 is no capacity.
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> SetCounterCapacity(string key, long amount)
        {
            string json = JsonUtility.ToJson(new CounterSetCapacityRequest {capacity = amount});
            return await SendRequestAsync($"/v1beta1/counters/{key}", json, "PATCH").ContinueWith(task => task.Result.ok);
        }

        /// <summary>
        /// GetListCapacityAsync returns the Capacity for a List, given the List's key (name).
        /// Always returns 0 if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>The List's capacity</returns>
        public async Task<long> GetListCapacity(string key)
        {
            var result =  await SendRequestAsync($"/v1beta1/lists/{key}", "{}", UnityWebRequest.kHttpVerbGET);
            if (!result.ok)
            {
                return 0;
            }

            if (Json.Deserialize(result.json) is not Dictionary<string, object> data
                || !data.TryGetValue("capacity", out object capacityObject)
                || capacityObject is not string capacityString
                || !long.TryParse(capacityString, out long capacity))
            {
                return 0;
            }

            return capacity;
        }

        private struct ListSetCapacityRequest {
            public long capacity;
        }

        /// <summary>
        /// SetListCapacityAsync sets the capacity for a given list. Capacity must be between 0 and 1000.
        /// Always returns false if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> SetListCapacity(string key, long amount)
        {
            string json = JsonUtility.ToJson(new ListSetCapacityRequest {
                capacity = amount
            });
            return await SendRequestAsync($"/v1beta1/lists/{key}", json, "PATCH").ContinueWith(task => task.Result.ok);
        }

        /// <summary>
        /// ListContainsAsync returns if a string exists in a List's values list, given the List's key
        /// and the string value. Search is case-sensitive.
        /// Always returns false if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>True if the value is found in the List</returns>
        public async Task<bool> ListContains(string key, string value)
        {
            var result =  await SendRequestAsync($"/v1beta1/lists/{key}", "{}", UnityWebRequest.kHttpVerbGET);
            
            if (!result.ok)
            {
                return false;
            }

            if (Json.Deserialize(result.json) is not Dictionary<string, object> data
                || !data.TryGetValue("values", out object listObject)
                || listObject is not List<object> list)
            {
                return false;
            }
            
            return list.Where(l => l is string).Select(l => l.ToString()).Contains(value);
        }

        /// <summary>
        /// GetListLengthAsync returns the length of the Values list for a List, given the List's key.
        /// Always returns 0 if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>The length of List's values array</returns>
        public async Task<int> GetListLength(string key)
        {
            var result =  await SendRequestAsync($"/v1beta1/lists/{key}", "{}", UnityWebRequest.kHttpVerbGET);
            
            if (!result.ok)
            {
                return 0;
            }

            if (Json.Deserialize(result.json) is not Dictionary<string, object> data
                || !data.TryGetValue("values", out object listObject)
                || listObject is not List<object> list)
            {
                return 0;
            }
            
            return list.Count();
        }

        /// <summary>
        /// GetListValuesAsync returns the Values for a List, given the List's key (name).
        /// Always returns an empty list if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>The List's values array</returns>
        public async Task<List<string>> GetListValues(string key)
        {
            var result =  await SendRequestAsync($"/v1beta1/lists/{key}", "{}", UnityWebRequest.kHttpVerbGET);
            
            if (!result.ok)
            {
                return new List<string>();
            }

            if (Json.Deserialize(result.json) is not Dictionary<string, object> data
                || !data.TryGetValue("values", out object listObject)
                || listObject is not List<object> list)
            {
                return new List<string>();
            }
            
            return list.Where(l => l is string).Select(l => l.ToString()).ToList();
        }
        
        private struct ListUpdateValuesRequest
        {
            public string value;
        }

        /// <summary>
        /// AppendListValueAsync appends a string to a List's values list, given the List's key (name)
        /// and the string value. Throws error if the string already exists in the list.
        /// Always returns false if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> AppendListValue(string key, string value)
        {
            string json = JsonUtility.ToJson(new ListUpdateValuesRequest {value = value});
            return await SendRequestAsync($"/v1beta1/lists/{key}:addValue", json, "POST").ContinueWith(task => task.Result.ok);
        }

        /// <summary>
        /// DeleteListValueAsync removes a string from a List's values list, given the List's key
        /// and the string value. Throws error if the string does not exist in the list.
        /// Always returns false if the key was not predefined in the GameServer resource on creation.
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> DeleteListValue(string key, string value)
        {
            string json = JsonUtility.ToJson(new ListUpdateValuesRequest {value = value});
            return await SendRequestAsync($"/v1beta1/lists/{key}:removeValue", json, "POST").ContinueWith(task => task.Result.ok);
        }

        #endregion

    }
}