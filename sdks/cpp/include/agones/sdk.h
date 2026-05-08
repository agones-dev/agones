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

#ifndef AGONES_CPP_SDK_H_
#define AGONES_CPP_SDK_H_

#include "agones_global.h"
#include "sdk.grpc.pb.h"

namespace agones {

// The Agones SDK
class SDK {
 public:
  // Creates a new instance of the SDK.
  // Does not connect to anything.
  AGONES_EXPORT SDK();
  AGONES_EXPORT ~SDK();
  SDK(const SDK&) = delete;
  SDK& operator=(const SDK&) = delete;

  // Must be called before any other functions on the SDK.
  // This will attempt to do a handshake with the sdk server, timing out
  // after 30 seconds.
  // Returns true if the connection was successful, false if not.
  AGONES_EXPORT bool Connect();

  // Marks the Game Server as ready to receive connections
  AGONES_EXPORT grpc::Status Ready();

  // Self marks this gameserver as Allocated.
  AGONES_EXPORT grpc::Status Allocate();

  // Marks the Game Server as Reserved for a given number of seconds, at which
  // point it will return the GameServer to a Ready state.
  AGONES_EXPORT grpc::Status Reserve(std::chrono::seconds seconds);

  // Send Health ping. This is a synchronous request.
  AGONES_EXPORT bool Health();

  // Retrieve the current GameServer data
  AGONES_EXPORT grpc::Status GameServer(agones::dev::sdk::GameServer* response);

  // Marks the Game Server as ready to shutdown
  AGONES_EXPORT grpc::Status Shutdown();

  // SetLabel sets a metadata label on the `GameServer` with the prefix
  // agones.dev/sdk-
  AGONES_EXPORT grpc::Status SetLabel(std::string key, std::string value);

  // SetAnnotation sets a metadata annotation on the `GameServer` with the
  // prefix agones.dev/sdk-
  AGONES_EXPORT grpc::Status SetAnnotation(std::string key, std::string value);

  // Watch the GameServer configuration, and fire the callback
  // when an update occurs.
  // This is a blocking function, and as such you will likely want to run it
  // inside a thread.
  AGONES_EXPORT grpc::Status WatchGameServer(
      const std::function<void(const agones::dev::sdk::GameServer&)>& callback);

 private:
  struct SDKImpl;
  std::unique_ptr<SDKImpl> pimpl_;
};

}  // namespace agones
#endif  // AGONES_CPP_SDK_H_
