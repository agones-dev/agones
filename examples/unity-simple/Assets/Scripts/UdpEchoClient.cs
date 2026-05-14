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

using System.Net;
using System.Net.Sockets;
using System.Text;
using UnityEngine;
using UnityEngine.UI;

namespace AgonesExample
{
    public class UdpEchoClient : MonoBehaviour
    {
        [SerializeField]
        private InputField sendTextField;
        [SerializeField]
        private Text receivedText;
        [SerializeField]
        private InputField serverAddressField;
        [SerializeField]
        private InputField serverPortField;

        private UdpClient client;
        public string ServerAddress { get; private set; } = "127.0.0.1";
        public int ServerPort { get; private set; } = 7777;

        void Start()
        {
            serverAddressField.text = ServerAddress;
            serverPortField.text = ServerPort.ToString();

            client = new UdpClient(ServerAddress, ServerPort);
        }

        void Update()
        {
            if (client.Available > 0)
            {
                IPEndPoint remote = null;
                byte[] rbytes = client.Receive(ref remote);
                string received = Encoding.UTF8.GetString(rbytes);

                Debug.Log($"Client - Recv {received}");

                receivedText.text = received;
            }
        }

        // Invoke by "Change Server" Button.
        public void ChangeServer()
        {
            if (IPAddress.TryParse(serverAddressField.text, out IPAddress ip))
            {
                ServerAddress = ip.ToString();
            }
            if (int.TryParse(serverPortField.text, out int port))
            {
                ServerPort = port;
            }

            client = new UdpClient(ServerAddress, ServerPort);

            Debug.Log($"Client - ChangeServer {ServerAddress}:{ServerPort}");
        }

        // Invoke by "Send" Button.
        public void SendTextToServer()
        {
            if (string.IsNullOrWhiteSpace(sendTextField.text))
            {
                return;
            }

            Debug.Log($"Client - SendText {sendTextField.text}");

            byte[] bytes = Encoding.UTF8.GetBytes(sendTextField.text);
            client.Send(bytes, bytes.Length);

            sendTextField.text = "";
        }
    }
}