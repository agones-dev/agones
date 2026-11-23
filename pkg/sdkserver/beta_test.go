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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/sdk/beta"
)

// recordingStableServer records the stable requests the beta adapter delegates, and returns
// canned responses. Methods not overridden here panic via the nil embedded interface.
type recordingStableServer struct {
	sdk.SDKServer
	requests []proto.Message
	counter  *sdk.Counter
	list     *sdk.List
	err      error
}

func (r *recordingStableServer) GetCounter(_ context.Context, in *sdk.GetCounterRequest) (*sdk.Counter, error) {
	r.requests = append(r.requests, in)
	return r.counter, r.err
}

func (r *recordingStableServer) UpdateCounter(_ context.Context, in *sdk.UpdateCounterRequest) (*sdk.Counter, error) {
	r.requests = append(r.requests, in)
	return r.counter, r.err
}

func (r *recordingStableServer) GetList(_ context.Context, in *sdk.GetListRequest) (*sdk.List, error) {
	r.requests = append(r.requests, in)
	return r.list, r.err
}

func (r *recordingStableServer) UpdateList(_ context.Context, in *sdk.UpdateListRequest) (*sdk.List, error) {
	r.requests = append(r.requests, in)
	return r.list, r.err
}

func (r *recordingStableServer) AddListValue(_ context.Context, in *sdk.AddListValueRequest) (*sdk.List, error) {
	r.requests = append(r.requests, in)
	return r.list, r.err
}

func (r *recordingStableServer) RemoveListValue(_ context.Context, in *sdk.RemoveListValueRequest) (*sdk.List, error) {
	r.requests = append(r.requests, in)
	return r.list, r.err
}

func TestBetaSDKServerConvertsCountersAndLists(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stableCounter := &sdk.Counter{Name: "rooms", Count: 3, Capacity: 10}
	stableList := &sdk.List{Name: "players", Capacity: 10, Values: []string{"a", "b"}}
	wantCounter := &beta.Counter{Name: "rooms", Count: 3, Capacity: 10}
	wantList := &beta.List{Name: "players", Capacity: 10, Values: []string{"a", "b"}}

	testCases := map[string]struct {
		call        func(beta.SDKServer) (proto.Message, error)
		wantRequest proto.Message
		want        proto.Message
	}{
		"GetCounter": {
			call: func(s beta.SDKServer) (proto.Message, error) {
				return s.GetCounter(ctx, &beta.GetCounterRequest{Name: "rooms"})
			},
			wantRequest: &sdk.GetCounterRequest{Name: "rooms"},
			want:        wantCounter,
		},
		"UpdateCounter count diff": {
			call: func(s beta.SDKServer) (proto.Message, error) {
				return s.UpdateCounter(ctx, &beta.UpdateCounterRequest{
					CounterUpdateRequest: &beta.CounterUpdateRequest{Name: "rooms", CountDiff: -2}})
			},
			// Count and Capacity must stay nil, as nil means "do not set".
			wantRequest: &sdk.UpdateCounterRequest{CounterUpdateRequest: &sdk.CounterUpdateRequest{Name: "rooms", CountDiff: -2}},
			want:        wantCounter,
		},
		"UpdateCounter count and capacity": {
			call: func(s beta.SDKServer) (proto.Message, error) {
				return s.UpdateCounter(ctx, &beta.UpdateCounterRequest{
					CounterUpdateRequest: &beta.CounterUpdateRequest{Name: "rooms", Count: wrapperspb.Int64(0), Capacity: wrapperspb.Int64(10)}})
			},
			wantRequest: &sdk.UpdateCounterRequest{CounterUpdateRequest: &sdk.CounterUpdateRequest{
				Name: "rooms", Count: wrapperspb.Int64(0), Capacity: wrapperspb.Int64(10)}},
			want: wantCounter,
		},
		"UpdateCounter nil request": {
			call: func(s beta.SDKServer) (proto.Message, error) {
				return s.UpdateCounter(ctx, &beta.UpdateCounterRequest{})
			},
			wantRequest: &sdk.UpdateCounterRequest{},
			want:        wantCounter,
		},
		"GetList": {
			call: func(s beta.SDKServer) (proto.Message, error) {
				return s.GetList(ctx, &beta.GetListRequest{Name: "players"})
			},
			wantRequest: &sdk.GetListRequest{Name: "players"},
			want:        wantList,
		},
		"UpdateList": {
			call: func(s beta.SDKServer) (proto.Message, error) {
				return s.UpdateList(ctx, &beta.UpdateListRequest{
					List:       &beta.List{Name: "players", Capacity: 5, Values: []string{"c"}},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity", "values"}},
				})
			},
			wantRequest: &sdk.UpdateListRequest{
				List:       &sdk.List{Name: "players", Capacity: 5, Values: []string{"c"}},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity", "values"}},
			},
			want: wantList,
		},
		"UpdateList nil list and mask": {
			call: func(s beta.SDKServer) (proto.Message, error) {
				return s.UpdateList(ctx, &beta.UpdateListRequest{})
			},
			wantRequest: &sdk.UpdateListRequest{},
			want:        wantList,
		},
		"AddListValue": {
			call: func(s beta.SDKServer) (proto.Message, error) {
				return s.AddListValue(ctx, &beta.AddListValueRequest{Name: "players", Value: "c"})
			},
			wantRequest: &sdk.AddListValueRequest{Name: "players", Value: "c"},
			want:        wantList,
		},
		"RemoveListValue": {
			call: func(s beta.SDKServer) (proto.Message, error) {
				return s.RemoveListValue(ctx, &beta.RemoveListValueRequest{Name: "players", Value: "a"})
			},
			wantRequest: &sdk.RemoveListValueRequest{Name: "players", Value: "a"},
			want:        wantList,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			r := &recordingStableServer{counter: stableCounter, list: stableList}
			got, err := tc.call(NewBetaSDKServer(r))
			require.NoError(t, err)
			require.Len(t, r.requests, 1)
			if diff := cmp.Diff(tc.wantRequest, r.requests[0], protocmp.Transform()); diff != "" {
				t.Errorf("unexpected delegated request (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("unexpected response (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBetaSDKServerReturnsErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	wantErr := errors.New("not found")
	s := NewBetaSDKServer(&recordingStableServer{err: wantErr})

	c, err := s.GetCounter(ctx, &beta.GetCounterRequest{Name: "missing"})
	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, c)

	c, err = s.UpdateCounter(ctx, &beta.UpdateCounterRequest{})
	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, c)

	l, err := s.GetList(ctx, &beta.GetListRequest{Name: "missing"})
	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, l)

	l, err = s.UpdateList(ctx, &beta.UpdateListRequest{})
	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, l)

	l, err = s.AddListValue(ctx, &beta.AddListValueRequest{Name: "missing"})
	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, l)

	l, err = s.RemoveListValue(ctx, &beta.RemoveListValueRequest{Name: "missing"})
	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, l)
}

func TestBetaSDKServerWithLocalSDKServer(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "beta"},
		Status: agonesv1.GameServerStatus{
			Counters: map[string]agonesv1.CounterStatus{"rooms": {Count: 1, Capacity: 10}},
			Lists:    map[string]agonesv1.ListStatus{"players": {Capacity: 10, Values: []string{"a", "b"}}},
		},
	}
	path, err := gsToTmpFile(fixture)
	require.NoError(t, err)
	l, err := NewLocalSDKServer(path, "", defaultTestListMaxCapacity)
	require.NoError(t, err)
	s := NewBetaSDKServer(l)

	// Counters
	counter, err := s.GetCounter(ctx, &beta.GetCounterRequest{Name: "rooms"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), counter.GetCount())
	assert.Equal(t, int64(10), counter.GetCapacity())

	counter, err = s.UpdateCounter(ctx, &beta.UpdateCounterRequest{
		CounterUpdateRequest: &beta.CounterUpdateRequest{Name: "rooms", CountDiff: 4}})
	require.NoError(t, err)
	assert.Equal(t, int64(5), counter.GetCount())

	counter, err = s.UpdateCounter(ctx, &beta.UpdateCounterRequest{
		CounterUpdateRequest: &beta.CounterUpdateRequest{Name: "rooms", Capacity: wrapperspb.Int64(20)}})
	require.NoError(t, err)
	assert.Equal(t, int64(20), counter.GetCapacity())

	_, err = s.UpdateCounter(ctx, &beta.UpdateCounterRequest{
		CounterUpdateRequest: &beta.CounterUpdateRequest{Name: "rooms", Count: wrapperspb.Int64(21)}})
	require.ErrorContains(t, err, "out of range")

	_, err = s.GetCounter(ctx, &beta.GetCounterRequest{Name: "missing"})
	require.ErrorContains(t, err, "not found")

	// The update made through the beta service is visible through the stable service.
	stableCounter, err := l.GetCounter(ctx, &sdk.GetCounterRequest{Name: "rooms"})
	require.NoError(t, err)
	assert.Equal(t, int64(5), stableCounter.GetCount())
	assert.Equal(t, int64(20), stableCounter.GetCapacity())

	// Lists
	list, err := s.GetList(ctx, &beta.GetListRequest{Name: "players"})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, list.GetValues())

	list, err = s.AddListValue(ctx, &beta.AddListValueRequest{Name: "players", Value: "c"})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, list.GetValues())

	_, err = s.AddListValue(ctx, &beta.AddListValueRequest{Name: "players", Value: "c"})
	require.ErrorContains(t, err, "already exists")

	list, err = s.RemoveListValue(ctx, &beta.RemoveListValueRequest{Name: "players", Value: "a"})
	require.NoError(t, err)
	assert.Equal(t, []string{"b", "c"}, list.GetValues())

	list, err = s.UpdateList(ctx, &beta.UpdateListRequest{
		List:       &beta.List{Name: "players", Capacity: 1},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity"}},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.GetCapacity())
	assert.Equal(t, []string{"b"}, list.GetValues())

	_, err = s.GetList(ctx, &beta.GetListRequest{Name: "missing"})
	require.ErrorContains(t, err, "not found")
}
