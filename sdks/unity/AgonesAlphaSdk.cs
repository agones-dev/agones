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
using MiniJSON;
using UnityEngine;
using UnityEngine.Networking;

namespace Agones
{
    /// <summary>
    /// Agones Alpha SDK for Unity.
    /// </summary>
    public class AgonesAlphaSdk : AgonesSdk
    {
        #region AgonesRestClient Public Methods

        private struct Player
        {
            public string playerID;

            public Player(string playerId)
            {
                this.playerID = playerId;
            }
        }

        /// <summary>
        /// This function increases the SDK’s stored player count by one, and appends this playerID to GameServer.Status.Players.IDs.
        /// Returns true and adds the playerID to the list of playerIDs if the playerIDs was not already in the list of connected playerIDs.
        /// </summary>
        /// <returns>True if the playerID was added to the list of playerIDs</returns>
        public async Task<bool> PlayerConnect(string id)
        {
            string json = JsonUtility.ToJson(new Player(playerId: id));
            return await SendRequestAsync("/alpha/player/connect", json).ContinueWith(task => task.Result.ok);
        }

        /// <summary>
        /// This function decreases the SDK’s stored player count by one, and removes the playerID from GameServer.Status.Players.IDs.
        /// Will return true and remove the supplied playerID from the list of connected playerIDs if the playerID value exists within the list.
        /// </summary>
        /// <returns>True if the playerID was removed from the list of playerIDs</returns>
        public async Task<bool> PlayerDisconnect(string id)
        {
            string json = JsonUtility.ToJson(new Player(playerId: id));
            return await SendRequestAsync("/alpha/player/disconnect", json).ContinueWith(task => task.Result.ok);
        }

        private struct Capacity
        {
            public long count;

            public Capacity(long count)
            {
                this.count = count;
            }
        }

        /// <summary>
        /// This changes the player capacity to a new value.
        /// </summary>
        /// <returns>gRPC Status of the request</returns>
        public async Task<bool> SetPlayerCapacity(long count)
        {
            string json = JsonUtility.ToJson(new Capacity(count: count));
            return await SendRequestAsync("/alpha/player/capacity", json, UnityWebRequest.kHttpVerbPUT).ContinueWith(task => task.Result.ok);
        }
        

        /// <summary>
        /// This returns the last player capacity that was set through the SDK.
        /// If the player capacity is set from outside the SDK, use SDK.GameServer() instead.
        /// </summary>
        /// <returns>Player capacity</returns>
        public async Task<long> GetPlayerCapacity()
        {
            var result =  await SendRequestAsync("/alpha/player/capacity", "{}", UnityWebRequest.kHttpVerbGET);
            
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

        /// <summary>
        /// Returns the current player count.
        /// </summary>
        /// <returns>Player count</returns>
        public async Task<long> GetPlayerCount()
        {
            var result =  await SendRequestAsync("/alpha/player/count", "{}", UnityWebRequest.kHttpVerbGET);
            
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

        /// <summary>
        /// This returns if the playerID is currently connected to the GameServer.
        /// This is always accurate, even if the value hasn’t been updated to the GameServer status yet.
        /// </summary>
        /// <returns>True if the playerID is currently connected</returns>
        public async Task<bool> IsPlayerConnected(string id)
        {
            var result = await SendRequestAsync($"/alpha/player/connected/{id}", "{}", UnityWebRequest.kHttpVerbGET);
            
            if (!result.ok)
            {
                return false;
            }

            if (Json.Deserialize(result.json) is not Dictionary<string, object> data
                || !data.TryGetValue("bool", out object boolObject)
                || boolObject is not bool resultBool)
            {
                return false;
            }
            
            return resultBool;
        }

        /// <summary>
        /// This returns the list of the currently connected player ids.
        /// This is always accurate, even if the value has not been updated to the Game Server status yet.
        /// </summary>
        /// <returns>The list of the currently connected player ids</returns>
        public async Task<List<string>> GetConnectedPlayers()
        {
            var result =  await SendRequestAsync("/alpha/player/connected", "{}", UnityWebRequest.kHttpVerbGET);
            
            if (!result.ok)
            {
                return new List<string>();
            }

            if (Json.Deserialize(result.json) is not Dictionary<string, object> data
                || !data.TryGetValue("list", out object listObject)
                || listObject is not List<object> list)
            {
                return new List<string>();
            }
            
            return list.Where(l => l is string).Select(l => l.ToString()).ToList();;
        }
        
        #endregion

    }
}