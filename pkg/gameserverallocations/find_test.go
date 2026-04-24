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
	"fmt"
	"testing"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestFindGameServerForAllocationPacked(t *testing.T) {
	t.Parallel()

	oneLabel := map[string]string{"role": "gameserver"}
	twoLabels := map[string]string{"role": "gameserver", "preferred": "true"}

	gsa := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{Namespace: defaultNs},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{
				MatchLabels: oneLabel,
			}}},
			Scheduling: apis.Packed,
		},
	}

	n := metav1.Now()
	twoLabelsGsa := gsa.DeepCopy()
	twoLabelsGsa.Spec.Selectors = append(
		[]allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{"preferred": "true"},
		}}}, twoLabelsGsa.Spec.Selectors...)

	fixtures := map[string]struct {
		list     []agonesv1.GameServer
		test     func(*testing.T, []*agonesv1.GameServer)
		features string
		gsa      *allocationv1.GameServerAllocation
	}{
		"empty selector": {
			list: []agonesv1.GameServer{{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, Labels: oneLabel}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}}},
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				require.Len(t, list, 1)

				emptyGSA := &allocationv1.GameServerAllocation{
					ObjectMeta: metav1.ObjectMeta{Namespace: defaultNs},
					Spec: allocationv1.GameServerAllocationSpec{
						Scheduling: apis.Packed,
					},
				}
				emptyGSA.ApplyDefaults()
				emptyGSA.Converter()
				allErrs := emptyGSA.Validate()
				require.Len(t, allErrs, 0)
				require.Len(t, emptyGSA.Spec.Selectors, 1)

				gs, index, err := findGameServerForAllocation(emptyGSA, list)
				assert.NotNil(t, gs)
				assert.Equal(t, 0, index)
				assert.NoError(t, err)
			},
		},
		"one label with player state (StateAllocationFilter)": {
			// nolint: dupl
			list: []agonesv1.GameServer{
				{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: defaultNs, Labels: oneLabel, DeletionTimestamp: &n}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, Labels: oneLabel}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, Labels: oneLabel}, Status: agonesv1.GameServerStatus{NodeName: "node2", State: agonesv1.GameServerStateReady}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, Labels: oneLabel}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateAllocated}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, Labels: oneLabel}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateAllocated}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, Labels: oneLabel}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateError}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: "does-not-apply", Labels: oneLabel}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}},
			},
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				require.Len(t, list, 5)
				require.Equal(t, agonesv1.GameServerStateReady, *gsa.Spec.Selectors[0].GameServerState)

				gs, index, err := findGameServerForAllocation(gsa, list)
				assert.NoError(t, err)
				require.NotNil(t, gs)
				assert.Equal(t, "node1", gs.Status.NodeName)
				assert.Equal(t, "gs1", gs.ObjectMeta.Name)
				assert.Equal(t, gs, list[index])
				assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)

				// remove the allocated game server
				list = append(list[:index], list[index+1:]...)
				assert.Len(t, list, 4)

				gs, index, err = findGameServerForAllocation(gsa, list)
				assert.NoError(t, err)
				require.NotNil(t, gs)

				assert.Equal(t, "node2", gs.Status.NodeName)
				assert.Equal(t, "gs2", gs.ObjectMeta.Name)
				assert.Equal(t, gs, list[index])
				assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)

				// now try an allocated state
				gsa := gsa.DeepCopy()
				allocated := agonesv1.GameServerStateAllocated
				gsa.Spec.Selectors[0].GameServerState = &allocated

				gs, index, err = findGameServerForAllocation(gsa, list)
				assert.NoError(t, err)
				require.NotNil(t, gs)
				assert.Equal(t, "node1", gs.Status.NodeName)
				// either is valid
				assert.Contains(t, []string{"gs3", "gs4"}, gs.ObjectMeta.Name)
				assert.Equal(t, gs, list[index])
				assert.Equal(t, allocated, gs.Status.State)

				// finally, we have nothing left
				list = nil
				gs, _, err = findGameServerForAllocation(gsa, list)
				assert.Error(t, err)
				assert.Equal(t, ErrNoGameServer, err)
				assert.Nil(t, gs)
			},
			features: fmt.Sprintf("%s=true", runtime.FeaturePlayerAllocationFilter),
		},
		"one label with player counts and state (PlayerAllocationFilter)": {
			list: []agonesv1.GameServer{
				{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, Labels: oneLabel}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, Labels: oneLabel}, Status: agonesv1.GameServerStatus{NodeName: "node2", State: agonesv1.GameServerStateReady}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, Labels: oneLabel},
					Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateAllocated,
						Players: &agonesv1.PlayerStatus{Count: 10, Capacity: 15}}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, Labels: oneLabel},
					Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateAllocated, Players: &agonesv1.PlayerStatus{
						Count:    3,
						Capacity: 15,
					}}},
			},
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				gsa := gsa.DeepCopy()
				allocated := agonesv1.GameServerStateAllocated
				gsa.Spec.Selectors[0].GameServerState = &allocated
				gsa.Spec.Selectors[0].Players = &allocationv1.PlayerSelector{
					MinAvailable: 1,
					MaxAvailable: 10,
				}
				require.Len(t, list, 4)

				gs, index, err := findGameServerForAllocation(gsa, list)
				assert.NoError(t, err)
				require.NotNil(t, gs)
				assert.Equal(t, "node1", gs.Status.NodeName)
				assert.Equal(t, "gs3", gs.ObjectMeta.Name)
				assert.Equal(t, gs, list[index])
				assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
			},
			features: fmt.Sprintf("%s=true", runtime.FeaturePlayerAllocationFilter),
		},
		"preferred": {
			list: []agonesv1.GameServer{
				{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, Labels: twoLabels}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, Labels: oneLabel}, Status: agonesv1.GameServerStatus{NodeName: "node2", State: agonesv1.GameServerStateReady}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, Labels: oneLabel}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, Labels: twoLabels}, Status: agonesv1.GameServerStatus{NodeName: "node2", State: agonesv1.GameServerStateReady}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, Labels: oneLabel}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: defaultNs, Labels: oneLabel}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}},
			},
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				assert.Len(t, list, 6)

				gs, index, err := findGameServerForAllocation(twoLabelsGsa, list)
				assert.NoError(t, err)
				assert.Equal(t, "node1", gs.Status.NodeName)
				assert.Equal(t, "gs1", gs.ObjectMeta.Name)
				assert.Equal(t, gs, list[index])
				assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)

				list = append(list[:index], list[index+1:]...)
				gs, index, err = findGameServerForAllocation(twoLabelsGsa, list)
				assert.NoError(t, err)
				assert.Equal(t, "node2", gs.Status.NodeName)
				assert.Equal(t, "gs4", gs.ObjectMeta.Name)
				assert.Equal(t, gs, list[index])
				assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)

				list = append(list[:index], list[index+1:]...)
				gs, index, err = findGameServerForAllocation(twoLabelsGsa, list)
				assert.NoError(t, err)
				assert.Equal(t, "node1", gs.Status.NodeName)
				assert.Contains(t, []string{"gs3", "gs5", "gs6"}, gs.ObjectMeta.Name)
				assert.Equal(t, gs, list[index])
				assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
			},
			features: fmt.Sprintf("%s=false", runtime.FeaturePlayerAllocationFilter),
		},
		"allocation trap": {
			list: []agonesv1.GameServer{
				{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Labels: oneLabel, Namespace: defaultNs}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateAllocated}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Labels: oneLabel, Namespace: defaultNs}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateAllocated}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Labels: oneLabel, Namespace: defaultNs}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateAllocated}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Labels: oneLabel, Namespace: defaultNs}, Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateAllocated}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Labels: oneLabel, Namespace: defaultNs}, Status: agonesv1.GameServerStatus{NodeName: "node2", State: agonesv1.GameServerStateReady}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Labels: oneLabel, Namespace: defaultNs}, Status: agonesv1.GameServerStatus{NodeName: "node2", State: agonesv1.GameServerStateReady}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs7", Labels: oneLabel, Namespace: defaultNs}, Status: agonesv1.GameServerStatus{NodeName: "node2", State: agonesv1.GameServerStateReady}},
				{ObjectMeta: metav1.ObjectMeta{Name: "gs8", Labels: oneLabel, Namespace: defaultNs}, Status: agonesv1.GameServerStatus{NodeName: "node2", State: agonesv1.GameServerStateReady}},
			},
			test: func(t *testing.T, list []*agonesv1.GameServer) {
				assert.Len(t, list, 8)

				gs, index, err := findGameServerForAllocation(gsa, list)
				assert.Nil(t, err)
				assert.Equal(t, "node2", gs.Status.NodeName)
				assert.Equal(t, gs, list[index])
				assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
			},
			features: fmt.Sprintf("%s=false", runtime.FeaturePlayerAllocationFilter),
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			runtime.FeatureTestMutex.Lock()
			defer runtime.FeatureTestMutex.Unlock()
			// we always set the feature flag in all these tests, so always process it.
			require.NoError(t, runtime.ParseFeatures(v.features))

			gsa.ApplyDefaults()
			allErrs := gsa.Validate()
			require.Len(t, allErrs, 0)

			twoLabelsGsa.ApplyDefaults()
			allErrs = twoLabelsGsa.Validate()
			require.Len(t, allErrs, 0)

			controller, m := newFakeController()
			c := controller.allocator.allocationCache

			m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
				return true, &agonesv1.GameServerList{Items: v.list}, nil
			})

			ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced)
			defer cancel()

			// This call initializes the cache
			err := c.syncCache()
			assert.Nil(t, err)

			err = c.counter.Run(ctx, 0)
			assert.Nil(t, err)

			list := c.ListSortedGameServers(v.gsa)
			v.test(t, list)
		})
	}
}

func TestFindGameServerForAllocationDistributed(t *testing.T) {
	t.Parallel()

	// TODO: remove when `CountsAndLists` feature flag is moved to stable.
	// NOTE: CountsAndLists has different behavior for Distributed, and the game server list is not random.
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=false"))

	controller, m := newFakeController()
	c := controller.allocator.allocationCache
	labels := map[string]string{"role": "gameserver"}

	gsa := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{Namespace: defaultNs},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{
				MatchLabels: labels,
			}}},
			Scheduling: apis.Distributed,
		},
	}
	gsa.ApplyDefaults()
	allErrs := gsa.Validate()
	require.Len(t, allErrs, 0)

	gsList := []agonesv1.GameServer{
		{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, Labels: labels},
			Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, Labels: labels},
			Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs3", Namespace: defaultNs, Labels: labels},
			Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs4", Namespace: defaultNs, Labels: labels},
			Status: agonesv1.GameServerStatus{NodeName: "node1", State: agonesv1.GameServerStateError}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs5", Namespace: defaultNs, Labels: labels},
			Status: agonesv1.GameServerStatus{NodeName: "node2", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs6", Namespace: defaultNs, Labels: labels},
			Status: agonesv1.GameServerStatus{NodeName: "node2", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gs7", Namespace: defaultNs, Labels: labels},
			Status: agonesv1.GameServerStatus{NodeName: "node3", State: agonesv1.GameServerStateReady}},
	}

	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, &agonesv1.GameServerList{Items: gsList}, nil
	})

	ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced)
	defer cancel()

	// This call initializes the cache
	err := c.syncCache()
	assert.Nil(t, err)

	err = c.counter.Run(ctx, 0)
	assert.Nil(t, err)

	list := c.ListSortedGameServers(gsa)
	assert.Len(t, list, 6)

	gs, index, err := findGameServerForAllocation(gsa, list)
	assert.NoError(t, err)
	assert.Equal(t, gs, list[index])
	assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)

	past := gs
	// we should get a different result in 10 tries, so we can see we get some randomness.
	for i := 0; i < 10; i++ {
		gs, index, err = findGameServerForAllocation(gsa, list)
		assert.NoError(t, err)
		assert.Equal(t, gs, list[index])
		assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)

		if gs.ObjectMeta.Name != past.ObjectMeta.Name {
			return
		}
	}

	assert.FailNow(t, "We should get a different gameserver by now")

}
