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
    tonic::include_proto!("agones.dev.sdk.beta");
}

use api::sdk_client::SdkClient;

/// Beta is an instance of the Agones Beta SDK
#[derive(Clone)]
pub struct Beta {
    client: SdkClient<Channel>,
}

impl Beta {

    /// new creates a new instance of the Beta SDK
    pub(crate) fn new(ch: Channel) -> Self {
        Self {
            client: SdkClient::new(ch),
        }
    }

    /// get_counter_count returns the Count for a Counter, given the Counter's key (name).
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn get_counter_count(&mut self, key: &str) -> Result<i64> {
        Ok(self
            .client
            .get_counter(api::GetCounterRequest { name: key.to_string() })
            .await
            .map(|c| c.into_inner().count)?)
    }

    /// increment_counter increases a counter by the given nonnegative integer amount.
    /// Will execute the increment operation against the current CRD value. Will max at max(int64).
    /// Will error if the key was not predefined in the GameServer resource on creation.
    /// Returns error if the count is at the current capacity (to the latest knowledge of the SDK),
    /// and no increment will occur.
    ///
    /// Note: A potential race condition here is that if count values are set from both the SDK and
    /// through the K8s API (Allocation or otherwise), since the SDK append operation back to the CRD
    /// value is batched asynchronous any value incremented past the capacity will be silently truncated.
    #[inline]
    pub async fn increment_counter(&mut self, key: &str, amount: i64) -> Result<()> {
        Ok(self
            .client
            .update_counter(api::UpdateCounterRequest {
                counter_update_request: Some(api::CounterUpdateRequest {
                    name: key.to_string(),
                    count: None,
                    capacity: None,
                    count_diff: amount,
                }),
            })
            .await
            .map(|_| ())?)
    }

    /// decrement_counter decreases the current count by the given nonnegative integer amount.
    /// The Counter Will not go below 0. Will execute the decrement operation against the current CRD value.
    /// Will error if the count is at 0 (to the latest knowledge of the SDK), and no decrement will occur.
    #[inline]
    pub async fn decrement_counter(&mut self, key: &str, amount: i64) -> Result<()> {
        Ok(self
            .client
            .update_counter(api::UpdateCounterRequest {
                counter_update_request: Some(api::CounterUpdateRequest {
                    name: key.to_string(),
                    count: None,
                    capacity: None,
                    count_diff: -amount,
                }),
            })
            .await
            .map(|_| ())?)
    }

    /// set_counter_count sets a count to the given value. Use with care, as this will overwrite any previous
    /// invocations’ value. Cannot be greater than Capacity.
    #[inline]
    pub async fn set_counter_count(&mut self, key: &str, amount: i64) -> Result<()> {
        Ok(self
            .client
            .update_counter(api::UpdateCounterRequest {
                counter_update_request: Some(api::CounterUpdateRequest {
                    name: key.to_string(),
                    count: Some(amount.into()),
                    capacity: None,
                    count_diff: 0,
                }),
            })
            .await
            .map(|_| ())?)
    }

    /// get_counter_capacity returns the Capacity for a Counter, given the Counter's key (name).
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn get_counter_capacity(&mut self, key: &str) -> Result<i64> {
        Ok(self
            .client
            .get_counter(api::GetCounterRequest { name: key.to_string() })
            .await
            .map(|c| c.into_inner().capacity)?)
    }

    /// set_counter_capacity sets the capacity for the given Counter. A capacity of 0 is no capacity.
    #[inline]
    pub async fn set_counter_capacity(&mut self, key: &str, amount: i64) -> Result<()> {
        Ok(self
            .client
            .update_counter(api::UpdateCounterRequest {
                counter_update_request: Some(api::CounterUpdateRequest {
                    name: key.to_string(),
                    count: None,
                    capacity: Some(amount.into()),
                    count_diff: 0,
                }),
            })
            .await
            .map(|_| ())?)
    }

    /// get_list_capacity returns the Capacity for a List, given the List's key (name).
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn get_list_capacity(&mut self, key: &str) -> Result<i64> {
        Ok(self
            .client
            .get_list(api::GetListRequest { name: key.to_string() })
            .await
            .map(|l| l.into_inner().capacity)?)
    }

    /// set_list_capacity sets the capacity for a given list. Capacity must be between 0 and 1000.
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn set_list_capacity(&mut self, key: &str, amount: i64) -> Result<()> {
        Ok(self
            .client
            .update_list(api::UpdateListRequest {
                list: Some(api::List {
                    name: key.to_string(),
                    capacity: amount,
                    values: vec![],
                }),
                update_mask: Some(prost_types::FieldMask { paths: vec!["capacity".to_string()] }),
            })
            .await
            .map(|_| ())?)
    }

    /// list_contains returns if a string exists in a List's values list, given the List's key (name)
    /// and the string value. Search is case-sensitive.
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn list_contains(&mut self, key: &str, value: &str) -> Result<bool> {
        Ok(self
            .client
            .get_list(api::GetListRequest { name: key.to_string() })
            .await
            .map(|l| l.into_inner().values.contains(&value.to_string()))?)
    }

    /// get_list_length returns the length of the Values list for a List, given the List's key (name).
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn get_list_length(&mut self, key: &str) -> Result<usize> {
        Ok(self
            .client
            .get_list(api::GetListRequest { name: key.to_string() })
            .await
            .map(|l| l.into_inner().values.len())?)
    }

    /// get_list_values returns the Values for a List, given the List's key (name).
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn get_list_values(&mut self, key: &str) -> Result<Vec<String>> {
        Ok(self
            .client
            .get_list(api::GetListRequest { name: key.to_string() })
            .await
            .map(|l| l.into_inner().values)?)
    }

    /// append_list_value appends a string to a List's values list, given the List's key (name)
    /// and the string value. Will error if the string already exists in the list.
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn append_list_value(&mut self, key: &str, value: &str) -> Result<()> {
        Ok(self
            .client
            .add_list_value(api::AddListValueRequest {
                name: key.to_string(),
                value: value.to_string(),
            })
            .await
            .map(|_| ())?)
    }

    /// delete_list_value removes a string from a List's values list, given the List's key (name)
    /// and the string value. Will error if the string does not exist in the list.
    /// Will error if the key was not predefined in the GameServer resource on creation.
    #[inline]
    pub async fn delete_list_value(&mut self, key: &str, value: &str) -> Result<()> {
        Ok(self
            .client
            .remove_list_value(api::RemoveListValueRequest {
                name: key.to_string(),
                value: value.to_string(),
            })
            .await
            .map(|_| ())?)
    }
}


#[cfg(test)]
mod tests {
    type Result<T> = std::result::Result<T, String>;
    use std::collections::HashMap;

    #[derive(Debug, PartialEq)]
    struct Counter {
        name: String,
        count: i64,
        capacity: i64,
    }

    #[derive(Debug, PartialEq)]
    struct List {
        name: String,
        values: Vec<String>,
        capacity: i64,
    }

    // MockBeta simulates Beta's implementation
    struct MockBeta {
        counters: HashMap<String, Counter>,
        lists: HashMap<String, List>,
    }

    impl MockBeta {
        fn new() -> Self {
            Self {
                counters: HashMap::new(),
                lists: HashMap::new(),
            }
        }

    // Counter methods
    async fn get_counter_count(&mut self, key: &str) -> Result<i64> {
            self.counters.get(key)
                .map(|c| c.count)
                .ok_or_else::<String, _>(|| format!("counter not found: {}", key))
        }

    async fn get_counter_capacity(&mut self, key: &str) -> Result<i64> {
            self.counters.get(key)
                .map(|c| c.capacity)
                .ok_or_else::<String, _>(|| format!("counter not found: {}", key))
        }

    async fn set_counter_capacity(&mut self, key: &str, amount: i64) -> Result<()> {
            let counter = self.counters.get_mut(key)
                .ok_or_else::<String, _>(|| format!("counter not found: {}", key))?;
            if amount < 0 {
                return Err("capacity must be >= 0".to_string());
            }
            counter.capacity = amount;
            Ok(())
        }

    async fn set_counter_count(&mut self, key: &str, amount: i64) -> Result<()> {
            let counter = self.counters.get_mut(key)
                .ok_or_else::<String, _>(|| format!("counter not found: {}", key))?;
            if amount < 0 || amount > counter.capacity {
                return Err("count out of range".to_string());
            }
            counter.count = amount;
            Ok(())
        }

    async fn increment_counter(&mut self, key: &str, amount: i64) -> Result<()> {
            let counter = self.counters.get_mut(key)
                .ok_or_else::<String, _>(|| format!("counter not found: {}", key))?;
            let new_count = counter.count + amount;
            if amount < 0 || new_count > counter.capacity {
                return Err("increment out of range".to_string());
            }
            counter.count = new_count;
            Ok(())
        }

    async fn decrement_counter(&mut self, key: &str, amount: i64) -> Result<()> {
            let counter = self.counters.get_mut(key)
                .ok_or_else::<String, _>(|| format!("counter not found: {}", key))?;
            let new_count = counter.count - amount;
            if amount < 0 || new_count < 0 {
                return Err("decrement out of range".to_string());
            }
            counter.count = new_count;
            Ok(())
        }

    // List methods
    async fn get_list_capacity(&mut self, key: &str) -> Result<i64> {
            self.lists.get(key)
                .map(|l| l.capacity)
                .ok_or_else::<String, _>(|| format!("list not found: {}", key))
        }

    async fn set_list_capacity(&mut self, key: &str, amount: i64) -> Result<()> {
            let list = self.lists.get_mut(key)
                .ok_or_else::<String, _>(|| format!("list not found: {}", key))?;
            if amount < 0 || amount > 1000 {
                return Err("capacity out of range".to_string());
            }
            list.capacity = amount;
            if list.values.len() > amount as usize {
                list.values.truncate(amount as usize);
            }
            Ok(())
        }

    async fn get_list_length(&mut self, key: &str) -> Result<usize> {
            self.lists.get(key)
                .map(|l| l.values.len())
                .ok_or_else::<String, _>(|| format!("list not found: {}", key))
        }

    async fn get_list_values(&mut self, key: &str) -> Result<Vec<String>> {
            self.lists.get(key)
                .map(|l| l.values.clone())
                .ok_or_else::<String, _>(|| format!("list not found: {}", key))
        }

    async fn list_contains(&mut self, key: &str, value: &str) -> Result<bool> {
            self.lists.get(key)
                .map(|l| l.values.contains(&value.to_string()))
                .ok_or_else::<String, _>(|| format!("list not found: {}", key))
        }

    async fn append_list_value(&mut self, key: &str, value: &str) -> Result<()> {
            let list = self.lists.get_mut(key)
                .ok_or_else::<String, _>(|| format!("list not found: {}", key))?;
            if list.values.len() >= list.capacity as usize {
                return Err("no available capacity".to_string());
            }
            if list.values.contains(&value.to_string()) {
                return Err("already exists".to_string());
            }
            list.values.push(value.to_string());
            Ok(())
        }

    async fn delete_list_value(&mut self, key: &str, value: &str) -> Result<()> {
            let list = self.lists.get_mut(key)
                .ok_or_else::<String, _>(|| format!("list not found: {}", key))?;
            if let Some(pos) = list.values.iter().position(|v| v == value) {
                list.values.remove(pos);
                Ok(())
            } else {
                Err("not found".to_string())
            }
        }
    }

    #[tokio::test]
    async fn test_beta_get_and_update_counter() {
        let mut beta = MockBeta::new();

        beta.counters.insert("sessions".to_string(), Counter { name: "sessions".to_string(), count: 21, capacity: 42 });
        beta.counters.insert("games".to_string(), Counter { name: "games".to_string(), count: 12, capacity: 24 });
        beta.counters.insert("gamers".to_string(), Counter { name: "gamers".to_string(), count: 263, capacity: 500 });

        // Set Counter and Set Capacity
        {
            let count = beta.get_counter_count("sessions").await.unwrap();
            assert_eq!(count, 21);

            let capacity = beta.get_counter_capacity("sessions").await.unwrap();
            assert_eq!(capacity, 42);

            let want_capacity = 25;
            beta.set_counter_capacity("sessions", want_capacity).await.unwrap();
            let capacity = beta.get_counter_capacity("sessions").await.unwrap();
            assert_eq!(capacity, want_capacity);

            let want_count = 10;
            beta.set_counter_count("sessions", want_count).await.unwrap();
            let count = beta.get_counter_count("sessions").await.unwrap();
            assert_eq!(count, want_count);
        }

        // Get and Set Non-Defined Counter
        {
            assert!(beta.get_counter_count("secessions").await.is_err());
            assert!(beta.get_counter_capacity("secessions").await.is_err());
            assert!(beta.set_counter_capacity("secessions", 100).await.is_err());
            assert!(beta.set_counter_count("secessions", 0).await.is_err());
        }

        // Decrement Counter Fails then Success
        {
            let count = beta.get_counter_count("games").await.unwrap();
            assert_eq!(count, 12);

            assert!(beta.decrement_counter("games", 21).await.is_err());
            let count = beta.get_counter_count("games").await.unwrap();
            assert_eq!(count, 12);

            assert!(beta.decrement_counter("games", -12).await.is_err());
            let count = beta.get_counter_count("games").await.unwrap();
            assert_eq!(count, 12);

            beta.decrement_counter("games", 12).await.unwrap();
            let count = beta.get_counter_count("games").await.unwrap();
            assert_eq!(count, 0);
        }
    }

    #[tokio::test]
    async fn test_beta_increment_counter_fails_then_success() {
        let mut beta = MockBeta::new();

        beta.counters.insert("gamers".to_string(), Counter { name: "gamers".to_string(), count: 263, capacity: 500 });

        // Increment Counter Fails then Success
        {
            let count = beta.get_counter_count("gamers").await.unwrap();
            assert_eq!(count, 263);

            assert!(beta.increment_counter("gamers", 250).await.is_err());
            let count = beta.get_counter_count("gamers").await.unwrap();
            assert_eq!(count, 263);

            assert!(beta.increment_counter("gamers", -237).await.is_err());
            let count = beta.get_counter_count("gamers").await.unwrap();
            assert_eq!(count, 263);

            beta.increment_counter("gamers", 237).await.unwrap();
            let count = beta.get_counter_count("gamers").await.unwrap();
            assert_eq!(count, 500);
        }
    }

    #[tokio::test]
    async fn test_beta_get_and_update_list() {
        let mut beta = MockBeta::new();

        beta.lists.insert("foo".to_string(), List { name: "foo".to_string(), values: vec![], capacity: 2 });
        beta.lists.insert("bar".to_string(), List { name: "bar".to_string(), values: vec!["abc".to_string(), "def".to_string()], capacity: 5 });
        beta.lists.insert("baz".to_string(), List { name: "baz".to_string(), values: vec!["123".to_string(), "456".to_string(), "789".to_string()], capacity: 5 });

        // Get and Set List Capacity
        {
            let capacity = beta.get_list_capacity("foo").await.unwrap();
            assert_eq!(capacity, 2);

            let want_capacity = 5;
            beta.set_list_capacity("foo", want_capacity).await.unwrap();
            let capacity = beta.get_list_capacity("foo").await.unwrap();
            assert_eq!(capacity, want_capacity);
        }

        // Get List Length, Get List Values, ListContains, and Append List Value
        {
            let length = beta.get_list_length("bar").await.unwrap();
            assert_eq!(length, 2);

            let values = beta.get_list_values("bar").await.unwrap();
            assert_eq!(values, vec!["abc".to_string(), "def".to_string()]);

            beta.append_list_value("bar", "ghi").await.unwrap();
            let length = beta.get_list_length("bar").await.unwrap();
            assert_eq!(length, 3);

            let want_values = vec!["abc".to_string(), "def".to_string(), "ghi".to_string()];
            let values = beta.get_list_values("bar").await.unwrap();
            assert_eq!(values, want_values);

            let contains = beta.list_contains("bar", "ghi").await.unwrap();
            assert!(contains);
        }

        // Get List Length, Get List Values, ListContains, and Delete List Value
        {
            let length = beta.get_list_length("baz").await.unwrap();
            assert_eq!(length, 3);

            let values = beta.get_list_values("baz").await.unwrap();
            assert_eq!(values, vec!["123".to_string(), "456".to_string(), "789".to_string()]);

            beta.delete_list_value("baz", "456").await.unwrap();
            let length = beta.get_list_length("baz").await.unwrap();
            assert_eq!(length, 2);

            let want_values = vec!["123".to_string(), "789".to_string()];
            let values = beta.get_list_values("baz").await.unwrap();
            assert_eq!(values, want_values);

            let contains = beta.list_contains("baz", "456").await.unwrap();
            assert!(!contains);
        }
    }
}