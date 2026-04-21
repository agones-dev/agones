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
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/gameserverallocations"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/sdk/alpha"
	"agones.dev/agones/pkg/sdk/beta"
	agtesting "agones.dev/agones/pkg/testing"
	agruntime "agones.dev/agones/pkg/util/runtime"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/clock"
	testclocks "k8s.io/utils/clock/testing"
)

// patchGameServer is a helper function for the AddReactor "patch" that creates and applies a patch
// to a gameserver. Returns a patched copy and does not modify the original game server.
func patchGameServer(t *testing.T, action k8stesting.Action, gs *agonesv1.GameServer) *agonesv1.GameServer {
	pa := action.(k8stesting.PatchAction)
	patchJSON := pa.GetPatch()
	patch, err := jsonpatch.DecodePatch(patchJSON)
	assert.NoError(t, err)
	gsCopy := gs.DeepCopy()
	gsJSON, err := json.Marshal(gsCopy)
	assert.NoError(t, err)
	patchedGs, err := patch.Apply(gsJSON)
	assert.NoError(t, err)
	err = json.Unmarshal(patchedGs, &gsCopy)
	assert.NoError(t, err)

	return gsCopy
}

func TestSidecarRun(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	nowTs, err := now.MarshalText()
	require.NoError(t, err)

	type expected struct {
		state       agonesv1.GameServerState
		labels      map[string]string
		annotations map[string]string
		recordings  []string
	}

	fixtures := map[string]struct {
		f        func(*SDKServer, context.Context)
		clock    clock.WithTickerAndDelayedExecution
		expected expected
	}{
		"ready": {
			f: func(sc *SDKServer, ctx context.Context) {
				sc.Ready(ctx, &sdk.Empty{}) // nolint: errcheck
			},
			expected: expected{
				state:      agonesv1.GameServerStateRequestReady,
				recordings: []string{"Normal " + string(agonesv1.GameServerStateRequestReady)},
			},
		},
		"shutdown": {
			f: func(sc *SDKServer, ctx context.Context) {
				sc.Shutdown(ctx, &sdk.Empty{}) // nolint: errcheck
			},
			expected: expected{
				state:      agonesv1.GameServerStateShutdown,
				recordings: []string{"Normal " + string(agonesv1.GameServerStateShutdown)},
			},
		},
		"unhealthy": {
			f: func(sc *SDKServer, _ context.Context) {
				time.Sleep(1 * time.Second)
				sc.checkHealthUpdateState() // normally invoked from health check loop
				time.Sleep(2 * time.Second) // exceed 1s timeout
				sc.checkHealthUpdateState() // normally invoked from health check loop
			},
			expected: expected{
				state:      agonesv1.GameServerStateUnhealthy,
				recordings: []string{"Warning " + string(agonesv1.GameServerStateUnhealthy)},
			},
		},
		"label": {
			f: func(sc *SDKServer, ctx context.Context) {
				_, err := sc.SetLabel(ctx, &sdk.KeyValue{Key: "foo", Value: "value-foo"})
				assert.Nil(t, err)
				_, err = sc.SetLabel(ctx, &sdk.KeyValue{Key: "bar", Value: "value-bar"})
				assert.Nil(t, err)
			},
			expected: expected{
				labels: map[string]string{
					metadataPrefix + "foo": "value-foo",
					metadataPrefix + "bar": "value-bar"},
			},
		},
		"annotation": {
			f: func(sc *SDKServer, ctx context.Context) {
				_, err := sc.SetAnnotation(ctx, &sdk.KeyValue{Key: "test-1", Value: "annotation-1"})
				assert.Nil(t, err)
				_, err = sc.SetAnnotation(ctx, &sdk.KeyValue{Key: "test-2", Value: "annotation-2"})
				assert.Nil(t, err)
			},
			expected: expected{
				annotations: map[string]string{
					metadataPrefix + "test-1": "annotation-1",
					metadataPrefix + "test-2": "annotation-2"},
			},
		},
		"allocated": {
			f: func(sc *SDKServer, ctx context.Context) {
				_, err := sc.Allocate(ctx, &sdk.Empty{})
				assert.NoError(t, err)
			},
			clock: testclocks.NewFakeClock(now),
			expected: expected{
				state:      agonesv1.GameServerStateAllocated,
				recordings: []string{string(agonesv1.GameServerStateAllocated)},
				annotations: map[string]string{
					gameserverallocations.LastAllocatedAnnotationKey: string(nowTs),
				},
			},
		},
		"reserved": {
			f: func(sc *SDKServer, ctx context.Context) {
				_, err := sc.Reserve(ctx, &sdk.Duration{Seconds: 10})
				assert.NoError(t, err)
			},
			expected: expected{
				state:      agonesv1.GameServerStateReserved,
				recordings: []string{string(agonesv1.GameServerStateReserved)},
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			m := agtesting.NewMocks()
			done := make(chan bool)

			gs := agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test", Namespace: "default", ResourceVersion: "0",
				},
				Spec: agonesv1.GameServerSpec{
					Health: agonesv1.Health{Disabled: false, FailureThreshold: 1, PeriodSeconds: 1, InitialDelaySeconds: 0},
				},
				Status: agonesv1.GameServerStatus{
					State: agonesv1.GameServerStateStarting,
				},
			}
			gs.ApplyDefaults()

			m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs.DeepCopy()}}, nil
			})

			m.AgonesClient.AddReactor("patch", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				defer close(done)

				gsCopy := patchGameServer(t, action, &gs)

				if v.expected.state != "" {
					assert.Equal(t, v.expected.state, gsCopy.Status.State)
				}
				for label, value := range v.expected.labels {
					assert.Equal(t, value, gsCopy.ObjectMeta.Labels[label])
				}
				for ann, value := range v.expected.annotations {
					assert.Equal(t, value, gsCopy.ObjectMeta.Annotations[ann])
				}
				return true, gsCopy, nil
			})

			sc, err := NewSDKServer("test", "default", m.KubeClient, m.AgonesClient, logrus.DebugLevel, 8080, 500*time.Millisecond)
			stop := make(chan struct{})
			defer close(stop)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			assert.NoError(t, sc.WaitForConnection(ctx))
			sc.informerFactory.Start(stop)
			assert.True(t, cache.WaitForCacheSync(stop, sc.gameServerSynced))

			assert.Nil(t, err)
			sc.recorder = m.FakeRecorder
			if v.clock != nil {
				sc.clock = v.clock
			}

			wg := sync.WaitGroup{}
			wg.Add(1)

			go func() {
				err := sc.Run(ctx)
				assert.Nil(t, err)
				wg.Done()
			}()
			v.f(sc, ctx)

			select {
			case <-done:
			case <-time.After(10 * time.Second):
				assert.Fail(t, "Timeout on Run")
			}

			logrus.Info("attempting to find event recording")
			for _, str := range v.expected.recordings {
				agtesting.AssertEventContains(t, m.FakeRecorder.Events, str)
			}

			cancel()
			wg.Wait()
		})
	}
}

func TestSDKServerSyncGameServer(t *testing.T) {
	t.Parallel()

	type expected struct {
		state       agonesv1.GameServerState
		labels      map[string]string
		annotations map[string]string
	}

	type scData struct {
		gsState       agonesv1.GameServerState
		gsLabels      map[string]string
		gsAnnotations map[string]string
	}

	fixtures := map[string]struct {
		expected expected
		key      string
		scData   scData
	}{
		"ready": {
			key: string(updateState),
			scData: scData{
				gsState: agonesv1.GameServerStateReady,
			},
			expected: expected{
				state: agonesv1.GameServerStateReady,
			},
		},
		"label": {
			key: string(updateLabel),
			scData: scData{
				gsLabels: map[string]string{"foo": "bar"},
			},
			expected: expected{
				labels: map[string]string{metadataPrefix + "foo": "bar"},
			},
		},
		"annotation": {
			key: string(updateAnnotation),
			scData: scData{
				gsAnnotations: map[string]string{"test": "annotation"},
			},
			expected: expected{
				annotations: map[string]string{metadataPrefix + "test": "annotation"},
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			m := agtesting.NewMocks()
			sc, err := defaultSidecar(m)
			assert.Nil(t, err)

			sc.gsState = v.scData.gsState
			sc.gsLabels = v.scData.gsLabels
			sc.gsAnnotations = v.scData.gsAnnotations

			updated := false
			gs := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{
				UID:  "1234",
				Name: sc.gameServerName, Namespace: sc.namespace, ResourceVersion: "0",
				Labels: map[string]string{}, Annotations: map[string]string{}},
			}

			m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs.DeepCopy()}}, nil
			})

			m.AgonesClient.AddReactor("patch", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				gsCopy := patchGameServer(t, action, &gs)

				if v.expected.state != "" {
					assert.Equal(t, v.expected.state, gsCopy.Status.State)
				}
				for label, value := range v.expected.labels {
					assert.Equal(t, value, gsCopy.ObjectMeta.Labels[label])
				}
				for ann, value := range v.expected.annotations {
					assert.Equal(t, value, gsCopy.ObjectMeta.Annotations[ann])
				}
				updated = true
				return false, gsCopy, nil
			})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sc.informerFactory.Start(ctx.Done())
			assert.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))
			sc.gsWaitForSync.Done()

			err = sc.syncGameServer(ctx, v.key)
			assert.Nil(t, err)
			assert.True(t, updated, "should have updated")
		})
	}
}

func TestSidecarUpdateState(t *testing.T) {
	t.Parallel()

	fixtures := map[string]struct {
		f func(gs *agonesv1.GameServer)
	}{
		"unhealthy": {
			f: func(gs *agonesv1.GameServer) {
				gs.Status.State = agonesv1.GameServerStateUnhealthy
			},
		},
		"shutdown": {
			f: func(gs *agonesv1.GameServer) {
				gs.Status.State = agonesv1.GameServerStateShutdown
			},
		},
		"deleted": {
			f: func(gs *agonesv1.GameServer) {
				now := metav1.Now()
				gs.ObjectMeta.DeletionTimestamp = &now
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			m := agtesting.NewMocks()
			sc, err := defaultSidecar(m)
			require.NoError(t, err)
			sc.gsState = agonesv1.GameServerStateReady

			updated := false

			m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				gs := agonesv1.GameServer{
					ObjectMeta: metav1.ObjectMeta{Name: sc.gameServerName, Namespace: sc.namespace, ResourceVersion: "0"},
					Status:     agonesv1.GameServerStatus{},
				}

				// apply mutation
				v.f(&gs)

				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
			})
			m.AgonesClient.AddReactor("patch", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				return true, nil, nil
			})

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sc.informerFactory.Start(ctx.Done())
			assert.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))
			sc.gsWaitForSync.Done()

			err = sc.updateState(ctx)
			assert.Nil(t, err)
			assert.False(t, updated)
		})
	}
}

func TestSidecarHealthLastUpdated(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	m := agtesting.NewMocks()

	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	sc.health = agonesv1.Health{Disabled: false}
	fc := testclocks.NewFakeClock(now)
	sc.clock = fc

	stream := newEmptyMockStream()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := sc.Health(stream)
		assert.Nil(t, err)
		wg.Done()
	}()

	// Test once with a single message
	fc.Step(3 * time.Second)
	stream.msgs <- &sdk.Empty{}

	err = waitForMessage(sc)
	assert.Nil(t, err)
	sc.healthMutex.RLock()
	assert.Equal(t, sc.clock.Now().UTC().String(), sc.healthLastUpdated.String())
	sc.healthMutex.RUnlock()

	// Test again, since the value has been set, that it is re-set
	fc.Step(3 * time.Second)
	stream.msgs <- &sdk.Empty{}
	err = waitForMessage(sc)
	assert.Nil(t, err)
	sc.healthMutex.RLock()
	assert.Equal(t, sc.clock.Now().UTC().String(), sc.healthLastUpdated.String())
	sc.healthMutex.RUnlock()

	// make sure closing doesn't change the time
	fc.Step(3 * time.Second)
	close(stream.msgs)
	assert.NotEqual(t, sc.clock.Now().UTC().String(), sc.healthLastUpdated.String())

	wg.Wait()
}

func TestSidecarUnhealthyMessage(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()
	sc, err := NewSDKServer("test", "default", m.KubeClient, m.AgonesClient, logrus.DebugLevel, 8080, 500*time.Millisecond)
	require.NoError(t, err)

	gs := agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test", Namespace: "default", ResourceVersion: "0",
		},
		Spec: agonesv1.GameServerSpec{},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateStarting,
		},
	}
	gs.ApplyDefaults()

	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs.DeepCopy()}}, nil
	})

	m.AgonesClient.AddReactor("patch", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gsCopy := patchGameServer(t, action, &gs)

		return true, gsCopy, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := make(chan struct{})
	defer close(stop)

	assert.NoError(t, sc.WaitForConnection(ctx))
	sc.informerFactory.Start(stop)
	assert.True(t, cache.WaitForCacheSync(stop, sc.gameServerSynced))

	sc.recorder = m.FakeRecorder

	go func() {
		err := sc.Run(ctx)
		assert.Nil(t, err)
	}()

	// manually push through an unhealthy state change
	sc.enqueueState(agonesv1.GameServerStateUnhealthy)
	agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Health check failure")

	// try to push back to Ready, enqueueState should block it.
	sc.enqueueState(agonesv1.GameServerStateRequestReady)
	sc.gsUpdateMutex.Lock()
	assert.Equal(t, agonesv1.GameServerStateUnhealthy, sc.gsState)
	sc.gsUpdateMutex.Unlock()
}

func TestSidecarHealthy(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()
	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	// manually set the values
	sc.health = agonesv1.Health{FailureThreshold: 1}
	sc.healthTimeout = 5 * time.Second
	sc.touchHealthLastUpdated()

	now := time.Now().UTC()
	fc := testclocks.NewFakeClock(now)
	sc.clock = fc

	stream := newEmptyMockStream()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := sc.Health(stream)
		assert.Nil(t, err)
		wg.Done()
	}()

	fixtures := map[string]struct {
		timeAdd         time.Duration
		disabled        bool
		expectedHealthy bool
	}{
		"disabled, under timeout": {disabled: true, timeAdd: time.Second, expectedHealthy: true},
		"disabled, over timeout":  {disabled: true, timeAdd: 15 * time.Second, expectedHealthy: true},
		"enabled, under timeout":  {disabled: false, timeAdd: time.Second, expectedHealthy: true},
		"enabled, over timeout":   {disabled: false, timeAdd: 15 * time.Second, expectedHealthy: false},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			logrus.WithField("test", k).Infof("Test Running")
			sc.health.Disabled = v.disabled
			fc.SetTime(time.Now().UTC())
			stream.msgs <- &sdk.Empty{}
			err = waitForMessage(sc)
			assert.Nil(t, err)

			fc.Step(v.timeAdd)
			sc.checkHealth()
			assert.Equal(t, v.expectedHealthy, sc.healthy())
		})
	}

	t.Run("initial delay", func(t *testing.T) {
		sc.health.Disabled = false
		fc.SetTime(time.Now().UTC())
		sc.touchHealthLastUpdated()

		// initial delay is handled by kubelet, runHealth() isn't
		// called until container starts.
		fc.Step(10 * time.Second)
		sc.touchHealthLastUpdated()
		sc.checkHealth()
		assert.True(t, sc.healthy())

		fc.Step(10 * time.Second)
		sc.checkHealth()
		assert.False(t, sc.healthy())
	})

	t.Run("health failure threshold", func(t *testing.T) {
		sc.health.Disabled = false
		sc.health.FailureThreshold = 3
		fc.SetTime(time.Now().UTC())
		sc.touchHealthLastUpdated()

		sc.checkHealth()
		assert.True(t, sc.healthy())
		assert.Equal(t, int32(0), sc.healthFailureCount)

		fc.Step(10 * time.Second)
		sc.checkHealth()
		assert.True(t, sc.healthy())
		sc.checkHealth()
		assert.True(t, sc.healthy())
		sc.checkHealth()
		assert.False(t, sc.healthy())

		stream.msgs <- &sdk.Empty{}
		err = waitForMessage(sc)
		assert.Nil(t, err)
		fc.Step(10 * time.Second)
		assert.True(t, sc.healthy())
	})

	close(stream.msgs)
	wg.Wait()
}

func TestSidecarHTTPHealthCheck(t *testing.T) {
	m := agtesting.NewMocks()
	sc, err := NewSDKServer("test", "default", m.KubeClient, m.AgonesClient, logrus.DebugLevel, 8080, 500*time.Millisecond)
	require.NoError(t, err)

	now := time.Now().Add(time.Hour).UTC()
	fc := testclocks.NewFakeClock(now)
	// now we control time - so slow machines won't fail anymore
	sc.clock = fc

	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		gs := agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: sc.gameServerName, Namespace: sc.namespace},
			Spec: agonesv1.GameServerSpec{
				Health: agonesv1.Health{Disabled: false, FailureThreshold: 1, PeriodSeconds: 1, InitialDelaySeconds: 0},
			},
		}

		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := sync.WaitGroup{}
	wg.Add(1)

	step := 2 * time.Second

	go func() {
		err := sc.Run(ctx)
		assert.Nil(t, err)
		// gate
		assert.Equal(t, 1*time.Second, sc.healthTimeout)
		wg.Done()
	}()

	testHTTPHealth(t, "http://localhost:8080/healthz", "ok", http.StatusOK)
	testHTTPHealth(t, "http://localhost:8080/gshealthz", "ok", http.StatusOK)

	assert.Equal(t, now, sc.healthLastUpdated)

	fc.Step(step)
	time.Sleep(step)
	sc.checkHealthUpdateState()
	assert.False(t, sc.healthy())
	cancel()
	wg.Wait() // wait for go routine test results.
}

func TestSDKServerGetGameServer(t *testing.T) {
	t.Parallel()

	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateReady,
		},
	}

	m := agtesting.NewMocks()
	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*fixture}}, nil
	})

	stop := make(chan struct{})
	defer close(stop)

	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	sc.informerFactory.Start(stop)
	assert.True(t, cache.WaitForCacheSync(stop, sc.gameServerSynced))
	sc.gsWaitForSync.Done()

	result, err := sc.GetGameServer(context.Background(), &sdk.Empty{})
	require.NoError(t, err)
	assert.Equal(t, fixture.ObjectMeta.Name, result.ObjectMeta.Name)
	assert.Equal(t, fixture.ObjectMeta.Namespace, result.ObjectMeta.Namespace)
	assert.Equal(t, string(fixture.Status.State), result.Status.State)
}

func TestSDKServerWatchGameServer(t *testing.T) {
	t.Parallel()

	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateReady,
		},
	}

	m := agtesting.NewMocks()
	fakeWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	sc, err := defaultSidecar(m)
	require.NoError(t, err)
	assert.Empty(t, sc.connectedStreams)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sc.ctx = ctx
	sc.informerFactory.Start(ctx.Done())

	fakeWatch.Add(fixture.DeepCopy())
	assert.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))
	sc.gsWaitForSync.Done()

	// wait for the GameServer to be populated, as we can't rely on WaitForCacheSync
	require.Eventually(t, func() bool {
		_, err := sc.gameServer()
		return err == nil
	}, time.Minute, time.Second, "Could not find the GameServer")

	stream := newGameServerMockStream()
	asyncWatchGameServer(t, sc, stream)

	require.Nil(t, waitConnectedStreamCount(sc, 1))
	require.Equal(t, stream, sc.connectedStreams[0])

	// modify for 2nd event in watch stream
	fixture.Status.State = agonesv1.GameServerStateAllocated
	fakeWatch.Modify(fixture.DeepCopy())

	totalSendCalls := 0
	running := true
	for running {
		select {
		case gs := <-stream.msgs:
			assert.Equal(t, fixture.ObjectMeta.Name, gs.ObjectMeta.Name)
			totalSendCalls++
			switch totalSendCalls {
			case 1:
				assert.Equal(t, string(agonesv1.GameServerStateReady), gs.Status.State)
			case 2:
				assert.Equal(t, string(agonesv1.GameServerStateAllocated), gs.Status.State)
			}
			// we shouldn't get more than 2, but let's put an upper bound on this
			// just in case we suddenly get way more than we expect.
			if totalSendCalls > 10 {
				assert.FailNow(t, "We should have only received two events. Got over 10 instead.")
			}
		case <-time.After(5 * time.Second):
			// we can't `break` out of the loop, hence we need `running`.
			running = false
		}
	}

	// There are two stream.Send() calls should happen: one in sendGameServerUpdate,
	// another one in WatchGameServer.
	assert.Equal(t, 2, totalSendCalls)
}

func TestSDKServerSendGameServerUpdate(t *testing.T) {
	t.Parallel()
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateReady,
		},
	}

	m := agtesting.NewMocks()
	fakeWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(fakeWatch, nil))
	sc, err := defaultSidecar(m)
	require.NoError(t, err)
	assert.Empty(t, sc.connectedStreams)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sc.ctx = ctx
	sc.informerFactory.Start(ctx.Done())

	fakeWatch.Add(fixture.DeepCopy())
	assert.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))
	sc.gsWaitForSync.Done()

	// wait for the GameServer to be populated, as we can't rely on WaitForCacheSync
	require.Eventually(t, func() bool {
		_, err := sc.gameServer()
		return err == nil
	}, time.Minute, time.Second, "Could not find the GameServer")

	stream := newGameServerMockStream()
	asyncWatchGameServer(t, sc, stream)
	assert.Nil(t, waitConnectedStreamCount(sc, 1))

	sc.sendGameServerUpdate(fixture)

	var sdkGS *sdk.GameServer
	select {
	case sdkGS = <-stream.msgs:
	case <-time.After(3 * time.Second):
		assert.Fail(t, "Event stream should not have timed out")
	}

	assert.Equal(t, fixture.ObjectMeta.Name, sdkGS.ObjectMeta.Name)
}

func TestSDKServer_SendGameServerUpdateRemovesDisconnectedStream(t *testing.T) {
	t.Parallel()

	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateReady,
		},
	}

	m := agtesting.NewMocks()
	fakeWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(fakeWatch, nil))
	sc, err := defaultSidecar(m)
	require.NoError(t, err)
	assert.Empty(t, sc.connectedStreams)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	sc.ctx = ctx
	sc.informerFactory.Start(ctx.Done())

	fakeWatch.Add(fixture.DeepCopy())
	assert.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))
	sc.gsWaitForSync.Done()

	// Wait for the GameServer to be populated, as we can't rely on WaitForCacheSync.
	require.Eventually(t, func() bool {
		_, err := sc.gameServer()
		return err == nil
	}, time.Minute, time.Second, "Could not find the GameServer")

	// Create and initialize two streams.
	streamOne := newGameServerMockStream()
	streamOneCtx, streamOneCancel := context.WithCancel(context.Background())
	t.Cleanup(streamOneCancel)
	streamOne.ctx = streamOneCtx
	asyncWatchGameServer(t, sc, streamOne)

	streamTwo := newGameServerMockStream()
	streamTwoCtx, streamTwoCancel := context.WithCancel(context.Background())
	t.Cleanup(streamTwoCancel)
	streamTwo.ctx = streamTwoCtx
	asyncWatchGameServer(t, sc, streamTwo)

	// Verify that two streams are connected.
	assert.Nil(t, waitConnectedStreamCount(sc, 2))
	streamOneCancel()
	streamTwoCancel()

	// Trigger stream removal by sending a game server update.
	sc.sendGameServerUpdate(fixture)
	// Verify that zero streams are connected.
	assert.Nil(t, waitConnectedStreamCount(sc, 0))
}

func TestSDKServerUpdateEventHandler(t *testing.T) {
	t.Parallel()
	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateReady,
		},
	}

	m := agtesting.NewMocks()
	fakeWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(fakeWatch, nil))
	sc, err := defaultSidecar(m)
	require.NoError(t, err)
	assert.Empty(t, sc.connectedStreams)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sc.ctx = ctx
	sc.informerFactory.Start(ctx.Done())

	fakeWatch.Add(fixture.DeepCopy())
	assert.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))
	sc.gsWaitForSync.Done()

	// wait for the GameServer to be populated, as we can't rely on WaitForCacheSync
	require.Eventually(t, func() bool {
		_, err := sc.gameServer()
		return err == nil
	}, time.Minute, time.Second, "Could not find the GameServer")
	stream := newGameServerMockStream()
	asyncWatchGameServer(t, sc, stream)
	assert.Nil(t, waitConnectedStreamCount(sc, 1))

	// need to add it before it can be modified
	fakeWatch.Add(fixture.DeepCopy())
	fakeWatch.Modify(fixture.DeepCopy())

	var sdkGS *sdk.GameServer
	select {
	case sdkGS = <-stream.msgs:
	case <-time.After(3 * time.Second):
		assert.Fail(t, "Event stream should not have timed out")
	}

	assert.NotNil(t, sdkGS)
	assert.Equal(t, fixture.ObjectMeta.Name, sdkGS.ObjectMeta.Name)
}

func TestSDKServerReserveTimeoutOnRun(t *testing.T) {
	t.Parallel()
	m := agtesting.NewMocks()

	updated := make(chan agonesv1.GameServerStatus, 1)

	gs := agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test", Namespace: "default", ResourceVersion: "0",
		},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateReserved,
		},
	}
	gs.ApplyDefaults()

	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		n := metav1.NewTime(metav1.Now().Add(time.Second))
		gsCopy := gs.DeepCopy()
		gsCopy.Status.ReservedUntil = &n

		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gsCopy}}, nil
	})

	m.AgonesClient.AddReactor("patch", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gsCopy := patchGameServer(t, action, &gs)

		updated <- gsCopy.Status

		return true, gsCopy, nil
	})

	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	assert.NoError(t, sc.WaitForConnection(ctx))
	sc.informerFactory.Start(ctx.Done())
	assert.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		err = sc.Run(ctx)
		assert.Nil(t, err)
		wg.Done()
	}()

	select {
	case status := <-updated:
		assert.Equal(t, agonesv1.GameServerStateRequestReady, status.State)
		assert.Nil(t, status.ReservedUntil)
	case <-time.After(5 * time.Second):
		assert.Fail(t, "should have been an update")
	}

	cancel()
	wg.Wait()
}

func TestSDKServerReserveTimeout(t *testing.T) {
	t.Parallel()
	m := agtesting.NewMocks()

	state := make(chan agonesv1.GameServerStatus, 100)
	defer close(state)

	gs := agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test", Namespace: "default", ResourceVersion: "0",
		},
		Spec: agonesv1.GameServerSpec{Health: agonesv1.Health{Disabled: true}},
	}
	gs.ApplyDefaults()

	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs.DeepCopy()}}, nil
	})

	m.AgonesClient.AddReactor("patch", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gsCopy := patchGameServer(t, action, &gs)

		state <- gsCopy.Status
		return true, gsCopy, nil
	})

	sc, err := defaultSidecar(m)

	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	assert.NoError(t, sc.WaitForConnection(ctx))
	sc.informerFactory.Start(ctx.Done())
	assert.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		err = sc.Run(ctx)
		assert.Nil(t, err)
		wg.Done()
	}()

	assertStateChange := func(expected agonesv1.GameServerState, additional func(status agonesv1.GameServerStatus)) {
		select {
		case current := <-state:
			assert.Equal(t, expected, current.State)
			additional(current)
		case <-time.After(5 * time.Second):
			assert.Fail(t, "should have gone to Reserved by now")
		}
	}
	assertReservedUntilDuration := func(d time.Duration) func(status agonesv1.GameServerStatus) {
		return func(status agonesv1.GameServerStatus) {
			assert.WithinDuration(t, time.Now().Add(d), status.ReservedUntil.Time, 1500*time.Millisecond)
		}
	}
	assertReservedUntilNil := func(status agonesv1.GameServerStatus) {
		assert.Nil(t, status.ReservedUntil)
	}

	_, err = sc.Reserve(context.Background(), &sdk.Duration{Seconds: 3})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateReserved, assertReservedUntilDuration(3*time.Second))

	// Wait for the game server to go back to being Ready.
	assertStateChange(agonesv1.GameServerStateRequestReady, func(status agonesv1.GameServerStatus) {
		assert.Nil(t, status.ReservedUntil)
	})

	// Test that a 0 second input into Reserved, never will go back to Ready
	_, err = sc.Reserve(context.Background(), &sdk.Duration{Seconds: 0})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateReserved, assertReservedUntilNil)
	assert.False(t, sc.reserveTimer.Stop())

	// Test that a negative input into Reserved, is the same as a 0 input
	_, err = sc.Reserve(context.Background(), &sdk.Duration{Seconds: -100})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateReserved, assertReservedUntilNil)
	assert.False(t, sc.reserveTimer.Stop())

	// Test that the timer to move Reserved->Ready is reset when requesting another state.

	// Test the return to a Ready state.
	_, err = sc.Reserve(context.Background(), &sdk.Duration{Seconds: 3})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateReserved, assertReservedUntilDuration(3*time.Second))

	_, err = sc.Ready(context.Background(), &sdk.Empty{})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateRequestReady, assertReservedUntilNil)
	assert.False(t, sc.reserveTimer.Stop())

	// Test Allocated resets the timer on Reserved->Ready
	_, err = sc.Reserve(context.Background(), &sdk.Duration{Seconds: 3})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateReserved, assertReservedUntilDuration(3*time.Second))

	_, err = sc.Allocate(context.Background(), &sdk.Empty{})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateAllocated, assertReservedUntilNil)
	assert.False(t, sc.reserveTimer.Stop())

	// Test Shutdown resets the timer on Reserved->Ready
	_, err = sc.Reserve(context.Background(), &sdk.Duration{Seconds: 3})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateReserved, assertReservedUntilDuration(3*time.Second))

	_, err = sc.Shutdown(context.Background(), &sdk.Empty{})
	assert.NoError(t, err)
	assertStateChange(agonesv1.GameServerStateShutdown, assertReservedUntilNil)
	assert.False(t, sc.reserveTimer.Stop())

	cancel()
	wg.Wait()
}

func TestSDKServerUpdateCounter(t *testing.T) {
	t.Parallel()
	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	err := agruntime.ParseFeatures(string(agruntime.FeatureCountsAndLists) + "=true")
	require.NoError(t, err, "Can not parse FeatureCountsAndLists feature")

	counters := map[string]agonesv1.CounterStatus{
		"widgets":  {Count: int64(10), Capacity: int64(100)},
		"foo":      {Count: int64(10), Capacity: int64(100)},
		"bar":      {Count: int64(10), Capacity: int64(100)},
		"baz":      {Count: int64(10), Capacity: int64(100)},
		"bazel":    {Count: int64(10), Capacity: int64(100)},
		"fish":     {Count: int64(10), Capacity: int64(100)},
		"onefish":  {Count: int64(10), Capacity: int64(100)},
		"twofish":  {Count: int64(10), Capacity: int64(100)},
		"redfish":  {Count: int64(10), Capacity: int64(100)},
		"bluefish": {Count: int64(10), Capacity: int64(100)},
		"fivefish": {Count: int64(10), Capacity: int64(100)},
		"starfish": {Count: int64(10), Capacity: int64(100)},
	}

	fixtures := map[string]struct {
		counterName string
		requests    []*beta.UpdateCounterRequest
		want        agonesv1.CounterStatus
		updateErrs  []bool
		updated     bool
	}{
		"increment": {
			counterName: "widgets",
			requests: []*beta.UpdateCounterRequest{{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:      "widgets",
					CountDiff: 9,
				}}},
			want:       agonesv1.CounterStatus{Count: int64(19), Capacity: int64(100)},
			updateErrs: []bool{false},
			updated:    true,
		},
		"increment illegal": {
			counterName: "foo",
			requests: []*beta.UpdateCounterRequest{{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:      "foo",
					CountDiff: 100,
				}}},
			want:       agonesv1.CounterStatus{Count: int64(10), Capacity: int64(100)},
			updateErrs: []bool{true},
			updated:    false,
		},
		"decrement": {
			counterName: "bar",
			requests: []*beta.UpdateCounterRequest{{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:      "bar",
					CountDiff: -1,
				}}},
			want:       agonesv1.CounterStatus{Count: int64(9), Capacity: int64(100)},
			updateErrs: []bool{false},
			updated:    true,
		},
		"decrement illegal": {
			counterName: "baz",
			requests: []*beta.UpdateCounterRequest{{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:      "baz",
					CountDiff: -11,
				}}},
			want:       agonesv1.CounterStatus{Count: int64(10), Capacity: int64(100)},
			updateErrs: []bool{true},
			updated:    false,
		},
		"set capacity": {
			counterName: "bazel",
			requests: []*beta.UpdateCounterRequest{{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:     "bazel",
					Capacity: wrapperspb.Int64(0),
				}}},
			want:       agonesv1.CounterStatus{Count: int64(0), Capacity: int64(0)},
			updateErrs: []bool{false},
			updated:    true,
		},
		"set capacity illegal": {
			counterName: "fish",
			requests: []*beta.UpdateCounterRequest{{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:     "fish",
					Capacity: wrapperspb.Int64(-1),
				}}},
			want:       agonesv1.CounterStatus{Count: int64(10), Capacity: int64(100)},
			updateErrs: []bool{true},
			updated:    false,
		},
		"set count": {
			counterName: "onefish",
			requests: []*beta.UpdateCounterRequest{{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:  "onefish",
					Count: wrapperspb.Int64(42),
				}}},
			want:       agonesv1.CounterStatus{Count: int64(42), Capacity: int64(100)},
			updateErrs: []bool{false},
			updated:    true,
		},
		"set count illegal": {
			counterName: "twofish",
			requests: []*beta.UpdateCounterRequest{{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:  "twofish",
					Count: wrapperspb.Int64(101),
				}}},
			want:       agonesv1.CounterStatus{Count: int64(10), Capacity: int64(100)},
			updateErrs: []bool{true},
			updated:    false,
		},
		"increment past set capacity illegal": {
			counterName: "redfish",
			requests: []*beta.UpdateCounterRequest{{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:     "redfish",
					Capacity: wrapperspb.Int64(0),
				}},
				{CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:      "redfish",
					CountDiff: 1,
				}}},
			want:       agonesv1.CounterStatus{Count: int64(0), Capacity: int64(0)},
			updateErrs: []bool{false, true},
			updated:    true,
		},
		"decrement past set capacity illegal": {
			counterName: "bluefish",
			requests: []*beta.UpdateCounterRequest{{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:     "bluefish",
					Capacity: wrapperspb.Int64(0),
				}},
				{CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:      "bluefish",
					CountDiff: -1,
				}}},
			want:       agonesv1.CounterStatus{Count: int64(0), Capacity: int64(0)},
			updateErrs: []bool{false, true},
			updated:    true,
		},
		"setcapacity, setcount, and diffcount": {
			counterName: "fivefish",
			requests: []*beta.UpdateCounterRequest{{
				CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:      "fivefish",
					Capacity:  wrapperspb.Int64(25),
					Count:     wrapperspb.Int64(0),
					CountDiff: 25,
				}}},
			want:       agonesv1.CounterStatus{Count: int64(25), Capacity: int64(25)},
			updateErrs: []bool{false},
			updated:    true,
		},
		"projected state returned": {
			counterName: "starfish",
			requests: []*beta.UpdateCounterRequest{
				{CounterUpdateRequest: &beta.CounterUpdateRequest{
					Name:      "starfish",
					CountDiff: 7,
				}},
			},
			want:       agonesv1.CounterStatus{Count: int64(17), Capacity: int64(100)},
			updateErrs: []bool{false},
			updated:    true,
		},
	}

	for test, testCase := range fixtures {
		t.Run(test, func(t *testing.T) {
			m := agtesting.NewMocks()

			gs := agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test", Namespace: "default", ResourceVersion: "0", Generation: 1,
				},
				Spec: agonesv1.GameServerSpec{
					SdkServer: agonesv1.SdkServer{
						LogLevel: "Debug",
					},
				},
				Status: agonesv1.GameServerStatus{
					Counters: counters,
				},
			}
			gs.ApplyDefaults()

			m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs.DeepCopy()}}, nil
			})

			updated := make(chan map[string]agonesv1.CounterStatus, 10)

			m.AgonesClient.AddReactor("patch", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				gsCopy := patchGameServer(t, action, &gs)

				updated <- gsCopy.Status.Counters
				return true, gsCopy, nil
			})

			ctx, cancel := context.WithCancel(context.Background())
			sc, err := defaultSidecar(m)
			require.NoError(t, err)
			assert.NoError(t, sc.WaitForConnection(ctx))
			sc.informerFactory.Start(ctx.Done())
			assert.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))

			wg := sync.WaitGroup{}
			wg.Add(1)

			go func() {
				err = sc.Run(ctx)
				assert.NoError(t, err)
				wg.Done()
			}()

			// check initial value comes through
			require.Eventually(t, func() bool {
				counter, err := sc.GetCounter(context.Background(), &beta.GetCounterRequest{Name: testCase.counterName})
				return counter.Count == 10 && counter.Capacity == 100 && err == nil
			}, 10*time.Second, time.Second)

			// Update the Counter
			for i, req := range testCase.requests {
				resp, err := sc.UpdateCounter(context.Background(), req)
				if testCase.updateErrs[i] {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, testCase.want.Count, resp.Count)
					assert.Equal(t, testCase.want.Capacity, resp.Capacity)

				}
			}

			got, err := sc.GetCounter(context.Background(), &beta.GetCounterRequest{Name: testCase.counterName})
			assert.NoError(t, err)
			assert.Equal(t, testCase.want.Count, got.Count)
			assert.Equal(t, testCase.want.Capacity, got.Capacity)

			// on an update, confirm that the update hits the K8s api
			if testCase.updated {
				select {
				case value := <-updated:
					assert.NotNil(t, value[testCase.counterName])
					assert.Equal(t,
						agonesv1.CounterStatus{Count: testCase.want.Count, Capacity: testCase.want.Capacity},
						value[testCase.counterName])
				case <-time.After(10 * time.Second):
					assert.Fail(t, "Counter should have been patched")
				}
			}

			cancel()
			wg.Wait()
		})
	}
}

func TestSDKServerAddListValue(t *testing.T) {
	t.Parallel()
	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	err := agruntime.ParseFeatures(string(agruntime.FeatureCountsAndLists) + "=true")
	require.NoError(t, err, "Can not parse FeatureCountsAndLists feature")

	lists := map[string]agonesv1.ListStatus{
		"foo": {Values: []string{"one", "two", "three", "four"}, Capacity: int64(10)},
		"bar": {Values: []string{"one", "two", "three", "four"}, Capacity: int64(10)},
		"baz": {Values: []string{"one", "two", "three", "four"}, Capacity: int64(10)},
	}

	fixtures := map[string]struct {
		listName                string
		requests                []*beta.AddListValueRequest
		want                    agonesv1.ListStatus
		updateErrs              []bool
		updated                 bool
		expectedUpdatesQueueLen int
	}{
		"Add value": {
			listName:                "foo",
			requests:                []*beta.AddListValueRequest{{Name: "foo", Value: "five"}},
			want:                    agonesv1.ListStatus{Values: []string{"one", "two", "three", "four", "five"}, Capacity: int64(10)},
			updateErrs:              []bool{false},
			updated:                 true,
			expectedUpdatesQueueLen: 0,
		},
		"Add multiple values including duplicates": {
			listName: "bar",
			requests: []*beta.AddListValueRequest{
				{Name: "bar", Value: "five"},
				{Name: "bar", Value: "one"},
				{Name: "bar", Value: "five"},
				{Name: "bar", Value: "zero"},
			},
			want:                    agonesv1.ListStatus{Values: []string{"one", "two", "three", "four", "five", "zero"}, Capacity: int64(10)},
			updateErrs:              []bool{false, true, true, false},
			updated:                 true,
			expectedUpdatesQueueLen: 0,
		},
		"Add multiple values past capacity": {
			listName: "baz",
			requests: []*beta.AddListValueRequest{
				{Name: "baz", Value: "five"},
				{Name: "baz", Value: "six"},
				{Name: "baz", Value: "seven"},
				{Name: "baz", Value: "eight"},
				{Name: "baz", Value: "nine"},
				{Name: "baz", Value: "ten"},
				{Name: "baz", Value: "eleven"},
			},
			want: agonesv1.ListStatus{
				Values:   []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"},
				Capacity: int64(10),
			},
			updateErrs:              []bool{false, false, false, false, false, false, true},
			updated:                 true,
			expectedUpdatesQueueLen: 0,
		},
	}

	// nolint:dupl  // Linter errors on lines are duplicate of TestSDKServerUpdateList, TestSDKServerRemoveListValue
	for test, testCase := range fixtures {
		t.Run(test, func(t *testing.T) {
			m := agtesting.NewMocks()

			gs := agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test", Namespace: "default", ResourceVersion: "0", Generation: 1,
				},
				Spec: agonesv1.GameServerSpec{
					SdkServer: agonesv1.SdkServer{
						LogLevel: "Debug",
					},
				},
				Status: agonesv1.GameServerStatus{
					Lists: lists,
				},
			}
			gs.ApplyDefaults()

			m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs.DeepCopy()}}, nil
			})

			updated := make(chan map[string]agonesv1.ListStatus, 10)

			m.AgonesClient.AddReactor("patch", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				gsCopy := patchGameServer(t, action, &gs)

				updated <- gsCopy.Status.Lists
				return true, gsCopy, nil
			})

			ctx, cancel := context.WithCancel(context.Background())
			sc, err := defaultSidecar(m)
			require.NoError(t, err)
			assert.NoError(t, sc.WaitForConnection(ctx))
			sc.informerFactory.Start(ctx.Done())
			require.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))
			sc.gsWaitForSync.Done()

			// check initial value comes through
			require.Eventually(t, func() bool {
				list, err := sc.GetList(context.Background(), &beta.GetListRequest{Name: testCase.listName})
				return cmp.Equal(list.Values, []string{"one", "two", "three", "four"}) && list.Capacity == 10 && err == nil
			}, 10*time.Second, time.Second)

			// Update the List
			for i, req := range testCase.requests {
				_, err = sc.AddListValue(context.Background(), req)
				if testCase.updateErrs[i] {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}

			got, err := sc.GetList(context.Background(), &beta.GetListRequest{Name: testCase.listName})
			assert.NoError(t, err)
			assert.Equal(t, testCase.want.Values, got.Values)
			assert.Equal(t, testCase.want.Capacity, got.Capacity)

			// start workerqueue processing at this point, so there is no chance of processing the above updates
			// earlier.
			sc.gsWaitForSync.Add(1)
			go func() {
				err = sc.Run(ctx)
				assert.NoError(t, err)
			}()

			// on an update, confirm that the update hits the K8s api
			if testCase.updated {
				select {
				case value := <-updated:
					assert.NotNil(t, value[testCase.listName])
					assert.Equal(t,
						agonesv1.ListStatus{Values: testCase.want.Values, Capacity: testCase.want.Capacity},
						value[testCase.listName])
				case <-time.After(10 * time.Second):
					assert.Fail(t, "List should have been patched")
				}
			}

			// on an update, confirms that the update queue list contains the right amount of items
			glu := sc.gsListUpdatesLen()
			assert.Equal(t, testCase.expectedUpdatesQueueLen, glu)

			cancel()
		})
	}
}

func TestSDKServerRemoveListValue(t *testing.T) {
	t.Parallel()
	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	err := agruntime.ParseFeatures(string(agruntime.FeatureCountsAndLists) + "=true")
	require.NoError(t, err, "Can not parse FeatureCountsAndLists feature")

	lists := map[string]agonesv1.ListStatus{
		"foo": {Values: []string{"one", "two", "three", "four"}, Capacity: int64(100)},
		"bar": {Values: []string{"one", "two", "three", "four"}, Capacity: int64(100)},
		"baz": {Values: []string{"one", "two", "three", "four"}, Capacity: int64(100)},
	}

	fixtures := map[string]struct {
		listName                string
		requests                []*beta.RemoveListValueRequest
		want                    agonesv1.ListStatus
		updateErrs              []bool
		updated                 bool
		expectedUpdatesQueueLen int
	}{
		"Remove value": {
			listName:                "foo",
			requests:                []*beta.RemoveListValueRequest{{Name: "foo", Value: "two"}},
			want:                    agonesv1.ListStatus{Values: []string{"one", "three", "four"}, Capacity: int64(100)},
			updateErrs:              []bool{false},
			updated:                 true,
			expectedUpdatesQueueLen: 0,
		},
		"Remove multiple values including duplicates": {
			listName: "bar",
			requests: []*beta.RemoveListValueRequest{
				{Name: "bar", Value: "two"},
				{Name: "bar", Value: "three"},
				{Name: "bar", Value: "two"},
			},
			want:                    agonesv1.ListStatus{Values: []string{"one", "four"}, Capacity: int64(100)},
			updateErrs:              []bool{false, false, true},
			updated:                 true,
			expectedUpdatesQueueLen: 0,
		},
		"Remove all values": {
			listName: "baz",
			requests: []*beta.RemoveListValueRequest{
				{Name: "baz", Value: "three"},
				{Name: "baz", Value: "two"},
				{Name: "baz", Value: "four"},
				{Name: "baz", Value: "one"},
			},
			want:                    agonesv1.ListStatus{Values: []string{}, Capacity: int64(100)},
			updateErrs:              []bool{false, false, false, false},
			updated:                 true,
			expectedUpdatesQueueLen: 0,
		},
	}

	// nolint:dupl  // Linter errors on lines are duplicate of TestSDKServerUpdateList, TestSDKServerAddListValue
	for test, testCase := range fixtures {
		t.Run(test, func(t *testing.T) {
			m := agtesting.NewMocks()

			gs := agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test", Namespace: "default", ResourceVersion: "0", Generation: 1,
				},
				Spec: agonesv1.GameServerSpec{
					SdkServer: agonesv1.SdkServer{
						LogLevel: "Debug",
					},
				},
				Status: agonesv1.GameServerStatus{
					Lists: lists,
				},
			}
			gs.ApplyDefaults()

			m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs.DeepCopy()}}, nil
			})

			updated := make(chan map[string]agonesv1.ListStatus, 10)

			m.AgonesClient.AddReactor("patch", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				gsCopy := patchGameServer(t, action, &gs)

				updated <- gsCopy.Status.Lists
				return true, gsCopy, nil
			})

			ctx, cancel := context.WithCancel(context.Background())
			sc, err := defaultSidecar(m)
			require.NoError(t, err)
			assert.NoError(t, sc.WaitForConnection(ctx))
			sc.informerFactory.Start(ctx.Done())
			require.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))
			sc.gsWaitForSync.Done()

			// check initial value comes through
			require.Eventually(t, func() bool {
				list, err := sc.GetList(context.Background(), &beta.GetListRequest{Name: testCase.listName})
				return cmp.Equal(list.Values, []string{"one", "two", "three", "four"}) && list.Capacity == 100 && err == nil
			}, 10*time.Second, time.Second)

			// Update the List
			for i, req := range testCase.requests {
				_, err = sc.RemoveListValue(context.Background(), req)
				if testCase.updateErrs[i] {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}

			got, err := sc.GetList(context.Background(), &beta.GetListRequest{Name: testCase.listName})
			assert.NoError(t, err)
			assert.Equal(t, testCase.want.Values, got.Values)
			assert.Equal(t, testCase.want.Capacity, got.Capacity)

			// start workerqueue processing at this point, so there is no chance of processing the above updates
			// earlier.
			sc.gsWaitForSync.Add(1)
			go func() {
				err = sc.Run(ctx)
				assert.NoError(t, err)
			}()

			// on an update, confirm that the update hits the K8s api
			if testCase.updated {
				select {
				case value := <-updated:
					assert.NotNil(t, value[testCase.listName])
					assert.Equal(t,
						agonesv1.ListStatus{Values: testCase.want.Values, Capacity: testCase.want.Capacity},
						value[testCase.listName])
				case <-time.After(10 * time.Second):
					assert.Fail(t, "List should have been patched")
				}
			}

			// on an update, confirms that the update queue list contains the right amount of items
			glu := sc.gsListUpdatesLen()
			assert.Equal(t, testCase.expectedUpdatesQueueLen, glu)

			cancel()
		})
	}
}

func TestSDKServerUpdateList(t *testing.T) {
	t.Parallel()
	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	err := agruntime.ParseFeatures(string(agruntime.FeatureCountsAndLists) + "=true")
	require.NoError(t, err, "Can not parse FeatureCountsAndLists feature")

	lists := map[string]agonesv1.ListStatus{
		"foo":  {Values: []string{"one", "two", "three", "four"}, Capacity: int64(100)},
		"bar":  {Values: []string{"one", "two", "three", "four"}, Capacity: int64(100)},
		"baz":  {Values: []string{"one", "two", "three", "four"}, Capacity: int64(100)},
		"qux":  {Values: []string{"one", "two", "three", "four"}, Capacity: int64(100)},
		"quux": {Values: []string{"one", "two", "three", "four"}, Capacity: int64(100)},
	}

	fixtures := map[string]struct {
		listName                string
		request                 *beta.UpdateListRequest
		want                    agonesv1.ListStatus
		updateErr               bool
		updated                 bool
		expectedUpdatesQueueLen int
	}{
		"set capacity to max": {
			listName: "foo",
			request: &beta.UpdateListRequest{
				List: &beta.List{
					Name:     "foo",
					Capacity: int64(1000),
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity"}},
			},
			want:                    agonesv1.ListStatus{Values: []string{"one", "two", "three", "four"}, Capacity: int64(1000)},
			updateErr:               false,
			updated:                 true,
			expectedUpdatesQueueLen: 0,
		},
		"set capacity to min values are truncated": {
			listName: "bar",
			request: &beta.UpdateListRequest{
				List: &beta.List{
					Name:     "bar",
					Capacity: int64(0),
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity"}},
			},
			want:                    agonesv1.ListStatus{Values: []string{}, Capacity: int64(0)},
			updateErr:               false,
			updated:                 true,
			expectedUpdatesQueueLen: 0,
		},
		"set capacity past max": {
			listName: "baz",
			request: &beta.UpdateListRequest{
				List: &beta.List{
					Name:     "baz",
					Capacity: int64(1001),
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity"}},
			},
			want:                    agonesv1.ListStatus{Values: []string{"one", "two", "three", "four"}, Capacity: int64(100)},
			updateErr:               true,
			updated:                 false,
			expectedUpdatesQueueLen: 0,
		},
		// New test cases to test updating values
		"update values below capacity": {
			listName: "qux",
			request: &beta.UpdateListRequest{
				List: &beta.List{
					Name:     "qux",
					Capacity: int64(100),
					Values:   []string{"one", "two", "three", "four", "five", "six"},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity", "values"}},
			},
			want:                    agonesv1.ListStatus{Values: []string{"one", "two", "three", "four", "five", "six"}, Capacity: int64(100)},
			updateErr:               false,
			updated:                 true,
			expectedUpdatesQueueLen: 0,
		},
		"update values above capacity": {
			listName: "quux",
			request: &beta.UpdateListRequest{
				List: &beta.List{
					Name:     "quux",
					Capacity: int64(4),
					Values:   []string{"one", "two", "three", "four", "five", "six"},
				},
				UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"capacity", "values"}},
			},
			want:                    agonesv1.ListStatus{Values: []string{"one", "two", "three", "four"}, Capacity: int64(4)},
			updateErr:               false,
			updated:                 true,
			expectedUpdatesQueueLen: 0,
		},
	}
	// Maximum capacity for the game server list.
	// of game server items the system can manage at once.
	GameServerListMaxCapacity = int64(1000)
	// nolint:dupl  // Linter errors on lines are duplicate of TestSDKServerAddListValue, TestSDKServerRemoveListValue
	for test, testCase := range fixtures {
		t.Run(test, func(t *testing.T) {
			m := agtesting.NewMocks()

			gs := agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test", Namespace: "default", ResourceVersion: "0", Generation: 1,
				},
				Spec: agonesv1.GameServerSpec{
					SdkServer: agonesv1.SdkServer{
						LogLevel: "Debug",
					},
				},
				Status: agonesv1.GameServerStatus{
					Lists: lists,
				},
			}
			gs.ApplyDefaults()

			m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs.DeepCopy()}}, nil
			})

			updated := make(chan map[string]agonesv1.ListStatus, 10)

			m.AgonesClient.AddReactor("patch", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				gsCopy := patchGameServer(t, action, &gs)

				updated <- gsCopy.Status.Lists
				return true, gsCopy, nil
			})

			ctx, cancel := context.WithCancel(context.Background())
			sc, err := defaultSidecar(m)
			require.NoError(t, err)
			assert.NoError(t, sc.WaitForConnection(ctx))
			sc.informerFactory.Start(ctx.Done())
			assert.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))

			wg := sync.WaitGroup{}
			wg.Add(1)

			go func() {
				err = sc.Run(ctx)
				assert.NoError(t, err)
				wg.Done()
			}()

			// check initial value comes through
			require.Eventually(t, func() bool {
				list, err := sc.GetList(context.Background(), &beta.GetListRequest{Name: testCase.listName})
				return cmp.Equal(list.Values, []string{"one", "two", "three", "four"}) && list.Capacity == 100 && err == nil
			}, 10*time.Second, time.Second)

			// Update the List
			_, err = sc.UpdateList(context.Background(), testCase.request)
			if testCase.updateErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			got, err := sc.GetList(context.Background(), &beta.GetListRequest{Name: testCase.listName})
			assert.NoError(t, err)
			assert.Equal(t, testCase.want.Values, got.Values)
			assert.Equal(t, testCase.want.Capacity, got.Capacity)

			// on an update, confirm that the update hits the K8s api
			if testCase.updated {
				select {
				case value := <-updated:
					assert.NotNil(t, value[testCase.listName])
					assert.Equal(t,
						agonesv1.ListStatus{Values: testCase.want.Values, Capacity: testCase.want.Capacity},
						value[testCase.listName])
				case <-time.After(10 * time.Second):
					assert.Fail(t, "List should have been patched")
				}
			}

			// on an update, confirm that the update queue list contains the right amount of items
			glu := sc.gsListUpdatesLen()
			assert.Equal(t, testCase.expectedUpdatesQueueLen, glu)

			cancel()
			wg.Wait()
		})
	}
}

func TestDeleteValues(t *testing.T) {
	t.Parallel()

	list := []string{"pDtUOSwMys", "MIaQYdeONT", "ZTwRNgZfxk", "ybtlfzfJau", "JwoYseCCyU", "JQJXhknLeG",
		"KDmxroeFvi", "fguLESWvmr", "xRUFzgrtuE", "UwElufBLtA", "jAySktznPe", "JZZRLkAtpQ", "BzHLffHxLd",
		"KWOyTiXsGP", "CtHFOMotCK", "SBOFIJBoBu", "gjYoIQLbAk", "krWVhxssxR", "ZTqRMKAqSx", "oDalBXZckY",
		"ZxATCXhBHk", "MTwgrrHePq", "KNGxlixHYt", "taZswVczZU", "beoXmuxAHE", "VbiLLJrRVs", "GrIEuiUlkB",
		"IPJhGxiKWY", "gYXZtGeFyd", "GYvKpRRsfj", "jRldDqcuEd", "ffPeeHOtMW", "AoEMlXWXVI", "HIjLrcvIqx",
		"GztXdbnxqg", "zSyNSIyQbp", "lntxdkIjVt", "jOgkkkaytV", "uHMvVtWKoc", "hetOAzBePn", "KqqkCbGLjS",
		"OQHRRtqIlq", "KFyHqLSACF", "nMZTcGlgAz", "iriNEjRLmh", "PRdGOtnyIo", "JDNDFYCIGi", "acalItODHz",
		"HJjxJnZWEu", "dmFWypNcDY", "fokGntWpON", "tQLmmXfDNW", "ZvyARYuebj", "ipHGcRmfWt", "MpTXveRDRg",
		"xPMoVLWeyj", "tXWeapJxkh", "KCMSWWiPMq", "fwsVKiWLuv", "AkKUUqwaOB", "DDlrgoWHGq", "DHScNuprJo",
		"PRMEGliSBU", "kqwktsjCNb", "vDuQZIhUHp", "YoazMkShki", "IwmXsZvlcp", "CJdrVMsjiD", "xNLnNvLRMN",
		"nKxDYSOkKx", "MWnrxVVOgK", "YnTHFAunKs", "DzUpkUxpuV", "kNVqCzjRxS", "IzqYWHDloX", "LvlVEniBqp",
		"CmdFcgTgzM", "qmORqLRaKv", "MxMnLiGOsY", "vAiAorAIdu", "pfhhTRFcpp", "ByqwQcKJYQ", "mKaeTCghbC",
		"eJssFVxVSI", "PGFMEopXax", "pYKCWZzGMf", "wIeRbiOdkf", "EKlxOXvqdF", "qOOorODUsn", "rcVUwlHOME",
		"etoDkduCkv", "iqUxYYUfpz", "ALyMkpYnbY", "TwfhVKGaIE", "zWsXruOeOn", "gNEmlDWmnj", "gEvodaSjIJ",
		"kOjWgLKjKE", "ATxBnODCKg", "liMbkiUTAs"}

	toDeleteMap := map[string]bool{"pDtUOSwMys": true, "beoXmuxAHE": true, "IPJhGxiKWY": true,
		"gYXZtGeFyd": true, "PRMEGliSBU": true, "kqwktsjCNb": true, "mKaeTCghbC": true,
		"PGFMEopXax": true, "qOOorODUsn": true, "rcVUwlHOME": true}

	newList := deleteValues(list, toDeleteMap)
	assert.Equal(t, len(list)-len(toDeleteMap), len(newList))
}

func TestSDKServerPlayerCapacity(t *testing.T) {
	t.Parallel()
	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	err := agruntime.ParseFeatures(string(agruntime.FeaturePlayerTracking) + "=true")
	require.NoError(t, err, "Can not parse FeaturePlayerTracking feature")

	m := agtesting.NewMocks()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	gs := agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test", Namespace: "default", ResourceVersion: "0",
		},
		Spec: agonesv1.GameServerSpec{
			SdkServer: agonesv1.SdkServer{
				LogLevel: "Debug",
			},
			Players: &agonesv1.PlayersSpec{
				InitialCapacity: 10,
			},
		},
	}
	gs.ApplyDefaults()

	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs.DeepCopy()}}, nil
	})

	updated := make(chan int64, 10)
	m.AgonesClient.AddReactor("patch", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {

		gsCopy := patchGameServer(t, action, &gs)

		updated <- gsCopy.Status.Players.Capacity
		return true, gsCopy, nil
	})

	assert.NoError(t, sc.WaitForConnection(ctx))
	sc.informerFactory.Start(ctx.Done())
	assert.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))

	go func() {
		err = sc.Run(ctx)
		assert.NoError(t, err)
	}()

	// check initial value comes through

	// async, so check after a period
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
		count, err := sc.GetPlayerCapacity(context.Background(), &alpha.Empty{})
		return count.Count == 10, err
	})
	assert.NoError(t, err)

	// on update from the SDK, the value is available from GetPlayerCapacity
	_, err = sc.SetPlayerCapacity(context.Background(), &alpha.Count{Count: 20})
	assert.NoError(t, err)

	count, err := sc.GetPlayerCapacity(context.Background(), &alpha.Empty{})
	require.NoError(t, err)
	assert.Equal(t, int64(20), count.Count)

	// on an update, confirm that the update hits the K8s api
	select {
	case value := <-updated:
		assert.Equal(t, int64(20), value)
	case <-time.After(time.Minute):
		assert.Fail(t, "Should have been patched")
	}

	agtesting.AssertEventContains(t, m.FakeRecorder.Events, "PlayerCapacity Set to 20")
}

func TestSDKServerPlayerConnectAndDisconnectWithoutPlayerTracking(t *testing.T) {
	t.Parallel()
	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	err := agruntime.ParseFeatures(string(agruntime.FeaturePlayerTracking) + "=false")
	require.NoError(t, err, "Can not parse FeaturePlayerTracking feature")

	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateReady,
		},
	}

	m := agtesting.NewMocks()
	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*fixture}}, nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	assert.NoError(t, sc.WaitForConnection(ctx))
	sc.informerFactory.Start(ctx.Done())
	assert.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))

	go func() {
		err = sc.Run(ctx)
		assert.NoError(t, err)
	}()

	// check initial value comes through
	// async, so check after a period
	e := &alpha.Empty{}
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
		count, err := sc.GetPlayerCapacity(context.Background(), e)

		assert.Nil(t, count)
		return false, err
	})
	assert.Error(t, err)

	count, err := sc.GetPlayerCount(context.Background(), e)
	require.Error(t, err)
	assert.Nil(t, count)

	list, err := sc.GetConnectedPlayers(context.Background(), e)
	require.Error(t, err)
	assert.Nil(t, list)

	id := &alpha.PlayerID{PlayerID: "test-player"}

	ok, err := sc.PlayerConnect(context.Background(), id)
	require.Error(t, err)
	assert.False(t, ok.Bool)

	ok, err = sc.IsPlayerConnected(context.Background(), id)
	require.Error(t, err)
	assert.False(t, ok.Bool)

	ok, err = sc.PlayerDisconnect(context.Background(), id)
	require.Error(t, err)
	assert.False(t, ok.Bool)
}

func TestSDKServerPlayerConnectAndDisconnect(t *testing.T) {
	t.Parallel()
	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	err := agruntime.ParseFeatures(string(agruntime.FeaturePlayerTracking) + "=true")
	require.NoError(t, err, "Can not parse FeaturePlayerTracking feature")

	m := agtesting.NewMocks()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	capacity := int64(3)
	gs := agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test", Namespace: "default", ResourceVersion: "0",
		},
		Spec: agonesv1.GameServerSpec{
			SdkServer: agonesv1.SdkServer{
				LogLevel: "Debug",
			},
			// this is here to give us a reference, so we know when sc.Run() has completed.
			Players: &agonesv1.PlayersSpec{
				InitialCapacity: capacity,
			},
		},
	}
	gs.ApplyDefaults()

	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs.DeepCopy()}}, nil
	})

	updated := make(chan *agonesv1.PlayerStatus, 10)
	m.AgonesClient.AddReactor("patch", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gsCopy := patchGameServer(t, action, &gs)
		updated <- gsCopy.Status.Players
		return true, gsCopy, nil
	})

	assert.NoError(t, sc.WaitForConnection(ctx))
	sc.informerFactory.Start(ctx.Done())
	assert.True(t, cache.WaitForCacheSync(ctx.Done(), sc.gameServerSynced))

	go func() {
		err = sc.Run(ctx)
		assert.NoError(t, err)
	}()

	// check initial value comes through
	// async, so check after a period
	e := &alpha.Empty{}
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
		count, err := sc.GetPlayerCapacity(context.Background(), e)
		return count.Count == capacity, err
	})
	assert.NoError(t, err)

	count, err := sc.GetPlayerCount(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count.Count)

	list, err := sc.GetConnectedPlayers(context.Background(), e)
	require.NoError(t, err)
	assert.Empty(t, list.List)

	ok, err := sc.IsPlayerConnected(context.Background(), &alpha.PlayerID{PlayerID: "1"})
	require.NoError(t, err)
	assert.False(t, ok.Bool, "no player connected yet")

	// sdk value should always be correct, even if we send more than one update per second.
	for i := int64(0); i < capacity; i++ {
		token := strconv.FormatInt(i, 10)
		id := &alpha.PlayerID{PlayerID: token}
		ok, err := sc.PlayerConnect(context.Background(), id)
		require.NoError(t, err)
		assert.True(t, ok.Bool, "Player "+token+" should not yet be connected")

		ok, err = sc.IsPlayerConnected(context.Background(), id)
		require.NoError(t, err)
		assert.True(t, ok.Bool, "Player "+token+" should be connected")
	}
	count, err = sc.GetPlayerCount(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, capacity, count.Count)

	list, err = sc.GetConnectedPlayers(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, []string{"0", "1", "2"}, list.List)

	// on an update, confirm that the update hits the K8s api, only once
	select {
	case value := <-updated:
		assert.Equal(t, capacity, value.Count)
		assert.Equal(t, []string{"0", "1", "2"}, value.IDs)
	case <-time.After(5 * time.Second):
		assert.Fail(t, "Should have been updated")
	}
	agtesting.AssertEventContains(t, m.FakeRecorder.Events, "PlayerCount Set to 3")

	// confirm there was only one update
	select {
	case <-updated:
		assert.Fail(t, "There should be only one update for the player connections")
	case <-time.After(2 * time.Second):
	}

	// should return an error if we try and add another, since we're at capacity
	nopePlayer := &alpha.PlayerID{PlayerID: "nope"}
	_, err = sc.PlayerConnect(context.Background(), nopePlayer)
	assert.EqualError(t, err, "players are already at capacity")

	// sdk value should always be correct, even if we send more than one update per second.
	// let's leave one player behind
	for i := int64(0); i < capacity-1; i++ {
		token := strconv.FormatInt(i, 10)
		id := &alpha.PlayerID{PlayerID: token}
		ok, err := sc.PlayerDisconnect(context.Background(), id)
		require.NoError(t, err)
		assert.Truef(t, ok.Bool, "Player %s should be disconnected", token)

		ok, err = sc.IsPlayerConnected(context.Background(), id)
		require.NoError(t, err)
		assert.Falsef(t, ok.Bool, "Player %s should be connected", token)
	}
	count, err = sc.GetPlayerCount(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count.Count)

	list, err = sc.GetConnectedPlayers(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, []string{"2"}, list.List)

	// on an update, confirm that the update hits the K8s api, only once
	select {
	case value := <-updated:
		assert.Equal(t, int64(1), value.Count)
		assert.Equal(t, []string{"2"}, value.IDs)
	case <-time.After(5 * time.Second):
		assert.Fail(t, "Should have been updated")
	}
	agtesting.AssertEventContains(t, m.FakeRecorder.Events, "PlayerCount Set to 1")

	// confirm there was only one update
	select {
	case <-updated:
		assert.Fail(t, "There should be only one update for the player disconnections")
	case <-time.After(2 * time.Second):
	}

	// last player is still there
	ok, err = sc.IsPlayerConnected(context.Background(), &alpha.PlayerID{PlayerID: "2"})
	require.NoError(t, err)
	assert.True(t, ok.Bool, "Player 2 should be connected")

	// finally, check idempotency of connect and disconnect
	id := &alpha.PlayerID{PlayerID: "2"} // only one left behind
	ok, err = sc.PlayerConnect(context.Background(), id)
	require.NoError(t, err)
	assert.False(t, ok.Bool, "Player 2 should already be connected")
	count, err = sc.GetPlayerCount(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count.Count)

	// no longer there.
	id.PlayerID = "0"
	ok, err = sc.PlayerDisconnect(context.Background(), id)
	require.NoError(t, err)
	assert.False(t, ok.Bool, "Player 2 should already be disconnected")
	count, err = sc.GetPlayerCount(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count.Count)

	agtesting.AssertNoEvent(t, m.FakeRecorder.Events)

	list, err = sc.GetConnectedPlayers(context.Background(), e)
	require.NoError(t, err)
	assert.Equal(t, []string{"2"}, list.List)
}

func TestSDKServerGracefulTerminationInterrupt(t *testing.T) {
	t.Parallel()
	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	m := agtesting.NewMocks()
	fakeWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	gs := agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test", Namespace: "default",
		},
		Spec: agonesv1.GameServerSpec{Health: agonesv1.Health{Disabled: true}},
	}
	gs.ApplyDefaults()
	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs.DeepCopy()}}, nil
	})
	sc, err := defaultSidecar(m)
	assert.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	sdkCtx := sc.NewSDKServerContext(ctx)
	assert.NoError(t, sc.WaitForConnection(sdkCtx))
	sc.informerFactory.Start(sdkCtx.Done())
	assert.True(t, cache.WaitForCacheSync(sdkCtx.Done(), sc.gameServerSynced))

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		err := sc.Run(sdkCtx)
		assert.Nil(t, err)
		wg.Done()
	}()

	assertContextCancelled := func(expected error, timeout time.Duration, ctx context.Context) {
		select {
		case <-ctx.Done():
			require.Equal(t, expected, ctx.Err())
		case <-time.After(timeout):
			require.Fail(t, "should have gone to Reserved by now")
		}
	}

	_, err = sc.Ready(sdkCtx, &sdk.Empty{})
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateRequestReady, sc.gsState)
	//	Mock interruption signal
	cancel()
	// Assert ctx is cancelled and sdkCtx is not cancelled
	assertContextCancelled(context.Canceled, 1*time.Second, ctx)
	assert.Nil(t, sdkCtx.Err())
	//	Assert gs is still requestReady
	assert.Equal(t, agonesv1.GameServerStateRequestReady, sc.gsState)
	// gs Shutdown
	gs.Status.State = agonesv1.GameServerStateShutdown
	fakeWatch.Modify(gs.DeepCopy())

	// Assert sdkCtx is cancelled after shutdown
	assertContextCancelled(context.Canceled, 1*time.Second, sdkCtx)
	wg.Wait()
}

func TestSDKServerGracefulTerminationShutdown(t *testing.T) {
	t.Parallel()
	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	m := agtesting.NewMocks()
	fakeWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	gs := agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test", Namespace: "default",
		},
		Spec: agonesv1.GameServerSpec{Health: agonesv1.Health{Disabled: true}},
	}
	gs.ApplyDefaults()
	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs.DeepCopy()}}, nil
	})

	sc, err := defaultSidecar(m)
	assert.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	sdkCtx := sc.NewSDKServerContext(ctx)
	assert.NoError(t, sc.WaitForConnection(sdkCtx))
	sc.informerFactory.Start(sdkCtx.Done())
	assert.True(t, cache.WaitForCacheSync(sdkCtx.Done(), sc.gameServerSynced))

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		err = sc.Run(sdkCtx)
		assert.Nil(t, err)
		wg.Done()
	}()

	assertContextCancelled := func(expected error, timeout time.Duration, ctx context.Context) {
		select {
		case <-ctx.Done():
			require.Equal(t, expected, ctx.Err())
		case <-time.After(timeout):
			require.Fail(t, "should have gone to Reserved by now")
		}
	}

	_, err = sc.Ready(sdkCtx, &sdk.Empty{})
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateRequestReady, sc.gsState)
	// gs Shutdown
	gs.Status.State = agonesv1.GameServerStateShutdown
	fakeWatch.Modify(gs.DeepCopy())

	// assert none of the context have been cancelled
	assert.Nil(t, sdkCtx.Err())
	assert.Nil(t, ctx.Err())
	//	Mock interruption signal
	cancel()
	// Assert ctx is cancelled and sdkCtx is not cancelled
	assertContextCancelled(context.Canceled, 2*time.Second, ctx)
	assertContextCancelled(context.Canceled, 2*time.Second, sdkCtx)
	wg.Wait()
}

func TestSDKServerGracefulTerminationGameServerStateChannel(t *testing.T) {
	t.Parallel()
	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	m := agtesting.NewMocks()
	fakeWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	gs := agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test", Namespace: "default",
		},
		Spec: agonesv1.GameServerSpec{Health: agonesv1.Health{Disabled: true}},
	}
	gs.ApplyDefaults()
	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs.DeepCopy()}}, nil
	})

	sc, err := defaultSidecar(m)
	assert.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sdkCtx := sc.NewSDKServerContext(ctx)
	sc.informerFactory.Start(sdkCtx.Done())
	assert.True(t, cache.WaitForCacheSync(sdkCtx.Done(), sc.gameServerSynced))

	gs.Status.State = agonesv1.GameServerStateShutdown
	fakeWatch.Modify(gs.DeepCopy())

	select {
	case current := <-sc.gsStateChannel:
		require.Equal(t, agonesv1.GameServerStateShutdown, current)
	case <-time.After(5 * time.Second):
		require.Fail(t, "should have gone to Shutdown by now")
	}
}

func defaultSidecar(m agtesting.Mocks) (*SDKServer, error) {
	server, err := NewSDKServer("test", "default", m.KubeClient, m.AgonesClient, logrus.DebugLevel, 8080, 500*time.Millisecond)
	if err != nil {
		return server, err
	}

	server.recorder = m.FakeRecorder
	return server, err
}

func waitForMessage(sc *SDKServer) error {
	return wait.PollUntilContextTimeout(context.Background(), time.Second, 5*time.Second, true, func(_ context.Context) (done bool, err error) {
		sc.healthMutex.RLock()
		defer sc.healthMutex.RUnlock()
		return sc.clock.Now().UTC() == sc.healthLastUpdated, nil
	})
}

func waitConnectedStreamCount(sc *SDKServer, count int) error { //nolint:unparam // Keep flexibility.
	return wait.PollUntilContextTimeout(context.Background(), 1*time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
		sc.streamMutex.RLock()
		defer sc.streamMutex.RUnlock()
		return len(sc.connectedStreams) == count, nil
	})
}

func asyncWatchGameServer(t *testing.T, sc *SDKServer, stream sdk.SDK_WatchGameServerServer) {
	// Note that WatchGameServer() uses getGameServer() and would block
	// if gsWaitForSync is not Done().
	go func() {
		err := sc.WatchGameServer(&sdk.Empty{}, stream)
		require.NoError(t, err)
	}()
}

func TestSetLabel_InvalidLabelRejected(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()

	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	// Invalid label key (spaces are not allowed)
	_, err = sc.SetLabel(context.Background(), &sdk.KeyValue{
		Key:   "INVALID KEY",
		Value: "x",
	})

	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok, "expected gRPC status error")
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestSetAnnotation_NilAndOverlimit(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()
	sc, err := defaultSidecar(m)
	require.NoError(t, err)

	testCases := []struct {
		name string
		kv   *sdk.KeyValue
	}{
		{
			name: "Nil KeyValue Object",
			kv:   nil,
		},
		{
			name: "Overlimit Key",
			kv:   &sdk.KeyValue{Key: strings.Repeat("a", 64), Value: "v"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := sc.SetAnnotation(context.Background(), tc.kv)
			require.Error(t, err, "Case %s should have failed", tc.name)
			st, ok := status.FromError(err)
			assert.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, st.Code())
		})
	}
}
