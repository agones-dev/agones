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

#include "agones/sdk.h"

#include <grpcpp/grpcpp.h>

#include <utility>

namespace agones {

struct SDK::SDKImpl {
  std::string host_;
  std::shared_ptr<grpc::Channel> channel_;
  std::unique_ptr<agones::dev::sdk::SDK::Stub> stub_;
  std::unique_ptr<grpc::ClientWriter<agones::dev::sdk::Empty>> health_;
  std::unique_ptr<grpc::ClientContext> health_context_;
};

SDK::SDK() : pimpl_{std::make_unique<SDKImpl>()} {
  const char* port = std::getenv("AGONES_SDK_GRPC_PORT");
  pimpl_->host_ = std::string("localhost:") + (port ? port : "9357");
  pimpl_->channel_ =
      grpc::CreateChannel(pimpl_->host_, grpc::InsecureChannelCredentials());
}

SDK::~SDK() {}

bool SDK::Connect() {
  if (!pimpl_->channel_->WaitForConnected(
          gpr_time_add(gpr_now(GPR_CLOCK_REALTIME),
                       gpr_time_from_seconds(30, GPR_TIMESPAN)))) {
    std::cerr << "Could not connect to the sidecar at " << pimpl_->host_
              << ".\n";
    return false;
  }

  pimpl_->stub_ = agones::dev::sdk::SDK::NewStub(pimpl_->channel_);

  // Make the health connection.
  agones::dev::sdk::Empty response;
  pimpl_->health_context_ =
      std::unique_ptr<grpc::ClientContext>(new grpc::ClientContext);
  pimpl_->health_ = pimpl_->stub_->Health(&*pimpl_->health_context_, &response);

  return true;
}

grpc::Status SDK::Ready() {
  grpc::ClientContext context;
  context.set_deadline(gpr_time_add(gpr_now(GPR_CLOCK_REALTIME),
                                    gpr_time_from_seconds(30, GPR_TIMESPAN)));
  agones::dev::sdk::Empty request;
  agones::dev::sdk::Empty response;

  return pimpl_->stub_->Ready(&context, request, &response);
}

grpc::Status SDK::Allocate() {
  grpc::ClientContext context;
  context.set_deadline(gpr_time_add(gpr_now(GPR_CLOCK_REALTIME),
                                    gpr_time_from_seconds(30, GPR_TIMESPAN)));
  agones::dev::sdk::Empty request;
  agones::dev::sdk::Empty response;

  return pimpl_->stub_->Allocate(&context, request, &response);
}

grpc::Status SDK::Reserve(std::chrono::seconds seconds) {
  grpc::ClientContext context;
  context.set_deadline(gpr_time_add(gpr_now(GPR_CLOCK_REALTIME),
                                    gpr_time_from_seconds(30, GPR_TIMESPAN)));

  agones::dev::sdk::Duration request;
  request.set_seconds(seconds.count());

  agones::dev::sdk::Empty response;

  return pimpl_->stub_->Reserve(&context, request, &response);
}

bool SDK::Health() {
  agones::dev::sdk::Empty request;
  return pimpl_->health_->Write(request);
}

grpc::Status SDK::GameServer(agones::dev::sdk::GameServer* response) {
  grpc::ClientContext context;
  context.set_deadline(gpr_time_add(gpr_now(GPR_CLOCK_REALTIME),
                                    gpr_time_from_seconds(30, GPR_TIMESPAN)));
  agones::dev::sdk::Empty request;

  return pimpl_->stub_->GetGameServer(&context, request, response);
}

grpc::Status SDK::WatchGameServer(
    const std::function<void(const agones::dev::sdk::GameServer&)>& callback) {
  grpc::ClientContext context;
  agones::dev::sdk::Empty request;
  agones::dev::sdk::GameServer gameServer;

  std::unique_ptr<grpc::ClientReader<agones::dev::sdk::GameServer>> reader =
      pimpl_->stub_->WatchGameServer(&context, request);
  while (reader->Read(&gameServer)) {
    callback(gameServer);
  }
  return reader->Finish();
}

grpc::Status SDK::Shutdown() {
  grpc::ClientContext context;
  context.set_deadline(gpr_time_add(gpr_now(GPR_CLOCK_REALTIME),
                                    gpr_time_from_seconds(30, GPR_TIMESPAN)));
  agones::dev::sdk::Empty request;
  agones::dev::sdk::Empty response;

  return pimpl_->stub_->Shutdown(&context, request, &response);
}

grpc::Status SDK::SetLabel(std::string key, std::string value) {
  grpc::ClientContext context;
  context.set_deadline(gpr_time_add(gpr_now(GPR_CLOCK_REALTIME),
                                    gpr_time_from_seconds(30, GPR_TIMESPAN)));

  agones::dev::sdk::KeyValue request;
  request.set_key(std::move(key));
  request.set_value(std::move(value));

  agones::dev::sdk::Empty response;

  return pimpl_->stub_->SetLabel(&context, request, &response);
}

grpc::Status SDK::SetAnnotation(std::string key, std::string value) {
  grpc::ClientContext context;
  context.set_deadline(gpr_time_add(gpr_now(GPR_CLOCK_REALTIME),
                                    gpr_time_from_seconds(30, GPR_TIMESPAN)));

  agones::dev::sdk::KeyValue request;
  request.set_key(std::move(key));
  request.set_value(std::move(value));

  agones::dev::sdk::Empty response;

  return pimpl_->stub_->SetAnnotation(&context, request, &response);
}
}  // namespace agones
