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
using Agones;
using System.Net;
using System.Net.Sockets;
using System.Text;
using UnityEngine;

namespace AgonesExample
{
    [RequireComponent(typeof(AgonesSdk))]
    public class UdpEchoServer : MonoBehaviour
    {
        private int Port { get; set; } = 7777;
        private UdpClient client = null;
        private AgonesSdk agones = null;

        async void Start()
        {
            client = new UdpClient(Port);

            agones = GetComponent<AgonesSdk>();
            bool ok = await agones.Connect();
            if (ok)
            {
                Debug.Log(("Server - Connected"));
            }
            else
            {
                Debug.Log(("Server - Failed to connect, exiting"));
                Application.Quit(1);
            }

            ok = await agones.Ready();
            if (ok)
            {
                Debug.Log($"Server - Ready");
            }
            else
            {
                Debug.Log($"Server - Ready failed");
                Application.Quit();
            }
        }

        async void Update()
        {
            if (client.Available > 0)
            {
                IPEndPoint remote = null;
                byte[] recvBytes = client.Receive(ref remote);
                string recvText = Encoding.UTF8.GetString(recvBytes);

                string[] recvTexts = recvText.Split(' ');
                byte[] echoBytes = null;
                bool ok = false;
                switch (recvTexts[0])
                {
                    case "Shutdown":
                        ok = await agones.Shutdown();
                        Debug.Log($"Server - Shutdown {ok}");

                        echoBytes = Encoding.UTF8.GetBytes($"Shutdown {ok}");
                        client.Send(echoBytes, echoBytes.Length, remote);
                        Application.Quit();
                        return;

                    case "Ready":
                        ok = await agones.Ready();
                        Debug.Log($"Server - Ready {ok}");

                        echoBytes = Encoding.UTF8.GetBytes($"Ready {ok}");
                        break;
                    
                    case "Allocate":
                        ok = await agones.Allocate();
                        Debug.Log($"Server - Allocate {ok}");

                        echoBytes = Encoding.UTF8.GetBytes($"Allocate {ok}");
                        break;

                    case "GameServer":
                        var gameserver = await agones.GameServer();
                        Debug.Log($"Server - GameServer {gameserver}");

                        ok = gameserver != null;
                        echoBytes = Encoding.UTF8.GetBytes(ok ? $"GameServer() Name: {gameserver.ObjectMeta.Name} {ok}" : $"GameServer(): {ok}");
                        break;

                    case "Label":
                        if (recvTexts.Length == 3)
                        {
                            (string key, string value) = (recvTexts[1], recvTexts[2]);
                            ok = await agones.SetLabel(key, value);
                            Debug.Log($"Server - SetLabel({recvTexts[1]}, {recvTexts[2]}) {ok}");

                            echoBytes = Encoding.UTF8.GetBytes($"SetLabel({recvTexts[1]}, {recvTexts[2]}) {ok}");
                        }
                        else
                        {
                            echoBytes = Encoding.UTF8.GetBytes($"ERROR: Invalid Label command, must use 2 arguments");
                        }

                        break;

                    case "Annotation":
                        if (recvTexts.Length == 3)
                        {
                            (string key, string value) = (recvTexts[1], recvTexts[2]);
                            ok = await agones.SetAnnotation(key, value);
                            Debug.Log($"Server - SetAnnotation({recvTexts[1]}, {recvTexts[2]}) {ok}");

                            echoBytes = Encoding.UTF8.GetBytes($"SetAnnotation({recvTexts[1]}, {recvTexts[2]}) {ok}");
                        }
                        else
                        {
                            echoBytes = Encoding.UTF8.GetBytes($"ERROR: Invalid Annotation command, must use 2 arguments");
                        }
                        break;
                    case "Reserve":
                        if (recvTexts.Length == 2)
                        {
                            TimeSpan duration = new TimeSpan(0, 0, Int32.Parse(recvTexts[1]));
                            ok = await agones.Reserve(duration);
                            Debug.Log($"Server - Reserve({recvTexts[1]} {ok}");

                            echoBytes = Encoding.UTF8.GetBytes($"Reserve({recvTexts[1]}) {ok}");
                        }
                        else
                        {
                            echoBytes = Encoding.UTF8.GetBytes($"ERROR: Invalid Reserve command, must use 1 argument");
                        }
                        break;
                    case "Watch":
                        agones.WatchGameServer(gameServer => Debug.Log($"Server - Watch {gameServer}"));
                        echoBytes = Encoding.UTF8.GetBytes("Watching()");
                        break;
                    default:
                        echoBytes = Encoding.UTF8.GetBytes($"Echo : {recvText}");
                        break;
                }

                client.Send(echoBytes, echoBytes.Length, remote);

                Debug.Log($"Server - Receive[{remote.ToString()}] : {recvText}");
            }
        }

        void OnDestroy()
        {
            client.Close();
            Debug.Log("Server - Close");
        }
    }
}