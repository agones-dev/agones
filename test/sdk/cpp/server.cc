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

// A server that starts, and then stops after 60 seconds.
// This example really does nothing other than show how to integrate
// the C++ SDK.

#include <agones/sdk.h>
#include <grpc++/grpc++.h>
#include <iostream>
#include <thread>

// send health check pings
void DoHealth(std::shared_ptr<agones::SDK> sdk) {
  bool ok = sdk->Health();
  std::cout << "Health ping " << (ok ? "sent" : "failed") << "\n" << std::flush;
}

// watch GameServer Updates
void WatchUpdates(std::shared_ptr<agones::SDK> sdk) {
  std::cout << "Starting to watch GameServer updates...\n" << std::flush;
  sdk->WatchGameServer([](const agones::dev::sdk::GameServer& gameserver) {
    std::cout << "GameServer Update:\n"                                 //
              << "\tname: " << gameserver.object_meta().name() << "\n"  //
              << "\tstate: " << gameserver.status().state() << "\n"
              << std::flush;
  });
}

int main() {
  std::cout << "C++ Game Server has started!\n"
            << "Getting the instance of the SDK.\n" << std::flush;
  auto sdk = std::make_shared<agones::SDK>();

  std::cout << "Attempting to connect...\n" << std::flush;
  if (!sdk->Connect()) {
    std::cerr << "Exiting!\n";
    return -1;
  }
  std::cout << "...handshake complete.\n" << std::flush;

  DoHealth(sdk);
  std::thread watch(WatchUpdates, sdk);

  std::cout << "Marking server as ready...\n" << std::flush;
  grpc::Status status = sdk->Ready();
  if (!status.ok()) {
    std::cerr << "Could not run Ready(): " << status.error_message()
              << ". Exiting!\n";
    return -1;
  }
  std::cout << "...marked Ready\n" << std::flush;

  status = sdk->Allocate();
  if (!status.ok()) {
    std::cerr << "Could not run Allocate(): " << status.error_message()
              << ". Exiting!\n";
    return -1;
  }
  std::cout << "...marked Allocated\n" << std::flush;

  std::chrono::seconds sec(1);
  status = sdk->Reserve(sec);
  if (!status.ok()) {
    std::cerr << "Could not run Reserve(): " << status.error_message()
              << ". Exiting!\n";
    return -1;
  }
  std::cout << "...marked Reserved\n" << std::flush;

  std::cout << "Getting GameServer details...\n" << std::flush;
  agones::dev::sdk::GameServer gameserver;
  status = sdk->GameServer(&gameserver);

  if (!status.ok()) {
    std::cerr << "Could not run GameServer(): " << status.error_message()
              << ". Exiting!\n";
    return -1;
  }

  std::cout << "GameServer name: " << gameserver.object_meta().name() << "\n"
            << std::flush;

  std::cout << "Setting a label\n" << std::flush;
  status = sdk->SetLabel(
      "test-label",
      std::to_string(gameserver.object_meta().creation_timestamp()));
  if (!status.ok()) {
    std::cerr << "Could not run SetLabel(): " << status.error_message()
              << ". Exiting!\n";
    return -1;
  }

  std::cout << "Setting an annotation\n" << std::flush;
  status =
      sdk->SetAnnotation("test-annotation", gameserver.object_meta().uid());
  if (!status.ok()) {
    std::cerr << "Could not run SetAnnotation(): " << status.error_message()
              << ". Exiting!\n";
    return -1;
  }

  for (int i = 0; i < 2; i++) {
    int time = i * 10;
    std::cout << "Running for " + std::to_string(time) + " seconds !\n"
              << std::flush;

    std::this_thread::sleep_for(std::chrono::seconds(10));

    if (i == 1) {
      std::cout << "Shutting down after 10 seconds...\n" << std::flush;
      grpc::Status status = sdk->Shutdown();
      if (!status.ok()) {
        std::cerr << "Could not run Shutdown():" << status.error_message()
                  << ". Exiting!\n";
        return -1;
      }
      std::cout << "...marked for Shutdown\n" << std::flush;
    }
  }
  watch.join();

  return 0;
}
