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
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/sdk/alpha"
	"agones.dev/agones/pkg/sdk/beta"
	"agones.dev/agones/pkg/util/runtime"
)

func TestLocal(t *testing.T) {
	ctx := context.Background()
	e := &sdk.Empty{}
	l, err := NewLocalSDKServer("", "")
	assert.Nil(t, err)

	_, err = l.Ready(ctx, e)
	assert.Nil(t, err, "Ready should not error")

	_, err = l.Shutdown(ctx, e)
	assert.Nil(t, err, "Shutdown should not error")

	wg := sync.WaitGroup{}
	wg.Add(1)
	stream := newEmptyMockStream()

	go func() {
		err = l.Health(stream)
		assert.Nil(t, err)
		wg.Done()
	}()

	stream.msgs <- &sdk.Empty{}
	close(stream.msgs)

	wg.Wait()

	gs, err := l.GetGameServer(ctx, e)
	assert.Nil(t, err)

	defaultGameServer := defaultGs()
	// do this to adjust for any time differences.
	// we only care about all the other values to be compared.
	defaultGameServer.ObjectMeta.CreationTimestamp = gs.GetObjectMeta().CreationTimestamp

	assert.Equal(t, defaultGameServer.GetObjectMeta(), gs.GetObjectMeta())
	assert.Equal(t, defaultGameServer.GetSpec(), gs.GetSpec())
	gsStatus := defaultGameServer.GetStatus()
	gsStatus.State = "Shutdown"
	assert.Equal(t, gsStatus, gs.GetStatus())
}

func TestLocalSDKWithTestMode(t *testing.T) {
	l, err := NewLocalSDKServer("", "")
	assert.NoError(t, err, "Should be able to create local SDK server")
	a := []string{"ready", "allocate", "setlabel", "setannotation", "gameserver", "health", "shutdown", "watch"}
	b := []string{"ready", "health", "ready", "watch", "allocate", "gameserver", "setlabel", "setannotation", "health", "health", "shutdown"}
	assert.True(t, l.EqualSets(a, a))
	assert.True(t, l.EqualSets(a, b))
	assert.True(t, l.EqualSets(b, a))
	assert.True(t, l.EqualSets(b, b))
	a[0] = "rady"
	assert.False(t, l.EqualSets(a, b))
	assert.False(t, l.EqualSets(b, a))
	a[0] = "ready"
	b[1] = "halth"
	assert.False(t, l.EqualSets(a, b))
	assert.False(t, l.EqualSets(b, a))
}

func TestLocalSDKWithGameServer(t *testing.T) {
	ctx := context.Background()
	e := &sdk.Empty{}

	fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "stuff"}}
	path, err := gsToTmpFile(fixture.DeepCopy())
	assert.Nil(t, err)

	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	gs, err := l.GetGameServer(ctx, e)
	assert.Nil(t, err)

	assert.Equal(t, fixture.ObjectMeta.Name, gs.ObjectMeta.Name)
}

// nolint:dupl
func TestLocalSDKWithLogLevel(t *testing.T) {
	ctx := context.Background()
	e := &sdk.Empty{}

	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Spec: agonesv1.GameServerSpec{
			SdkServer: agonesv1.SdkServer{LogLevel: "debug"},
		},
	}
	path, err := gsToTmpFile(fixture.DeepCopy())
	assert.Nil(t, err)

	l, err := NewLocalSDKServer(path, "test")
	assert.Nil(t, err)

	_, err = l.GetGameServer(ctx, e)
	assert.Nil(t, err)

	// Check if the LocalSDKServer's logger.LogLevel equal fixture's
	assert.Equal(t, string(fixture.Spec.SdkServer.LogLevel), l.logger.Logger.Level.String())
}

// nolint:dupl
func TestLocalSDKServerSetLabel(t *testing.T) {
	t.Parallel()

	fixtures := map[string]struct {
		gs *agonesv1.GameServer
	}{
		"default": {
			gs: nil,
		},
		"no labels": {
			gs: &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "empty"}},
		},
		"empty": {
			gs: &agonesv1.GameServer{},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			ctx := context.Background()
			e := &sdk.Empty{}
			path, err := gsToTmpFile(v.gs)
			assert.Nil(t, err)

			l, err := NewLocalSDKServer(path, "")
			assert.Nil(t, err)
			kv := &sdk.KeyValue{Key: "foo", Value: "bar"}

			stream := newGameServerMockStream()
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := l.WatchGameServer(e, stream)
				assert.Nil(t, err)
			}()
			assertInitialWatchUpdate(t, stream)

			// make sure length of l.updateObservers is at least 1
			err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
				ret := false
				l.updateObservers.Range(func(_, _ interface{}) bool {
					ret = true
					return false
				})

				return ret, nil
			})
			assert.Nil(t, err)

			_, err = l.SetLabel(ctx, kv)
			assert.Nil(t, err)

			gs, err := l.GetGameServer(ctx, e)
			assert.Nil(t, err)
			assert.Equal(t, gs.ObjectMeta.Labels[metadataPrefix+"foo"], "bar")

			assertWatchUpdate(t, stream, "bar", func(gs *sdk.GameServer) interface{} {
				return gs.ObjectMeta.Labels[metadataPrefix+"foo"]
			})

			l.Close()
			wg.Wait()
		})
	}
}

// nolint:dupl
func TestLocalSDKServerSetAnnotation(t *testing.T) {
	t.Parallel()

	fixtures := map[string]struct {
		gs *agonesv1.GameServer
	}{
		"default": {
			gs: nil,
		},
		"no annotation": {
			gs: &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "empty"}},
		},
		"empty": {
			gs: &agonesv1.GameServer{},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			ctx := context.Background()
			e := &sdk.Empty{}
			path, err := gsToTmpFile(v.gs)
			assert.Nil(t, err)

			l, err := NewLocalSDKServer(path, "")
			assert.Nil(t, err)

			kv := &sdk.KeyValue{Key: "bar", Value: "foo"}

			stream := newGameServerMockStream()
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := l.WatchGameServer(e, stream)
				assert.Nil(t, err)
			}()
			assertInitialWatchUpdate(t, stream)

			// make sure length of l.updateObservers is at least 1
			err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
				ret := false
				l.updateObservers.Range(func(_, _ interface{}) bool {
					ret = true
					return false
				})

				return ret, nil
			})
			assert.Nil(t, err)

			_, err = l.SetAnnotation(ctx, kv)
			assert.Nil(t, err)

			gs, err := l.GetGameServer(ctx, e)
			assert.Nil(t, err)
			assert.Equal(t, gs.ObjectMeta.Annotations[metadataPrefix+"bar"], "foo")

			assertWatchUpdate(t, stream, "foo", func(gs *sdk.GameServer) interface{} {
				return gs.ObjectMeta.Annotations[metadataPrefix+"bar"]
			})

			l.Close()
			wg.Wait()
		})
	}
}

func TestLocalSDKServerWatchGameServer(t *testing.T) {
	t.Parallel()

	fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "stuff"}}
	path, err := gsToTmpFile(fixture)
	assert.Nil(t, err)

	e := &sdk.Empty{}
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(e, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

	// wait for watching to begin
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
		found := false
		l.updateObservers.Range(func(_, _ interface{}) bool {
			found = true
			return false
		})
		return found, nil
	})
	assert.NoError(t, err)

	assertNoWatchUpdate(t, stream)
	fixture.ObjectMeta.Annotations = map[string]string{"foo": "bar"}
	j, err := json.Marshal(fixture)
	assert.Nil(t, err)

	err = os.WriteFile(path, j, os.ModeDevice)
	assert.Nil(t, err)

	assertWatchUpdate(t, stream, "bar", func(gs *sdk.GameServer) interface{} {
		return gs.ObjectMeta.Annotations["foo"]
	})
}

func TestLocalSDKServerPlayerCapacity(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeaturePlayerTracking)+"=true"))

	fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "stuff"}}

	e := &alpha.Empty{}
	path, err := gsToTmpFile(fixture)
	assert.NoError(t, err)
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

	// wait for watching to begin
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
		found := false
		l.updateObservers.Range(func(_, _ interface{}) bool {
			found = true
			return false
		})
		return found, nil
	})
	assert.NoError(t, err)

	c, err := l.GetPlayerCapacity(context.Background(), e)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), c.Count)

	_, err = l.SetPlayerCapacity(context.Background(), &alpha.Count{Count: 10})
	assert.NoError(t, err)

	select {
	case msg := <-stream.msgs:
		assert.Equal(t, int64(10), msg.Status.Players.Capacity)
	case <-time.After(10 * time.Second):
		assert.Fail(t, "timeout getting watch")
	}

	c, err = l.GetPlayerCapacity(context.Background(), e)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), c.Count)

	gs, err := l.GetGameServer(context.Background(), &sdk.Empty{})
	assert.NoError(t, err)
	assert.Equal(t, int64(10), gs.Status.Players.Capacity)
}

func TestLocalSDKServerPlayerConnectAndDisconnectWithoutPlayerTracking(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	require.NoError(t, runtime.ParseFeatures(string(runtime.FeaturePlayerTracking)+"=false"))

	l, err := NewLocalSDKServer("", "")
	assert.Nil(t, err)

	e := &alpha.Empty{}
	capacity, err := l.GetPlayerCapacity(context.Background(), e)
	assert.Nil(t, capacity)
	assert.Error(t, err)

	count, err := l.GetPlayerCount(context.Background(), e)
	assert.Error(t, err)
	assert.Nil(t, count)

	list, err := l.GetConnectedPlayers(context.Background(), e)
	assert.Error(t, err)
	assert.Nil(t, list)

	id := &alpha.PlayerID{PlayerID: "test-player"}

	ok, err := l.PlayerConnect(context.Background(), id)
	assert.Error(t, err)
	assert.False(t, ok.Bool)

	ok, err = l.IsPlayerConnected(context.Background(), id)
	assert.Error(t, err)
	assert.False(t, ok.Bool)

	ok, err = l.PlayerDisconnect(context.Background(), id)
	assert.Error(t, err)
	assert.False(t, ok.Bool)
}

func TestLocalSDKServerPlayerConnectAndDisconnect(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeaturePlayerTracking)+"=true"))

	gs := func() *agonesv1.GameServer {
		return &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
			Status: agonesv1.GameServerStatus{
				Players: &agonesv1.PlayerStatus{
					Capacity: 1,
				},
			}}
	}

	e := &alpha.Empty{}

	fixtures := map[string]struct {
		testMode bool
		gs       *agonesv1.GameServer
		useFile  bool
	}{
		"test mode on, gs with Status.Players": {
			testMode: true,
			gs:       gs(),
			useFile:  true,
		},
		"test mode off, gs with Status.Players": {
			testMode: false,
			gs:       gs(),
			useFile:  true,
		},
		"test mode on, gs without Status.Players": {
			testMode: true,
			useFile:  true,
		},
		"test mode off, gs without Status.Players": {
			testMode: false,
			useFile:  true,
		},
		"test mode on, no filePath": {
			testMode: true,
			useFile:  false,
		},
		"test mode off, no filePath": {
			testMode: false,
			useFile:  false,
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			var l *LocalSDKServer
			var err error
			if v.useFile {
				path, pathErr := gsToTmpFile(v.gs)
				assert.NoError(t, pathErr)
				l, err = NewLocalSDKServer(path, "")
			} else {
				l, err = NewLocalSDKServer("", "")
			}
			assert.Nil(t, err)
			l.SetTestMode(v.testMode)

			stream := newGameServerMockStream()
			go func() {
				err := l.WatchGameServer(&sdk.Empty{}, stream)
				assert.Nil(t, err)
			}()
			assertInitialWatchUpdate(t, stream)

			// wait for watching to begin
			err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
				found := false
				l.updateObservers.Range(func(_, _ interface{}) bool {
					found = true
					return false
				})
				return found, nil
			})
			assert.NoError(t, err)

			if !v.useFile || v.gs == nil {
				_, err := l.SetPlayerCapacity(context.Background(), &alpha.Count{
					Count: 1,
				})
				assert.NoError(t, err)
				expected := &sdk.GameServer_Status_PlayerStatus{
					Capacity: 1,
				}
				assertWatchUpdate(t, stream, expected, func(gs *sdk.GameServer) interface{} {
					return gs.Status.Players
				})
			}

			id := &alpha.PlayerID{PlayerID: "one"}
			ok, err := l.IsPlayerConnected(context.Background(), id)
			assert.NoError(t, err)
			if assert.NotNil(t, ok) {
				assert.False(t, ok.Bool, "player should not be connected")
			}

			count, err := l.GetPlayerCount(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, int64(0), count.Count)

			list, err := l.GetConnectedPlayers(context.Background(), e)
			assert.NoError(t, err)
			assert.Empty(t, list.List)

			// connect a player
			ok, err = l.PlayerConnect(context.Background(), id)
			assert.NoError(t, err)
			assert.True(t, ok.Bool, "Player should not exist yet")

			count, err = l.GetPlayerCount(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, int64(1), count.Count)

			expected := &sdk.GameServer_Status_PlayerStatus{
				Count:    1,
				Capacity: 1,
				Ids:      []string{id.PlayerID},
			}
			assertWatchUpdate(t, stream, expected, func(gs *sdk.GameServer) interface{} {
				return gs.Status.Players
			})

			ok, err = l.IsPlayerConnected(context.Background(), id)
			assert.NoError(t, err)
			assert.True(t, ok.Bool, "player should be connected")

			list, err = l.GetConnectedPlayers(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, []string{id.PlayerID}, list.List)

			// add same player
			ok, err = l.PlayerConnect(context.Background(), id)
			assert.NoError(t, err)
			assert.False(t, ok.Bool, "Player already exists")

			count, err = l.GetPlayerCount(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, int64(1), count.Count)
			assertNoWatchUpdate(t, stream)

			list, err = l.GetConnectedPlayers(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, []string{id.PlayerID}, list.List)

			// should return an error if we try to add another, since we're at capacity
			nopePlayer := &alpha.PlayerID{PlayerID: "nope"}
			_, err = l.PlayerConnect(context.Background(), nopePlayer)
			assert.EqualError(t, err, "Players are already at capacity")

			ok, err = l.IsPlayerConnected(context.Background(), nopePlayer)
			assert.NoError(t, err)
			assert.False(t, ok.Bool)

			// disconnect a player
			ok, err = l.PlayerDisconnect(context.Background(), id)
			assert.NoError(t, err)
			assert.True(t, ok.Bool, "Player should be removed")
			count, err = l.GetPlayerCount(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, int64(0), count.Count)

			expected = &sdk.GameServer_Status_PlayerStatus{
				Count:    0,
				Capacity: 1,
				Ids:      []string{},
			}
			assertWatchUpdate(t, stream, expected, func(gs *sdk.GameServer) interface{} {
				return gs.Status.Players
			})

			list, err = l.GetConnectedPlayers(context.Background(), e)
			assert.NoError(t, err)
			assert.Empty(t, list.List)

			// remove same player
			ok, err = l.PlayerDisconnect(context.Background(), id)
			assert.NoError(t, err)
			assert.False(t, ok.Bool, "Player already be gone")
			count, err = l.GetPlayerCount(context.Background(), e)
			assert.NoError(t, err)
			assert.Equal(t, int64(0), count.Count)
			assertNoWatchUpdate(t, stream)
		})
	}
}

func TestLocalSDKServerGetCounter(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	counters := map[string]agonesv1.CounterStatus{
		"sessions": {Count: int64(1), Capacity: int64(100)},
	}
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Status: agonesv1.GameServerStatus{
			Counters: counters,
		},
	}

	path, err := gsToTmpFile(fixture)
	assert.NoError(t, err)
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

	// wait for watching to begin
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
		found := false
		l.updateObservers.Range(func(_, _ interface{}) bool {
			found = true
			return false
		})
		return found, nil
	})
	assert.NoError(t, err)

	testScenarios := map[string]struct {
		name    string
		want    *beta.Counter
		wantErr error
	}{
		"Counter exists": {
			name: "sessions",
			want: &beta.Counter{Name: "sessions", Count: int64(1), Capacity: int64(100)},
		},
		"Counter does not exist": {
			name:    "noName",
			wantErr: errors.Errorf("not found. %s Counter not found", "noName"),
		},
	}

	for testName, testScenario := range testScenarios {
		t.Run(testName, func(t *testing.T) {
			got, err := l.GetCounter(context.Background(), &beta.GetCounterRequest{Name: testScenario.name})
			// Check tests expecting non-errors
			if testScenario.want != nil {
				assert.NoError(t, err)
				if diff := cmp.Diff(testScenario.want, got, protocmp.Transform()); diff != "" {
					t.Errorf("Unexpected difference:\n%v", diff)
				}
			} else {
				// Check tests expecting errors
				assert.EqualError(t, err, testScenario.wantErr.Error())
			}
		})
	}
}

func TestLocalSDKServerUpdateCounter(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	counters := map[string]agonesv1.CounterStatus{
		"sessions": {Count: 1, Capacity: 100},
		"players":  {Count: 100, Capacity: 100},
		"lobbies":  {Count: 0, Capacity: 0},
		"games":    {Count: 5, Capacity: 10},
		"npcs":     {Count: 6, Capacity: 10},
	}
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Status: agonesv1.GameServerStatus{
			Counters: counters,
		},
	}

	path, err := gsToTmpFile(fixture)
	assert.NoError(t, err)
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

	// wait for watching to begin
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
		found := false
		l.updateObservers.Range(func(_, _ interface{}) bool {
			found = true
			return false
		})
		return found, nil
	})
	assert.NoError(t, err)

	testScenarios := map[string]struct {
		updateRequest *beta.UpdateCounterRequest
		want          *beta.Counter
		wantErr       error
	}{
		"Set Counter Capacity": {
			updateRequest: &beta.UpdateCounterRequest{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:     "lobbies",
					Capacity: wrapperspb.Int64(10),
				}},
			want: &beta.Counter{
				Name: "lobbies", Count: 0, Capacity: 10,
			},
		},
		"Set Counter Count": {
			updateRequest: &beta.UpdateCounterRequest{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:  "npcs",
					Count: wrapperspb.Int64(10),
				}},
			want: &beta.Counter{
				Name: "npcs", Count: 10, Capacity: 10,
			},
		},
		"Decrement Counter Count": {
			updateRequest: &beta.UpdateCounterRequest{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:      "games",
					CountDiff: -5,
				}},
			want: &beta.Counter{
				Name: "games", Count: 0, Capacity: 10,
			},
		},
		"Cannot Decrement Counter": {
			updateRequest: &beta.UpdateCounterRequest{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:      "sessions",
					CountDiff: -2,
				}},
			wantErr: errors.Errorf("out of range. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", -1, 100),
		},
		"Cannot Increment Counter": {
			updateRequest: &beta.UpdateCounterRequest{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:      "players",
					CountDiff: 1,
				}},
			wantErr: errors.Errorf("out of range. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", 101, 100),
		},
		"Counter does not exist": {
			updateRequest: &beta.UpdateCounterRequest{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:      "dragons",
					CountDiff: 1,
				}},
			wantErr: errors.Errorf("not found. %s Counter not found", "dragons"),
		},
		"request Counter is nil": {
			updateRequest: &beta.UpdateCounterRequest{
				CounterUpdateRequest: nil,
			},
			wantErr: errors.Errorf("invalid argument. CounterUpdateRequest cannot be nil"),
		},
		"capacity is less than zero": {
			updateRequest: &beta.UpdateCounterRequest{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:     "lobbies",
					Capacity: wrapperspb.Int64(-1),
				}},
			wantErr: errors.Errorf("out of range. Capacity must be greater than or equal to 0. Found Capacity: %d", -1),
		},
		"count is less than zero": {
			updateRequest: &beta.UpdateCounterRequest{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:  "players",
					Count: wrapperspb.Int64(-1),
				}},
			wantErr: errors.Errorf("out of range. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", -1, 100),
		},
		"count is greater than capacity": {
			updateRequest: &beta.UpdateCounterRequest{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:  "players",
					Count: wrapperspb.Int64(101),
				}},
			wantErr: errors.Errorf("out of range. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d", 101, 100),
		},
	}

	for testName, testScenario := range testScenarios {
		t.Run(testName, func(t *testing.T) {
			got, err := l.UpdateCounter(context.Background(), testScenario.updateRequest)
			// Check tests expecting non-errors
			if testScenario.want != nil {
				assert.NoError(t, err)
				if diff := cmp.Diff(testScenario.want, got, protocmp.Transform()); diff != "" {
					t.Errorf("Unexpected difference:\n%v", diff)
				}
			} else {
				// Check tests expecting errors
				assert.ErrorContains(t, err, testScenario.wantErr.Error())
			}
		})
	}
}

func TestLocalSDKServerGetList(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	lists := map[string]agonesv1.ListStatus{
		"games": {Capacity: int64(100), Values: []string{"game1", "game2"}},
	}
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Status: agonesv1.GameServerStatus{
			Lists: lists,
		},
	}

	path, err := gsToTmpFile(fixture)
	assert.NoError(t, err)
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

	// wait for watching to begin
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
		found := false
		l.updateObservers.Range(func(_, _ interface{}) bool {
			found = true
			return false
		})
		return found, nil
	})
	assert.NoError(t, err)

	testScenarios := map[string]struct {
		name    string
		want    *beta.List
		wantErr error
	}{
		"List exists": {
			name: "games",
			want: &beta.List{Name: "games", Capacity: int64(100), Values: []string{"game1", "game2"}},
		},
		"List does not exist": {
			name:    "noName",
			wantErr: errors.Errorf("not found. %s List not found", "noName"),
		},
	}

	for testName, testScenario := range testScenarios {
		t.Run(testName, func(t *testing.T) {
			got, err := l.GetList(context.Background(), &beta.GetListRequest{Name: testScenario.name})
			// Check tests expecting non-errors
			if testScenario.want != nil {
				assert.NoError(t, err)
				if diff := cmp.Diff(testScenario.want, got, protocmp.Transform()); diff != "" {
					t.Errorf("Unexpected difference:\n%v", diff)
				}
			} else {
				// Check tests expecting errors
				assert.EqualError(t, err, testScenario.wantErr.Error())
			}
		})
	}
}

func TestLocalSDKServerUpdateList(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	lists := map[string]agonesv1.ListStatus{
		"players":  {Capacity: 1000},
		"games":    {Capacity: 100, Values: []string{"game1", "game2"}},
		"unicorns": {Capacity: 1000, Values: []string{"unicorn1", "unicorn2"}},
		"clients":  {Capacity: 10, Values: []string{}},
		"assets":   {Capacity: 1, Values: []string{"asset1"}},
		"models":   {Capacity: 11, Values: []string{"model1", "model2"}},
	}
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Status: agonesv1.GameServerStatus{
			Lists: lists,
		},
	}

	path, err := gsToTmpFile(fixture)
	assert.NoError(t, err)
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

	// wait for watching to begin
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
		found := false
		l.updateObservers.Range(func(_, _ interface{}) bool {
			found = true
			return false
		})
		return found, nil
	})
	assert.NoError(t, err)

	testScenarios := map[string]struct {
		updateRequest *beta.UpdateListRequest
		want          *beta.List
		wantErr       error
	}{
		"only updates fields in the FieldMask": {
			updateRequest: &beta.UpdateListRequest{
				List: &beta.List{
					Name:     "games",
					Capacity: int64(999),
					Values:   []string{"game3"},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity"}},
			},
			want: &beta.List{
				Name:     "games",
				Capacity: int64(999),
				Values:   []string{"game1", "game2"},
			},
		},
		"updates both fields in the FieldMask": {
			updateRequest: &beta.UpdateListRequest{
				List: &beta.List{
					Name:     "unicorns",
					Capacity: int64(42),
					Values:   []string{"unicorn0"},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"values", "capacity"}},
			},
			want: &beta.List{
				Name:     "unicorns",
				Capacity: int64(42),
				Values:   []string{"unicorn0"},
			},
		},
		"default value for Capacity applied": {
			updateRequest: &beta.UpdateListRequest{
				List: &beta.List{
					Name: "clients",
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity"}},
			},
			want: &beta.List{
				Name:     "clients",
				Capacity: int64(0),
				Values:   []string{},
			},
		},
		"default value for Values applied": {
			updateRequest: &beta.UpdateListRequest{
				List: &beta.List{
					Name: "assets",
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"values"}},
			},
			want: &beta.List{
				Name:     "assets",
				Capacity: int64(1),
				Values:   []string{},
			},
		},
		"List does not exist": {
			updateRequest: &beta.UpdateListRequest{
				List: &beta.List{
					Name: "dragons",
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity"}},
			},
			wantErr: errors.Errorf("not found. %s List not found", "dragons"),
		},
		"request List is nil": {
			updateRequest: &beta.UpdateListRequest{
				List:       nil,
				UpdateMask: &fieldmaskpb.FieldMask{},
			},
			wantErr: errors.Errorf("invalid argument. List: %v and UpdateMask %v cannot be nil", nil, &fieldmaskpb.FieldMask{}),
		},
		"request UpdateMask is nil": {
			updateRequest: &beta.UpdateListRequest{
				List:       &beta.List{},
				UpdateMask: nil,
			},
			wantErr: errors.Errorf("invalid argument. List: %v and UpdateMask %v cannot be nil", &beta.List{}, nil),
		},
		"updateMask contains invalid path": {
			updateRequest: &beta.UpdateListRequest{
				List: &beta.List{
					Name: "assets",
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"foo"}},
			},
			wantErr: errors.Errorf("invalid argument. Field Mask Path(s): [foo] are invalid for List. Use valid field name(s): "),
		},
		"updateMask is empty": {
			updateRequest: &beta.UpdateListRequest{
				List: &beta.List{
					Name: "unicorns",
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{""}},
			},
			wantErr: errors.Errorf("invalid argument. Field Mask Path(s): [] are invalid for List. Use valid field name(s): "),
		},
		"capacity is less than zero": {
			updateRequest: &beta.UpdateListRequest{
				List: &beta.List{
					Name:     "clients",
					Capacity: -1,
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity"}},
			},
			wantErr: errors.Errorf("out of range. Capacity must be within range [0,1000]. Found Capacity: %d", -1),
		},
		"capacity greater than max capacity (1000)": {
			updateRequest: &beta.UpdateListRequest{
				List: &beta.List{
					Name:     "clients",
					Capacity: 1001,
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity"}},
			},
			wantErr: errors.Errorf("out of range. Capacity must be within range [0,1000]. Found Capacity: %d", 1001),
		},
		"capacity is less than List length": {
			updateRequest: &beta.UpdateListRequest{
				List: &beta.List{
					Name:     "models",
					Capacity: 1,
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity"}},
			},
			want: &beta.List{
				Name:     "models",
				Capacity: int64(1),
				Values:   []string{"model1"},
			},
		},
	}

	for testName, testScenario := range testScenarios {
		t.Run(testName, func(t *testing.T) {
			got, err := l.UpdateList(context.Background(), testScenario.updateRequest)
			// Check tests expecting non-errors
			if testScenario.want != nil {
				assert.NoError(t, err)
				if diff := cmp.Diff(testScenario.want, got, protocmp.Transform()); diff != "" {
					t.Errorf("Unexpected difference:\n%v", diff)
				}
			} else {
				// Check tests expecting errors
				assert.ErrorContains(t, err, testScenario.wantErr.Error())
			}
		})
	}
}

func TestLocalSDKServerAddListValue(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	lists := map[string]agonesv1.ListStatus{
		"lemmings": {Capacity: int64(100), Values: []string{"lemming1", "lemming2"}},
		"hacks":    {Capacity: int64(2), Values: []string{"hack1", "hack2"}},
	}
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Status: agonesv1.GameServerStatus{
			Lists: lists,
		},
	}

	path, err := gsToTmpFile(fixture)
	assert.NoError(t, err)
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

	// wait for watching to begin
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
		found := false
		l.updateObservers.Range(func(_, _ interface{}) bool {
			found = true
			return false
		})
		return found, nil
	})
	assert.NoError(t, err)

	testScenarios := map[string]struct {
		addRequest *beta.AddListValueRequest
		want       *beta.List
		wantErr    error
	}{
		"add List value": {
			addRequest: &beta.AddListValueRequest{
				Name:  "lemmings",
				Value: "lemming3",
			},
			want: &beta.List{Name: "lemmings", Capacity: int64(100), Values: []string{"lemming1", "lemming2", "lemming3"}},
		},
		"List does not exist": {
			addRequest: &beta.AddListValueRequest{
				Name: "dragons",
			},
			wantErr: errors.Errorf("not found. %s List not found", "dragons"),
		},
		"add more values than capacity": {
			addRequest: &beta.AddListValueRequest{
				Name:  "hacks",
				Value: "hack3",
			},
			wantErr: errors.Errorf("out of range. No available capacity. Current Capacity: %d, List Size: %d", int64(2), int64(2)),
		},
		"add existing value": {
			addRequest: &beta.AddListValueRequest{
				Name:  "lemmings",
				Value: "lemming1",
			},
			wantErr: errors.Errorf("already exists. Value: %s already in List: %s", "lemming1", "lemmings"),
		},
	}

	for testName, testScenario := range testScenarios {
		t.Run(testName, func(t *testing.T) {
			got, err := l.AddListValue(context.Background(), testScenario.addRequest)
			// Check tests expecting non-errors
			if testScenario.want != nil {
				assert.NoError(t, err)
				if diff := cmp.Diff(testScenario.want, got, protocmp.Transform()); diff != "" {
					t.Errorf("Unexpected difference:\n%v", diff)
				}
			} else {
				// Check tests expecting errors
				assert.ErrorContains(t, err, testScenario.wantErr.Error())
			}
		})
	}
}

func TestLocalSDKServerRemoveListValue(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	lists := map[string]agonesv1.ListStatus{
		"players": {Capacity: int64(100), Values: []string{"player1", "player2"}},
		"items":   {Capacity: int64(1000), Values: []string{"item1", "item2"}},
	}
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "stuff"},
		Status: agonesv1.GameServerStatus{
			Lists: lists,
		},
	}

	path, err := gsToTmpFile(fixture)
	assert.NoError(t, err)
	l, err := NewLocalSDKServer(path, "")
	assert.Nil(t, err)

	stream := newGameServerMockStream()
	go func() {
		err := l.WatchGameServer(&sdk.Empty{}, stream)
		assert.Nil(t, err)
	}()
	assertInitialWatchUpdate(t, stream)

	// wait for watching to begin
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
		found := false
		l.updateObservers.Range(func(_, _ interface{}) bool {
			found = true
			return false
		})
		return found, nil
	})
	assert.NoError(t, err)

	testScenarios := map[string]struct {
		removeRequest *beta.RemoveListValueRequest
		want          *beta.List
		wantErr       error
	}{
		"remove List value": {
			removeRequest: &beta.RemoveListValueRequest{
				Name:  "players",
				Value: "player1",
			},
			want: &beta.List{Name: "players", Capacity: int64(100), Values: []string{"player2"}},
		},
		"List does not exist": {
			removeRequest: &beta.RemoveListValueRequest{
				Name: "dragons",
			},
			wantErr: errors.Errorf("not found. %s List not found", "dragons"),
		},
		"value does not exist": {
			removeRequest: &beta.RemoveListValueRequest{
				Name:  "items",
				Value: "item3",
			},
			wantErr: errors.Errorf("not found. Value: %s not found in List: %s", "item3", "items"),
		},
	}

	for testName, testScenario := range testScenarios {
		t.Run(testName, func(t *testing.T) {
			got, err := l.RemoveListValue(context.Background(), testScenario.removeRequest)
			// Check tests expecting non-errors
			if testScenario.want != nil {
				assert.NoError(t, err)
				if diff := cmp.Diff(testScenario.want, got, protocmp.Transform()); diff != "" {
					t.Errorf("Unexpected difference:\n%v", diff)
				}
			} else {
				// Check tests expecting errors
				assert.ErrorContains(t, err, testScenario.wantErr.Error())
			}
		})
	}
}

// TestLocalSDKServerStateUpdates verify that SDK functions changes the state of the
// GameServer object
func TestLocalSDKServerStateUpdates(t *testing.T) {
	t.Parallel()
	l, err := NewLocalSDKServer("", "")
	assert.Nil(t, err)

	ctx := context.Background()
	e := &sdk.Empty{}
	_, err = l.Ready(ctx, e)
	assert.Nil(t, err)

	gs, err := l.GetGameServer(ctx, e)
	assert.Nil(t, err)
	assert.Equal(t, gs.Status.State, string(agonesv1.GameServerStateReady))

	seconds := &sdk.Duration{Seconds: 2}
	_, err = l.Reserve(ctx, seconds)
	assert.Nil(t, err)

	gs, err = l.GetGameServer(ctx, e)
	assert.Nil(t, err)
	assert.Equal(t, gs.Status.State, string(agonesv1.GameServerStateReserved))

	_, err = l.Allocate(ctx, e)
	assert.Nil(t, err)

	gs, err = l.GetGameServer(ctx, e)
	assert.Nil(t, err)
	assert.Equal(t, gs.Status.State, string(agonesv1.GameServerStateAllocated))

	_, err = l.Shutdown(ctx, e)
	assert.Nil(t, err)

	gs, err = l.GetGameServer(ctx, e)
	assert.Nil(t, err)
	assert.Equal(t, gs.Status.State, string(agonesv1.GameServerStateShutdown))
}

// TestSDKConformanceFunctionality - run a number of record requests in parallel
func TestSDKConformanceFunctionality(t *testing.T) {
	t.Parallel()

	l, err := NewLocalSDKServer("", "")
	assert.Nil(t, err)
	l.testMode = true
	l.recordRequest("")
	l.gs = &sdk.GameServer{ObjectMeta: &sdk.GameServer_ObjectMeta{Name: "empty"}}
	exampleUID := "052fb0f4-3d50-11e5-b066-42010af0d7b6"
	// field which is tested
	setAnnotation := "setannotation"
	l.gs.ObjectMeta.Uid = exampleUID

	var expected []string
	expected = append(expected, "", setAnnotation)

	wg := sync.WaitGroup{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		str := fmt.Sprintf("%d", i)
		expected = append(expected, str)

		go func() {
			l.recordRequest(str)
			l.recordRequestWithValue(setAnnotation, exampleUID, "UID")
			wg.Done()
		}()
	}
	wg.Wait()

	l.SetExpectedSequence(expected)
	b := l.EqualSets(l.expectedSequence, l.requestSequence)
	assert.True(t, b, "we should receive strings from all go routines %v %v", l.expectedSequence, l.requestSequence)
}

func TestAlphaSDKConformanceFunctionality(t *testing.T) {
	t.Parallel()
	lStable, err := NewLocalSDKServer("", "")
	assert.Nil(t, err)
	v := int64(0)
	lStable.recordRequestWithValue("setplayercapacity", strconv.FormatInt(v, 10), "PlayerCapacity")
	lStable.recordRequestWithValue("isplayerconnected", "", "PlayerIDs")

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	require.NoError(t, runtime.ParseFeatures(string(runtime.FeaturePlayerTracking)+"=true"))
	l, err := NewLocalSDKServer("", "")
	assert.Nil(t, err)
	l.testMode = true
	l.recordRequestWithValue("setplayercapacity", strconv.FormatInt(v, 10), "PlayerCapacity")
	l.recordRequestWithValue("isplayerconnected", "", "PlayerIDs")

}

func gsToTmpFile(gs *agonesv1.GameServer) (string, error) {
	file, err := os.CreateTemp(os.TempDir(), "gameserver-")
	if err != nil {
		return file.Name(), err
	}

	err = json.NewEncoder(file).Encode(gs)
	return file.Name(), err
}

// assertWatchUpdate checks the values of an update message when a GameServer value has been changed
func assertWatchUpdate(t *testing.T, stream *gameServerMockStream, expected interface{}, actual func(gs *sdk.GameServer) interface{}) {
	select {
	case msg := <-stream.msgs:
		assert.Equal(t, expected, actual(msg))
	case <-time.After(20 * time.Second):
		assert.Fail(t, "timeout on receiving messages")
	}
}

// assertNoWatchUpdate checks that no update message has been sent for changes to the GameServer
func assertNoWatchUpdate(t *testing.T, stream *gameServerMockStream) {
	select {
	case <-stream.msgs:
		assert.Fail(t, "should not get a message")
	case <-time.After(time.Second):
	}
}

// assertInitialWatchUpdate checks that the initial GameServer state is sent immediately after WatchGameServer
func assertInitialWatchUpdate(t *testing.T, stream *gameServerMockStream) {
	select {
	case <-stream.msgs:
	case <-time.After(time.Second):
		assert.Fail(t, "timeout on receiving initial message")
	}
}
