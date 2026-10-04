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

package sdkserver

import (
	"context"
	"slices"

	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/sdk/beta"
)

// sdkServer is implemented by both the local and Kubernetes SDK servers.
// Counter/List handlers retain their beta types while both API versions are served.
type sdkServer interface {
	beta.SDKServer
	Ready(context.Context, *sdk.Empty) (*sdk.Empty, error)
	Allocate(context.Context, *sdk.Empty) (*sdk.Empty, error)
	Shutdown(context.Context, *sdk.Empty) (*sdk.Empty, error)
	Health(sdk.SDK_HealthServer) error
	GetGameServer(context.Context, *sdk.Empty) (*sdk.GameServer, error)
	WatchGameServer(*sdk.Empty, sdk.SDK_WatchGameServerServer) error
	SetLabel(context.Context, *sdk.KeyValue) (*sdk.Empty, error)
	SetAnnotation(context.Context, *sdk.KeyValue) (*sdk.Empty, error)
	Reserve(context.Context, *sdk.Duration) (*sdk.Empty, error)
}

type stableSDKServer struct {
	sdkServer
}

var _ sdk.SDKServer = &stableSDKServer{}

// NewStableSDKServer exposes stable RPCs using the same handlers as the beta SDK.
// Go cannot implement the two services' identically named Counter/List methods
// on one type because their protobuf request and response types differ.
func NewStableSDKServer(server sdkServer) sdk.SDKServer {
	return &stableSDKServer{sdkServer: server}
}

func (s *stableSDKServer) GetCounter(ctx context.Context, in *sdk.GetCounterRequest) (*sdk.Counter, error) {
	out, err := s.sdkServer.GetCounter(ctx, &beta.GetCounterRequest{Name: in.GetName()})
	return stableCounter(out), err
}

func (s *stableSDKServer) UpdateCounter(ctx context.Context, in *sdk.UpdateCounterRequest) (*sdk.Counter, error) {
	request := &beta.UpdateCounterRequest{}
	if update := in.GetCounterUpdateRequest(); update != nil {
		request.CounterUpdateRequest = &beta.CounterUpdateRequest{
			Name: update.Name, Count: update.Count, Capacity: update.Capacity, CountDiff: update.CountDiff,
		}
	}
	out, err := s.sdkServer.UpdateCounter(ctx, request)
	return stableCounter(out), err
}

func (s *stableSDKServer) GetList(ctx context.Context, in *sdk.GetListRequest) (*sdk.List, error) {
	out, err := s.sdkServer.GetList(ctx, &beta.GetListRequest{Name: in.GetName()})
	return stableList(out), err
}

func (s *stableSDKServer) UpdateList(ctx context.Context, in *sdk.UpdateListRequest) (*sdk.List, error) {
	request := &beta.UpdateListRequest{UpdateMask: in.GetUpdateMask()}
	if list := in.GetList(); list != nil {
		request.List = &beta.List{Name: list.Name, Capacity: list.Capacity, Values: slices.Clone(list.Values)}
	}
	out, err := s.sdkServer.UpdateList(ctx, request)
	return stableList(out), err
}

func (s *stableSDKServer) AddListValue(ctx context.Context, in *sdk.AddListValueRequest) (*sdk.List, error) {
	out, err := s.sdkServer.AddListValue(ctx, &beta.AddListValueRequest{Name: in.GetName(), Value: in.GetValue()})
	return stableList(out), err
}

func (s *stableSDKServer) RemoveListValue(ctx context.Context, in *sdk.RemoveListValueRequest) (*sdk.List, error) {
	out, err := s.sdkServer.RemoveListValue(ctx, &beta.RemoveListValueRequest{Name: in.GetName(), Value: in.GetValue()})
	return stableList(out), err
}

func stableCounter(counter *beta.Counter) *sdk.Counter {
	if counter == nil {
		return nil
	}
	return &sdk.Counter{Name: counter.Name, Count: counter.Count, Capacity: counter.Capacity}
}

func stableList(list *beta.List) *sdk.List {
	if list == nil {
		return nil
	}
	return &sdk.List{Name: list.Name, Capacity: list.Capacity, Values: slices.Clone(list.Values)}
}
