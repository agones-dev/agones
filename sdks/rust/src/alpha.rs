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

use crate::errors::Result;
use tonic::transport::Channel;

mod api {
    tonic::include_proto!("agones.dev.sdk.alpha");
}

use api::sdk_client::SdkClient;

/// Alpha is an instance of the Agones Alpha SDK
#[derive(Clone)]
pub struct Alpha {
    client: SdkClient<Channel>,
}

impl Alpha {
    /// new creates a new instance of the Alpha SDK
    pub(crate) fn new(ch: Channel) -> Self {
        Self {
            client: SdkClient::new(ch),
        }
    }

    /// This returns the last player capacity that was set through the SDK.
    /// If the player capacity is set from outside the SDK, use
    /// [`Sdk::get_gameserver`] instead.
    #[inline]
    pub async fn get_player_capacity(&mut self) -> Result<i64> {
        Ok(self
            .client
            .get_player_capacity(api::Empty {})
            .await
            .map(|c| c.into_inner().count)?)
    }

    /// This changes the player capacity to a new value.
    #[inline]
    pub async fn set_player_capacity(&mut self, count: i64) -> Result<()> {
        Ok(self
            .client
            .set_player_capacity(api::Count { count })
            .await
            .map(|_| ())?)
    }

    /// This function increases the SDK’s stored player count by one, and appends
    /// this playerID to `GameServer.status.players.ids`.
    ///
    /// Returns true and adds the playerID to the list of playerIDs if the
    /// playerIDs was not already in the list of connected playerIDs.
    #[inline]
    pub async fn player_connect(&mut self, id: impl Into<String>) -> Result<bool> {
        Ok(self
            .client
            .player_connect(api::PlayerId {
                player_id: id.into(),
            })
            .await
            .map(|b| b.into_inner().bool)?)
    }

    /// This function decreases the SDK’s stored player count by one, and removes
    /// the playerID from GameServer.status.players.ids.
    ///
    /// Will return true and remove the supplied playerID from the list of
    /// connected playerIDs if the playerID value exists within the list.
    #[inline]
    pub async fn player_disconnect(&mut self, id: impl Into<String>) -> Result<bool> {
        Ok(self
            .client
            .player_disconnect(api::PlayerId {
                player_id: id.into(),
            })
            .await
            .map(|b| b.into_inner().bool)?)
    }

    /// Returns the current player count.
    #[inline]
    pub async fn get_player_count(&mut self) -> Result<i64> {
        Ok(self
            .client
            .get_player_count(api::Empty {})
            .await
            .map(|c| c.into_inner().count)?)
    }

    /// This returns if the playerID is currently connected to the GameServer.
    /// This is always accurate, even if the value hasn’t been updated to the
    /// Game Server status yet.
    #[inline]
    pub async fn is_player_connected(&mut self, id: impl Into<String>) -> Result<bool> {
        Ok(self
            .client
            .is_player_connected(api::PlayerId {
                player_id: id.into(),
            })
            .await
            .map(|b| b.into_inner().bool)?)
    }

    /// This returns the list of the currently connected player ids.
    /// This is always accurate, even if the value has not been updated to the
    /// Game Server status yet.
    #[inline]
    pub async fn get_connected_players(&mut self) -> Result<Vec<String>> {
        Ok(self
            .client
            .get_connected_players(api::Empty {})
            .await
            .map(|pl| pl.into_inner().list)?)
    }
}


#[cfg(test)]
mod tests {
    use tokio;

    // MockAlpha simulates Alpha's async methods for unit testing
    struct MockAlpha {
        capacity: i64,
        player_count: i64,
        player_connected: Option<String>,
        player_disconnected: Option<String>,
    }

    impl MockAlpha {
        fn new() -> Self {
            Self {
                capacity: 0,
                player_count: 0,
                player_connected: None,
                player_disconnected: None,
            }
        }

        async fn get_player_capacity(&mut self) -> i64 {
            self.capacity
        }

        async fn set_player_capacity(&mut self, count: i64) {
            self.capacity = count;
        }

        async fn player_connect(&mut self, id: impl Into<String>) -> bool {
            let id = id.into();
            self.player_connected = Some(id.clone());
            self.player_count += 1;
            true
        }

        async fn player_disconnect(&mut self, id: impl Into<String>) -> bool {
            let id = id.into();
            self.player_disconnected = Some(id.clone());
            if self.player_count > 0 {
                self.player_count -= 1;
            }
            true
        }

        async fn get_player_count(&mut self) -> i64 {
            self.player_count
        }

        async fn is_player_connected(&mut self, id: impl Into<String>) -> bool {
            match &self.player_connected {
                Some(connected) => id.into() == *connected,
                None => false,
            }
        }

        async fn get_connected_players(&mut self) -> Vec<String> {
            match &self.player_connected {
                Some(id) => vec![id.clone()],
                None => vec![],
            }
        }
    }

    #[tokio::test]
    async fn test_alpha_player_flow() {
        let mut alpha = MockAlpha::new();

        // Set and get player capacity
        alpha.set_player_capacity(15).await;
        assert_eq!(alpha.capacity, 15);

        let capacity = alpha.get_player_capacity().await;
        assert_eq!(capacity, 15);

        // Connect player
        let player_id = "one";
        let ok = alpha.player_connect(player_id).await;
        assert!(ok);
        assert_eq!(alpha.player_connected.as_deref(), Some(player_id));

        // Get player count
        let count = alpha.get_player_count().await;
        assert_eq!(count, 1);

        // Disconnect player
        let ok = alpha.player_disconnect(player_id).await;
        assert!(ok);
        assert_eq!(alpha.player_disconnected.as_deref(), Some(player_id));

        // Put the player back in
        let ok = alpha.player_connect(player_id).await;
        assert!(ok);
        let count = alpha.get_player_count().await;
        assert_eq!(count, 1);

        // Is player connected (should be true)
        let ok = alpha.is_player_connected(player_id).await;
        assert!(ok, "Player should be connected");

        // Is player connected (should be false)
        let ok = alpha.is_player_connected("false").await;
        assert!(!ok, "Player should not be connected");

        // Get connected players
        let list = alpha.get_connected_players().await;
        assert_eq!(list, vec![player_id]);
    }
}