// Copyright Contributors to Agones a Series of LF Projects, LLC.
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

use tonic::transport::Channel;

mod api {
    tonic::include_proto!("agones.dev.sdk.alpha");
}

use api::sdk_client::SdkClient;

/// Alpha is an instance of the Agones Alpha SDK
#[derive(Clone)]
pub struct Alpha {
    // Not currently used by any method, but kept for future Alpha SDK
    // features.
    #[allow(dead_code)]
    client: SdkClient<Channel>,
}

impl Alpha {
    /// new creates a new instance of the Alpha SDK
    pub(crate) fn new(ch: Channel) -> Self {
        Self {
            client: SdkClient::new(ch),
        }
    }
}


#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_new() {
        // connect_lazy() defers the actual connection until first use, so
        // this doesn't require a running server (sdk.rs uses the same
        // pattern to build its own client), but it still needs a Tokio
        // runtime in context.
        let channel = Channel::from_static("http://[::1]:9357").connect_lazy();
        let alpha = Alpha::new(channel);
        let _client = alpha.client;
    }
}