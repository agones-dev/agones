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
    /// Agones SDK for Unity.
    /// </summary>
    public class AgonesSdk : MonoBehaviour, IRequestSender
    {
        /// <summary>
        /// Handles sending HTTP requests to the Agones sidecar.
        /// </summary>
        public IRequestSender requestSender;
        /// <summary>
        /// Interval of the server sending a health ping to the Agones sidecar.
        /// </summary>
        [Range(0.01f, 5)] public float healthIntervalSecond = 5.0f;

        /// <summary>
        /// Whether the server sends a health ping to the Agones sidecar.
        /// </summary>
        public bool healthEnabled = true;

        /// <summary>
        /// Debug Logging Enabled. Debug logging for development of this Plugin.
        /// </summary>
        public bool logEnabled = false;

        private string sidecarAddress;
        private readonly CancellationTokenSource cancellationTokenSource = new CancellationTokenSource();

        private struct KeyValueMessage
        {
            public string key;
            public string value;
            public KeyValueMessage(string k, string v) => (key, value) = (k, v);
        }

        private List<WatchGameServerCallback> watchCallbacks = new List<WatchGameServerCallback>();
        private bool watchingForUpdates = false;

        #region Unity Methods
        // Use this for initialization.
        private void Awake()
        {
            String port = Environment.GetEnvironmentVariable("AGONES_SDK_HTTP_PORT");
            sidecarAddress = "http://localhost:" + (port ?? "9358");
        }

        private void Start()
        {
            requestSender ??= this;
            HealthCheckAsync();
        }

        private void OnApplicationQuit()
        {
            cancellationTokenSource.Dispose();
        }
        #endregion

        #region AgonesRestClient Public Methods

        /// <summary>
        /// Async method that waits to connect to the SDK Server. Will timeout
        /// and return false after 30 seconds.
        /// </summary>
        /// <returns>A task that indicated whether it was successful or not</returns>
        public async Task<bool> Connect()
        {
            for (var i = 0; i < 30; i++)
            {
                Log($"Attempting to connect...{i + 1}");
                try
                {
                    var gameServer = await GameServer();
                    if (gameServer != null)
                    {
                        Log("Connected!");
                        return true;
                    }
                }
                catch (Exception ex)
                {
                    Log($"Connection exception: {ex.Message}");
                }

                Log("Connection failed, retrying.");
                await Task.Delay(1000);
            }

            return false;
        }

        /// <summary>
        /// Marks this Game Server as ready to receive connections.
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> Ready()
        {
            return await requestSender.SendRequestAsync("/ready", "{}").ContinueWith(task => task.Result.ok);
        }

        /// <summary>
        /// Retrieve the GameServer details
        /// </summary>
        /// <returns>The current GameServer configuration</returns>
        public async Task<GameServer> GameServer()
        {
            var result = await requestSender.SendRequestAsync("/gameserver", "{}", UnityWebRequest.kHttpVerbGET);
            if (!result.ok)
            {
                return null;
            }

            var data = Json.Deserialize(result.json) as Dictionary<string, object>;
            return new GameServer(data);
        }

        /// <summary>
        /// Marks this Game Server as ready to shutdown.
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> Shutdown()
        {
            return await requestSender.SendRequestAsync("/shutdown", "{}").ContinueWith(task => task.Result.ok);
        }

        /// <summary>
        /// Marks this Game Server as Allocated.
        /// </summary>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> Allocate()
        {
            return await requestSender.SendRequestAsync("/allocate", "{}").ContinueWith(task => task.Result.ok);
        }

        /// <summary>
        /// Set a metadata label that is stored in k8s.
        /// </summary>
        /// <param name="key">label key</param>
        /// <param name="value">label value</param>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> SetLabel(string key, string value)
        {
            string json = JsonUtility.ToJson(new KeyValueMessage(key, value));
            return await requestSender.SendRequestAsync("/metadata/label", json, UnityWebRequest.kHttpVerbPUT)
                .ContinueWith(task => task.Result.ok);
        }

        /// <summary>
        /// Set a metadata annotation that is stored in k8s.
        /// </summary>
        /// <param name="key">annotation key</param>
        /// <param name="value">annotation value</param>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful.
        /// </returns>
        public async Task<bool> SetAnnotation(string key, string value)
        {
            string json = JsonUtility.ToJson(new KeyValueMessage(key, value));
            return await requestSender.SendRequestAsync("/metadata/annotation", json, UnityWebRequest.kHttpVerbPUT)
                .ContinueWith(task => task.Result.ok);
        }

        private struct Duration
        {
            public int seconds;

            public Duration(int seconds)
            {
                this.seconds = seconds;
            }
        }

        /// <summary>
        /// Move the GameServer into the Reserved state for the specified Timespan (0 seconds is forever)
        /// Smallest unit is seconds.
        /// </summary>
        /// <param name="duration">The time span to reserve for</param>
        /// <returns>
        /// A task that represents the asynchronous operation and returns true if the request was successful
        /// </returns>
        public async Task<bool> Reserve(TimeSpan duration)
        {
            string json = JsonUtility.ToJson(new Duration(seconds: duration.Seconds));
            return await requestSender.SendRequestAsync("/reserve", json).ContinueWith(task => task.Result.ok);
        }

        /// <summary>
        /// WatchGameServerCallback is the callback that will be executed every time
        /// a GameServer is changed and WatchGameServer is notified
        /// </summary>
        /// <param name="gameServer">The GameServer value</param>
        public delegate void WatchGameServerCallback(GameServer gameServer);

        /// <summary>
        /// WatchGameServer watches for changes in the backing GameServer configuration.
        /// </summary>
        /// <param name="callback">This callback is executed whenever a GameServer configuration change occurs</param>
        public void WatchGameServer(WatchGameServerCallback callback)
        {
            this.watchCallbacks.Add(callback);
            if (!this.watchingForUpdates)
            {
                StartWatchingForUpdates();
            }
        }
        #endregion

        #region AgonesRestClient Private Methods

        private void NotifyWatchUpdates(GameServer gs)
        {
            this.watchCallbacks.ForEach((callback) =>
            {
                try
                {
                    callback(gs);
                }
                catch (Exception ignore) { } // Ignore callback exceptions
            });
        }

        private void StartWatchingForUpdates()
        {
            var req = new UnityWebRequest(sidecarAddress + "/watch/gameserver", UnityWebRequest.kHttpVerbGET);
            req.downloadHandler = new GameServerHandler(this);
            req.SetRequestHeader("Content-Type", "application/json");
            req.SendWebRequest();
            this.watchingForUpdates = true;
            Log("Agones Watch Started");
        }

        private async void HealthCheckAsync()
        {
            while (healthEnabled)
            {
                await Task.Delay(TimeSpan.FromSeconds(healthIntervalSecond));

                try
                {
                    await requestSender.SendRequestAsync("/health", "{}");
                }
                catch (ObjectDisposedException)
                {
                    break;
                }
            }
        }

        /// <summary>
        /// Result of a Async HTTP request
        /// </summary>
        public struct AsyncResult
        {
            public bool ok;
            public string json;
        }

        public async Task<AsyncResult> SendRequestAsync(string api, string json,
            string method = UnityWebRequest.kHttpVerbPOST)
        {
            // To prevent that an async method leaks after destroying this gameObject.
            cancellationTokenSource.Token.ThrowIfCancellationRequested();

            var req = new UnityWebRequest(sidecarAddress + api, method)
            {
                uploadHandler = new UploadHandlerRaw(Encoding.UTF8.GetBytes(json)),
                downloadHandler = new DownloadHandlerBuffer()
            };
            req.SetRequestHeader("Content-Type", "application/json");

            await new AgonesAsyncOperationWrapper(req.SendWebRequest());

            var result = new AsyncResult();

            result.ok = req.responseCode == (long)HttpStatusCode.OK;

            if (result.ok)
            {
                result.json = req.downloadHandler.text;
                Log($"Agones SendRequest ok: {method} {api} {json} {req.downloadHandler.text}");
            }
            else
            {
                Log($"Agones SendRequest failed: {method} {api} {json} {req.error}");
            }

            req.Dispose();

            return result;
        }

        private void Log(object message)
        {
            if (!logEnabled)
            {
                return;
            }

            Debug.Log(message);
        }
        #endregion

        #region AgonesRestClient Nested Classes
        private class AgonesAsyncOperationWrapper
        {
            public UnityWebRequestAsyncOperation AsyncOp { get; }
            public AgonesAsyncOperationWrapper(UnityWebRequestAsyncOperation unityOp)
            {
                AsyncOp = unityOp;
            }

            public AgonesAsyncOperationAwaiter GetAwaiter()
            {
                return new AgonesAsyncOperationAwaiter(this);
            }
        }

        private class AgonesAsyncOperationAwaiter : INotifyCompletion
        {
            private UnityWebRequestAsyncOperation asyncOp;
            private Action continuation;
            public bool IsCompleted => asyncOp.isDone;

            public AgonesAsyncOperationAwaiter(AgonesAsyncOperationWrapper wrapper)
            {
                asyncOp = wrapper.AsyncOp;
                asyncOp.completed += OnRequestCompleted;
            }

            // C# Awaiter Pattern requires that the GetAwaiter method has GetResult(),
            // And AgonesAsyncOperationAwaiter does not return a value in this case.
            public void GetResult()
            {
                asyncOp.completed -= OnRequestCompleted;
            }

            public void OnCompleted(Action continuation)
            {
                this.continuation = continuation;
            }

            private void OnRequestCompleted(AsyncOperation _)
            {
                continuation?.Invoke();
                continuation = null;
            }
        }

        /// <summary>
        /// Custom UnityWebRequest http data handler
        /// that fires a callback whenever it receives data
        /// from the SDK.Watch() REST endpoint 
        /// </summary>
        private class GameServerHandler : DownloadHandlerScript
        {
            private AgonesSdk sdk;
            private StringBuilder stringBuilder;

            public GameServerHandler(AgonesSdk sdk)
            {
                this.sdk = sdk;
                this.stringBuilder = new StringBuilder();
            }

            protected override bool ReceiveData(byte[] data, int dataLength)
            {
                string dataString = Encoding.UTF8.GetString(data);
                this.stringBuilder.Append(dataString);

                string bufferString = stringBuilder.ToString();
                int newlineIndex;

                while ((newlineIndex = bufferString.IndexOf('\n')) >= 0)
                {
                    string fullLine = bufferString.Substring(0, newlineIndex);
                    try
                    {
                        var dictionary = (Dictionary<string, object>)Json.Deserialize(fullLine);
                        var gameServer = new GameServer(dictionary["result"] as Dictionary<string, object>);
                        this.sdk.NotifyWatchUpdates(gameServer);
                    }
                    catch (Exception ignore) { } // Ignore parse errors
                    bufferString = bufferString.Substring(newlineIndex + 1);
                }

                stringBuilder.Clear();
                stringBuilder.Append(bufferString);
                return true;
            }

            protected override void CompleteContent()
            {
                base.CompleteContent();
                this.sdk.StartWatchingForUpdates();
            }
        }
        #endregion
    }
}
