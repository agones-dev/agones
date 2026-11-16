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

	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/sdk/beta"
)

// stableServer is the subset of the stable SDK service implemented by both SDKServer and LocalSDKServer,
// plus their Counter and List handlers. Those handlers use the beta types, as Go cannot have two methods
// of the same name on one type, so stableSDKServer converts between the stable and beta types.
type stableServer interface {
	Ready(context.Context, *sdk.Empty) (*sdk.Empty, error)
	Allocate(context.Context, *sdk.Empty) (*sdk.Empty, error)
	Shutdown(context.Context, *sdk.Empty) (*sdk.Empty, error)
	Health(sdk.SDK_HealthServer) error
	GetGameServer(context.Context, *sdk.Empty) (*sdk.GameServer, error)
	WatchGameServer(*sdk.Empty, sdk.SDK_WatchGameServerServer) error
	SetLabel(context.Context, *sdk.KeyValue) (*sdk.Empty, error)
	SetAnnotation(context.Context, *sdk.KeyValue) (*sdk.Empty, error)
	Reserve(context.Context, *sdk.Duration) (*sdk.Empty, error)

	GetCounter(context.Context, *beta.GetCounterRequest) (*beta.Counter, error)
	UpdateCounter(context.Context, *beta.UpdateCounterRequest) (*beta.Counter, error)
	GetList(context.Context, *beta.GetListRequest) (*beta.List, error)
	UpdateList(context.Context, *beta.UpdateListRequest) (*beta.List, error)
	AddListValue(context.Context, *beta.AddListValueRequest) (*beta.List, error)
	RemoveListValue(context.Context, *beta.RemoveListValueRequest) (*beta.List, error)
}

var (
	_ sdk.SDKServer = NewStableSDKServer(&SDKServer{})
	_ sdk.SDKServer = NewStableSDKServer(&LocalSDKServer{})
)

// unimplementedSDKServer is nested one level deeper than stableServer, so it only supplies the
// methods neither stableServer nor stableSDKServer defines (forward compatibility for new RPCs).
type unimplementedSDKServer struct {
	sdk.UnimplementedSDKServer
}

// stableSDKServer serves the stable SDK service. The Counter and List RPCs defined on it shadow the
// beta-typed methods promoted from stableServer, and delegate to them, so the stable and beta
// services share a single implementation.
type stableSDKServer struct {
	unimplementedSDKServer
	stableServer
}

// NewStableSDKServer returns an implementation of the stable SDK service backed by s.
func NewStableSDKServer(s stableServer) sdk.SDKServer {
	return &stableSDKServer{stableServer: s}
}

// GetCounter returns a Counter. Returns NOT_FOUND if the Counter does not exist.
func (s *stableSDKServer) GetCounter(ctx context.Context, in *sdk.GetCounterRequest) (*sdk.Counter, error) {
	c, err := s.stableServer.GetCounter(ctx, &beta.GetCounterRequest{Name: in.GetName()})
	return toStableCounter(c), err
}

// UpdateCounter updates a Counter, and returns the updated Counter.
func (s *stableSDKServer) UpdateCounter(ctx context.Context, in *sdk.UpdateCounterRequest) (*sdk.Counter, error) {
	req := &beta.UpdateCounterRequest{}
	if u := in.GetCounterUpdateRequest(); u != nil {
		req.CounterUpdateRequest = &beta.CounterUpdateRequest{
			Name:      u.GetName(),
			Count:     cloneInt64Value(u.GetCount()),
			Capacity:  cloneInt64Value(u.GetCapacity()),
			CountDiff: u.GetCountDiff(),
		}
	}
	c, err := s.stableServer.UpdateCounter(ctx, req)
	return toStableCounter(c), err
}

// GetList returns a List. Returns NOT_FOUND if the List does not exist.
func (s *stableSDKServer) GetList(ctx context.Context, in *sdk.GetListRequest) (*sdk.List, error) {
	l, err := s.stableServer.GetList(ctx, &beta.GetListRequest{Name: in.GetName()})
	return toStableList(l), err
}

// UpdateList updates a List, and returns the updated List.
func (s *stableSDKServer) UpdateList(ctx context.Context, in *sdk.UpdateListRequest) (*sdk.List, error) {
	req := &beta.UpdateListRequest{}
	if l := in.GetList(); l != nil {
		req.List = &beta.List{
			Name:     l.GetName(),
			Capacity: l.GetCapacity(),
			Values:   append([]string(nil), l.GetValues()...),
		}
	}
	if m := in.GetUpdateMask(); m != nil {
		req.UpdateMask = &fieldmaskpb.FieldMask{Paths: append([]string(nil), m.GetPaths()...)}
	}
	l, err := s.stableServer.UpdateList(ctx, req)
	return toStableList(l), err
}

// AddListValue adds a value to a List, and returns the updated List.
func (s *stableSDKServer) AddListValue(ctx context.Context, in *sdk.AddListValueRequest) (*sdk.List, error) {
	l, err := s.stableServer.AddListValue(ctx, &beta.AddListValueRequest{Name: in.GetName(), Value: in.GetValue()})
	return toStableList(l), err
}

// RemoveListValue removes a value from a List, and returns the updated List.
func (s *stableSDKServer) RemoveListValue(ctx context.Context, in *sdk.RemoveListValueRequest) (*sdk.List, error) {
	l, err := s.stableServer.RemoveListValue(ctx, &beta.RemoveListValueRequest{Name: in.GetName(), Value: in.GetValue()})
	return toStableList(l), err
}

// toStableCounter converts a beta Counter to a stable Counter. Returns nil if c is nil.
func toStableCounter(c *beta.Counter) *sdk.Counter {
	if c == nil {
		return nil
	}
	return &sdk.Counter{Name: c.GetName(), Count: c.GetCount(), Capacity: c.GetCapacity()}
}

// toStableList converts a beta List to a stable List. Returns nil if l is nil.
func toStableList(l *beta.List) *sdk.List {
	if l == nil {
		return nil
	}
	return &sdk.List{Name: l.GetName(), Capacity: l.GetCapacity(), Values: append([]string(nil), l.GetValues()...)}
}

// cloneInt64Value copies v, preserving nil, as nil has meaning in a CounterUpdateRequest.
func cloneInt64Value(v *wrapperspb.Int64Value) *wrapperspb.Int64Value {
	if v == nil {
		return nil
	}
	return wrapperspb.Int64(v.GetValue())
}
