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
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gateway "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/sdk/beta"
	"agones.dev/agones/pkg/util/runtime"
)

func stableTestConnection(t *testing.T) *grpc.ClientConn {
	t.Helper()
	runtime.FeatureTestMutex.Lock()
	t.Cleanup(runtime.FeatureTestMutex.Unlock)
	enabled := runtime.FeatureEnabled(runtime.FeatureCountsAndLists)
	t.Cleanup(func() {
		value := "false"
		if enabled {
			value = "true"
		}
		assert.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"="+value))
	})
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	local, err := NewLocalSDKServer("", "", defaultTestListMaxCapacity)
	require.NoError(t, err)
	t.Cleanup(local.Close)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	sdk.RegisterSDKServer(server, NewStableSDKServer(local))
	beta.RegisterSDKServer(server, local)
	go func() {
		assert.NoError(t, server.Serve(listener))
	}()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient("passthrough:///"+listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, conn.Close()) })
	return conn
}

func TestStableSDKServerCountersAndLists(t *testing.T) {
	t.Parallel()
	conn := stableTestConnection(t)
	stableClient, betaClient := sdk.NewSDKClient(conn), beta.NewSDKClient(conn)
	ctx := t.Context()

	gs, err := stableClient.GetGameServer(ctx, &sdk.Empty{})
	require.NoError(t, err)
	assert.Equal(t, "local", gs.ObjectMeta.Name)

	counter, err := stableClient.GetCounter(ctx, &sdk.GetCounterRequest{Name: "rooms"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), counter.Count)
	assert.Equal(t, int64(10), counter.Capacity)
	counter, err = stableClient.UpdateCounter(ctx, &sdk.UpdateCounterRequest{
		CounterUpdateRequest: &sdk.CounterUpdateRequest{Name: "rooms", Count: wrapperspb.Int64(0), Capacity: wrapperspb.Int64(20)},
	})
	require.NoError(t, err)
	assert.Zero(t, counter.Count)
	assert.Equal(t, int64(20), counter.Capacity)
	betaCounter, err := betaClient.GetCounter(ctx, &beta.GetCounterRequest{Name: "rooms"})
	require.NoError(t, err)
	assert.Zero(t, betaCounter.Count)
	_, err = betaClient.UpdateCounter(ctx, &beta.UpdateCounterRequest{
		CounterUpdateRequest: &beta.CounterUpdateRequest{Name: "rooms", CountDiff: 3},
	})
	require.NoError(t, err)
	counter, err = stableClient.UpdateCounter(ctx, &sdk.UpdateCounterRequest{
		CounterUpdateRequest: &sdk.CounterUpdateRequest{Name: "rooms", CountDiff: -1},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), counter.Count)
	assert.Equal(t, int64(20), counter.Capacity)

	list, err := stableClient.GetList(ctx, &sdk.GetListRequest{Name: "players"})
	require.NoError(t, err)
	assert.Equal(t, []string{"test0", "test1", "test2"}, list.Values)
	list, err = stableClient.UpdateList(ctx, &sdk.UpdateListRequest{
		List:       &sdk.List{Name: "players", Capacity: 4, Values: []string{"ignored"}},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity"}},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(4), list.Capacity)
	assert.Equal(t, []string{"test0", "test1", "test2"}, list.Values)
	_, err = stableClient.AddListValue(ctx, &sdk.AddListValueRequest{Name: "players", Value: "test3"})
	require.NoError(t, err)
	betaList, err := betaClient.GetList(ctx, &beta.GetListRequest{Name: "players"})
	require.NoError(t, err)
	assert.Equal(t, []string{"test0", "test1", "test2", "test3"}, betaList.Values)
	_, err = betaClient.RemoveListValue(ctx, &beta.RemoveListValueRequest{Name: "players", Value: "test0"})
	require.NoError(t, err)
	list, err = stableClient.RemoveListValue(ctx, &sdk.RemoveListValueRequest{Name: "players", Value: "test3"})
	require.NoError(t, err)
	assert.Equal(t, []string{"test1", "test2"}, list.Values)

	_, stableErr := stableClient.GetCounter(ctx, &sdk.GetCounterRequest{Name: "missing"})
	_, betaErr := betaClient.GetCounter(ctx, &beta.GetCounterRequest{Name: "missing"})
	require.Error(t, stableErr)
	require.Error(t, betaErr)
	assert.Equal(t, status.Code(betaErr), status.Code(stableErr))
	assert.Equal(t, status.Convert(betaErr).Message(), status.Convert(stableErr).Message())
}

func TestStableSDKServerRESTCountersAndLists(t *testing.T) {
	t.Parallel()
	conn := stableTestConnection(t)
	mux := gateway.NewServeMux()
	require.NoError(t, sdk.RegisterSDKHandler(t.Context(), mux, conn))
	require.NoError(t, beta.RegisterSDKHandler(t.Context(), mux, conn))

	requests := []struct {
		method, path, body string
		count, capacity    int64
		values             []string
		counter            bool
	}{
		{method: http.MethodGet, path: "/counters/rooms", count: 1, capacity: 10, counter: true},
		{method: http.MethodPatch, path: "/counters/rooms", body: `{"count":"0","capacity":"20"}`, count: 0, capacity: 20, counter: true},
		{method: http.MethodGet, path: "/v1beta1/counters/rooms", count: 0, capacity: 20, counter: true},
		{method: http.MethodPatch, path: "/v1beta1/counters/rooms", body: `{"countDiff":"2"}`, count: 2, capacity: 20, counter: true},
		{method: http.MethodGet, path: "/counters/rooms", count: 2, capacity: 20, counter: true},
		{method: http.MethodGet, path: "/lists/players", capacity: 100, values: []string{"test0", "test1", "test2"}},
		{method: http.MethodPatch, path: "/lists/players?updateMask=capacity", body: `{"capacity":"4","values":["ignored"]}`, capacity: 4, values: []string{"test0", "test1", "test2"}},
		{method: http.MethodPost, path: "/lists/players:addValue", body: `{"value":"test3"}`, capacity: 4, values: []string{"test0", "test1", "test2", "test3"}},
		{method: http.MethodGet, path: "/v1beta1/lists/players", capacity: 4, values: []string{"test0", "test1", "test2", "test3"}},
		{method: http.MethodPost, path: "/lists/players:removeValue", body: `{"value":"test0"}`, capacity: 4, values: []string{"test1", "test2", "test3"}},
		{method: http.MethodPatch, path: "/v1beta1/lists/players?updateMask=values", body: `{"values":[]}`, capacity: 4, values: []string{}},
		{method: http.MethodPost, path: "/v1beta1/lists/players:addValue", body: `{"value":"beta"}`, capacity: 4, values: []string{"beta"}},
		{method: http.MethodPost, path: "/v1beta1/lists/players:removeValue", body: `{"value":"beta"}`, capacity: 4, values: []string{}},
		{method: http.MethodGet, path: "/lists/players", capacity: 4, values: []string{}},
	}
	for _, testCase := range requests {
		t.Run(testCase.method+" "+testCase.path, func(t *testing.T) {
			request := httptest.NewRequestWithContext(t.Context(), testCase.method, testCase.path, strings.NewReader(testCase.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			if testCase.counter {
				result := &sdk.Counter{}
				require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), result))
				assert.Equal(t, "rooms", result.Name)
				assert.Equal(t, testCase.count, result.Count)
				assert.Equal(t, testCase.capacity, result.Capacity)
			} else {
				result := &sdk.List{}
				require.NoError(t, protojson.Unmarshal(response.Body.Bytes(), result))
				assert.Equal(t, "players", result.Name)
				assert.Equal(t, testCase.capacity, result.Capacity)
				if len(testCase.values) == 0 {
					assert.Empty(t, result.Values)
				} else {
					assert.Equal(t, testCase.values, result.Values)
				}
			}
		})
	}
}
