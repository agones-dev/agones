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
	"errors"
	"fmt"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	multiclusterv1 "agones.dev/agones/pkg/apis/multicluster/v1"
	"agones.dev/agones/pkg/gameservers"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/test/e2e/framework"
	"github.com/heptiolabs/healthcheck"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestAllocatorAllocate(t *testing.T) {
	t.Parallel()

	// TODO: remove when `CountsAndLists` feature flag is moved to stable.
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	f, gsList := defaultFixtures(4)
	a, m := newFakeAllocator()
	n := metav1.Now()
	labels := map[string]string{"mode": "deathmatch"}
	annotations := map[string]string{"map": "searide"}
	fam := allocationv1.MetaPatch{Labels: labels, Annotations: annotations}

	gsList[3].ObjectMeta.DeletionTimestamp = &n

	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, &agonesv1.GameServerList{Items: gsList}, nil
	})

	updated := false
	gsWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gs := ua.GetObject().(*agonesv1.GameServer)

		updated = true
		assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
		gsWatch.Modify(gs)

		return true, gs, nil
	})

	ctx, cancel := agtesting.StartInformers(m, a.allocationCache.gameServerSynced)
	defer cancel()

	require.NoError(t, a.Run(ctx))
	// wait for it to be up and running
	err := wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (done bool, err error) {
		return a.allocationCache.workerqueue.RunCount() == 1, nil
	})
	require.NoError(t, err)

	gsa := allocationv1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{Name: "gsa-1", Namespace: defaultNs},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: f.ObjectMeta.Name}}}},
			MetaPatch: fam,
		}}
	gsa.ApplyDefaults()
	errs := gsa.Validate()
	require.Len(t, errs, 0)

	gs, err := a.allocate(ctx, &gsa)
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
	assert.True(t, updated)
	for key, value := range fam.Labels {
		v, ok := gs.ObjectMeta.Labels[key]
		assert.True(t, ok)
		assert.Equal(t, v, value)
	}
	for key, value := range fam.Annotations {
		v, ok := gs.ObjectMeta.Annotations[key]
		assert.True(t, ok)
		assert.Equal(t, v, value)
	}

	updated = false
	gs, err = a.allocate(ctx, &gsa)
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
	assert.True(t, updated)

	updated = false
	gs, err = a.allocate(ctx, &gsa)
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
	assert.True(t, updated)

	updated = false
	_, err = a.allocate(ctx, &gsa)
	require.Error(t, err)
	assert.Equal(t, ErrNoGameServer, err)
	assert.False(t, updated)
}

func TestAllocatorAllocatePriority(t *testing.T) {
	t.Parallel()

	// TODO: remove when `CountsAndLists` feature flag is moved to stable.
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	run := func(t *testing.T, name string, test func(t *testing.T, a *Allocator, gas *allocationv1.GameServerAllocation)) {
		f, gsList := defaultFixtures(4)
		a, m := newFakeAllocator()

		gsList[0].Status.NodeName = n1
		gsList[1].Status.NodeName = n2
		gsList[2].Status.NodeName = n1
		gsList[3].Status.NodeName = n1

		m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
			return true, &agonesv1.GameServerList{Items: gsList}, nil
		})

		gsWatch := watch.NewFake()
		m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			gsWatch.Modify(gs)

			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(m, a.allocationCache.gameServerSynced)
		defer cancel()

		require.NoError(t, a.Run(ctx))
		// wait for it to be up and running
		err := wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (done bool, err error) {
			return a.allocationCache.workerqueue.RunCount() == 1, nil
		})
		require.NoError(t, err)

		gsa := &allocationv1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{Name: "fa-1", Namespace: defaultNs},
			Spec: allocationv1.GameServerAllocationSpec{
				Selectors: []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: f.ObjectMeta.Name}}}},
			}}
		gsa.ApplyDefaults()
		errs := gsa.Validate()
		require.Len(t, errs, 0)

		t.Run(name, func(t *testing.T) {
			test(t, a, gsa.DeepCopy())
		})
	}

	run(t, "packed", func(t *testing.T, a *Allocator, gas *allocationv1.GameServerAllocation) {
		ctx := context.Background()
		// priority should be node1, then node2
		gs1, err := a.allocate(ctx, gas)
		assert.NoError(t, err)
		assert.Equal(t, n1, gs1.Status.NodeName)

		gs2, err := a.allocate(ctx, gas)
		assert.NoError(t, err)
		assert.Equal(t, n1, gs2.Status.NodeName)
		assert.NotEqual(t, gs1.ObjectMeta.Name, gs2.ObjectMeta.Name)

		gs3, err := a.allocate(ctx, gas)
		assert.NoError(t, err)
		assert.Equal(t, n1, gs3.Status.NodeName)
		assert.NotContains(t, []string{gs1.ObjectMeta.Name, gs2.ObjectMeta.Name}, gs3.ObjectMeta.Name)

		gs4, err := a.allocate(ctx, gas)
		assert.NoError(t, err)
		assert.Equal(t, n2, gs4.Status.NodeName)
		assert.NotContains(t, []string{gs1.ObjectMeta.Name, gs2.ObjectMeta.Name, gs3.ObjectMeta.Name}, gs4.ObjectMeta.Name)

		// should have none left
		_, err = a.allocate(ctx, gas)
		assert.Equal(t, err, ErrNoGameServer)
	})

	run(t, "distributed", func(t *testing.T, a *Allocator, gas *allocationv1.GameServerAllocation) {
		// make a copy, to avoid the race check
		gas = gas.DeepCopy()
		gas.Spec.Scheduling = apis.Distributed

		// distributed is randomised, so no set pattern
		ctx := context.Background()

		gs1, err := a.allocate(ctx, gas)
		assert.NoError(t, err)

		gs2, err := a.allocate(ctx, gas)
		assert.NoError(t, err)
		assert.NotEqual(t, gs1.ObjectMeta.Name, gs2.ObjectMeta.Name)

		gs3, err := a.allocate(ctx, gas)
		assert.NoError(t, err)
		assert.NotContains(t, []string{gs1.ObjectMeta.Name, gs2.ObjectMeta.Name}, gs3.ObjectMeta.Name)

		gs4, err := a.allocate(ctx, gas)
		assert.NoError(t, err)
		assert.NotContains(t, []string{gs1.ObjectMeta.Name, gs2.ObjectMeta.Name, gs3.ObjectMeta.Name}, gs4.ObjectMeta.Name)

		// should have none left
		_, err = a.allocate(ctx, gas)
		assert.Equal(t, err, ErrNoGameServer)
	})
}

func TestAllocatorApplyAllocationToGameServer(t *testing.T) {
	t.Parallel()
	m := agtesting.NewMocks()
	ctx := context.Background()

	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gs := ua.GetObject().(*agonesv1.GameServer)
		return true, gs, nil
	})

	allocator := NewAllocator(m.AgonesInformerFactory.Multicluster().V1().GameServerAllocationPolicies(),
		m.KubeInformerFactory.Core().V1().Secrets(),
		m.AgonesClient.AgonesV1(), m.KubeClient,
		NewAllocationCache(m.AgonesInformerFactory.Agones().V1().GameServers(), gameservers.NewPerNodeCounter(m.KubeInformerFactory, m.AgonesInformerFactory), healthcheck.NewHandler()),
		time.Second, 5*time.Second, 500*time.Millisecond,
	)

	gs, err := allocator.applyAllocationToGameServer(ctx, allocationv1.MetaPatch{}, &agonesv1.GameServer{}, &allocationv1.GameServerAllocation{})
	assert.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
	assert.NotNil(t, gs.ObjectMeta.Annotations["agones.dev/last-allocated"])
	var ts time.Time
	assert.NoError(t, ts.UnmarshalText([]byte(gs.ObjectMeta.Annotations[LastAllocatedAnnotationKey])))

	gs, err = allocator.applyAllocationToGameServer(ctx, allocationv1.MetaPatch{Labels: map[string]string{"foo": "bar"}}, &agonesv1.GameServer{}, &allocationv1.GameServerAllocation{})
	assert.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
	assert.Equal(t, "bar", gs.ObjectMeta.Labels["foo"])
	assert.NotNil(t, gs.ObjectMeta.Annotations["agones.dev/last-allocated"])

	gs, err = allocator.applyAllocationToGameServer(ctx,
		allocationv1.MetaPatch{Labels: map[string]string{"foo": "bar"}, Annotations: map[string]string{"bar": "foo"}},
		&agonesv1.GameServer{}, &allocationv1.GameServerAllocation{})
	assert.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
	assert.Equal(t, "bar", gs.ObjectMeta.Labels["foo"])
	assert.Equal(t, "foo", gs.ObjectMeta.Annotations["bar"])
	assert.NotNil(t, gs.ObjectMeta.Annotations[LastAllocatedAnnotationKey])
}

func TestAllocatorApplyAllocationToGameServerCountsListsActions(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()
	ctx := context.Background()
	mp := allocationv1.MetaPatch{}

	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gs := ua.GetObject().(*agonesv1.GameServer)
		return true, gs, nil
	})

	allocator := NewAllocator(m.AgonesInformerFactory.Multicluster().V1().GameServerAllocationPolicies(),
		m.KubeInformerFactory.Core().V1().Secrets(),
		m.AgonesClient.AgonesV1(), m.KubeClient,
		NewAllocationCache(m.AgonesInformerFactory.Agones().V1().GameServers(), gameservers.NewPerNodeCounter(m.KubeInformerFactory, m.AgonesInformerFactory), healthcheck.NewHandler()),
		time.Second, 5*time.Second, 500*time.Millisecond,
	)

	ONE := int64(1)
	FORTY := int64(40)
	THOUSAND := int64(1000)
	INCREMENT := "Increment"
	READY := agonesv1.GameServerStateReady

	gs1 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, UID: "1"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Lists: map[string]agonesv1.ListStatus{
				"players": {
					Values:   []string{"alice", "bob", "cat"},
					Capacity: 100,
				},
			},
			Counters: map[string]agonesv1.CounterStatus{
				"rooms": {
					Count:    101,
					Capacity: 1000,
				}}}}
	gs2 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, UID: "2"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Lists: map[string]agonesv1.ListStatus{
				"players": {
					Values:   []string{},
					Capacity: 100,
				},
			},
			Counters: map[string]agonesv1.CounterStatus{
				"rooms": {
					Count:    101,
					Capacity: 1000,
				}}}}

	testScenarios := map[string]struct {
		features     string
		gs           *agonesv1.GameServer
		gsa          *allocationv1.GameServerAllocation
		wantCounters map[string]agonesv1.CounterStatus
		wantLists    map[string]agonesv1.ListStatus
	}{
		"CounterActions increment and ListActions add, delete, and update capacity": {
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			gs:       &gs1,
			gsa: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{Namespace: defaultNs},
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{{
						GameServerState: &READY,
					}},
					Scheduling: apis.Packed,
					Counters: map[string]allocationv1.CounterAction{
						"rooms": {
							Action: &INCREMENT,
							Amount: &ONE,
						}},
					Lists: map[string]allocationv1.ListAction{
						"players": {
							AddValues:    []string{"x7un", "8inz"},
							Capacity:     &FORTY,
							DeleteValues: []string{"bob"},
						}}}},
			wantCounters: map[string]agonesv1.CounterStatus{
				"rooms": {
					Count:    102,
					Capacity: 1000,
				}},
			wantLists: map[string]agonesv1.ListStatus{
				"players": {
					Values:   []string{"alice", "cat", "x7un", "8inz"},
					Capacity: 40,
				}},
		},
		"CounterActions and ListActions truncate counter Count and update list capacity": {
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			gs:       &gs2,
			gsa: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{Namespace: defaultNs},
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{{
						GameServerState: &READY,
					}},
					Scheduling: apis.Packed,
					Counters: map[string]allocationv1.CounterAction{
						"rooms": {
							Action: &INCREMENT,
							Amount: &THOUSAND,
						}},
					Lists: map[string]allocationv1.ListAction{
						"players": {
							AddValues: []string{"x7un", "8inz"},
							Capacity:  &ONE,
						}}}},
			wantCounters: map[string]agonesv1.CounterStatus{
				"rooms": {
					Count:    1000,
					Capacity: 1000,
				}},
			wantLists: map[string]agonesv1.ListStatus{
				"players": {
					Values:   []string{"x7un"},
					Capacity: 1,
				}},
		},
	}

	for test, testScenario := range testScenarios {
		t.Run(test, func(t *testing.T) {
			runtime.FeatureTestMutex.Lock()
			defer runtime.FeatureTestMutex.Unlock()
			// we always set the feature flag in all these tests, so always process it.
			require.NoError(t, runtime.ParseFeatures(testScenario.features))

			foundGs, err := allocator.applyAllocationToGameServer(ctx, mp, testScenario.gs, testScenario.gsa)
			assert.NoError(t, err)
			for counter, counterStatus := range testScenario.wantCounters {
				if gsCounter, ok := foundGs.Status.Counters[counter]; ok {
					assert.Equal(t, counterStatus, gsCounter)
				}
			}
			for list, listStatus := range testScenario.wantLists {
				if gsList, ok := foundGs.Status.Lists[list]; ok {
					assert.Equal(t, listStatus, gsList)
				}
			}
		})
	}
}

func TestAllocationApplyAllocationError(t *testing.T) {
	t.Parallel()
	m := agtesting.NewMocks()
	ctx := context.Background()

	m.AgonesClient.AddReactor("update", "gameservers", func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, nil, errors.New("failed to update")
	})

	allocator := NewAllocator(m.AgonesInformerFactory.Multicluster().V1().GameServerAllocationPolicies(),
		m.KubeInformerFactory.Core().V1().Secrets(),
		m.AgonesClient.AgonesV1(), m.KubeClient,
		NewAllocationCache(m.AgonesInformerFactory.Agones().V1().GameServers(), gameservers.NewPerNodeCounter(m.KubeInformerFactory, m.AgonesInformerFactory), healthcheck.NewHandler()),
		time.Second, 5*time.Second, 500*time.Millisecond,
	)

	gsa, err := allocator.applyAllocationToGameServer(ctx, allocationv1.MetaPatch{}, &agonesv1.GameServer{}, &allocationv1.GameServerAllocation{})
	logrus.WithError(err).WithField("gsa", gsa).WithField("test", t.Name()).Info("Allocation should fail")
	assert.Error(t, err)
}

func TestAllocatorAllocateOnGameServerUpdateError(t *testing.T) {
	t.Parallel()

	// TODO: remove when `CountsAndLists` feature flag is moved to stable.
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=false&%s=false", runtime.FeaturePlayerAllocationFilter, runtime.FeatureCountsAndLists)))

	a, m := newFakeAllocator()
	log := framework.TestLogger(t)

	// make sure there is more than can be retried, so there is always at least some.
	gsLen := allocationRetry.Steps * 2
	_, gsList := defaultFixtures(gsLen)
	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, &agonesv1.GameServerList{Items: gsList}, nil
	})
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gs := ua.GetObject().(*agonesv1.GameServer)

		return true, gs, errors.New("failed to update")
	})

	ctx, cancel := agtesting.StartInformers(m, a.allocationCache.gameServerSynced)
	defer cancel()

	require.NoError(t, a.Run(ctx))
	// wait for all the gameservers to be in the cache
	require.Eventuallyf(t, func() bool {
		return a.allocationCache.cache.Len() == gsLen
	}, 10*time.Second, time.Second, fmt.Sprintf("should be %d items in the cache", gsLen))

	gsa := allocationv1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{Name: "gsa-1", Namespace: defaultNs},
		Spec: allocationv1.GameServerAllocationSpec{},
	}

	gsa.ApplyDefaults()
	// without converter, we don't end up with at least one selector
	gsa.Converter()
	errs := gsa.Validate()
	require.Len(t, errs, 0)
	require.Len(t, gsa.Spec.Selectors, 1)

	// try the private method
	_, err := a.allocate(ctx, gsa.DeepCopy())
	log.WithError(err).Info("allocate (private): failed allocation")
	require.NotEqual(t, ErrNoGameServer, err)
	require.EqualError(t, err, ErrGameServerUpdateConflict.Error())

	// make sure we aren't in the same batch!
	time.Sleep(2 * a.batchWaitTime)

	// wait for all the gameservers to be in the cache
	require.Eventuallyf(t, func() bool {
		return a.allocationCache.cache.Len() == gsLen
	}, 10*time.Second, time.Second, fmt.Sprintf("should be %d items in the cache", gsLen))

	// try the public method
	result, err := a.Allocate(ctx, gsa.DeepCopy())
	log.WithField("result", result).WithError(err).Info("Allocate (public): failed allocation")
	require.Nil(t, result)
	require.NotEqual(t, ErrNoGameServer, err)
	require.EqualError(t, err, ErrGameServerUpdateConflict.Error())
}

func TestAllocatorRunLocalAllocations(t *testing.T) {
	t.Parallel()

	// TODO: remove when `CountsAndLists` feature flag is moved to stable.
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	t.Run("no problems", func(t *testing.T) {
		f, gsList := defaultFixtures(5)
		gsList[0].Status.NodeName = "special"

		a, m := newFakeAllocator()
		m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
			return true, &agonesv1.GameServerList{Items: gsList}, nil
		})
		updateCount := 0
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			updateCount++

			uo := action.(k8stesting.UpdateAction)
			gs := uo.GetObject().(*agonesv1.GameServer)

			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(m, a.allocationCache.gameServerSynced)
		defer cancel()

		// This call initializes the cache
		err := a.allocationCache.syncCache()
		assert.Nil(t, err)

		err = a.allocationCache.counter.Run(ctx, 0)
		assert.Nil(t, err)

		gsa := &allocationv1.GameServerAllocation{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: defaultNs,
			},
			Spec: allocationv1.GameServerAllocationSpec{
				Selectors: []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: f.ObjectMeta.Name}}}},
			}}
		gsa.ApplyDefaults()
		errs := gsa.Validate()
		require.Len(t, errs, 0)

		// line up 3 in a batch
		j1 := request{gsa: gsa.DeepCopy(), response: make(chan response, 1), ctx: context.Background()}
		a.pendingRequests <- j1
		j2 := request{gsa: gsa.DeepCopy(), response: make(chan response, 1), ctx: context.Background()}
		a.pendingRequests <- j2
		j3 := request{gsa: gsa.DeepCopy(), response: make(chan response, 1), ctx: context.Background()}
		a.pendingRequests <- j3

		go a.ListenAndAllocate(ctx, 3)

		res1 := <-j1.response
		assert.NoError(t, res1.err)
		assert.NotNil(t, res1.gs)

		// since we gave gsList[0] a different nodename, it should always come first
		assert.Contains(t, []string{"gs2", "gs3", "gs4", "gs5"}, res1.gs.ObjectMeta.Name)
		assert.Equal(t, agonesv1.GameServerStateAllocated, res1.gs.Status.State)

		res2 := <-j2.response
		assert.NoError(t, res2.err)
		assert.NotNil(t, res2.gs)
		assert.NotEqual(t, res1.gs.ObjectMeta.Name, res2.gs.ObjectMeta.Name)
		assert.Equal(t, agonesv1.GameServerStateAllocated, res2.gs.Status.State)

		res3 := <-j3.response
		assert.NoError(t, res3.err)
		assert.NotNil(t, res3.gs)
		assert.Equal(t, agonesv1.GameServerStateAllocated, res3.gs.Status.State)
		assert.NotEqual(t, res1.gs.ObjectMeta.Name, res3.gs.ObjectMeta.Name)
		assert.NotEqual(t, res2.gs.ObjectMeta.Name, res3.gs.ObjectMeta.Name)

		assert.Equal(t, 3, updateCount)
	})

	t.Run("no gameservers", func(t *testing.T) {
		a, m := newFakeAllocator()
		ctx, cancel := agtesting.StartInformers(m, a.allocationCache.gameServerSynced)
		defer cancel()

		// This call initializes the cache
		err := a.allocationCache.syncCache()
		assert.Nil(t, err)

		err = a.allocationCache.counter.Run(ctx, 0)
		assert.Nil(t, err)

		gsa := &allocationv1.GameServerAllocation{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: defaultNs,
			},
			Spec: allocationv1.GameServerAllocationSpec{
				Selectors: []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: "thereisnofleet"}}}},
			}}
		gsa.ApplyDefaults()
		errs := gsa.Validate()
		require.Len(t, errs, 0)

		j1 := request{gsa: gsa.DeepCopy(), response: make(chan response, 1), ctx: context.Background()}
		a.pendingRequests <- j1

		go a.ListenAndAllocate(ctx, 3)

		res1 := <-j1.response
		assert.Nil(t, res1.gs)
		assert.Error(t, res1.err)
		assert.Equal(t, ErrNoGameServer, res1.err)
	})

	t.Run("do not propagate allocation if request context cancelled", func(t *testing.T) {
		a, m := newFakeAllocator()
		ctx, cancel := agtesting.StartInformers(m, a.allocationCache.gameServerSynced)
		defer cancel()

		// This call initializes the cache
		err := a.allocationCache.syncCache()
		assert.Nil(t, err)

		err = a.allocationCache.counter.Run(ctx, 0)
		assert.Nil(t, err)

		gsa := &allocationv1.GameServerAllocation{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: defaultNs,
			},
			Spec: allocationv1.GameServerAllocationSpec{
				Selectors: []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: "thereisnofleet"}}}},
			}}
		gsa.ApplyDefaults()
		errs := gsa.Validate()
		require.Len(t, errs, 0)

		reqCtx, reqCancel := context.WithTimeout(context.Background(), time.Millisecond)
		reqCancel()
		j1 := request{gsa: gsa.DeepCopy(), response: make(chan response, 1), ctx: reqCtx}
		a.pendingRequests <- j1

		go a.ListenAndAllocate(ctx, 3)

		res1 := <-j1.response
		assert.Nil(t, res1.gs)
		assert.Error(t, res1.err)
		assert.Equal(t, ErrTotalTimeoutExceeded, res1.err)
	})
}

func TestAllocatorRunLocalAllocationsCountsAndLists(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

	a, m := newFakeAllocator()

	gs1 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, UID: "1"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Counters: map[string]agonesv1.CounterStatus{
				"foo": { // Available Capacity == 1000
					Count:    0,
					Capacity: 1000,
				}}}}
	gs2 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, UID: "2"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Counters: map[string]agonesv1.CounterStatus{
				"foo": { // Available Capacity == 900
					Count:    100,
					Capacity: 1000,
				}}}}
	gs3 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, UID: "3"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Counters: map[string]agonesv1.CounterStatus{
				"foo": { // Available Capacity == 1
					Count:    999,
					Capacity: 1000,
				}}}}
	gs4 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, UID: "4"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Lists: map[string]agonesv1.ListStatus{
				"foo": { // Available Capacity == 10
					Values:   []string{},
					Capacity: 10,
				}}}}
	gs5 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, UID: "5"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Lists: map[string]agonesv1.ListStatus{
				"foo": { // Available Capacity == 9
					Values:   []string{"1"},
					Capacity: 10,
				}}}}
	gs6 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: defaultNs, UID: "6"},
		Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady,
			Lists: map[string]agonesv1.ListStatus{
				"foo": { // Available Capacity == 1
					Values:   []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"},
					Capacity: 10,
				}}}}

	gsList := []agonesv1.GameServer{gs1, gs2, gs3, gs4, gs5, gs6}

	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, &agonesv1.GameServerList{Items: gsList}, nil
	})

	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		uo := action.(k8stesting.UpdateAction)
		gs := uo.GetObject().(*agonesv1.GameServer)
		return true, gs, nil
	})

	ctx, cancel := agtesting.StartInformers(m, a.allocationCache.gameServerSynced)
	defer cancel()

	// This call initializes the cache
	err := a.allocationCache.syncCache()
	assert.Nil(t, err)

	err = a.allocationCache.counter.Run(ctx, 0)
	assert.Nil(t, err)

	READY := agonesv1.GameServerStateReady

	gsaAscending := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{Namespace: defaultNs},
		Spec: allocationv1.GameServerAllocationSpec{
			Scheduling: apis.Packed,
			Selectors: []allocationv1.GameServerSelector{{
				GameServerState: &READY,
			}},
			Priorities: []agonesv1.Priority{
				{Type: agonesv1.GameServerPriorityCounter,
					Key:   "foo",
					Order: agonesv1.GameServerPriorityAscending},
			},
		}}
	gsaDescending := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{Namespace: defaultNs},
		Spec: allocationv1.GameServerAllocationSpec{
			Scheduling: apis.Packed,
			Selectors: []allocationv1.GameServerSelector{{
				GameServerState: &READY,
			}},
			Priorities: []agonesv1.Priority{
				{Type: agonesv1.GameServerPriorityCounter,
					Key:   "foo",
					Order: agonesv1.GameServerPriorityDescending},
			},
		}}
	gsaDistributed := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{Namespace: defaultNs},
		Spec: allocationv1.GameServerAllocationSpec{
			Scheduling: apis.Distributed,
			Selectors: []allocationv1.GameServerSelector{{
				GameServerState: &READY,
			}},
			Priorities: []agonesv1.Priority{
				{Type: agonesv1.GameServerPriorityCounter,
					Key:   "foo",
					Order: agonesv1.GameServerPriorityDescending},
			},
		}}
	gsaListAscending := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{Namespace: defaultNs},
		Spec: allocationv1.GameServerAllocationSpec{
			Scheduling: apis.Packed,
			Selectors: []allocationv1.GameServerSelector{{
				GameServerState: &READY,
			}},
			Priorities: []agonesv1.Priority{
				{Type: agonesv1.GameServerPriorityList,
					Key:   "foo",
					Order: agonesv1.GameServerPriorityAscending},
			},
		}}
	gsaListDescending := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{Namespace: defaultNs},
		Spec: allocationv1.GameServerAllocationSpec{
			Scheduling: apis.Packed,
			Selectors: []allocationv1.GameServerSelector{{
				GameServerState: &READY,
			}},
			Priorities: []agonesv1.Priority{
				{Type: agonesv1.GameServerPriorityList,
					Key:   "foo",
					Order: agonesv1.GameServerPriorityDescending},
			},
		}}
	gsaListDistributed := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{Namespace: defaultNs},
		Spec: allocationv1.GameServerAllocationSpec{
			Scheduling: apis.Distributed,
			Selectors: []allocationv1.GameServerSelector{{
				GameServerState: &READY,
			}},
			Priorities: []agonesv1.Priority{
				{Type: agonesv1.GameServerPriorityList,
					Key:   "foo",
					Order: agonesv1.GameServerPriorityDescending},
			},
		}}

	j1 := request{gsa: gsaDescending.DeepCopy(), response: make(chan response, 1), ctx: context.Background()}
	a.pendingRequests <- j1
	j2 := request{gsa: gsaAscending.DeepCopy(), response: make(chan response, 1), ctx: context.Background()}
	a.pendingRequests <- j2
	j3 := request{gsa: gsaDistributed.DeepCopy(), response: make(chan response, 1), ctx: context.Background()}
	a.pendingRequests <- j3
	j4 := request{gsa: gsaListDescending.DeepCopy(), response: make(chan response, 1), ctx: context.Background()}
	a.pendingRequests <- j4
	j5 := request{gsa: gsaListAscending.DeepCopy(), response: make(chan response, 1), ctx: context.Background()}
	a.pendingRequests <- j5
	j6 := request{gsa: gsaListDistributed.DeepCopy(), response: make(chan response, 1), ctx: context.Background()}
	a.pendingRequests <- j6

	go a.ListenAndAllocate(ctx, 5)

	res1 := <-j1.response
	assert.NoError(t, res1.err)
	assert.NotNil(t, res1.gs)
	assert.Equal(t, agonesv1.GameServerStateAllocated, res1.gs.Status.State)
	assert.Equal(t, gs1.ObjectMeta.Name, res1.gs.ObjectMeta.Name)
	assert.Equal(t, gs1.Status.Counters["foo"].Count, res1.gs.Status.Counters["foo"].Count)
	assert.Equal(t, gs1.Status.Counters["foo"].Capacity, res1.gs.Status.Counters["foo"].Capacity)

	res2 := <-j2.response
	assert.NoError(t, res2.err)
	assert.NotNil(t, res2.gs)
	assert.Equal(t, agonesv1.GameServerStateAllocated, res2.gs.Status.State)
	assert.Equal(t, gs3.ObjectMeta.Name, res2.gs.ObjectMeta.Name)
	assert.Equal(t, gs3.Status.Counters["foo"].Count, res2.gs.Status.Counters["foo"].Count)
	assert.Equal(t, gs3.Status.Counters["foo"].Capacity, res2.gs.Status.Counters["foo"].Capacity)

	res3 := <-j3.response
	assert.NoError(t, res3.err)
	assert.NotNil(t, res3.gs)
	assert.Equal(t, agonesv1.GameServerStateAllocated, res3.gs.Status.State)
	assert.Equal(t, gs2.ObjectMeta.Name, res3.gs.ObjectMeta.Name)
	assert.Equal(t, gs2.Status.Counters["foo"].Count, res3.gs.Status.Counters["foo"].Count)
	assert.Equal(t, gs2.Status.Counters["foo"].Capacity, res3.gs.Status.Counters["foo"].Capacity)

	res4 := <-j4.response
	assert.NoError(t, res4.err)
	assert.NotNil(t, res4.gs)
	assert.Equal(t, agonesv1.GameServerStateAllocated, res4.gs.Status.State)
	assert.Equal(t, gs4.ObjectMeta.Name, res4.gs.ObjectMeta.Name)
	assert.Equal(t, gs4.Status.Lists["foo"].Values, res4.gs.Status.Lists["foo"].Values)
	assert.Equal(t, gs4.Status.Lists["foo"].Capacity, res4.gs.Status.Lists["foo"].Capacity)

	res5 := <-j5.response
	assert.NoError(t, res5.err)
	assert.NotNil(t, res5.gs)
	assert.Equal(t, agonesv1.GameServerStateAllocated, res5.gs.Status.State)
	assert.Equal(t, gs6.ObjectMeta.Name, res5.gs.ObjectMeta.Name)
	assert.Equal(t, gs6.Status.Lists["foo"].Values, res5.gs.Status.Lists["foo"].Values)
	assert.Equal(t, gs6.Status.Lists["foo"].Capacity, res5.gs.Status.Lists["foo"].Capacity)

	res6 := <-j6.response
	assert.NoError(t, res6.err)
	assert.NotNil(t, res6.gs)
	assert.Equal(t, agonesv1.GameServerStateAllocated, res6.gs.Status.State)
	assert.Equal(t, gs5.ObjectMeta.Name, res6.gs.ObjectMeta.Name)
	assert.Equal(t, gs5.Status.Lists["foo"].Values, res6.gs.Status.Lists["foo"].Values)
	assert.Equal(t, gs5.Status.Lists["foo"].Capacity, res6.gs.Status.Lists["foo"].Capacity)

}

func TestControllerAllocationUpdateWorkers(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		a, m := newFakeAllocator()

		updated := false
		gs1 := &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: "gs1"},
		}
		r := response{
			request: request{
				gsa:      &allocationv1.GameServerAllocation{},
				response: make(chan response, 1),
				ctx:      context.Background(),
			},
			gs: gs1,
		}

		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			updated = true

			uo := action.(k8stesting.UpdateAction)
			gs := uo.GetObject().(*agonesv1.GameServer)

			assert.Equal(t, gs1.ObjectMeta.Name, gs.ObjectMeta.Name)
			assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)

			return true, gs, nil
		})

		updateQueue := a.allocationUpdateWorkers(context.Background(), 1)

		go func() {
			updateQueue <- r
		}()

		r = <-r.request.response

		assert.True(t, updated)
		assert.NoError(t, r.err)
		assert.Equal(t, gs1.ObjectMeta.Name, r.gs.ObjectMeta.Name)
		assert.Equal(t, agonesv1.GameServerStateAllocated, r.gs.Status.State)

		// Verify that the GameServer was added back to the cache after successful allocation
		key, err := cache.MetaNamespaceKeyFunc(gs1)
		require.NoError(t, err)
		cached, ok := a.allocationCache.cache.Load(key)
		require.True(t, ok, "GameServer should be in the cache after successful allocation")
		require.Equal(t, gs1.ObjectMeta.Name, cached.ObjectMeta.Name)

		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Allocated")

		// make sure we can do more allocations than number of workers
		gs2 := &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: "gs1"},
		}
		r = response{
			request: request{
				gsa:      &allocationv1.GameServerAllocation{},
				response: make(chan response, 1),
				ctx:      context.Background(),
			},
			gs: gs2,
		}

		go func() {
			updateQueue <- r
		}()

		r = <-r.request.response

		assert.True(t, updated)
		assert.NoError(t, r.err)
		assert.Equal(t, gs2.ObjectMeta.Name, r.gs.ObjectMeta.Name)
		assert.Equal(t, agonesv1.GameServerStateAllocated, r.gs.Status.State)

		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Allocated")
	})

	t.Run("error on update", func(t *testing.T) {
		a, m := newFakeAllocator()

		updated := false
		gs1 := &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: "gs1"},
		}
		key, err := cache.MetaNamespaceKeyFunc(gs1)
		assert.NoError(t, err)

		_, ok := a.allocationCache.cache.Load(key)
		assert.False(t, ok)

		r := response{
			request: request{
				gsa:      &allocationv1.GameServerAllocation{},
				response: make(chan response, 1),
				ctx:      context.Background(),
			},
			gs: gs1,
		}

		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			updated = true

			uo := action.(k8stesting.UpdateAction)
			gs := uo.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, gs1.ObjectMeta.Name, gs.ObjectMeta.Name)
			assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)

			return true, gs, errors.New("something went wrong")
		})

		updateQueue := a.allocationUpdateWorkers(context.Background(), 1)

		go func() {
			updateQueue <- r
		}()

		r = <-r.request.response

		assert.True(t, updated)
		assert.EqualError(t, r.err, ErrGameServerUpdateConflict.Error())
		assert.Equal(t, gs1, r.gs)
		agtesting.AssertNoEvent(t, m.FakeRecorder.Events)

		var cached *agonesv1.GameServer
		cached, ok = a.allocationCache.cache.Load(key)
		assert.True(t, ok)
		assert.Equal(t, gs1.ObjectMeta.Name, cached.ObjectMeta.Name)
	})
}

func TestAllocatorCreateRestClientError(t *testing.T) {
	t.Parallel()
	t.Run("Missing secret", func(t *testing.T) {
		a, _ := newFakeAllocator()

		connectionInfo := &multiclusterv1.ClusterConnectionInfo{
			SecretName: "secret-name",
		}
		_, err := a.createRemoteClusterDialOption(defaultNs, connectionInfo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret-name")
	})
	t.Run("Missing cert", func(t *testing.T) {
		a, m := newFakeAllocator()

		m.KubeClient.AddReactor("list", "secrets",
			func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
				return true, &corev1.SecretList{
					Items: []corev1.Secret{{
						Data: map[string][]byte{
							"tls.crt": clientCert,
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "secret-name",
							Namespace: defaultNs,
						},
					}}}, nil
			})

		_, cancel := agtesting.StartInformers(m, a.secretSynced)
		defer cancel()

		connectionInfo := &multiclusterv1.ClusterConnectionInfo{
			SecretName: "secret-name",
		}
		_, err := a.createRemoteClusterDialOption(defaultNs, connectionInfo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing client certificate key pair in secret secret-name")
	})
	t.Run("Bad client cert", func(t *testing.T) {
		a, m := newFakeAllocator()

		m.KubeClient.AddReactor("list", "secrets",
			func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
				return true, &corev1.SecretList{
					Items: []corev1.Secret{{
						Data: map[string][]byte{
							"tls.crt": []byte("XXX"),
							"tls.key": []byte("XXX"),
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "secret-name",
							Namespace: defaultNs,
						},
					}}}, nil
			})

		_, cancel := agtesting.StartInformers(m, a.secretSynced)
		defer cancel()

		connectionInfo := &multiclusterv1.ClusterConnectionInfo{
			SecretName: "secret-name",
		}
		_, err := a.createRemoteClusterDialOption(defaultNs, connectionInfo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find any PEM data in certificate input")
	})
	t.Run("Bad CA cert", func(t *testing.T) {
		a, m := newFakeAllocator()

		m.KubeClient.AddReactor("list", "secrets",
			func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
				return true, getTestSecret("secret-name", clientCert), nil
			})

		_, cancel := agtesting.StartInformers(m, a.secretSynced)
		defer cancel()

		connectionInfo := &multiclusterv1.ClusterConnectionInfo{
			SecretName: "secret-name",
			ServerCA:   []byte("XXX"),
		}
		_, err := a.createRemoteClusterDialOption(defaultNs, connectionInfo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PEM format")
	})
	t.Run("Bad client CA cert", func(t *testing.T) {
		a, m := newFakeAllocator()

		m.KubeClient.AddReactor("list", "secrets",
			func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
				return true, getTestSecret("secret-name", []byte("XXX")), nil
			})

		_, cancel := agtesting.StartInformers(m, a.secretSynced)
		defer cancel()

		connectionInfo := &multiclusterv1.ClusterConnectionInfo{
			SecretName: "secret-name",
		}
		_, err := a.createRemoteClusterDialOption(defaultNs, connectionInfo)
		assert.Nil(t, err)
	})
}

// newFakeAllocator returns a fake allocator.
func newFakeAllocator() (*Allocator, agtesting.Mocks) {
	m := agtesting.NewMocks()

	counter := gameservers.NewPerNodeCounter(m.KubeInformerFactory, m.AgonesInformerFactory)
	a := NewAllocator(
		m.AgonesInformerFactory.Multicluster().V1().GameServerAllocationPolicies(),
		m.KubeInformerFactory.Core().V1().Secrets(),
		m.AgonesClient.AgonesV1(),
		m.KubeClient,
		NewAllocationCache(m.AgonesInformerFactory.Agones().V1().GameServers(), counter, healthcheck.NewHandler()),
		time.Second,
		5*time.Second,
		500*time.Millisecond)
	a.recorder = m.FakeRecorder

	return a, m
}
