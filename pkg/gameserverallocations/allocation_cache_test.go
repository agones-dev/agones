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

package gameserverallocations

import (
	"context"
	"fmt"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/gameservers"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/heptiolabs/healthcheck"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
)

func TestAllocationCacheListSortedGameServers(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	gs1 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, UID: "1"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}}
	gs2 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, UID: "2"},
		Status: agonesv1.GameServerStatus{NodeName: "node2", State: agonesv1.GameServerStateReady}}
	gs3 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, UID: "3"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateAllocated}}
	gs4 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, UID: "4"},
		Status: agonesv1.GameServerStatus{NodeName: "node2", State: agonesv1.GameServerStateReady}}
	gs5 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, UID: "5"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Players: &agonesv1.PlayerStatus{
				Count:    0,
				Capacity: 10,
			},
			Counters: map[string]agonesv1.CounterStatus{
				"players": {
					Count:    4,
					Capacity: 40, // Available Capacity == 36
				},
			}},
	}
	gs6 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: defaultNs, UID: "6"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady, Players: &agonesv1.PlayerStatus{
			Count:    2,
			Capacity: 10,
		},
			Counters: map[string]agonesv1.CounterStatus{
				"players": {
					Count:    14,
					Capacity: 40, // Available Capacity == 26
				},
			}},
	}

	fixtures := map[string]struct {
		list     []agonesv1.GameServer
		test     func(*testing.T, []*agonesv1.GameServer)
		features string
		gsa      *allocationv1.GameServerAllocation
	}{
		"allocated first (StateAllocationFilter)": {
			list: []agonesv1.GameServer{gs1, gs2, gs3},
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				assert.Equal(t, []*agonesv1.GameServer{&gs3, &gs1, &gs2}, list)
			},
		},
		"nil player status (PlayerAllocationFilter)": {
			list:     []agonesv1.GameServer{gs1, gs2, gs4},
			features: fmt.Sprintf("%s=true", runtime.FeaturePlayerAllocationFilter),
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				require.Len(t, list, 3)
				// first two items can come in any order
				assert.ElementsMatchf(t, []*agonesv1.GameServer{&gs2, &gs4}, list[:2], "GameServer Names")
				assert.Equal(t, &gs1, list[2])
			},
		},
		"least player capacity (PlayerAllocationFilter)": {
			list:     []agonesv1.GameServer{gs5, gs6},
			features: fmt.Sprintf("%s=true", runtime.FeaturePlayerAllocationFilter),
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				assert.Equal(t, []*agonesv1.GameServer{&gs6, &gs5}, list)
			},
		},
		"list ready": {
			// node1: 1 ready, node2: 2 ready
			list: []agonesv1.GameServer{gs1, gs2, gs4},
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				assert.Len(t, list, 3)
				// first two items can come in any order
				assert.ElementsMatchf(t, []*agonesv1.GameServer{&gs2, &gs4}, list[:2], "GameServer Names")
				assert.Equal(t, &gs1, list[2])
			},
		},
		"lexicographical (node name)": {
			list: []agonesv1.GameServer{gs2, gs1},
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				assert.Len(t, list, 2)
				if !assert.Equal(t, []*agonesv1.GameServer{&gs1, &gs2}, list) {
					for _, gs := range list {
						logrus.WithField("name", gs.Name).Info("game server")
					}
				}
			},
		},
		"counters Descending": {
			list:     []agonesv1.GameServer{gs1, gs2, gs3, gs4, gs5, gs6},
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			gsa: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Priorities: []agonesv1.Priority{
						{
							Type:  "Counter",
							Key:   "players",
							Order: "Descending",
						},
					},
				},
			},
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				assert.Len(t, list, 6)
				if !assert.Equal(t, []*agonesv1.GameServer{&gs3, &gs5, &gs6, &gs1, &gs2, &gs4}, list) {
					for _, gs := range list {
						logrus.WithField("game", gs.Name).Info("game server")
					}
				}
			},
		},
		"counters Ascending": {
			list:     []agonesv1.GameServer{gs1, gs2, gs3, gs4, gs5, gs6},
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			gsa: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Priorities: []agonesv1.Priority{
						{
							Type:  "Counter",
							Key:   "players",
							Order: "Ascending",
						},
					},
				},
			},
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				assert.Len(t, list, 6)
				if !assert.Equal(t, []*agonesv1.GameServer{&gs3, &gs6, &gs5, &gs1, &gs2, &gs4}, list) {
					for _, gs := range list {
						logrus.WithField("game", gs.Name).Info("game server")
					}
				}
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			// deliberately not resetting the Feature state, to catch any possible unknown regressions with the
			// new feature flags
			if v.features != "" {
				require.NoError(t, runtime.ParseFeatures(v.features))
			}

			cache, m := newFakeAllocationCache()

			gsList := v.list

			m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
				return true, &agonesv1.GameServerList{Items: gsList}, nil
			})

			ctx, cancel := agtesting.StartInformers(m, cache.gameServerSynced)
			defer cancel()

			// This call initializes the cache
			err := cache.syncCache()
			assert.Nil(t, err)

			err = cache.counter.Run(ctx, 0)
			assert.Nil(t, err)

			list := cache.ListSortedGameServers(v.gsa)

			v.test(t, list)
		})
	}
}

func TestListSortedGameServersPriorities(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	gs1 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, UID: "1"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Lists: map[string]agonesv1.ListStatus{
				"players": {
					Values:   []string{"player1"},
					Capacity: 100, // Available Capacity == 99
				},
				"layers": {
					Values:   []string{"layer1", "layer2", "layer3"},
					Capacity: 100, // Available Capacity == 97
				}}}}
	gs2 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, UID: "2"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Lists: map[string]agonesv1.ListStatus{
				"players": {
					Values:   []string{},
					Capacity: 100, // Available Capacity == 100
				},
			},
			Counters: map[string]agonesv1.CounterStatus{
				"assets": {
					Count:    101,
					Capacity: 1000, // Available Capacity = 899
				},
			}}}
	gs3 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, UID: "3"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Lists: map[string]agonesv1.ListStatus{
				"players": {
					Values:   []string{"player2", "player3"},
					Capacity: 100, // Available Capacity == 98
				}},
			Counters: map[string]agonesv1.CounterStatus{
				"sessions": {
					Count:    9,
					Capacity: 1000, // Available Capacity == 991
				},
				"assets": {
					Count:    100,
					Capacity: 1000, // Available Capacity == 900
				},
			}}}
	gs4 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, UID: "4"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Counters: map[string]agonesv1.CounterStatus{
				"sessions": {
					Count:    99,
					Capacity: 1000, // Available Capacity == 901
				},
			},
			Lists: map[string]agonesv1.ListStatus{
				"players": {
					Values:   []string{"player4"},
					Capacity: 100, // Available Capacity == 99
				},
				"layers": {
					Values:   []string{"layer4, layer5"},
					Capacity: 100, // Available Capacity == 98
				}}}}
	gs5 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, UID: "5"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Counters: map[string]agonesv1.CounterStatus{
				"sessions": {
					Count:    9,
					Capacity: 1000, // Available Capacity == 991
				},
				"assets": {
					Count:    99,
					Capacity: 1000, // Available Capacity == 901
				},
			},
			Lists: map[string]agonesv1.ListStatus{
				"layers": {
					Values:   []string{},
					Capacity: 100, // Available Capacity == 100
				}}}}
	gs6 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: defaultNs, UID: "6"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Counters: map[string]agonesv1.CounterStatus{
				"sessions": {
					Count:    999,
					Capacity: 1000, // Available Capacity == 1
				},
			}}}

	testScenarios := map[string]struct {
		list []agonesv1.GameServer
		gsa  *allocationv1.GameServerAllocation
		want []*agonesv1.GameServer
	}{
		"Sort by one Priority Counter Ascending": {
			list: []agonesv1.GameServer{gs4, gs5, gs6},
			gsa: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Priorities: []agonesv1.Priority{
						{
							Type:  "Counter",
							Key:   "sessions",
							Order: "Ascending",
						},
					},
				},
			},
			want: []*agonesv1.GameServer{&gs6, &gs4, &gs5},
		},
		"Sort by one Priority Counter Descending": {
			list: []agonesv1.GameServer{gs4, gs5, gs6},
			gsa: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Priorities: []agonesv1.Priority{
						{
							Type:  "Counter",
							Key:   "sessions",
							Order: "Descending",
						},
					},
				},
			},
			want: []*agonesv1.GameServer{&gs5, &gs4, &gs6},
		},
		"Sort by two Priority Counter Ascending and Ascending": {
			list: []agonesv1.GameServer{gs3, gs5, gs6, gs4, gs1, gs2},
			gsa: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Priorities: []agonesv1.Priority{
						{
							Type:  "Counter",
							Key:   "sessions",
							Order: "Ascending",
						},
						{
							Type:  "Counter",
							Key:   "assets",
							Order: "Ascending",
						},
					},
				},
			},
			want: []*agonesv1.GameServer{&gs6, &gs4, &gs3, &gs5, &gs2, &gs1},
		},
		"Sort by two Priority Counter Ascending and Descending": {
			list: []agonesv1.GameServer{gs3, gs5, gs6, gs4, gs1, gs2},
			gsa: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Priorities: []agonesv1.Priority{
						{
							Type:  "Counter",
							Key:   "sessions",
							Order: "Ascending",
						},
						{
							Type:  "Counter",
							Key:   "assets",
							Order: "Descending",
						},
					},
				},
			},
			want: []*agonesv1.GameServer{&gs6, &gs4, &gs5, &gs3, &gs2, &gs1},
		},
		"Sort by one Priority Counter game server without Counter": {
			list: []agonesv1.GameServer{gs1, gs5, gs6, gs4},
			gsa: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Priorities: []agonesv1.Priority{
						{
							Type:  "Counter",
							Key:   "sessions",
							Order: "Ascending",
						},
					},
				},
			},
			want: []*agonesv1.GameServer{&gs6, &gs4, &gs5, &gs1},
		},
		"Sort by one Priority List Ascending": {
			list: []agonesv1.GameServer{gs3, gs2, gs1},
			gsa: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Priorities: []agonesv1.Priority{
						{
							Type:  "List",
							Key:   "players",
							Order: "Ascending",
						},
					},
				},
			},
			want: []*agonesv1.GameServer{&gs3, &gs1, &gs2},
		},
		"Sort by one Priority List Descending": {
			list: []agonesv1.GameServer{gs3, gs2, gs1},
			gsa: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Priorities: []agonesv1.Priority{
						{
							Type:  "List",
							Key:   "players",
							Order: "Descending",
						},
					},
				},
			},
			want: []*agonesv1.GameServer{&gs2, &gs1, &gs3},
		},
		"Sort by two Priority List Descending and Ascending": {
			list: []agonesv1.GameServer{gs1, gs2, gs3, gs4},
			gsa: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Priorities: []agonesv1.Priority{
						{
							Type:  "List",
							Key:   "players",
							Order: "Descending",
						},
						{
							Type:  "List",
							Key:   "layers",
							Order: "Ascending",
						},
					},
				},
			},
			want: []*agonesv1.GameServer{&gs2, &gs1, &gs4, &gs3},
		},
		"Sort by two Priority List Descending and Descending": {
			list: []agonesv1.GameServer{gs6, gs5, gs4, gs3, gs2, gs1},
			gsa: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Priorities: []agonesv1.Priority{
						{
							Type:  "List",
							Key:   "players",
							Order: "Descending",
						},
						{
							Type:  "List",
							Key:   "layers",
							Order: "Descending",
						},
					},
				},
			},
			want: []*agonesv1.GameServer{&gs2, &gs4, &gs1, &gs3, &gs5, &gs6},
		},
		"Sort lexigraphically as no game server has the priority": {
			list: []agonesv1.GameServer{gs6, gs5, gs4, gs3, gs2, gs1},
			gsa: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Priorities: []agonesv1.Priority{
						{
							Type:  "Counter",
							Key:   "sayers",
							Order: "Ascending",
						},
					},
				},
			},
			want: []*agonesv1.GameServer{&gs1, &gs2, &gs3, &gs4, &gs5, &gs6},
		},
	}

	for testName, testScenario := range testScenarios {
		t.Run(testName, func(t *testing.T) {

			cache, m := newFakeAllocationCache()

			m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
				return true, &agonesv1.GameServerList{Items: testScenario.list}, nil
			})

			ctx, cancel := agtesting.StartInformers(m, cache.gameServerSynced)
			defer cancel()

			// This call initializes the cache
			err := cache.syncCache()
			assert.Nil(t, err)

			err = cache.counter.Run(ctx, 0)
			assert.Nil(t, err)

			got := cache.ListSortedGameServersPriorities(testScenario.gsa)

			assert.Equal(t, testScenario.want, got)
		})
	}
}

func TestAllocatorRunCacheSync(t *testing.T) {
	t.Parallel()
	cache, m := newFakeAllocationCache()
	gsWatch := watch.NewFake()

	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))

	ctx, cancel := agtesting.StartInformers(m, cache.gameServerSynced)
	defer cancel()

	assertCacheEntries := func(expected int) {
		count := 0
		err := wait.PollUntilContextTimeout(context.Background(), time.Second, 5*time.Second, true, func(_ context.Context) (done bool, err error) {
			count = 0
			cache.cache.Range(func(_ string, _ *agonesv1.GameServer) bool {
				count++
				return true
			})

			return count == expected, nil
		})

		assert.NoError(t, err, fmt.Sprintf("Should be %d values", expected))
	}

	go func() {
		err := cache.Run(ctx)
		assert.Nil(t, err)
	}()

	gs := agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: "default", ResourceVersion: "1"},
		Status:     agonesv1.GameServerStatus{State: agonesv1.GameServerStateStarting},
	}

	logrus.Info("adding ready game server")
	gsWatch.Add(gs.DeepCopy())

	assertCacheEntries(0)

	gs.Status.State = agonesv1.GameServerStateReady
	gs.ObjectMeta.ResourceVersion = "2"
	gsWatch.Modify(gs.DeepCopy())

	assertCacheEntries(1)

	// try again, should be no change
	gs.Status.State = agonesv1.GameServerStateReady
	gs.ObjectMeta.ResourceVersion = "3"
	gsWatch.Modify(gs.DeepCopy())

	assertCacheEntries(1)

	// now move it to Shutdown
	gs.Status.State = agonesv1.GameServerStateShutdown
	gs.ObjectMeta.ResourceVersion = "4"
	gsWatch.Modify(gs.DeepCopy())
	assertCacheEntries(0)

	// add it back in as Allocated
	gs.Status.State = agonesv1.GameServerStateAllocated
	gs.ObjectMeta.ResourceVersion = "5"
	gsWatch.Modify(gs.DeepCopy())
	assertCacheEntries(1)

	// now move it to Shutdown
	gs.Status.State = agonesv1.GameServerStateShutdown
	gs.ObjectMeta.ResourceVersion = "6"
	gsWatch.Modify(gs.DeepCopy())
	assertCacheEntries(0)

	// do not add back in with stale resource version
	gs.Status.State = agonesv1.GameServerStateAllocated
	gs.ObjectMeta.ResourceVersion = "6"
	gsWatch.Modify(gs.DeepCopy())
	assertCacheEntries(0)

	// add back in ready gameserver
	gs.Status.State = agonesv1.GameServerStateReady
	gs.ObjectMeta.ResourceVersion = "7"
	gsWatch.Modify(gs.DeepCopy())
	assertCacheEntries(1)

	// update with deletion timestamp
	n := metav1.Now()
	deletedCopy := gs.DeepCopy()
	deletedCopy.ObjectMeta.DeletionTimestamp = &n
	deletedCopy.ObjectMeta.ResourceVersion = "8"
	gsWatch.Modify(deletedCopy.DeepCopy())
	assertCacheEntries(0)

	// add back in ready gameserver
	gs.Status.State = agonesv1.GameServerStateReady
	deletedCopy.ObjectMeta.ResourceVersion = "9"
	gsWatch.Modify(gs.DeepCopy())
	assertCacheEntries(1)

	// now actually delete it
	gsWatch.Delete(gs.DeepCopy())
	assertCacheEntries(0)
}

func TestAllocationCacheReorderGameServerAfterAllocation(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	gs0Allocated := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs0", Namespace: defaultNs, UID: "0"},
		Status: agonesv1.GameServerStatus{NodeName: "node0", State: agonesv1.GameServerStateAllocated}}
	gs1 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, UID: "1"},
		Status: agonesv1.GameServerStatus{NodeName: "node0", State: agonesv1.GameServerStateReady}}
	gs1Allocated := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, UID: "1"},
		Status: agonesv1.GameServerStatus{NodeName: "node0", State: agonesv1.GameServerStateAllocated}}
	gs2 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, UID: "2"},
		Status: agonesv1.GameServerStatus{NodeName: "node0", State: agonesv1.GameServerStateReady}}
	gs2Allocated := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, UID: "2"},
		Status: agonesv1.GameServerStatus{NodeName: "node0", State: agonesv1.GameServerStateAllocated}}
	gs3 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, UID: "3"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}}
	gs4 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, UID: "4"},
		Status: agonesv1.GameServerStatus{
			NodeName: "node1",
			State:    agonesv1.GameServerStateAllocated,
			Players: &agonesv1.PlayerStatus{
				Count:    3,
				Capacity: 10,
			},
			Counters: map[string]agonesv1.CounterStatus{
				"players": {
					Count:    3,
					Capacity: 10,
				},
			},
			Lists: map[string]agonesv1.ListStatus{
				"players": {
					Values:   []string{"player0", "player1", "player2"},
					Capacity: 10,
				},
			},
		},
	}
	gs5 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, UID: "5"},
		Status: agonesv1.GameServerStatus{
			NodeName: "node1",
			State:    agonesv1.GameServerStateAllocated,
			Players: &agonesv1.PlayerStatus{
				Count:    2,
				Capacity: 10,
			},
			Counters: map[string]agonesv1.CounterStatus{
				"players": {
					Count:    2,
					Capacity: 10,
				},
			},
			Lists: map[string]agonesv1.ListStatus{
				"players": {
					Values:   []string{"player0", "player1"},
					Capacity: 10,
				},
			},
		},
	}
	gs5Allocated := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, UID: "5"},
		Status: agonesv1.GameServerStatus{
			NodeName: "node1",
			State:    agonesv1.GameServerStateAllocated,
			Players: &agonesv1.PlayerStatus{
				Count:    5,
				Capacity: 10,
			},
			Counters: map[string]agonesv1.CounterStatus{
				"players": {
					Count:    5,
					Capacity: 10,
				},
			},
			Lists: map[string]agonesv1.ListStatus{
				"players": {
					Values:   []string{"player0", "player1", "player2", "player3", "player4"},
					Capacity: 10,
				},
			},
		},
	}
	gs6 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: defaultNs, UID: "6"},
		Status: agonesv1.GameServerStatus{
			NodeName: "node1",
			State:    agonesv1.GameServerStateAllocated,
			Players: &agonesv1.PlayerStatus{
				Count:    1,
				Capacity: 10,
			},
			Counters: map[string]agonesv1.CounterStatus{
				"players": {
					Count:    1,
					Capacity: 10,
				},
			},
			Lists: map[string]agonesv1.ListStatus{
				"players": {
					Values:   []string{"player0"},
					Capacity: 10,
				},
			},
		},
	}

	fixtures := map[string]struct {
		features         string
		list             []*agonesv1.GameServer
		priorities       []agonesv1.Priority
		packingStrategy  apis.SchedulingStrategy
		gsToReorder      *agonesv1.GameServer
		gsToReorderIndex int
		want             []*agonesv1.GameServer
	}{
		"pakced (no change)": {
			list:             []*agonesv1.GameServer{&gs0Allocated, &gs1, &gs2, &gs3},
			gsToReorder:      &gs1Allocated,
			gsToReorderIndex: 1,
			want:             []*agonesv1.GameServer{&gs0Allocated, &gs1Allocated, &gs2, &gs3},
		},
		"packed": {
			list:             []*agonesv1.GameServer{&gs0Allocated, &gs1, &gs2, &gs3},
			gsToReorder:      &gs2Allocated,
			gsToReorderIndex: 2,
			want:             []*agonesv1.GameServer{&gs0Allocated, &gs2Allocated, &gs1, &gs3},
		},
		"packed (sort by name)": {
			list:             []*agonesv1.GameServer{&gs0Allocated, &gs2Allocated, &gs1, &gs3},
			gsToReorder:      &gs1Allocated,
			gsToReorderIndex: 2,
			want:             []*agonesv1.GameServer{&gs0Allocated, &gs1Allocated, &gs2Allocated, &gs3},
		},
		"packed (all ready)": {
			list:             []*agonesv1.GameServer{&gs1, &gs2, &gs3},
			gsToReorder:      &gs1Allocated,
			gsToReorderIndex: 0,
			want:             []*agonesv1.GameServer{&gs1Allocated, &gs2, &gs3},
		},
		"packed (only one)": {
			list:             []*agonesv1.GameServer{&gs1},
			gsToReorder:      &gs1Allocated,
			gsToReorderIndex: 0,
			want:             []*agonesv1.GameServer{&gs1Allocated},
		},
		"packed (priority counter)": {
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			list:     []*agonesv1.GameServer{&gs4, &gs5, &gs6},
			priorities: []agonesv1.Priority{
				{
					Type:  "Counter",
					Key:   "players",
					Order: "Ascending",
				},
			},
			gsToReorder:      &gs5Allocated,
			gsToReorderIndex: 1,
			want:             []*agonesv1.GameServer{&gs5Allocated, &gs4, &gs6},
		},
		"packed (priority list)": {
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			list:     []*agonesv1.GameServer{&gs6, &gs5, &gs4},
			priorities: []agonesv1.Priority{
				{
					Type:  "List",
					Key:   "players",
					Order: "Descending",
				},
			},
			gsToReorder:      &gs5Allocated,
			gsToReorderIndex: 1,
			want:             []*agonesv1.GameServer{&gs6, &gs4, &gs5Allocated},
		},
		"packed (FeaturePlayerAllocationFilter)": {
			features:         fmt.Sprintf("%s=true", runtime.FeaturePlayerAllocationFilter),
			list:             []*agonesv1.GameServer{&gs4, &gs5, &gs6},
			gsToReorder:      &gs5Allocated,
			gsToReorderIndex: 1,
			want:             []*agonesv1.GameServer{&gs5Allocated, &gs4, &gs6},
		},
		"distributed (no change)": {
			list:             []*agonesv1.GameServer{&gs0Allocated, &gs1, &gs2, &gs3},
			packingStrategy:  apis.Distributed,
			gsToReorder:      &gs2Allocated,
			gsToReorderIndex: 2,
			want:             []*agonesv1.GameServer{&gs0Allocated, &gs1, &gs2Allocated, &gs3},
		},
		"distributed (only one)": {
			list:             []*agonesv1.GameServer{&gs1},
			packingStrategy:  apis.Distributed,
			gsToReorder:      &gs1Allocated,
			gsToReorderIndex: 0,
			want:             []*agonesv1.GameServer{&gs1Allocated},
		},
		"distributed (priority counter)": {
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			list:     []*agonesv1.GameServer{&gs4, &gs5, &gs6},
			priorities: []agonesv1.Priority{
				{
					Type:  "Counter",
					Key:   "players",
					Order: "Ascending",
				},
			},
			packingStrategy:  apis.Distributed,
			gsToReorder:      &gs5Allocated,
			gsToReorderIndex: 1,
			want:             []*agonesv1.GameServer{&gs5Allocated, &gs4, &gs6},
		},
		"distributed (priority list)": {
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			list:     []*agonesv1.GameServer{&gs6, &gs5, &gs4},
			priorities: []agonesv1.Priority{
				{
					Type:  "List",
					Key:   "players",
					Order: "Descending",
				},
			},
			packingStrategy:  apis.Distributed,
			gsToReorder:      &gs5Allocated,
			gsToReorderIndex: 1,
			want:             []*agonesv1.GameServer{&gs6, &gs4, &gs5Allocated},
		},
	}

	for testName, testScenario := range fixtures {
		t.Run(testName, func(t *testing.T) {
			// deliberately not resetting the Feature state, to catch any possible unknown regressions
			if testScenario.features != "" {
				require.NoError(t, runtime.ParseFeatures(testScenario.features))
			}

			cache, m := newFakeAllocationCache()

			m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
				return true, &agonesv1.GameServerList{
					Items: func(input []*agonesv1.GameServer) []agonesv1.GameServer {
						result := make([]agonesv1.GameServer, len(input))
						for i, gs := range input {
							result[i] = *gs
						}
						return result
					}(testScenario.list),
				}, nil
			})

			ctx, cancel := agtesting.StartInformers(m, cache.gameServerSynced)
			defer cancel()

			err := cache.syncCache()
			assert.Nil(t, err)

			err = cache.counter.Run(ctx, 0)
			assert.Nil(t, err)

			strategy := apis.Packed
			if testScenario.packingStrategy != "" {
				strategy = testScenario.packingStrategy
			}

			cache.ReorderGameServerAfterAllocation(
				testScenario.list,
				testScenario.gsToReorderIndex, testScenario.gsToReorder,
				testScenario.priorities, strategy)

			if !assert.Equal(t, testScenario.want, testScenario.list, "reordered list should match expected") {
				for _, gs := range testScenario.list {
					t.Logf("%s, ", gs.Name)
				}
			}
		})
	}
}

func newFakeAllocationCache() (*AllocationCache, agtesting.Mocks) {
	m := agtesting.NewMocks()
	cache := NewAllocationCache(m.AgonesInformerFactory.Agones().V1().GameServers(), gameservers.NewPerNodeCounter(m.KubeInformerFactory, m.AgonesInformerFactory), healthcheck.NewHandler())
	return cache, m
}
