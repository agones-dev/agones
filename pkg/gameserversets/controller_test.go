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
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/cloudproduct/generic"
	"agones.dev/agones/pkg/gameservers"
	agtesting "agones.dev/agones/pkg/testing"
	utilruntime "agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/webhooks"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func gsWithState(st agonesv1.GameServerState) *agonesv1.GameServer {
	return &agonesv1.GameServer{Status: agonesv1.GameServerStatus{State: st}}
}

func gsPendingDeletionWithState(st agonesv1.GameServerState) *agonesv1.GameServer {
	return &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			DeletionTimestamp: &deletionTime,
		},
		Status: agonesv1.GameServerStatus{State: st},
	}
}

const (
	maxTestCreationsPerBatch = 3
	maxTestDeletionsPerBatch = 3
	maxTestPendingPerBatch   = 3
)

func TestComputeReconciliationAction(t *testing.T) {
	t.Parallel()

	cases := []struct {
		desc                   string
		list                   []*agonesv1.GameServer
		targetReplicaCount     int
		wantNumServersToAdd    int
		wantNumServersToDelete int
		wantIsPartial          bool
		priorities             []agonesv1.Priority
	}{
		{
			desc: "Empty",
		},
		{
			desc: "AddServers",
			list: []*agonesv1.GameServer{
				gsWithState(agonesv1.GameServerStateReady),
			},
			targetReplicaCount:  3,
			wantNumServersToAdd: 2,
		},
		{
			// 1 ready servers, target is 30 but we can only create 3 at a time.
			desc: "AddServersPartial",
			list: []*agonesv1.GameServer{
				gsWithState(agonesv1.GameServerStateReady),
			},
			targetReplicaCount:  30,
			wantNumServersToAdd: 3,
			wantIsPartial:       true, // max 3 creations per action
		},
		{
			// 0 ready servers, target is 30 but we can only have 3 in-flight.
			desc: "AddServersExceedsInFlightLimit",
			list: []*agonesv1.GameServer{
				gsWithState(agonesv1.GameServerStateCreating),
				gsWithState(agonesv1.GameServerStatePortAllocation),
			},
			targetReplicaCount:  30,
			wantNumServersToAdd: 1,
			wantIsPartial:       true,
		}, {
			desc: "DeleteServers",
			list: []*agonesv1.GameServer{
				gsWithState(agonesv1.GameServerStateReady),
				gsWithState(agonesv1.GameServerStateReserved),
				gsWithState(agonesv1.GameServerStateReady),
			},
			targetReplicaCount:     1,
			wantNumServersToDelete: 2,
		},
		{
			// 6 ready servers, target is 1 but we can only delete 3 at a time.
			desc: "DeleteServerPartial",
			list: []*agonesv1.GameServer{
				gsWithState(agonesv1.GameServerStateReady),
				gsWithState(agonesv1.GameServerStateReady),
				gsWithState(agonesv1.GameServerStateReady),
				gsWithState(agonesv1.GameServerStateReady),
				gsWithState(agonesv1.GameServerStateReady),
				gsWithState(agonesv1.GameServerStateReady),
			},
			targetReplicaCount:     1,
			wantNumServersToDelete: 3,
			wantIsPartial:          true, // max 3 deletions per action
		},
		{
			desc: "DeleteIgnoresAllocatedServers",
			list: []*agonesv1.GameServer{
				gsWithState(agonesv1.GameServerStateReady),
				gsWithState(agonesv1.GameServerStateAllocated),
				gsWithState(agonesv1.GameServerStateAllocated),
			},
			targetReplicaCount:     0,
			wantNumServersToDelete: 1,
		},
		{
			desc: "DeleteIgnoresReservedServers",
			list: []*agonesv1.GameServer{
				gsWithState(agonesv1.GameServerStateReady),
				gsWithState(agonesv1.GameServerStateReserved),
				gsWithState(agonesv1.GameServerStateReserved),
			},
			targetReplicaCount:     0,
			wantNumServersToDelete: 1,
		},
		{
			desc: "CreateWhileDeletionsPending",
			list: []*agonesv1.GameServer{
				// 2 being deleted, one ready, target is 4, we add 3 more.
				gsPendingDeletionWithState(agonesv1.GameServerStateUnhealthy),
				gsPendingDeletionWithState(agonesv1.GameServerStateUnhealthy),
				gsWithState(agonesv1.GameServerStateReady),
			},
			targetReplicaCount:  4,
			wantNumServersToAdd: 3,
		},
		{
			desc: "PendingDeletionsCountTowardsTargetReplicaCount",
			list: []*agonesv1.GameServer{
				// 6 being deleted now, we want 10 but that would exceed in-flight limit by a lot.
				gsWithState(agonesv1.GameServerStateCreating),
				gsWithState(agonesv1.GameServerStatePortAllocation),
				gsWithState(agonesv1.GameServerStateCreating),
				gsWithState(agonesv1.GameServerStatePortAllocation),
				gsWithState(agonesv1.GameServerStateCreating),
				gsWithState(agonesv1.GameServerStatePortAllocation),
			},
			targetReplicaCount:  10,
			wantNumServersToAdd: 0,
			wantIsPartial:       true,
		},
		{
			desc: "DeletingUnhealthyGameServers",
			list: []*agonesv1.GameServer{
				gsWithState(agonesv1.GameServerStateReady),
				gsWithState(agonesv1.GameServerStateUnhealthy),
				gsWithState(agonesv1.GameServerStateUnhealthy),
			},
			targetReplicaCount:     3,
			wantNumServersToAdd:    2,
			wantNumServersToDelete: 2,
		},
		{
			desc: "DeletingErrorGameServers",
			list: []*agonesv1.GameServer{
				gsWithState(agonesv1.GameServerStateReady),
				gsWithState(agonesv1.GameServerStateError),
				gsWithState(agonesv1.GameServerStateError),
			},
			targetReplicaCount:     3,
			wantNumServersToAdd:    2,
			wantNumServersToDelete: 2,
		},
		{
			desc: "DeletingPartialGameServers",
			list: []*agonesv1.GameServer{
				gsWithState(agonesv1.GameServerStateReady),
				gsWithState(agonesv1.GameServerStateUnhealthy),
				gsWithState(agonesv1.GameServerStateError),
				gsWithState(agonesv1.GameServerStateUnhealthy),
				gsWithState(agonesv1.GameServerStateError),
				gsWithState(agonesv1.GameServerStateUnhealthy),
				gsWithState(agonesv1.GameServerStateError),
			},
			targetReplicaCount:     3,
			wantNumServersToAdd:    2,
			wantNumServersToDelete: 3,
			wantIsPartial:          true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			toAdd, toDelete, isPartial := computeReconciliationAction(apis.Distributed, tc.list, map[string]gameservers.NodeCount{},
				tc.targetReplicaCount, maxTestCreationsPerBatch, maxTestDeletionsPerBatch, maxTestPendingPerBatch, tc.priorities)

			assert.Equal(t, tc.wantNumServersToAdd, toAdd, "# of GameServers to add")
			assert.Len(t, toDelete, tc.wantNumServersToDelete, "# of GameServers to delete")
			assert.Equal(t, tc.wantIsPartial, isPartial, "is partial reconciliation")
		})
	}

	t.Run("test packed scale down", func(t *testing.T) {
		list := []*agonesv1.GameServer{
			{ObjectMeta: metav1.ObjectMeta{Name: "gs1"}, Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady, NodeName: "node3"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs2"}, Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady, NodeName: "node1"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs3"}, Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady, NodeName: "node3"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs4"}, Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady, NodeName: ""}},
		}

		counts := map[string]gameservers.NodeCount{"node1": {Ready: 1}, "node3": {Ready: 2}}
		toAdd, toDelete, isPartial := computeReconciliationAction(apis.Packed, list, counts, 2,
			1000, 1000, 1000, nil)

		assert.Empty(t, toAdd)
		assert.False(t, isPartial, "shouldn't be partial")

		assert.Len(t, toDelete, 2)
		assert.Equal(t, "gs4", toDelete[0].ObjectMeta.Name)
		assert.Equal(t, "gs2", toDelete[1].ObjectMeta.Name)
	})

	t.Run("test distributed scale down", func(t *testing.T) {
		now := metav1.Now()

		list := []*agonesv1.GameServer{
			{ObjectMeta: metav1.ObjectMeta{Name: "gs1",
				CreationTimestamp: metav1.Time{Time: now.Add(10 * time.Second)}}, Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs2",
				CreationTimestamp: now}, Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs3",
				CreationTimestamp: metav1.Time{Time: now.Add(40 * time.Second)}}, Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}},
			{ObjectMeta: metav1.ObjectMeta{Name: "gs4",
				CreationTimestamp: metav1.Time{Time: now.Add(30 * time.Second)}}, Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}},
		}

		toAdd, toDelete, isPartial := computeReconciliationAction(apis.Distributed, list, map[string]gameservers.NodeCount{},
			2, 1000, 1000, 1000, nil)

		assert.Empty(t, toAdd)
		assert.False(t, isPartial, "shouldn't be partial")

		assert.Len(t, toDelete, 2)
		assert.Equal(t, "gs2", toDelete[0].ObjectMeta.Name)
		assert.Equal(t, "gs1", toDelete[1].ObjectMeta.Name)
	})
}

func TestComputeStatus(t *testing.T) {
	t.Parallel()

	t.Run("compute status", func(t *testing.T) {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()

		require.NoError(t, utilruntime.ParseFeatures(fmt.Sprintf("%s=false", utilruntime.FeatureCountsAndLists)))
		gsSet := defaultFixture()
		cases := []struct {
			list       []*agonesv1.GameServer
			wantStatus agonesv1.GameServerSetStatus
		}{
			{[]*agonesv1.GameServer{}, agonesv1.GameServerSetStatus{}},
			{[]*agonesv1.GameServer{
				gsWithState(agonesv1.GameServerStateCreating),
				gsWithState(agonesv1.GameServerStateReady),
			}, agonesv1.GameServerSetStatus{ReadyReplicas: 1, Replicas: 2}},
			{[]*agonesv1.GameServer{
				gsWithState(agonesv1.GameServerStateAllocated),
				gsWithState(agonesv1.GameServerStateAllocated),
				gsWithState(agonesv1.GameServerStateCreating),
				gsWithState(agonesv1.GameServerStateReady),
			}, agonesv1.GameServerSetStatus{ReadyReplicas: 1, AllocatedReplicas: 2, Replicas: 4}},
			{
				list: []*agonesv1.GameServer{
					gsWithState(agonesv1.GameServerStateReserved),
					gsWithState(agonesv1.GameServerStateReserved),
					gsWithState(agonesv1.GameServerStateReady),
				},
				wantStatus: agonesv1.GameServerSetStatus{Replicas: 3, ReadyReplicas: 1, ReservedReplicas: 2},
			},
		}

		for _, tc := range cases {
			assert.Equal(t, tc.wantStatus, computeStatus(gsSet, tc.list))
		}
	})

	t.Run("player tracking", func(t *testing.T) {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()

		require.NoError(t, utilruntime.ParseFeatures(fmt.Sprintf("%s=true", utilruntime.FeaturePlayerTracking)))

		gsSet := defaultFixture()
		var list []*agonesv1.GameServer
		gs1 := gsWithState(agonesv1.GameServerStateAllocated)
		gs1.Status.Players = &agonesv1.PlayerStatus{Count: 5, Capacity: 10}
		gs2 := gsWithState(agonesv1.GameServerStateReserved)
		gs2.Status.Players = &agonesv1.PlayerStatus{Count: 10, Capacity: 15}
		gs3 := gsWithState(agonesv1.GameServerStateCreating)
		gs3.Status.Players = &agonesv1.PlayerStatus{Count: 20, Capacity: 30}
		gs4 := gsWithState(agonesv1.GameServerStateReady)
		gs4.Status.Players = &agonesv1.PlayerStatus{Count: 15, Capacity: 30}
		list = append(list, gs1, gs2, gs3, gs4)

		expected := agonesv1.GameServerSetStatus{
			Replicas:          4,
			ReadyReplicas:     1,
			ReservedReplicas:  1,
			AllocatedReplicas: 1,
			Players: &agonesv1.AggregatedPlayerStatus{
				Count:    30,
				Capacity: 55,
			},
			Counters: map[string]agonesv1.AggregatedCounterStatus{},
			Lists:    map[string]agonesv1.AggregatedListStatus{},
		}

		assert.Equal(t, expected, computeStatus(gsSet, list))
	})

	t.Run("counters", func(t *testing.T) {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()

		require.NoError(t, utilruntime.ParseFeatures(fmt.Sprintf("%s=true", utilruntime.FeatureCountsAndLists)))

		gsSet := defaultFixture()
		var list []*agonesv1.GameServer
		gs1 := gsWithState(agonesv1.GameServerStateAllocated)
		gs1.Status.Counters = map[string]agonesv1.CounterStatus{
			"firstCounter":  {Count: 5, Capacity: 10},
			"secondCounter": {Count: 100, Capacity: 1000},
			"fullCounter":   {Count: 9223372036854775807, Capacity: 9223372036854775807},
		}
		gs2 := gsWithState(agonesv1.GameServerStateReserved)
		gs2.Status.Counters = map[string]agonesv1.CounterStatus{
			"firstCounter": {Count: 10, Capacity: 15},
			"fullCounter":  {Count: 9223372036854775807, Capacity: 9223372036854775807},
		}
		gs3 := gsWithState(agonesv1.GameServerStateCreating)
		gs3.Status.Counters = map[string]agonesv1.CounterStatus{
			"firstCounter":  {Count: 20, Capacity: 30},
			"secondCounter": {Count: 100, Capacity: 1000},
			"fullCounter":   {Count: 9223372036854775807, Capacity: 9223372036854775807},
		}
		gs4 := gsWithState(agonesv1.GameServerStateReady)
		gs4.Status.Counters = map[string]agonesv1.CounterStatus{
			"firstCounter":  {Count: 15, Capacity: 30},
			"secondCounter": {Count: 20, Capacity: 200},
			"fullCounter":   {Count: 9223372036854775807, Capacity: 9223372036854775807},
		}
		list = append(list, gs1, gs2, gs3, gs4)

		expected := agonesv1.GameServerSetStatus{
			Replicas:          4,
			ReadyReplicas:     1,
			ReservedReplicas:  1,
			AllocatedReplicas: 1,
			Counters: map[string]agonesv1.AggregatedCounterStatus{
				"firstCounter": {
					AllocatedCount:    5,
					AllocatedCapacity: 10,
					Count:             50,
					Capacity:          85,
				},
				"secondCounter": {
					AllocatedCount:    100,
					AllocatedCapacity: 1000,
					Count:             220,
					Capacity:          2200,
				},
				"fullCounter": {
					AllocatedCount:    9223372036854775807,
					AllocatedCapacity: 9223372036854775807,
					Count:             9223372036854775807,
					Capacity:          9223372036854775807,
				},
			},
			Lists: map[string]agonesv1.AggregatedListStatus{},
		}

		assert.Equal(t, expected, computeStatus(gsSet, list))
	})

	t.Run("counters with no gameservers", func(t *testing.T) {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()

		require.NoError(t, utilruntime.ParseFeatures(fmt.Sprintf("%s=true", utilruntime.FeatureCountsAndLists)))

		gsSet := defaultFixture()
		gsSet.Spec.Template.Spec.Counters = map[string]agonesv1.CounterStatus{
			"firstCounter":  {Capacity: 10, Count: 1},
			"secondCounter": {Capacity: 10, Count: 1},
		}
		var list []*agonesv1.GameServer

		expected := agonesv1.GameServerSetStatus{
			Replicas:          0,
			ReadyReplicas:     0,
			ReservedReplicas:  0,
			AllocatedReplicas: 0,
			Lists:             map[string]agonesv1.AggregatedListStatus{},
			Counters: map[string]agonesv1.AggregatedCounterStatus{
				"firstCounter": {
					AllocatedCount:    0,
					AllocatedCapacity: 0,
					Capacity:          0,
					Count:             0,
				},
				"secondCounter": {
					AllocatedCount:    0,
					AllocatedCapacity: 0,
					Capacity:          0,
					Count:             0,
				},
			},
		}

		assert.Equal(t, expected, computeStatus(gsSet, list))
	})

	t.Run("lists", func(t *testing.T) {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()

		require.NoError(t, utilruntime.ParseFeatures(fmt.Sprintf("%s=true", utilruntime.FeatureCountsAndLists)))

		gsSet := defaultFixture()
		var list []*agonesv1.GameServer
		gs1 := gsWithState(agonesv1.GameServerStateAllocated)
		gs1.Status.Lists = map[string]agonesv1.ListStatus{
			"firstList":  {Capacity: 10, Values: []string{"a", "b"}},
			"secondList": {Capacity: 1000, Values: []string{"1", "2"}},
		}
		gs2 := gsWithState(agonesv1.GameServerStateReserved)
		gs2.Status.Lists = map[string]agonesv1.ListStatus{
			"firstList": {Capacity: 15, Values: []string{"c"}},
		}
		gs3 := gsWithState(agonesv1.GameServerStateCreating)
		gs3.Status.Lists = map[string]agonesv1.ListStatus{
			"firstList":  {Capacity: 30, Values: []string{"d"}},
			"secondList": {Capacity: 1000, Values: []string{"3"}},
		}
		gs4 := gsWithState(agonesv1.GameServerStateReady)
		gs4.Status.Lists = map[string]agonesv1.ListStatus{
			"firstList":  {Capacity: 30},
			"secondList": {Capacity: 100, Values: []string{"4", "5", "6"}},
		}
		list = append(list, gs1, gs2, gs3, gs4)

		expected := agonesv1.GameServerSetStatus{
			Replicas:          4,
			ReadyReplicas:     1,
			ReservedReplicas:  1,
			AllocatedReplicas: 1,
			Counters:          map[string]agonesv1.AggregatedCounterStatus{},
			Lists: map[string]agonesv1.AggregatedListStatus{
				"firstList": {
					AllocatedCount:    2,
					AllocatedCapacity: 10,
					Capacity:          85,
					Count:             4,
				},
				"secondList": {
					AllocatedCount:    2,
					AllocatedCapacity: 1000,
					Capacity:          2100,
					Count:             6,
				},
			},
		}

		assert.Equal(t, expected, computeStatus(gsSet, list))
	})

	t.Run("lists with no gameservers", func(t *testing.T) {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()

		require.NoError(t, utilruntime.ParseFeatures(fmt.Sprintf("%s=true", utilruntime.FeatureCountsAndLists)))

		gsSet := defaultFixture()
		gsSet.Spec.Template.Spec.Lists = map[string]agonesv1.ListStatus{
			"firstList":  {Capacity: 10, Values: []string{"a", "b"}},
			"secondList": {Capacity: 1000, Values: []string{"1", "2"}},
		}
		var list []*agonesv1.GameServer

		expected := agonesv1.GameServerSetStatus{
			Replicas:          0,
			ReadyReplicas:     0,
			ReservedReplicas:  0,
			AllocatedReplicas: 0,
			Counters:          map[string]agonesv1.AggregatedCounterStatus{},
			Lists: map[string]agonesv1.AggregatedListStatus{
				"firstList": {
					AllocatedCount:    0,
					AllocatedCapacity: 0,
					Capacity:          0,
					Count:             0,
				},
				"secondList": {
					AllocatedCount:    0,
					AllocatedCapacity: 0,
					Capacity:          0,
					Count:             0,
				},
			},
		}

		assert.Equal(t, expected, computeStatus(gsSet, list))
	})
}

// Test that the aggregated Counters and Lists are removed from the Game Server Set status if the
// FeatureCountsAndLists flag is set to false.
func TestGameServerSetDropCountsAndListsStatus(t *testing.T) {
	t.Parallel()
	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	gss := defaultFixture()
	c, m := newFakeController()

	list := createGameServers(gss, 2)
	list[0].Status.Counters = map[string]agonesv1.CounterStatus{
		"firstCounter": {Count: 5, Capacity: 10},
	}
	list[1].Status.Lists = map[string]agonesv1.ListStatus{
		"firstList": {Capacity: 100, Values: []string{"4", "5", "6"}},
	}
	gsList := []*agonesv1.GameServer{&list[0], &list[1]}

	expectedCounterStatus := map[string]agonesv1.AggregatedCounterStatus{
		"firstCounter": {
			AllocatedCount:    0,
			AllocatedCapacity: 0,
			Capacity:          10,
			Count:             5,
		},
	}
	expectedListStatus := map[string]agonesv1.AggregatedListStatus{
		"firstList": {
			AllocatedCount:    0,
			AllocatedCapacity: 0,
			Capacity:          100,
			Count:             3,
		},
	}

	flag := ""
	updated := false

	m.AgonesClient.AddReactor("update", "gameserversets",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			gsSet := ua.GetObject().(*agonesv1.GameServerSet)

			switch flag {
			case string(utilruntime.FeatureCountsAndLists) + "=true":
				assert.Equal(t, expectedCounterStatus, gsSet.Status.Counters)
				assert.Equal(t, expectedListStatus, gsSet.Status.Lists)
			case string(utilruntime.FeatureCountsAndLists) + "=false":
				assert.Nil(t, gsSet.Status.Counters)
				assert.Nil(t, gsSet.Status.Lists)
			default:
				return false, nil, errors.Errorf("Flag string(utilruntime.FeatureCountsAndLists) should be set")
			}

			return true, gsSet, nil
		})

	// Expect starting fleet to have Aggregated Counter and List Statuses
	flag = string(utilruntime.FeatureCountsAndLists) + "=true"
	require.NoError(t, utilruntime.ParseFeatures(flag))
	err := c.syncGameServerSetStatus(context.Background(), gss, gsList)
	assert.Nil(t, err)
	assert.True(t, updated)

	updated = false
	flag = string(utilruntime.FeatureCountsAndLists) + "=false"
	require.NoError(t, utilruntime.ParseFeatures(flag))
	err = c.syncGameServerSetStatus(context.Background(), gss, gsList)
	assert.Nil(t, err)
	assert.True(t, updated)

	updated = false
	flag = string(utilruntime.FeatureCountsAndLists) + "=true"
	require.NoError(t, utilruntime.ParseFeatures(flag))
	err = c.syncGameServerSetStatus(context.Background(), gss, gsList)
	assert.Nil(t, err)
	assert.True(t, updated)
}

func TestControllerWatchGameServers(t *testing.T) {
	t.Parallel()
	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	gsSet := defaultFixture()

	c, m := newFakeController()

	received := make(chan string)
	defer close(received)

	m.ExtClient.AddReactor("get", "customresourcedefinitions", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, agtesting.NewEstablishedCRD(), nil
	})
	gsSetWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameserversets", k8stesting.DefaultWatchReactor(gsSetWatch, nil))
	gsWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))

	c.workerqueue.SyncHandler = func(_ context.Context, name string) error {
		received <- name
		return nil
	}

	ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced)
	defer cancel()

	go func() {
		err := c.Run(ctx, 1)
		assert.Nil(t, err)
	}()

	f := func() string {
		select {
		case result := <-received:
			return result
		case <-time.After(3 * time.Second):
			assert.FailNow(t, "timeout occurred")
		}
		return ""
	}

	expected, err := cache.MetaNamespaceKeyFunc(gsSet)
	require.NoError(t, err)

	// gsSet add
	logrus.Info("adding gsSet")
	gsSetWatch.Add(gsSet.DeepCopy())

	assert.Equal(t, expected, f())
	// gsSet update
	logrus.Info("modify gsSet")
	gsSetCopy := gsSet.DeepCopy()
	gsSetCopy.Spec.Replicas = 5
	gsSetWatch.Modify(gsSetCopy)
	assert.Equal(t, expected, f())

	gs := gsSet.GameServer()
	gs.ObjectMeta.Name = "test-gs"
	// gs add
	logrus.Info("add gs")
	gsWatch.Add(gs.DeepCopy())
	assert.Equal(t, expected, f())

	// gs update
	gsCopy := gs.DeepCopy()
	now := metav1.Now()
	gsCopy.ObjectMeta.DeletionTimestamp = &now

	logrus.Info("modify gs - noop")
	gsWatch.Modify(gsCopy.DeepCopy())
	select {
	case <-received:
		assert.Fail(t, "Should be no value")
	case <-time.After(time.Second):
	}

	gsCopy = gs.DeepCopy()
	gsCopy.Status.State = agonesv1.GameServerStateUnhealthy
	logrus.Info("modify gs - unhealthy")
	gsWatch.Modify(gsCopy.DeepCopy())
	assert.Equal(t, expected, f())

	// gs delete
	logrus.Info("delete gs")
	gsWatch.Delete(gsCopy.DeepCopy())
	assert.Equal(t, expected, f())
}

func TestSyncGameServerSet(t *testing.T) {
	t.Parallel()

	t.Run("gameservers are not recreated when set is marked for deletion", func(t *testing.T) {
		gsSet := defaultFixture()
		gsSet.DeletionTimestamp = &metav1.Time{
			Time: time.Now(),
		}
		list := createGameServers(gsSet, 5)

		// mark some as shutdown
		list[0].Status.State = agonesv1.GameServerStateShutdown
		list[1].Status.State = agonesv1.GameServerStateShutdown

		c, m := newFakeController()
		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})
		m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerList{Items: list}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "gameserver should not update")
			return false, nil, nil
		})
		m.AgonesClient.AddReactor("create", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			assert.FailNow(t, "new gameservers should not be created")

			return false, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.gameServerSetSynced, c.gameServerSynced)
		defer cancel()

		c.syncGameServerSet(ctx, gsSet.ObjectMeta.Namespace+"/"+gsSet.ObjectMeta.Name) // nolint: errcheck
	})

	t.Run("adding and deleting unhealthy gameservers", func(t *testing.T) {
		gsSet := defaultFixture()
		list := createGameServers(gsSet, 5)

		// make some as unhealthy
		list[0].Status.State = agonesv1.GameServerStateUnhealthy

		updated := false
		count := 0

		c, m := newFakeController()
		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})
		m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerList{Items: list}, nil
		})

		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, gs.Status.State, agonesv1.GameServerStateShutdown)

			updated = true
			assert.Equal(t, "test-0", gs.GetName())
			return true, nil, nil
		})
		m.AgonesClient.AddReactor("create", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.CreateAction)
			gs := ca.GetObject().(*agonesv1.GameServer)

			assert.True(t, metav1.IsControlledBy(gs, gsSet))
			count++
			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.gameServerSetSynced, c.gameServerSynced)
		defer cancel()

		c.syncGameServerSet(ctx, gsSet.ObjectMeta.Namespace+"/"+gsSet.ObjectMeta.Name) // nolint: errcheck

		assert.Equal(t, 6, count)
		assert.True(t, updated, "A game servers should have been updated")
	})

	t.Run("adding and deleting errored gameservers", func(t *testing.T) {
		gsSet := defaultFixture()
		list := createGameServers(gsSet, 5)

		// make some as unhealthy
		list[0].Annotations = map[string]string{agonesv1.GameServerErroredAtAnnotation: time.Now().Add(-30 * time.Second).UTC().Format(time.RFC3339)}
		list[0].Status.State = agonesv1.GameServerStateError

		updated := false
		count := 0

		c, m := newFakeController()
		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})
		m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerList{Items: list}, nil
		})

		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, gs.Status.State, agonesv1.GameServerStateShutdown)

			updated = true
			assert.Equal(t, "test-0", gs.GetName())
			return true, nil, nil
		})
		m.AgonesClient.AddReactor("create", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.CreateAction)
			gs := ca.GetObject().(*agonesv1.GameServer)

			assert.True(t, metav1.IsControlledBy(gs, gsSet))
			count++
			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.gameServerSetSynced, c.gameServerSynced)
		defer cancel()

		c.syncGameServerSet(ctx, gsSet.ObjectMeta.Namespace+"/"+gsSet.ObjectMeta.Name) // nolint: errcheck

		assert.Equal(t, 6, count)
		assert.True(t, updated, "A game servers should have been updated")
	})

	t.Run("adding and delay deleting errored gameservers", func(t *testing.T) {
		gsSet := defaultFixture()
		list := createGameServers(gsSet, 5)

		// make some as unhealthy
		list[0].Annotations = map[string]string{agonesv1.GameServerErroredAtAnnotation: time.Now().UTC().Format(time.RFC3339)}
		list[0].Status.State = agonesv1.GameServerStateError

		updated := false
		count := 0

		c, m := newFakeController()
		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})
		m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerList{Items: list}, nil
		})

		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, gs.Status.State, agonesv1.GameServerStateShutdown)

			updated = true
			assert.Equal(t, "test-0", gs.GetName())
			return true, nil, nil
		})
		m.AgonesClient.AddReactor("create", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.CreateAction)
			gs := ca.GetObject().(*agonesv1.GameServer)

			assert.True(t, metav1.IsControlledBy(gs, gsSet))
			count++
			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.gameServerSetSynced, c.gameServerSynced)
		defer cancel()

		c.syncGameServerSet(ctx, gsSet.ObjectMeta.Namespace+"/"+gsSet.ObjectMeta.Name) // nolint: errcheck

		assert.Equal(t, 5, count)
		assert.False(t, updated, "A game servers should not have been updated")
	})

	t.Run("removing gamservers", func(t *testing.T) {
		gsSet := defaultFixture()
		list := createGameServers(gsSet, 15)
		count := 0

		c, m := newFakeController()
		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})
		m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerList{Items: list}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			count++
			return true, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.gameServerSetSynced, c.gameServerSynced)
		defer cancel()

		c.syncGameServerSet(ctx, gsSet.ObjectMeta.Namespace+"/"+gsSet.ObjectMeta.Name) // nolint: errcheck

		assert.Equal(t, 5, count)
	})

	t.Run("Starting GameServers get deleted first", func(t *testing.T) {
		gsSet := defaultFixture()
		list := createGameServers(gsSet, 12)

		list[0].Status.State = agonesv1.GameServerStateStarting
		list[1].Status.State = agonesv1.GameServerStateCreating

		rand.Shuffle(len(list), func(i, j int) {
			list[i], list[j] = list[j], list[i]
		})

		var deleted []string

		c, m := newFakeController()
		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})
		m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerList{Items: list}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			require.Equal(t, gs.Status.State, agonesv1.GameServerStateShutdown)

			deleted = append(deleted, gs.ObjectMeta.Name)
			return true, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.gameServerSetSynced, c.gameServerSynced)
		defer cancel()
		require.NoError(t, c.syncGameServerSet(ctx, gsSet.ObjectMeta.Namespace+"/"+gsSet.ObjectMeta.Name))

		require.Len(t, deleted, 2)
		require.ElementsMatchf(t, []string{"test-0", "test-1"}, deleted, "should be the non-ready GameServers")
	})
}

func TestControllerSyncUnhealthyGameServers(t *testing.T) {
	t.Parallel()

	gsSet := defaultFixture()

	gs1 := gsSet.GameServer()
	gs1.ObjectMeta.Name = "test-1"
	gs1.Status = agonesv1.GameServerStatus{State: agonesv1.GameServerStateUnhealthy}

	gs2 := gsSet.GameServer()
	gs2.ObjectMeta.Name = "test-2"
	gs2.Status = agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}

	gs3 := gsSet.GameServer()
	gs3.ObjectMeta.Name = "test-3"
	now := metav1.Now()
	gs3.ObjectMeta.DeletionTimestamp = &now
	gs3.Status = agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}

	t.Run("valid case", func(t *testing.T) {
		var updatedCount int
		c, m := newFakeController()
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)

			assert.Equal(t, gs.Status.State, agonesv1.GameServerStateShutdown)

			updatedCount++
			return true, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m)
		defer cancel()

		err := c.deleteGameServers(ctx, gsSet, []*agonesv1.GameServer{gs1, gs2, gs3})
		assert.Nil(t, err)

		assert.Equal(t, 3, updatedCount, "Updates should have occurred")
	})

	t.Run("error on update step", func(t *testing.T) {
		c, m := newFakeController()
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)

			assert.Equal(t, gs.Status.State, agonesv1.GameServerStateShutdown)

			return true, nil, errors.New("update-err")
		})

		ctx, cancel := agtesting.StartInformers(m)
		defer cancel()

		err := c.deleteGameServers(ctx, gsSet, []*agonesv1.GameServer{gs1, gs2, gs3})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error updating gameserver")
	})
}

func TestSyncMoreGameServers(t *testing.T) {
	t.Parallel()
	gsSet := defaultFixture()

	t.Run("valid case", func(t *testing.T) {

		c, m := newFakeController()
		expected := 10
		count := 0

		m.AgonesClient.AddReactor("create", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.CreateAction)
			gs := ca.GetObject().(*agonesv1.GameServer)

			assert.True(t, metav1.IsControlledBy(gs, gsSet))
			count++

			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(m)
		defer cancel()

		err := c.addMoreGameServers(ctx, gsSet, expected)
		assert.Nil(t, err)
		assert.Equal(t, expected, count)
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "SuccessfulCreate")
	})

	t.Run("error on create step", func(t *testing.T) {
		gsSet := defaultFixture()
		c, m := newFakeController()
		expected := 10
		m.AgonesClient.AddReactor("create", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.CreateAction)
			gs := ca.GetObject().(*agonesv1.GameServer)

			assert.True(t, metav1.IsControlledBy(gs, gsSet))

			return true, gs, errors.New("create-err")
		})

		ctx, cancel := agtesting.StartInformers(m)
		defer cancel()

		err := c.addMoreGameServers(ctx, gsSet, expected)
		require.Error(t, err)
		assert.Equal(t, "error creating gameserver for gameserverset test: create-err", err.Error())
	})
}

func TestControllerSyncGameServerSetStatus(t *testing.T) {
	t.Parallel()

	t.Run("all ready list", func(t *testing.T) {
		gsSet := defaultFixture()
		c, m := newFakeController()

		updated := false
		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			gsSet := ua.GetObject().(*agonesv1.GameServerSet)

			assert.Equal(t, int32(1), gsSet.Status.Replicas)
			assert.Equal(t, int32(1), gsSet.Status.ReadyReplicas)
			assert.Equal(t, int32(0), gsSet.Status.AllocatedReplicas)

			return true, nil, nil
		})

		list := []*agonesv1.GameServer{{Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}}}
		err := c.syncGameServerSetStatus(context.Background(), gsSet, list)
		assert.Nil(t, err)
		assert.True(t, updated)
	})

	t.Run("only some ready list", func(t *testing.T) {
		gsSet := defaultFixture()
		c, m := newFakeController()

		updated := false
		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			gsSet := ua.GetObject().(*agonesv1.GameServerSet)

			assert.Equal(t, int32(8), gsSet.Status.Replicas)
			assert.Equal(t, int32(1), gsSet.Status.ReadyReplicas)
			assert.Equal(t, int32(2), gsSet.Status.AllocatedReplicas)

			return true, nil, nil
		})

		list := []*agonesv1.GameServer{
			{Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}},
			{Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateStarting}},
			{Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateUnhealthy}},
			{Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStatePortAllocation}},
			{Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateError}},
			{Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateCreating}},
			{Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateAllocated}},
			{Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateAllocated}},
		}
		err := c.syncGameServerSetStatus(context.Background(), gsSet, list)
		assert.Nil(t, err)
		assert.True(t, updated)
	})
}

func TestControllerUpdateValidationHandler(t *testing.T) {
	t.Parallel()

	ext := newFakeExtensions()
	gvk := metav1.GroupVersionKind(agonesv1.SchemeGroupVersion.WithKind("GameServerSet"))
	fixture := &agonesv1.GameServerSet{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: agonesv1.GameServerSetSpec{Replicas: 5},
	}
	raw, err := json.Marshal(fixture)
	require.NoError(t, err)

	t.Run("valid gameserverset update", func(t *testing.T) {
		newGSS := fixture.DeepCopy()
		newGSS.Spec.Replicas = 10
		newRaw, err := json.Marshal(newGSS)
		require.NoError(t, err)

		review := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Kind:      gvk,
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw: newRaw,
				},
				OldObject: runtime.RawExtension{
					Raw: raw,
				},
			},
			Response: &admissionv1.AdmissionResponse{Allowed: true},
		}

		result, err := ext.updateValidationHandler(review)
		require.NoError(t, err)
		if !assert.True(t, result.Response.Allowed) {
			// show the reason of the failure
			require.NotNil(t, result.Response.Result)
			require.NotNil(t, result.Response.Result.Details)
			require.NotEmpty(t, result.Response.Result.Details.Causes)
		}
	})

	t.Run("new object is nil, err excpected", func(t *testing.T) {
		review := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Kind:      gvk,
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw: nil,
				},
				OldObject: runtime.RawExtension{
					Raw: raw,
				},
			},
			Response: &admissionv1.AdmissionResponse{Allowed: true},
		}

		_, err := ext.updateValidationHandler(review)
		require.Error(t, err)
		assert.Equal(t, "error unmarshalling new GameServerSet json: : unexpected end of JSON input", err.Error())
	})

	t.Run("old object is nil, err excpected", func(t *testing.T) {
		newGSS := fixture.DeepCopy()
		newGSS.Spec.Replicas = 10
		newRaw, err := json.Marshal(newGSS)
		require.NoError(t, err)

		review := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Kind:      gvk,
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw: newRaw,
				},
				OldObject: runtime.RawExtension{
					Raw: nil,
				},
			},
			Response: &admissionv1.AdmissionResponse{Allowed: true},
		}

		_, err = ext.updateValidationHandler(review)
		require.Error(t, err)
		assert.Equal(t, "error unmarshalling old GameServerSet json: : unexpected end of JSON input", err.Error())
	})

	t.Run("invalid gameserverset update", func(t *testing.T) {
		newGSS := fixture.DeepCopy()
		newGSS.Spec.Template = agonesv1.GameServerTemplateSpec{
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Static}},
			},
		}
		newRaw, err := json.Marshal(newGSS)
		require.NoError(t, err)

		assert.NotEqual(t, string(raw), string(newRaw))

		review := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Kind:      gvk,
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw: newRaw,
				},
				OldObject: runtime.RawExtension{
					Raw: raw,
				},
			},
			Response: &admissionv1.AdmissionResponse{Allowed: true},
		}

		result, err := ext.updateValidationHandler(review)
		require.NoError(t, err)
		require.NotNil(t, result.Response)
		require.NotNil(t, result.Response.Result)
		require.NotNil(t, result.Response.Result.Details)
		assert.False(t, result.Response.Allowed)
		assert.NotEmpty(t, result.Response.Result.Details.Causes)
		assert.Equal(t, metav1.StatusFailure, result.Response.Result.Status)
		assert.Equal(t, metav1.StatusReasonInvalid, result.Response.Result.Reason)
		assert.Contains(t, result.Response.Result.Message, "GameServerSet.agones.dev \"\" is invalid")
	})
}

func TestCreationValidationHandler(t *testing.T) {
	t.Parallel()

	ext := newFakeExtensions()

	gvk := metav1.GroupVersionKind(agonesv1.SchemeGroupVersion.WithKind("GameServerSet"))
	fixture := &agonesv1.GameServerSet{ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "default"},
		Spec: agonesv1.GameServerSetSpec{
			Replicas: 5,
			Template: agonesv1.GameServerTemplateSpec{
				Spec: agonesv1.GameServerSpec{Container: "test",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "c1"}},
						},
					},
				},
			},
		},
	}
	raw, err := json.Marshal(fixture)
	require.NoError(t, err)

	t.Run("valid gameserverset create", func(t *testing.T) {
		newGSS := fixture.DeepCopy()
		newGSS.Spec.Replicas = 10
		newRaw, err := json.Marshal(newGSS)
		require.NoError(t, err)

		review := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Kind:      gvk,
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw: newRaw,
				},
			},
			Response: &admissionv1.AdmissionResponse{Allowed: true},
		}

		result, err := ext.creationValidationHandler(review)
		require.NoError(t, err)
		if !assert.True(t, result.Response.Allowed) {
			// show the reason of the failure
			require.NotNil(t, result.Response.Result)
			require.NotNil(t, result.Response.Result.Details)
			require.NotEmpty(t, result.Response.Result.Details.Causes)
		}
	})

	t.Run("object is nil, err excpected", func(t *testing.T) {
		review := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Kind:      gvk,
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw: nil,
				},
			},
			Response: &admissionv1.AdmissionResponse{Allowed: true},
		}

		_, err := ext.creationValidationHandler(review)
		require.Error(t, err)
		assert.Equal(t, "error unmarshalling GameServerSet json after schema validation: : unexpected end of JSON input", err.Error())
	})

	t.Run("invalid gameserverset create", func(t *testing.T) {
		newGSS := fixture.DeepCopy()
		newGSS.Spec.Template = agonesv1.GameServerTemplateSpec{
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{{PortPolicy: agonesv1.Static}},
			},
		}
		newRaw, err := json.Marshal(newGSS)
		require.NoError(t, err)

		assert.NotEqual(t, string(raw), string(newRaw))

		review := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Kind:      gvk,
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw: newRaw,
				},
			},
			Response: &admissionv1.AdmissionResponse{Allowed: true},
		}

		result, err := ext.creationValidationHandler(review)
		require.NoError(t, err)
		require.NotNil(t, result.Response)
		require.NotNil(t, result.Response.Result)
		require.NotNil(t, result.Response.Result.Details)
		assert.False(t, result.Response.Allowed)
		assert.NotEmpty(t, result.Response.Result.Details.Causes)
		assert.Equal(t, metav1.StatusFailure, result.Response.Result.Status)
		assert.Equal(t, metav1.StatusReasonInvalid, result.Response.Result.Reason)
		assert.Contains(t, result.Response.Result.Message, "GameServerSet.agones.dev \"\" is invalid")
	})
}

// defaultFixture creates the default GameServerSet fixture
func defaultFixture() *agonesv1.GameServerSet {
	gsSet := &agonesv1.GameServerSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", UID: "1234"},
		Spec: agonesv1.GameServerSetSpec{
			Replicas:   10,
			Scheduling: apis.Packed,
			Template:   agonesv1.GameServerTemplateSpec{},
		},
	}
	return gsSet
}

// createGameServers create an array of GameServers from the GameServerSet
func createGameServers(gsSet *agonesv1.GameServerSet, size int) []agonesv1.GameServer {
	var list []agonesv1.GameServer
	for i := 0; i < size; i++ {
		gs := gsSet.GameServer()
		gs.Name = gs.GenerateName + strconv.Itoa(i)
		gs.Status = agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}
		list = append(list, *gs)
	}
	return list
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() (*Controller, agtesting.Mocks) {
	m := agtesting.NewMocks()
	counter := gameservers.NewPerNodeCounter(m.KubeInformerFactory, m.AgonesInformerFactory)
	c := NewController(healthcheck.NewHandler(), counter, m.KubeClient, m.ExtClient, m.AgonesClient, m.AgonesInformerFactory, 16, 64, 64, 64, 5000)
	c.recorder = m.FakeRecorder
	return c, m
}

// newFakeExtensions returns an extensions struct
func newFakeExtensions() *Extensions {
	return NewExtensions(generic.New(), webhooks.NewWebHook(http.NewServeMux()))
}
