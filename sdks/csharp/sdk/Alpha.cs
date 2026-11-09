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
using System.Threading;
using Grpc.Net.Client;

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
