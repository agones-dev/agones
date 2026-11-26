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

package gameserversets

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/gameservers"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
	testclocks "k8s.io/utils/clock/testing"
)

var testGameServerCount atomic.Int64

func uniqueGameServerName() string {
	return fmt.Sprintf("gs-%d", testGameServerCount.Add(1))
}

func unhealthyBeforeReadyGameServer(createdAt, failedAt time.Time) *agonesv1.GameServer {
	return &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:              uniqueGameServerName(),
			CreationTimestamp: metav1.NewTime(createdAt),
			Annotations: map[string]string{
				agonesv1.GameServerUnhealthyBeforeReadyAtAnnotation: failedAt.UTC().Format(time.RFC3339),
			},
		},
		Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateUnhealthy},
	}
}

func gameServerCreatedAt(state agonesv1.GameServerState, createdAt time.Time) *agonesv1.GameServer {
	return &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: uniqueGameServerName(), CreationTimestamp: metav1.NewTime(createdAt)},
		Status:     agonesv1.GameServerStatus{State: state},
	}
}

func failBeforeReady(gs *agonesv1.GameServer, failedAt time.Time) {
	gs.ObjectMeta.Annotations = map[string]string{
		agonesv1.GameServerUnhealthyBeforeReadyAtAnnotation: failedAt.UTC().Format(time.RFC3339),
	}
	gs.Status.State = agonesv1.GameServerStateUnhealthy
}

// startUnhealthyBackoffDelay fails a GameServer at 5s and another at 15s, leaving the backoff at its base delay.
func startUnhealthyBackoffDelay(t *testing.T, b *unhealthyBackoff, gsSet *agonesv1.GameServerSet, at func(int) time.Time) []*agonesv1.GameServer {
	t.Helper()

	list := []*agonesv1.GameServer{unhealthyBeforeReadyGameServer(at(0), at(5))}
	require.Equal(t, time.Duration(0), b.observe(gsSet, list, at(5)))

	list = []*agonesv1.GameServer{unhealthyBeforeReadyGameServer(at(5), at(15))}
	require.Equal(t, unhealthyBackoffBase, b.observe(gsSet, list, at(15)))
	return list
}

func TestUnhealthyBackoffObserve(t *testing.T) {
	t.Parallel()

	gsSet := &agonesv1.GameServerSet{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gsSet"}}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(seconds int) time.Time { return start.Add(time.Duration(seconds) * time.Second) }

	t.Run("no failures, no delay", func(t *testing.T) {
		b := newUnhealthyBackoff(40 * time.Second)
		list := []*agonesv1.GameServer{
			gameServerCreatedAt(agonesv1.GameServerStateReady, at(0)),
			gameServerCreatedAt(agonesv1.GameServerStateUnhealthy, at(0)),
		}
		assert.Equal(t, time.Duration(0), b.observe(gsSet, list, at(5)))
		assert.Empty(t, b.entries)
	})

	t.Run("the first failure is replaced straight away, then the delay doubles up to the max", func(t *testing.T) {
		b := newUnhealthyBackoff(40 * time.Second)

		list := []*agonesv1.GameServer{unhealthyBeforeReadyGameServer(at(0), at(5))}
		assert.Equal(t, time.Duration(0), b.observe(gsSet, list, at(5)))
		assert.Equal(t, time.Duration(0), b.observe(gsSet, list, at(6)))

		list = []*agonesv1.GameServer{unhealthyBeforeReadyGameServer(at(5), at(15))}
		assert.Equal(t, 10*time.Second, b.observe(gsSet, list, at(15)))
		assert.Equal(t, 10*time.Second, b.observe(gsSet, list, at(24)))

		list = []*agonesv1.GameServer{unhealthyBeforeReadyGameServer(at(25), at(30))}
		assert.Equal(t, 20*time.Second, b.observe(gsSet, list, at(30)))

		list = []*agonesv1.GameServer{unhealthyBeforeReadyGameServer(at(50), at(55))}
		assert.Equal(t, 40*time.Second, b.observe(gsSet, list, at(55)))

		list = []*agonesv1.GameServer{unhealthyBeforeReadyGameServer(at(95), at(100))}
		assert.Equal(t, 40*time.Second, b.observe(gsSet, list, at(100)))
	})

	t.Run("failures soon after the first one are replaced straight away too", func(t *testing.T) {
		b := newUnhealthyBackoff(40 * time.Second)

		first := unhealthyBeforeReadyGameServer(at(0), at(5))
		second := gameServerCreatedAt(agonesv1.GameServerStateScheduled, at(0))
		third := gameServerCreatedAt(agonesv1.GameServerStateScheduled, at(0))
		list := []*agonesv1.GameServer{first, second, third}
		assert.Equal(t, time.Duration(0), b.observe(gsSet, list, at(5)))

		failBeforeReady(second, at(7))
		assert.Equal(t, time.Duration(0), b.observe(gsSet, list, at(7)))

		failBeforeReady(third, at(9))
		assert.Equal(t, time.Duration(0), b.observe(gsSet, list, at(9)))

		list = append(list, unhealthyBeforeReadyGameServer(at(5), at(15)))
		assert.Equal(t, 10*time.Second, b.observe(gsSet, list, at(15)))
	})

	t.Run("failures within the same delay period count once", func(t *testing.T) {
		b := newUnhealthyBackoff(40 * time.Second)
		list := startUnhealthyBackoffDelay(t, b, gsSet, at)

		list = append(list, unhealthyBeforeReadyGameServer(at(7), at(17)))
		assert.Equal(t, 10*time.Second, b.observe(gsSet, list, at(17)))

		list = append(list, unhealthyBeforeReadyGameServer(at(9), at(19)))
		assert.Equal(t, 10*time.Second, b.observe(gsSet, list, at(19)))
	})

	t.Run("a GameServer created with the failed one clears the delay when it becomes usable", func(t *testing.T) {
		b := newUnhealthyBackoff(40 * time.Second)
		list := startUnhealthyBackoffDelay(t, b, gsSet, at)

		failed := unhealthyBeforeReadyGameServer(at(20), at(25))
		sibling := gameServerCreatedAt(agonesv1.GameServerStateScheduled, at(20))
		list = append(list, failed, sibling)
		assert.Equal(t, 20*time.Second, b.observe(gsSet, list, at(25)))

		sibling.Status.State = agonesv1.GameServerStateReady
		assert.Equal(t, time.Duration(0), b.observe(gsSet, list, at(32)))
		assert.Empty(t, b.entries)
	})

	t.Run("becoming usable and a new failure seen together clears the delay", func(t *testing.T) {
		b := newUnhealthyBackoff(40 * time.Second)
		startUnhealthyBackoffDelay(t, b, gsSet, at)

		list := []*agonesv1.GameServer{
			unhealthyBeforeReadyGameServer(at(20), at(25)),
			gameServerCreatedAt(agonesv1.GameServerStateReady, at(20)),
		}
		assert.Equal(t, time.Duration(0), b.observe(gsSet, list, at(25)))
		assert.Empty(t, b.entries)
	})

	t.Run("a GameServer that becomes usable after the last failure clears the delay", func(t *testing.T) {
		b := newUnhealthyBackoff(40 * time.Second)

		// was already Ready before the failures, so says nothing about whether they are over
		older := gameServerCreatedAt(agonesv1.GameServerStateReady, at(0))
		assert.Equal(t, time.Duration(0), b.observe(gsSet, []*agonesv1.GameServer{older, unhealthyBeforeReadyGameServer(at(0), at(5))}, at(5)))
		failed := unhealthyBeforeReadyGameServer(at(5), at(15))
		assert.Equal(t, 10*time.Second, b.observe(gsSet, []*agonesv1.GameServer{older, failed}, at(15)))

		starting := gameServerCreatedAt(agonesv1.GameServerStateScheduled, at(20))
		assert.Equal(t, 10*time.Second, b.observe(gsSet, []*agonesv1.GameServer{older, failed, starting}, at(21)))

		starting.Status.State = agonesv1.GameServerStateReady
		assert.Equal(t, time.Duration(0), b.observe(gsSet, []*agonesv1.GameServer{older, failed, starting}, at(23)))
		assert.Empty(t, b.entries)
	})

	t.Run("the replacement of the first failure becoming usable clears the state", func(t *testing.T) {
		b := newUnhealthyBackoff(40 * time.Second)

		list := []*agonesv1.GameServer{unhealthyBeforeReadyGameServer(at(0), at(5))}
		assert.Equal(t, time.Duration(0), b.observe(gsSet, list, at(5)))
		assert.Len(t, b.entries, 1)

		list = []*agonesv1.GameServer{gameServerCreatedAt(agonesv1.GameServerStateReady, at(5))}
		assert.Equal(t, time.Duration(0), b.observe(gsSet, list, at(10)))
		assert.Empty(t, b.entries)
	})

	t.Run("GameServers being deleted are ignored", func(t *testing.T) {
		b := newUnhealthyBackoff(40 * time.Second)

		failed := unhealthyBeforeReadyGameServer(at(0), at(5))
		deletedAt := metav1.NewTime(at(6))
		failed.ObjectMeta.DeletionTimestamp = &deletedAt
		assert.Equal(t, time.Duration(0), b.observe(gsSet, []*agonesv1.GameServer{failed}, at(6)))
	})

	t.Run("state expires when there have been no failures for a long time", func(t *testing.T) {
		b := newUnhealthyBackoff(40 * time.Second)
		startUnhealthyBackoffDelay(t, b, gsSet, at)

		assert.Equal(t, 10*time.Second, b.observe(gsSet, nil, at(65)))
		assert.Equal(t, time.Duration(0), b.observe(gsSet, nil, at(96)))
		assert.Empty(t, b.entries)
	})

	t.Run("a failure a long time after the last one starts again", func(t *testing.T) {
		b := newUnhealthyBackoff(40 * time.Second)
		startUnhealthyBackoffDelay(t, b, gsSet, at)

		list := []*agonesv1.GameServer{unhealthyBeforeReadyGameServer(at(90), at(96))}
		assert.Equal(t, time.Duration(0), b.observe(gsSet, list, at(96)))

		list = []*agonesv1.GameServer{unhealthyBeforeReadyGameServer(at(96), at(106))}
		assert.Equal(t, 10*time.Second, b.observe(gsSet, list, at(106)))
	})

	t.Run("deleting the GameServerSet clears its state", func(t *testing.T) {
		b := newUnhealthyBackoff(40 * time.Second)

		list := []*agonesv1.GameServer{unhealthyBeforeReadyGameServer(at(0), at(5))}
		assert.Equal(t, time.Duration(0), b.observe(gsSet, list, at(5)))
		assert.Len(t, b.entries, 1)
		b.deleteGameServerSet(gsSet)
		assert.Empty(t, b.entries)
	})
}

func TestNextUnhealthyDelayEnd(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	at := func(seconds int) time.Time { return start.Add(time.Duration(seconds) * time.Second) }

	list := []*agonesv1.GameServer{
		gameServerCreatedAt(agonesv1.GameServerStateReady, at(0)),
		gameServerCreatedAt(agonesv1.GameServerStateUnhealthy, at(0)),
		unhealthyBeforeReadyGameServer(at(0), at(8)),
		unhealthyBeforeReadyGameServer(at(0), at(5)),
		unhealthyBeforeReadyGameServer(at(0), at(1)),
	}

	next, ok := nextUnhealthyDelayEnd(list, 10*time.Second, at(12))
	assert.True(t, ok)
	assert.Equal(t, 3*time.Second, next)

	_, ok = nextUnhealthyDelayEnd(list, 10*time.Second, at(18))
	assert.False(t, ok)

	_, ok = nextUnhealthyDelayEnd(list, 0, at(12))
	assert.False(t, ok)
}

func TestComputeReconciliationActionUnhealthyBeforeReady(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	compute := func(list []*agonesv1.GameServer, replicas int, delay time.Duration) (int, []*agonesv1.GameServer) {
		toAdd, toDelete, _ := computeReconciliationAction(apis.Distributed, list, map[string]gameservers.NodeCount{},
			replicas, 1000, 1000, 1000, nil, delay, now)
		return toAdd, toDelete
	}

	t.Run("kept while within the delay", func(t *testing.T) {
		list := []*agonesv1.GameServer{
			unhealthyBeforeReadyGameServer(now, now.Add(-5*time.Second)),
			gameServerCreatedAt(agonesv1.GameServerStateReady, now),
		}
		toAdd, toDelete := compute(list, 2, 30*time.Second)
		assert.Equal(t, 0, toAdd)
		assert.Empty(t, toDelete)
	})

	t.Run("replaced once the delay has passed", func(t *testing.T) {
		list := []*agonesv1.GameServer{
			unhealthyBeforeReadyGameServer(now, now.Add(-45*time.Second)),
			gameServerCreatedAt(agonesv1.GameServerStateReady, now),
		}
		toAdd, toDelete := compute(list, 2, 30*time.Second)
		assert.Equal(t, 1, toAdd)
		assert.Equal(t, []*agonesv1.GameServer{list[0]}, toDelete)
	})

	t.Run("replaced straight away when there is no delay", func(t *testing.T) {
		list := []*agonesv1.GameServer{unhealthyBeforeReadyGameServer(now, now)}
		toAdd, toDelete := compute(list, 1, 0)
		assert.Equal(t, 1, toAdd)
		assert.Len(t, toDelete, 1)
	})

	t.Run("Unhealthy after Ready is replaced straight away", func(t *testing.T) {
		list := []*agonesv1.GameServer{gameServerCreatedAt(agonesv1.GameServerStateUnhealthy, now)}
		toAdd, toDelete := compute(list, 1, 30*time.Second)
		assert.Equal(t, 1, toAdd)
		assert.Len(t, toDelete, 1)
	})

	t.Run("kept GameServers are removed on scale down", func(t *testing.T) {
		list := []*agonesv1.GameServer{
			unhealthyBeforeReadyGameServer(now, now),
			unhealthyBeforeReadyGameServer(now, now),
		}
		toAdd, toDelete := compute(list, 0, 30*time.Second)
		assert.Equal(t, 0, toAdd)
		assert.Len(t, toDelete, 2)
	})

	t.Run("kept GameServers are removed before Ready ones on scale down", func(t *testing.T) {
		list := []*agonesv1.GameServer{
			gameServerCreatedAt(agonesv1.GameServerStateReady, now.Add(-time.Minute)),
			unhealthyBeforeReadyGameServer(now, now),
			gameServerCreatedAt(agonesv1.GameServerStateReady, now.Add(-2*time.Minute)),
		}
		toAdd, toDelete := compute(list, 2, 30*time.Second)
		assert.Equal(t, 0, toAdd)
		assert.Equal(t, []*agonesv1.GameServer{list[1]}, toDelete)

		toAdd, toDelete = compute(list, 1, 30*time.Second)
		assert.Equal(t, 0, toAdd)
		assert.Equal(t, []*agonesv1.GameServer{list[1], list[2]}, toDelete)
	})
}

func TestSyncGameServerSetUnhealthyBeforeReady(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	gsSet := defaultFixture()
	gsSet.Spec.Replicas = 1
	key := gsSet.ObjectMeta.Namespace + "/" + gsSet.ObjectMeta.Name

	list := createGameServers(gsSet, 1)
	failBeforeReady(&list[0], start)

	c, m := newFakeController()
	fc := testclocks.NewFakeClock(start.Add(5 * time.Second))
	c.clock = fc
	c.unhealthyBackoff.entries[key] = &unhealthyBackoffEntry{delay: unhealthyBackoffBase, lastStep: start, lastFailure: start}

	var deleted, created atomic.Int64
	m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
	})
	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: list}, nil
	})
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gs := action.(k8stesting.UpdateAction).GetObject().(*agonesv1.GameServer)
		assert.Equal(t, agonesv1.GameServerStateShutdown, gs.Status.State)
		deleted.Add(1)
		return true, gs, nil
	})
	m.AgonesClient.AddReactor("create", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		created.Add(1)
		return true, action.(k8stesting.CreateAction).GetObject(), nil
	})

	ctx, cancel := agtesting.StartInformers(m, c.gameServerSetSynced, c.gameServerSynced)
	defer cancel()

	require.NoError(t, c.syncGameServerSet(ctx, key))
	assert.Equal(t, int64(0), deleted.Load())
	assert.Equal(t, int64(0), created.Load())

	fc.SetTime(start.Add(unhealthyBackoffBase))
	require.NoError(t, c.syncGameServerSet(ctx, key))
	assert.Equal(t, int64(1), deleted.Load())
	assert.Equal(t, int64(1), created.Load())
}
