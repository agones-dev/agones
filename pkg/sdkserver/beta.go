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

var (
	_ beta.SDKServer = NewBetaSDKServer(&SDKServer{})
	_ beta.SDKServer = NewBetaSDKServer(&LocalSDKServer{})
)

// betaSDKServer serves the deprecated beta SDK service, by converting the beta request/response messages
// to and from the stable types, and delegating to the stable SDK service. Remove it along with the beta SDK.
type betaSDKServer struct {
	beta.UnimplementedSDKServer
	stable sdk.SDKServer
}

// NewBetaSDKServer returns an implementation of the beta SDK service backed by the stable SDK service s.
func NewBetaSDKServer(s sdk.SDKServer) beta.SDKServer {
	return &betaSDKServer{stable: s}
}

// GetCounter returns a Counter. Returns NOT_FOUND if the Counter does not exist.
func (b *betaSDKServer) GetCounter(ctx context.Context, in *beta.GetCounterRequest) (*beta.Counter, error) {
	c, err := b.stable.GetCounter(ctx, &sdk.GetCounterRequest{Name: in.GetName()})
	return toBetaCounter(c), err
}

// UpdateCounter updates a Counter, and returns the updated Counter.
func (b *betaSDKServer) UpdateCounter(ctx context.Context, in *beta.UpdateCounterRequest) (*beta.Counter, error) {
	req := &sdk.UpdateCounterRequest{}
	if u := in.GetCounterUpdateRequest(); u != nil {
		req.CounterUpdateRequest = &sdk.CounterUpdateRequest{
			Name:      u.GetName(),
			Count:     cloneInt64Value(u.GetCount()),
			Capacity:  cloneInt64Value(u.GetCapacity()),
			CountDiff: u.GetCountDiff(),
		}
	}
	c, err := b.stable.UpdateCounter(ctx, req)
	return toBetaCounter(c), err
}

// GetList returns a List. Returns NOT_FOUND if the List does not exist.
func (b *betaSDKServer) GetList(ctx context.Context, in *beta.GetListRequest) (*beta.List, error) {
	l, err := b.stable.GetList(ctx, &sdk.GetListRequest{Name: in.GetName()})
	return toBetaList(l), err
}

// UpdateList updates a List, and returns the updated List.
func (b *betaSDKServer) UpdateList(ctx context.Context, in *beta.UpdateListRequest) (*beta.List, error) {
	req := &sdk.UpdateListRequest{}
	if l := in.GetList(); l != nil {
		req.List = &sdk.List{
			Name:     l.GetName(),
			Capacity: l.GetCapacity(),
			Values:   append([]string(nil), l.GetValues()...),
		}
	}
	if m := in.GetUpdateMask(); m != nil {
		req.UpdateMask = &fieldmaskpb.FieldMask{Paths: append([]string(nil), m.GetPaths()...)}
	}
	l, err := b.stable.UpdateList(ctx, req)
	return toBetaList(l), err
}

// AddListValue adds a value to a List, and returns the updated List.
func (b *betaSDKServer) AddListValue(ctx context.Context, in *beta.AddListValueRequest) (*beta.List, error) {
	l, err := b.stable.AddListValue(ctx, &sdk.AddListValueRequest{Name: in.GetName(), Value: in.GetValue()})
	return toBetaList(l), err
}

// RemoveListValue removes a value from a List, and returns the updated List.
func (b *betaSDKServer) RemoveListValue(ctx context.Context, in *beta.RemoveListValueRequest) (*beta.List, error) {
	l, err := b.stable.RemoveListValue(ctx, &sdk.RemoveListValueRequest{Name: in.GetName(), Value: in.GetValue()})
	return toBetaList(l), err
}

// toBetaCounter converts a stable Counter to a beta Counter. Returns nil if c is nil.
func toBetaCounter(c *sdk.Counter) *beta.Counter {
	if c == nil {
		return nil
	}
	return &beta.Counter{Name: c.GetName(), Count: c.GetCount(), Capacity: c.GetCapacity()}
}

// toBetaList converts a stable List to a beta List. Returns nil if l is nil.
func toBetaList(l *sdk.List) *beta.List {
	if l == nil {
		return nil
	}
	return &beta.List{Name: l.GetName(), Capacity: l.GetCapacity(), Values: append([]string(nil), l.GetValues()...)}
}

// cloneInt64Value copies v, preserving nil, as nil has meaning in a CounterUpdateRequest.
func cloneInt64Value(v *wrapperspb.Int64Value) *wrapperspb.Int64Value {
	if v == nil {
		return nil
	}
	return wrapperspb.Int64(v.GetValue())
}
