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

	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/sdk/beta"
)

// stableServer is implemented by both SDKServer and LocalSDKServer. Their Counter and List
// handlers use the beta types, as Go cannot have two methods of the same name on one type.
type stableServer interface {
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

var (
	_ sdk.SDKServer = NewStableSDKServer(&SDKServer{})
	_ sdk.SDKServer = NewStableSDKServer(&LocalSDKServer{})
)

// stableSDKServer serves the stable SDK service, converting the Counter and List
// requests and responses to and from their beta equivalents.
type stableSDKServer struct {
	stableServer
}

// NewStableSDKServer returns an implementation of the stable SDK service backed by s.
func NewStableSDKServer(s stableServer) sdk.SDKServer {
	return &stableSDKServer{stableServer: s}
}

// GetCounter returns a Counter. Returns error if the counter does not exist.
func (s *stableSDKServer) GetCounter(ctx context.Context, in *sdk.GetCounterRequest) (*sdk.Counter, error) {
	c, err := s.stableServer.GetCounter(ctx, &beta.GetCounterRequest{Name: in.GetName()})
	return toStableCounter(c), err
}

// UpdateCounter updates the given Counter, and returns the updated Counter.
func (s *stableSDKServer) UpdateCounter(ctx context.Context, in *sdk.UpdateCounterRequest) (*sdk.Counter, error) {
	req := &beta.UpdateCounterRequest{}
	if u := in.GetCounterUpdateRequest(); u != nil {
		req.CounterUpdateRequest = &beta.CounterUpdateRequest{
			Name:      u.GetName(),
			Count:     u.GetCount(),
			Capacity:  u.GetCapacity(),
			CountDiff: u.GetCountDiff(),
		}
	}
	c, err := s.stableServer.UpdateCounter(ctx, req)
	return toStableCounter(c), err
}

// GetList returns a List. Returns error if the list does not exist.
func (s *stableSDKServer) GetList(ctx context.Context, in *sdk.GetListRequest) (*sdk.List, error) {
	l, err := s.stableServer.GetList(ctx, &beta.GetListRequest{Name: in.GetName()})
	return toStableList(l), err
}

// UpdateList updates the fields in the update mask of the given List, and returns the updated List.
func (s *stableSDKServer) UpdateList(ctx context.Context, in *sdk.UpdateListRequest) (*sdk.List, error) {
	req := &beta.UpdateListRequest{UpdateMask: in.GetUpdateMask()}
	if l := in.GetList(); l != nil {
		req.List = &beta.List{Name: l.GetName(), Capacity: l.GetCapacity(), Values: l.GetValues()}
	}
	l, err := s.stableServer.UpdateList(ctx, req)
	return toStableList(l), err
}

// AddListValue adds a value to the given List, and returns the updated List.
func (s *stableSDKServer) AddListValue(ctx context.Context, in *sdk.AddListValueRequest) (*sdk.List, error) {
	l, err := s.stableServer.AddListValue(ctx, &beta.AddListValueRequest{Name: in.GetName(), Value: in.GetValue()})
	return toStableList(l), err
}

// RemoveListValue removes a value from the given List, and returns the updated List.
func (s *stableSDKServer) RemoveListValue(ctx context.Context, in *sdk.RemoveListValueRequest) (*sdk.List, error) {
	l, err := s.stableServer.RemoveListValue(ctx, &beta.RemoveListValueRequest{Name: in.GetName(), Value: in.GetValue()})
	return toStableList(l), err
}

func toStableCounter(c *beta.Counter) *sdk.Counter {
	if c == nil {
		return nil
	}
	return &sdk.Counter{Name: c.GetName(), Count: c.GetCount(), Capacity: c.GetCapacity()}
}

func toStableList(l *beta.List) *sdk.List {
	if l == nil {
		return nil
	}
	return &sdk.List{Name: l.GetName(), Capacity: l.GetCapacity(), Values: l.GetValues()}
}
