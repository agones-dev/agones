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

use std::{env, time::Duration};
use tonic::transport::Channel;

mod api {
    tonic::include_proto!("agones.dev.sdk");
}

use api::sdk_client::SdkClient;
pub use api::{
    game_server::{
        status::{PlayerStatus, Port},
        ObjectMeta, Spec, Status,
    },
    GameServer,
};

pub type WatchStream = tonic::Streaming<GameServer>;

use crate::{alpha::Alpha, beta::Beta, errors::Result};

#[inline]
fn empty() -> api::Empty {
    api::Empty {}
}

/// SDK is an instance of the Agones SDK
#[derive(Clone)]
pub struct Sdk {
    client: SdkClient<Channel>,
    alpha: Alpha,
    beta: Beta,
}

impl Sdk {
    /// Starts a new SDK instance, and connects to localhost on the port specified
    /// or else falls back to the `AGONES_SDK_GRPC_PORT` environment variable,
    /// or defaults to 9357.
    ///
    /// # Errors
    ///
    /// - The port specified in `AGONES_SDK_GRPC_PORT` can't be parsed as a `u16`.
    /// - A connection cannot be established with an Agones SDK server
    /// - The handshake takes longer than 30 seconds
    pub async fn new(port: Option<u16>, keep_alive: Option<Duration>) -> Result<Self> {
        let addr: http::Uri = format!(
            "http://localhost:{}",
            port.unwrap_or_else(|| {
                env::var("AGONES_SDK_GRPC_PORT")
                    .ok()
                    .and_then(|s| s.parse().ok())
                    .unwrap_or(9357)
            })
        )
        .parse()?;

        Self::new_internal(addr, keep_alive).await
    }

    pub async fn new_with_host(
        host: Option<String>,
        port: Option<u16>,
        keep_alive: Option<Duration>,
    ) -> Result<Self> {
        let addr: http::Uri = format!(
            "{}:{}",
            host.unwrap_or_else(|| {
                env::var("AGONES_SDK_GRPC_HOST")
                    .ok()
                    .and_then(|s| s.parse().ok())
                    .unwrap_or("http://localhost".to_owned())
            }),
            port.unwrap_or_else(|| {
                env::var("AGONES_SDK_GRPC_PORT")
                    .ok()
                    .and_then(|s| s.parse().ok())
                    .unwrap_or(9357)
            })
        )
        .parse()?;

        Self::new_internal(addr, keep_alive).await
    }

    async fn new_internal(addr: http::Uri, keep_alive: Option<Duration>) -> Result<Self> {
        let builder = tonic::transport::channel::Channel::builder(addr)
            .connect_timeout(Duration::from_secs(30))
            .keep_alive_timeout(keep_alive.unwrap_or_else(|| Duration::from_secs(30)));

        // will only attempt to connect on first invocation, so won't exit straight away.
        let channel = builder.connect_lazy();
        let mut client = SdkClient::new(channel.clone());
        let alpha = Alpha::new(channel.clone());
        let beta = Beta::new(channel);

        tokio::time::timeout(Duration::from_secs(30), async {
            let mut connect_interval = tokio::time::interval(Duration::from_millis(100));

            loop {
                connect_interval.tick().await;
                if client.get_game_server(empty()).await.is_ok() {
                    break;
                }
            }
        })
        .await?;

        Ok(Self { client, alpha, beta })
    }

    /// Alpha returns the Alpha SDK
    #[inline]
    pub fn alpha(&self) -> &Alpha {
        &self.alpha
    }

    /// Beta returns the Beta SDK
    #[inline]
    pub fn beta(&self) -> &Beta {
        &self.beta
    }

    /// Marks the Game Server as ready to receive connections
    pub async fn ready(&mut self) -> Result<()> {
        Ok(self.client.ready(empty()).await.map(|_| ())?)
    }

    /// Allocate the Game Server
    pub async fn allocate(&mut self) -> Result<()> {
        Ok(self.client.allocate(empty()).await.map(|_| ())?)
    }

    /// Marks the Game Server as ready to shutdown
    pub async fn shutdown(&mut self) -> Result<()> {
        Ok(self.client.shutdown(empty()).await.map(|_| ())?)
    }

    /// Returns a [`tokio::sync::mpsc::Sender`] that will emit a health check
    /// every time a message is sent on the channel.
    pub fn health_check(&self) -> tokio::sync::mpsc::Sender<()> {
        let mut health_client = self.clone();
        let (tx, mut rx) = tokio::sync::mpsc::channel(10);

        tokio::task::spawn(async move {
            let health_stream = async_stream::stream! {
                while rx.recv().await.is_some() {
                    yield empty();
                }
            };

            let _ = health_client.client.health(health_stream).await;
        });

        tx
    }

    /// Set a Label value on the backing GameServer record that is stored in Kubernetes
    pub async fn set_label(
        &mut self,
        key: impl Into<String>,
        value: impl Into<String>,
    ) -> Result<()> {
        Ok(self
            .client
            .set_label(api::KeyValue {
                key: key.into(),
                value: value.into(),
            })
            .await
            .map(|_| ())?)
    }

    /// Set a Annotation value on the backing Gameserver record that is stored in Kubernetes
    pub async fn set_annotation(
        &mut self,
        key: impl Into<String>,
        value: impl Into<String>,
    ) -> Result<()> {
        Ok(self
            .client
            .set_annotation(api::KeyValue {
                key: key.into(),
                value: value.into(),
            })
            .await
            .map(|_| ())?)
    }

    /// Returns most of the backing GameServer configuration and Status
    pub async fn get_gameserver(&mut self) -> Result<GameServer> {
        Ok(self
            .client
            .get_game_server(empty())
            .await
            .map(|res| res.into_inner())?)
    }

    /// Reserve marks the Game Server as Reserved for a given duration, at which point
    /// it will return the GameServer to a Ready state.
    /// Do note, the smallest unit available in the time.Duration argument is one second.
    pub async fn reserve(&mut self, duration: Duration) -> Result<()> {
        Ok(self
            .client
            .reserve(api::Duration {
                seconds: std::cmp::max(duration.as_secs() as i64, 1),
            })
            .await
            .map(|_| ())?)
    }

    /// Watch the backing GameServer configuration on updated
    pub async fn watch_gameserver(&mut self) -> Result<WatchStream> {
        Ok(self
            .client
            .watch_game_server(empty())
            .await
            .map(|stream| stream.into_inner())?)
    }
}
