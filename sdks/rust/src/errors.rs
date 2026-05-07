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

pub type Result<T> = std::result::Result<T, Error>;

#[derive(thiserror::Error, Debug)]
pub enum Error {
    #[error("health ping connection failure: `{0}`")]
    HealthPingConnectionFailure(String),
    #[error(transparent)]
    TimedOut(#[from] tokio::time::error::Elapsed),
    #[error("failed to parse connection uri")]
    InvalidUri(#[from] http::uri::InvalidUri),
    #[error("rpc failure")]
    Rpc(#[from] tonic::Status),
    #[error("transport failure")]
    Transport(#[from] tonic::transport::Error),
}
