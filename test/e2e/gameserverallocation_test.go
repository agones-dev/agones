// Copyright 2018 Google LLC All Rights Reserved.
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

package e2e

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	multiclusterv1 "agones.dev/agones/pkg/apis/multicluster/v1"
	"agones.dev/agones/pkg/util/runtime"
	e2e "agones.dev/agones/test/e2e/framework"
)

func TestCreateFleetAndGameServerAllocate(t *testing.T) {
	t.Parallel()

	fixtures := []apis.SchedulingStrategy{apis.Packed, apis.Distributed}

	for _, strategy := range fixtures {
		t.Run(string(strategy), func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
			fleet := defaultFleet(framework.Namespace)
			fleet.Spec.Scheduling = strategy
			flt, err := fleets.Create(ctx, fleet, metav1.CreateOptions{})
			if strategy != apis.Packed && framework.CloudProduct == "gke-autopilot" {
				// test that Autopilot rejects anything but Packed and skip the rest of the test
				assert.ErrorContains(t, err, "Invalid value")
				return
			}
			if assert.NoError(t, err) {
				defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
			}

			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

			gsaList := []*allocationv1.GameServerAllocation{
				{Spec: allocationv1.GameServerAllocationSpec{
					Scheduling: strategy,
					Selectors:  []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}}}},
				},
				{Spec: allocationv1.GameServerAllocationSpec{
					Scheduling: strategy,
					Required:   allocationv1.GameServerSelector{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}}},
				},
			}

			for _, gsa := range gsaList {
				gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa, metav1.CreateOptions{})
				if assert.NoError(t, err) {
					assert.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(gsa.Status.State))
				}
			}
		})
	}
}

func TestCreateFleetAndGameServerStateFilterAllocation(t *testing.T) {
	t.Parallel()

	fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
	fleet := defaultFleet(framework.Namespace)
	ctx := context.Background()

	flt, err := fleets.Create(ctx, fleet, metav1.CreateOptions{})
	require.NoError(t, err)
	defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetSelector := metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}
	gsa := &allocationv1.GameServerAllocation{
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{{LabelSelector: fleetSelector}},
		}}

	// standard allocation
	gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa, metav1.CreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(gsa.Status.State))

	gs1, err := framework.AgonesClient.AgonesV1().GameServers(fleet.ObjectMeta.Namespace).Get(ctx, gsa.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs1.Status.State)
	assert.NotNil(t, gs1.ObjectMeta.Annotations["agones.dev/last-allocated"])

	// now let's get it back again
	gsa = gsa.DeepCopy()
	allocated := agonesv1.GameServerStateAllocated
	gsa.Spec.Selectors[0].GameServerState = &allocated

	gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa, metav1.CreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(gsa.Status.State))
	assert.Equal(t, gs1.ObjectMeta.Name, gsa.Status.GameServerName)

	gs2, err := framework.AgonesClient.AgonesV1().GameServers(fleet.ObjectMeta.Namespace).Get(ctx, gsa.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs2.Status.State)

	require.Equal(t, gs1.ObjectMeta.Name, gs2.ObjectMeta.Name)
	require.NotEqual(t, gs1.ObjectMeta.ResourceVersion, gs2.ObjectMeta.ResourceVersion)
	require.NotEqual(t, gs1.ObjectMeta.Annotations["agones.dev/last-allocated"], gs2.ObjectMeta.Annotations["agones.dev/last-allocated"])
}

func TestHighDensityGameServerFlow(t *testing.T) {
	t.Parallel()
	log := e2e.TestLogger(t)
	ctx := context.Background()

	fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
	fleet := defaultFleet(framework.Namespace)
	lockLabel := "agones.dev/sdk-available"
	// to start they are all available
	fleet.Spec.Template.ObjectMeta.Labels = map[string]string{lockLabel: "true"}

	flt, err := fleets.Create(ctx, fleet, metav1.CreateOptions{})
	require.NoError(t, err)
	defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetSelector := metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}
	allocatedSelector := fleetSelector.DeepCopy()

	allocated := agonesv1.GameServerStateAllocated
	allocatedSelector.MatchLabels[lockLabel] = "true"
	gsa := &allocationv1.GameServerAllocation{
		Spec: allocationv1.GameServerAllocationSpec{
			MetaPatch: allocationv1.MetaPatch{Labels: map[string]string{lockLabel: "false"}},
			Selectors: []allocationv1.GameServerSelector{
				{LabelSelector: *allocatedSelector, GameServerState: &allocated},
				{LabelSelector: fleetSelector},
			},
		}}

	// standard allocation
	result, err := framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa, metav1.CreateOptions{})
	require.NoError(t, err)
	require.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(result.Status.State))

	gs, err := framework.AgonesClient.AgonesV1().GameServers(fleet.ObjectMeta.Namespace).Get(ctx, result.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, allocated, gs.Status.State)

	// set the label to being available again
	_, err = framework.SendGameServerUDP(t, gs, "LABEL available true")
	require.NoError(t, err)

	// wait for the label to be applied!
	require.Eventuallyf(t, func() bool {
		gs, err := framework.AgonesClient.AgonesV1().GameServers(fleet.ObjectMeta.Namespace).Get(ctx, result.Status.GameServerName, metav1.GetOptions{})
		require.NoError(t, err)
		log.WithField("labels", gs.ObjectMeta.Labels).Info("checking labels")
		return gs.ObjectMeta.Labels[lockLabel] == "true"
	}, time.Minute, time.Second, "GameServer did not unlock")

	// Run the same allocation again, we should get back the preferred item.
	expected := result.Status.GameServerName

	// we will run this as an Eventually, as caches are eventually consistent
	require.Eventuallyf(t, func() bool {
		result, err = framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa, metav1.CreateOptions{})
		require.NoError(t, err)
		require.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(result.Status.State))

		if expected != result.Status.GameServerName {
			log.WithField("expected", expected).WithField("gsa", result).Info("Re-allocation attempt failed. Retrying.")
			return false
		}

		return true
	}, time.Minute, time.Second, "Could not re-allocation")
}

func TestCreateFleetAndGameServerPlayerCapacityAllocation(t *testing.T) {
	if !(runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter)) {
		t.SkipNow()
	}
	t.Parallel()

	fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
	fleet := defaultFleet(framework.Namespace)
	fleet.Spec.Template.Spec.Players = &agonesv1.PlayersSpec{InitialCapacity: 10}
	ctx := context.Background()

	flt, err := fleets.Create(ctx, fleet, metav1.CreateOptions{})
	require.NoError(t, err)
	defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetSelector := metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}
	allocated := agonesv1.GameServerStateAllocated
	gsa := &allocationv1.GameServerAllocation{
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{
				{
					LabelSelector:   fleetSelector,
					GameServerState: &allocated,
					Players: &allocationv1.PlayerSelector{
						MinAvailable: 1,
						MaxAvailable: 99,
					},
				},
				{LabelSelector: fleetSelector, Players: &allocationv1.PlayerSelector{MinAvailable: 5, MaxAvailable: 10}},
			},
		}}

	// first try should give me a Ready->Allocated server
	gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(gsa.Status.State))

	gs1, err := framework.AgonesClient.AgonesV1().GameServers(fleet.ObjectMeta.Namespace).Get(ctx, gsa.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs1.Status.State)
	assert.NotNil(t, gs1.ObjectMeta.Annotations["agones.dev/last-allocated"])

	// second try should give me the same allocated server
	gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(gsa.Status.State))
	assert.Equal(t, gs1.ObjectMeta.Name, gsa.Status.GameServerName)

	gs2, err := framework.AgonesClient.AgonesV1().GameServers(fleet.ObjectMeta.Namespace).Get(ctx, gsa.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs2.Status.State)

	require.Equal(t, gs1.ObjectMeta.Name, gs2.ObjectMeta.Name)
	require.NotEqual(t, gs1.ObjectMeta.ResourceVersion, gs2.ObjectMeta.ResourceVersion)
	require.NotEqual(t, gs1.ObjectMeta.Annotations["agones.dev/last-allocated"], gs2.ObjectMeta.Annotations["agones.dev/last-allocated"])
}

func TestCounterAndListGameServerAllocation(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		t.SkipNow()
	}
	t.Parallel()
	ctx := context.Background()
	client := framework.AgonesClient.AgonesV1()

	flt := defaultFleet(framework.Namespace)
	initialCounters := map[string]agonesv1.CounterStatus{
		"games": {
			Count:    2,
			Capacity: 10,
		},
	}
	initialLists := map[string]agonesv1.ListStatus{
		"players": {
			Values:   []string{"player0"},
			Capacity: 10,
		},
	}
	flt.Spec.Template.Spec.Counters = initialCounters
	flt.Spec.Template.Spec.Lists = initialLists

	flt, err := client.Fleets(framework.Namespace).Create(ctx, flt.DeepCopy(), metav1.CreateOptions{})
	require.NoError(t, err)
	defer client.Fleets(framework.Namespace).Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	// Need fleetSelector to get the correct fleet, otherwise GSA will return game servers from any fleet in the namespace.
	fleetSelector := metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}
	stateAllocated := agonesv1.GameServerStateAllocated
	ready := agonesv1.GameServerStateReady
	allocated := allocationv1.GameServerAllocationAllocated
	unallocated := allocationv1.GameServerAllocationUnAllocated

	testCases := map[string]struct {
		gsa           allocationv1.GameServerAllocation
		wantGsaErr    bool                                   // For invalid GSA
		wantAllocated allocationv1.GameServerAllocationState // For a valid GSA: "allocated" if you expect the GSA to succed in allocating a GameServer, "unallocated" if not
		wantState     agonesv1.GameServerState
	}{
		"Counter Allocate to same GameServer MinAvailable (available capacity)": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{
							LabelSelector:   fleetSelector,
							GameServerState: &stateAllocated,
							Counters: map[string]allocationv1.CounterSelector{
								"games": {
									MinAvailable: 5,
								}}}, {
							LabelSelector:   fleetSelector,
							GameServerState: &ready,
							Counters: map[string]allocationv1.CounterSelector{
								"games": {
									MinAvailable: 5,
								}}}}}},
			wantGsaErr:    false,
			wantAllocated: allocated,
			wantState:     stateAllocated,
		},
		"List Allocate to same GameServer MinAvailable (available capacity)": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{
							LabelSelector:   fleetSelector,
							GameServerState: &stateAllocated,
							Lists: map[string]allocationv1.ListSelector{
								"players": {
									MinAvailable: 9,
								}}}, {
							LabelSelector:   fleetSelector,
							GameServerState: &ready,
							Lists: map[string]allocationv1.ListSelector{
								"players": {
									MinAvailable: 9,
								}}}}}},
			wantGsaErr:    false,
			wantAllocated: allocated,
			wantState:     stateAllocated,
		},
		"Counter Allocate to same GameServer MaxAvailable": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{
							LabelSelector:   fleetSelector,
							GameServerState: &stateAllocated,
							Counters: map[string]allocationv1.CounterSelector{
								"games": {
									MaxAvailable: 10,
								}}}, {
							LabelSelector:   fleetSelector,
							GameServerState: &ready,
							Counters: map[string]allocationv1.CounterSelector{
								"games": {
									MaxAvailable: 10,
								}}}}}},
			wantGsaErr:    false,
			wantAllocated: allocated,
			wantState:     stateAllocated,
		},
		"List Allocate to same GameServer MaxAvailable": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{
							LabelSelector:   fleetSelector,
							GameServerState: &stateAllocated,
							Lists: map[string]allocationv1.ListSelector{
								"players": {
									MaxAvailable: 9,
								}}}, {
							LabelSelector:   fleetSelector,
							GameServerState: &ready,
							Lists: map[string]allocationv1.ListSelector{
								"players": {
									MaxAvailable: 9,
								}}}}}},
			wantGsaErr:    false,
			wantAllocated: allocated,
			wantState:     stateAllocated,
		},
		"Allocate to same GameServer MinCount (count value)": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{
							LabelSelector:   fleetSelector,
							GameServerState: &stateAllocated,
							Counters: map[string]allocationv1.CounterSelector{
								"games": {
									MinCount: 2,
								}}}, {
							LabelSelector:   fleetSelector,
							GameServerState: &ready,
							Counters: map[string]allocationv1.CounterSelector{
								"games": {
									MinCount: 1,
								}}}}}},
			wantGsaErr:    false,
			wantAllocated: allocated,
			wantState:     stateAllocated,
		},
		"Allocate to same GameServer MaxCount (count value)": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{
							LabelSelector:   fleetSelector,
							GameServerState: &stateAllocated,
							Counters: map[string]allocationv1.CounterSelector{
								"games": {
									MaxCount: 3,
								}}}, {
							LabelSelector:   fleetSelector,
							GameServerState: &ready,
							Counters: map[string]allocationv1.CounterSelector{
								"games": {
									MaxCount: 2,
								}}}}}},
			wantGsaErr:    false,
			wantAllocated: allocated,
			wantState:     stateAllocated,
		},
		"List Allocate to same GameServer Contains Value": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{
							LabelSelector:   fleetSelector,
							GameServerState: &stateAllocated,
							Lists: map[string]allocationv1.ListSelector{
								"players": {
									ContainsValue: "player0",
								}}}, {
							LabelSelector:   fleetSelector,
							GameServerState: &ready,
							Lists: map[string]allocationv1.ListSelector{
								"players": {
									ContainsValue: "player0",
								}}}}}},
			wantGsaErr:    false,
			wantAllocated: allocated,
			wantState:     stateAllocated,
		},
		// 0 for MaxCount or MaxAvailable means unlimited maximum. Default for all fields: 0
		"Allocate to same GameServer no Counter values": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{
							LabelSelector:   fleetSelector,
							GameServerState: &stateAllocated,
							Counters: map[string]allocationv1.CounterSelector{
								"games": {}}}, {
							LabelSelector:   fleetSelector,
							GameServerState: &ready,
							Counters: map[string]allocationv1.CounterSelector{
								"games": {}}}}}},
			wantGsaErr:    false,
			wantAllocated: allocated,
			wantState:     stateAllocated,
		},
		"Allocate to same GameServer no List values": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{
							LabelSelector:   fleetSelector,
							GameServerState: &stateAllocated,
							Lists: map[string]allocationv1.ListSelector{
								"players": {}}}, {
							LabelSelector:   fleetSelector,
							GameServerState: &ready,
							Lists: map[string]allocationv1.ListSelector{
								"players": {}}}}}},
			wantGsaErr:    false,
			wantAllocated: allocated,
			wantState:     stateAllocated,
		},
		"Counter does not exist": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{{
						LabelSelector:   fleetSelector,
						GameServerState: &ready,
						Counters: map[string]allocationv1.CounterSelector{
							"players": {
								MinAvailable: 1,
							}}}}}},
			wantGsaErr:    false,
			wantAllocated: unallocated,
		},
		"List does not exist": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{{
						LabelSelector:   fleetSelector,
						GameServerState: &ready,
						Lists: map[string]allocationv1.ListSelector{
							"games": {
								MinAvailable: 1,
							}}}}}},
			wantGsaErr:    false,
			wantAllocated: unallocated,
		},
		"Counter MaxAvailable < MinAvailable": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{{
						LabelSelector:   fleetSelector,
						GameServerState: &ready,
						Counters: map[string]allocationv1.CounterSelector{
							"games": {
								MaxAvailable: 1,
								MinAvailable: 2,
							}}}}}},
			wantGsaErr: true,
		},
		"List MaxAvailable < MinAvailable": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{{
						LabelSelector:   fleetSelector,
						GameServerState: &ready,
						Lists: map[string]allocationv1.ListSelector{
							"players": {
								MaxAvailable: 8,
								MinAvailable: 9,
							}}}}}},
			wantGsaErr: true,
		},
		"Counter Maxcount < MinCount": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{{
						LabelSelector:   fleetSelector,
						GameServerState: &ready,
						Counters: map[string]allocationv1.CounterSelector{
							"games": {
								MaxCount: 1,
								MinCount: 2,
							}}}}}},
			wantGsaErr: true,
		},
		"List Value does not exist": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{{
						LabelSelector:   fleetSelector,
						GameServerState: &ready,
						Lists: map[string]allocationv1.ListSelector{
							"players": {
								ContainsValue: "player1",
							}}}}}},
			wantGsaErr:    false,
			wantAllocated: unallocated,
		},
		"Negative values for MinCount, MaxCount, MaxAvailable, MinAvailable": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{{
						LabelSelector:   fleetSelector,
						GameServerState: &ready,
						Counters: map[string]allocationv1.CounterSelector{
							"games": {
								MaxCount:     -1,
								MinCount:     -2,
								MaxAvailable: -10,
								MinAvailable: -1,
							}}}}}},
			wantGsaErr: true,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {

			// First allocation
			gsa, err := framework.AgonesClient.AllocationV1().GameServerAllocations(flt.ObjectMeta.Namespace).Create(ctx, testCase.gsa.DeepCopy(), metav1.CreateOptions{})
			if testCase.wantGsaErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, string(testCase.wantAllocated), string(gsa.Status.State))
			if gsa.Status.State == allocated {
				assert.Equal(t, initialCounters, gsa.Status.Counters)
				assert.Equal(t, initialLists, gsa.Status.Lists)
			}

			gs1, err := framework.AgonesClient.AgonesV1().GameServers(flt.ObjectMeta.Namespace).Get(ctx, gsa.Status.GameServerName, metav1.GetOptions{})
			if testCase.wantAllocated == unallocated {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.wantState, gs1.Status.State)
			assert.NotNil(t, gs1.ObjectMeta.Annotations["agones.dev/last-allocated"])

			// Second allocation
			gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(flt.ObjectMeta.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
			require.NoError(t, err)
			assert.Equal(t, string(testCase.wantAllocated), string(gsa.Status.State))
			assert.Equal(t, gs1.ObjectMeta.Name, gsa.Status.GameServerName)
			if gsa.Status.State == allocated {
				assert.Equal(t, initialCounters, gsa.Status.Counters)
				assert.Equal(t, initialLists, gsa.Status.Lists)
			}

			gs2, err := framework.AgonesClient.AgonesV1().GameServers(flt.ObjectMeta.Namespace).Get(ctx, gsa.Status.GameServerName, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, testCase.wantState, gs2.Status.State)

			// Confirm allocated to the same GameServer (gs2 == gs1)
			require.Equal(t, gs1.ObjectMeta.Name, gs2.ObjectMeta.Name)
			require.NotEqual(t, gs1.ObjectMeta.ResourceVersion, gs2.ObjectMeta.ResourceVersion)
			require.NotEqual(t, gs1.ObjectMeta.Annotations["agones.dev/last-allocated"], gs2.ObjectMeta.Annotations["agones.dev/last-allocated"])

			// Reset any GameServers in state Allocated -> Ready. Note: This does not reset any changes to Counters or Lists.
			list, err := framework.ListGameServersFromFleet(flt)
			require.NoError(t, err)
			for _, gs := range list {
				if gs.Status.State == ready {
					continue
				}
				gsCopy := gs.DeepCopy()
				gsCopy.Status.State = ready
				reqReadyGs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
				require.NoError(t, err)
				require.Equal(t, ready, reqReadyGs.Status.State)
			}
		})
	}
}

func TestCounterGameServerAllocationActions(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		t.SkipNow()
	}
	t.Parallel()
	ctx := context.Background()
	client := framework.AgonesClient.AgonesV1()

	counters := map[string]agonesv1.CounterStatus{}
	counters["games"] = agonesv1.CounterStatus{
		Count:    5,
		Capacity: 10,
	}

	flt := defaultFleet(framework.Namespace)
	flt.Spec.Template.Spec.Counters = counters

	flt, err := client.Fleets(framework.Namespace).Create(ctx, flt.DeepCopy(), metav1.CreateOptions{})
	require.NoError(t, err)
	defer client.Fleets(framework.Namespace).Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetSelector := metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}
	allocated := agonesv1.GameServerStateAllocated
	ready := agonesv1.GameServerStateReady
	increment := agonesv1.GameServerPriorityIncrement
	decrement := agonesv1.GameServerPriorityDecrement
	zero := int64(0)
	one := int64(1)
	five := int64(5)
	six := int64(6)
	ten := int64(10)
	negativeOne := int64(-1)

	testCases := map[string]struct {
		gsa          allocationv1.GameServerAllocation
		wantGsaErr   bool
		wantCount    *int64
		wantCapacity *int64
	}{
		"increment": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{LabelSelector: fleetSelector},
					},
					Counters: map[string]allocationv1.CounterAction{
						"games": {
							Action: &increment,
							Amount: &one,
						}}}},
			wantGsaErr:   false,
			wantCount:    &six,
			wantCapacity: &ten,
		},
		"decrement": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{LabelSelector: fleetSelector},
					},
					Counters: map[string]allocationv1.CounterAction{
						"games": {
							Action: &decrement,
							Amount: &five,
						}}}},
			wantGsaErr:   false,
			wantCount:    &zero,
			wantCapacity: &ten,
		},
		"change capacity to less than count also updates count": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{LabelSelector: fleetSelector},
					},
					Counters: map[string]allocationv1.CounterAction{
						"games": {
							Capacity: &zero,
						}}}},
			wantGsaErr:   false,
			wantCount:    &zero,
			wantCapacity: &zero,
		},
		"decrement past zero truncated": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{LabelSelector: fleetSelector},
					},
					Counters: map[string]allocationv1.CounterAction{
						"games": {
							Action: &decrement,
							Amount: &six,
						}}}},
			wantGsaErr:   false,
			wantCount:    &zero,
			wantCapacity: &ten,
		},
		"decrement negative": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{LabelSelector: fleetSelector},
					},
					Counters: map[string]allocationv1.CounterAction{
						"games": {
							Action: &decrement,
							Amount: &negativeOne,
						}}}},
			wantGsaErr: true,
		},
		"increment past capacity truncated": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{LabelSelector: fleetSelector},
					},
					Counters: map[string]allocationv1.CounterAction{
						"games": {
							Action: &increment,
							Amount: &six,
						}}}},
			wantGsaErr:   false,
			wantCount:    &ten,
			wantCapacity: &ten,
		},
		"increment negative": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{LabelSelector: fleetSelector},
					},
					Counters: map[string]allocationv1.CounterAction{
						"games": {
							Action: &increment,
							Amount: &negativeOne,
						}}}},
			wantGsaErr: true,
		},
		"change capacity negative": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{LabelSelector: fleetSelector},
					},
					Counters: map[string]allocationv1.CounterAction{
						"games": {
							Capacity: &negativeOne,
						}}}},
			wantGsaErr: true,
		},
		// Note: a gameserver is still allocated even though the counter does not exist (and thus the
		// action cannot be performed). gsa.Validate() is not able to see the state of Counters in the
		// fleet, so the GSA is not able to validate the existence of a Counter. Use the
		// GameServerSelector to filter the Counters.
		"Counter does not exist": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{LabelSelector: fleetSelector},
					},
					Counters: map[string]allocationv1.CounterAction{
						"lames": {
							Action: &increment,
							Amount: &one,
						}}}},
			wantGsaErr: false,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {

			gsa, err := framework.AgonesClient.AllocationV1().GameServerAllocations(flt.ObjectMeta.Namespace).Create(ctx, testCase.gsa.DeepCopy(), metav1.CreateOptions{})
			if testCase.wantGsaErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, string(allocated), string(gsa.Status.State))
			counterStatus, ok := gsa.Status.Counters["games"]
			assert.True(t, ok)
			if testCase.wantCount != nil {
				assert.Equal(t, *testCase.wantCount, counterStatus.Count)
			}
			if testCase.wantCapacity != nil {
				assert.Equal(t, *testCase.wantCapacity, counterStatus.Capacity)
			}

			gs1, err := framework.AgonesClient.AgonesV1().GameServers(flt.ObjectMeta.Namespace).Get(ctx, gsa.Status.GameServerName, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, allocated, gs1.Status.State)
			assert.NotNil(t, gs1.ObjectMeta.Annotations["agones.dev/last-allocated"])

			counter, ok := gs1.Status.Counters["games"]
			assert.True(t, ok)
			if testCase.wantCount != nil {
				assert.Equal(t, *testCase.wantCount, counter.Count)
			}
			if testCase.wantCapacity != nil {
				assert.Equal(t, *testCase.wantCapacity, counter.Capacity)
			}

			// Reset any GameServers in state Allocated -> Ready, and reset any changes to Counters.
			gsList, err := framework.ListGameServersFromFleet(flt)
			require.NoError(t, err)
			for _, gs := range gsList {
				if gs.Status.State == ready && cmp.Equal(gs.Status.Counters, counters) {
					continue
				}
				gsCopy := gs.DeepCopy()
				gsCopy.Status.State = ready
				gsCopy.Status.Counters = counters
				reqReadyGs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
				require.NoError(t, err)
				require.Equal(t, ready, reqReadyGs.Status.State)
			}
		})
	}
}

func TestListGameServerAllocationActions(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		t.SkipNow()
	}
	t.Parallel()
	ctx := context.Background()
	client := framework.AgonesClient.AgonesV1()

	lists := map[string]agonesv1.ListStatus{}
	lists["players"] = agonesv1.ListStatus{
		Values:   []string{"player0", "player1", "player2"},
		Capacity: 8,
	}
	lists["rooms"] = agonesv1.ListStatus{
		Values:   []string{"room0", "room1", "room2"},
		Capacity: 20,
	}

	flt := defaultFleet(framework.Namespace)
	flt.Spec.Template.Spec.Lists = lists

	flt, err := client.Fleets(framework.Namespace).Create(ctx, flt.DeepCopy(), metav1.CreateOptions{})
	require.NoError(t, err)
	defer client.Fleets(framework.Namespace).Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	fleetSelector := metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}
	allocated := agonesv1.GameServerStateAllocated
	ready := agonesv1.GameServerStateReady
	one := int64(1)

	testCases := map[string]struct {
		gsa          allocationv1.GameServerAllocation
		listName     string
		wantGsaErr   bool
		wantCapacity *int64
		wantValues   []string
	}{
		"add List values": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{LabelSelector: fleetSelector},
					},
					Lists: map[string]allocationv1.ListAction{
						"players": {
							AddValues: []string{"player3", "player4", "player5"},
						}}}},
			listName:   "players",
			wantGsaErr: false,
			wantValues: []string{"player0", "player1", "player2", "player3", "player4", "player5"},
		},
		"change List capacity truncates": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{LabelSelector: fleetSelector},
					},
					Lists: map[string]allocationv1.ListAction{
						"players": {
							Capacity: &one,
						}}}},
			listName:     "players",
			wantGsaErr:   false,
			wantValues:   []string{"player0"},
			wantCapacity: &one,
		},
		"delete List values": {
			gsa: allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Selectors: []allocationv1.GameServerSelector{
						{LabelSelector: fleetSelector},
					},
					Lists: map[string]allocationv1.ListAction{
						"rooms": {
							DeleteValues: []string{"room1", "room0"},
						}}}},
			listName:   "rooms",
			wantGsaErr: false,
			wantValues: []string{"room2"},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {

			gsa, err := framework.AgonesClient.AllocationV1().GameServerAllocations(flt.ObjectMeta.Namespace).Create(ctx, testCase.gsa.DeepCopy(), metav1.CreateOptions{})
			if testCase.wantGsaErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, string(allocated), string(gsa.Status.State))
			listStatus, ok := gsa.Status.Lists[testCase.listName]
			assert.True(t, ok)
			if testCase.wantCapacity != nil {
				assert.Equal(t, *testCase.wantCapacity, listStatus.Capacity)
			}
			assert.Equal(t, testCase.wantValues, listStatus.Values)

			gs1, err := framework.AgonesClient.AgonesV1().GameServers(flt.ObjectMeta.Namespace).Get(ctx, gsa.Status.GameServerName, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, allocated, gs1.Status.State)
			assert.NotNil(t, gs1.ObjectMeta.Annotations["agones.dev/last-allocated"])

			list, ok := gs1.Status.Lists[testCase.listName]
			assert.True(t, ok)
			if testCase.wantCapacity != nil {
				assert.Equal(t, *testCase.wantCapacity, list.Capacity)
			}
			assert.Equal(t, testCase.wantValues, list.Values)

			// Reset any GameServers in state Allocated -> Ready, and reset any changes to Lists.
			gsList, err := framework.ListGameServersFromFleet(flt)
			require.NoError(t, err)
			for _, gs := range gsList {
				if gs.Status.State == ready && cmp.Equal(gs.Status.Lists, lists) {
					continue
				}
				gsCopy := gs.DeepCopy()
				gsCopy.Status.State = ready
				gsCopy.Status.Lists = lists
				reqReadyGs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Update(ctx, gsCopy, metav1.UpdateOptions{})
				require.NoError(t, err)
				require.Equal(t, ready, reqReadyGs.Status.State)
			}
		})
	}
}

func TestMultiClusterAllocationOnLocalCluster(t *testing.T) {
	t.Parallel()

	fixtures := []apis.SchedulingStrategy{apis.Packed, apis.Distributed}
	for _, strategy := range fixtures {
		t.Run(string(strategy), func(t *testing.T) {
			if strategy == apis.Distributed {
				framework.SkipOnCloudProduct(t, "gke-autopilot", "Autopilot does not support Distributed scheduling")
			}
			t.Parallel()
			ctx := context.Background()

			namespace := fmt.Sprintf("gsa-multicluster-local-%s", uuid.NewUUID())
			err := framework.CreateNamespace(namespace)
			if !assert.Nil(t, err) {
				return
			}
			defer func() {
				if derr := framework.DeleteNamespace(namespace); derr != nil {
					t.Error(derr)
				}
			}()

			fleets := framework.AgonesClient.AgonesV1().Fleets(namespace)
			fleet := defaultFleet(namespace)
			fleet.Spec.Scheduling = strategy
			flt, err := fleets.Create(ctx, fleet, metav1.CreateOptions{})
			if assert.Nil(t, err) {
				defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
			}

			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

			// Allocation Policy #1: local cluster with desired label.
			// This policy allocates locally on the cluster due to matching namespace with gsa and not setting AllocationEndpoints.
			mca := &multiclusterv1.GameServerAllocationPolicy{
				Spec: multiclusterv1.GameServerAllocationPolicySpec{
					Priority: 1,
					Weight:   100,
					ConnectionInfo: multiclusterv1.ClusterConnectionInfo{
						ClusterName: "multicluster1",
						SecretName:  "sec1",
						Namespace:   namespace,
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels:       map[string]string{"cluster": "onprem"},
					GenerateName: "allocationpolicy-",
				},
			}
			resp, err := framework.AgonesClient.MulticlusterV1().GameServerAllocationPolicies(fleet.ObjectMeta.Namespace).Create(ctx, mca, metav1.CreateOptions{})
			if !assert.Nil(t, err) {
				assert.FailNowf(t, "GameServerAllocationPolicies(%v).Create(ctx, %v, metav1.CreateOptions{})", fleet.ObjectMeta.Namespace, mca)
			}
			assert.Equal(t, mca.Spec, resp.Spec)

			// Allocation Policy #2: another cluster with desired label, but lower priority.
			// If the policy is selected due to a bug the request fails as it cannot find the secret.
			mca = &multiclusterv1.GameServerAllocationPolicy{
				Spec: multiclusterv1.GameServerAllocationPolicySpec{
					Priority: 2,
					Weight:   100,
					ConnectionInfo: multiclusterv1.ClusterConnectionInfo{
						AllocationEndpoints: []string{"another-endpoint"},
						ClusterName:         "multicluster2",
						SecretName:          "sec2",
						Namespace:           namespace,
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels:       map[string]string{"cluster": "onprem"},
					GenerateName: "allocationpolicy-",
				},
			}
			resp, err = framework.AgonesClient.MulticlusterV1().GameServerAllocationPolicies(fleet.ObjectMeta.Namespace).Create(ctx, mca, metav1.CreateOptions{})
			if assert.Nil(t, err) {
				assert.Equal(t, mca.Spec, resp.Spec)
			}

			// Allocation Policy #3: another cluster with highest priority, but missing desired label (will not be selected)
			mca = &multiclusterv1.GameServerAllocationPolicy{
				Spec: multiclusterv1.GameServerAllocationPolicySpec{
					Priority: 1,
					Weight:   10,
					ConnectionInfo: multiclusterv1.ClusterConnectionInfo{
						AllocationEndpoints: []string{"another-endpoint"},
						ClusterName:         "multicluster3",
						SecretName:          "sec3",
						Namespace:           namespace,
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "allocationpolicy-",
				},
			}
			resp, err = framework.AgonesClient.MulticlusterV1().GameServerAllocationPolicies(fleet.ObjectMeta.Namespace).Create(ctx, mca, metav1.CreateOptions{})
			if assert.Nil(t, err) {
				assert.Equal(t, mca.Spec, resp.Spec)
			}

			gsa := &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Scheduling: strategy,
					Selectors:  []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}}},
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: true,
						PolicySelector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"cluster": "onprem",
							},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "allocation-",
					Namespace:    namespace,
				},
			}

			// wait for the allocation policies to be added.
			err = wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
				gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(fleet.ObjectMeta.Namespace).Create(ctx, gsa, metav1.CreateOptions{})
				if err != nil {
					t.Logf("GameServerAllocations(%v).Create(ctx, %v, metav1.CreateOptions{}) failed: %s", fleet.ObjectMeta.Namespace, gsa, err)
					return false, nil
				}

				assert.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(gsa.Status.State))
				return true, nil
			})

			assert.NoError(t, err)
		})
	}
}

// Can't allocate more GameServers if a fleet is fully used.
func TestCreateFullFleetAndCantGameServerAllocate(t *testing.T) {
	t.Parallel()

	fixtures := []apis.SchedulingStrategy{apis.Packed, apis.Distributed}

	for _, strategy := range fixtures {
		t.Run(string(strategy), func(t *testing.T) {
			if strategy == apis.Distributed {
				framework.SkipOnCloudProduct(t, "gke-autopilot", "Autopilot does not support Distributed scheduling")
			}
			t.Parallel()
			ctx := context.Background()

			fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
			fleet := defaultFleet(framework.Namespace)
			fleet.Spec.Scheduling = strategy
			flt, err := fleets.Create(ctx, fleet, metav1.CreateOptions{})
			if assert.Nil(t, err) {
				defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
			}

			framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

			gsa := &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Scheduling: strategy,
					Selectors:  []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}}},
				}}

			for i := 0; i < replicasCount; i++ {
				var gsa2 *allocationv1.GameServerAllocation
				gsa2, err = framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
				if assert.Nil(t, err) {
					assert.Equal(t, allocationv1.GameServerAllocationAllocated, gsa2.Status.State)
				}
			}

			framework.AssertFleetCondition(t, flt, func(_ *logrus.Entry, fleet *agonesv1.Fleet) bool {
				return fleet.Status.AllocatedReplicas == replicasCount
			})

			gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
			if assert.Nil(t, err) {
				assert.Equal(t, string(allocationv1.GameServerAllocationUnAllocated), string(gsa.Status.State))
			}
		})
	}
}

func TestGameServerAllocationMetaDataPatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	log := e2e.TestLogger(t)
	createAndAllocate := func(input *allocationv1.GameServerAllocation) *allocationv1.GameServerAllocation {
		gs := framework.DefaultGameServer(framework.Namespace)
		gs.ObjectMeta.Labels = map[string]string{"test": t.Name()}
		gs, err := framework.CreateGameServerAndWaitUntilReady(t, framework.Namespace, gs)
		require.NoError(t, err)

		log.WithField("gs", gs.ObjectMeta.Name).Info("👍 created and ready")

		// poll, as it may take a moment for the allocation cache to be populated
		err = wait.PollUntilContextTimeout(context.Background(), time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
			input, err = framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, input, metav1.CreateOptions{})
			if err != nil {
				log.WithError(err).Info("Failed, trying again...")
				return false, err
			}

			return allocationv1.GameServerAllocationAllocated == input.Status.State, nil
		})
		require.NoError(t, err)
		return input
	}

	// two standard labels
	gsa := &allocationv1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"test": t.Name()}}}},
			MetaPatch: allocationv1.MetaPatch{
				Labels:      map[string]string{"red": "blue"},
				Annotations: map[string]string{"dog": "good"},
			},
		}}
	result := createAndAllocate(gsa)
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, result.Status.GameServerName, metav1.DeleteOptions{}) // nolint: errcheck

	gs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, result.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "blue", gs.ObjectMeta.Labels["red"])
	assert.Equal(t, "good", gs.ObjectMeta.Annotations["dog"])

	// use special characters that are valid
	gsa.Spec.MetaPatch = allocationv1.MetaPatch{Labels: map[string]string{"blue-frog.fred_thing": "test"}}
	result = createAndAllocate(gsa)
	defer framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Delete(ctx, result.Status.GameServerName, metav1.DeleteOptions{}) // nolint: errcheck

	gs, err = framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, result.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "test", gs.ObjectMeta.Labels["blue-frog.fred_thing"])

	// throw something invalid at it.
	gsa.Spec.MetaPatch = allocationv1.MetaPatch{Labels: map[string]string{"$$$$$$$": "test"}}
	result, err = framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	log.WithField("result", result).WithError(err).Info("Failed allocation")
	require.Error(t, err)
	require.Contains(t, err.Error(), `GameServerAllocation.allocation.agones.dev "" is invalid`)
}

func TestGameServerAllocationPreferredSelection(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
	gameServers := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	label := map[string]string{"role": t.Name()}

	preferred := defaultFleet(framework.Namespace)
	preferred.ObjectMeta.GenerateName = "preferred-"
	preferred.Spec.Replicas = 1
	preferred.Spec.Template.ObjectMeta.Labels = label
	preferred, err := fleets.Create(ctx, preferred, metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleets.Delete(ctx, preferred.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		assert.FailNow(t, "could not create first fleet")
	}

	required := defaultFleet(framework.Namespace)
	required.ObjectMeta.GenerateName = "required-"
	required.Spec.Replicas = 2
	required.Spec.Template.ObjectMeta.Labels = label
	required, err = fleets.Create(ctx, required, metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleets.Delete(ctx, required.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		assert.FailNow(t, "could not create second fleet")
	}

	framework.AssertFleetCondition(t, preferred, e2e.FleetReadyCount(preferred.Spec.Replicas))
	framework.AssertFleetCondition(t, required, e2e.FleetReadyCount(required.Spec.Replicas))

	gsa := &allocationv1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{
				{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: preferred.ObjectMeta.Name}}},
				{LabelSelector: metav1.LabelSelector{MatchLabels: label}},
			},
		}}

	gsa1, err := framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		assert.Equal(t, allocationv1.GameServerAllocationAllocated, gsa1.Status.State)
		gs, err := gameServers.Get(ctx, gsa1.Status.GameServerName, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, preferred.ObjectMeta.Name, gs.ObjectMeta.Labels[agonesv1.FleetNameLabel])
	} else {
		assert.FailNow(t, "could not completed gsa1 allocation")
	}

	gs2, err := framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		assert.Equal(t, allocationv1.GameServerAllocationAllocated, gs2.Status.State)
		gs, err := gameServers.Get(ctx, gs2.Status.GameServerName, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, required.ObjectMeta.Name, gs.ObjectMeta.Labels[agonesv1.FleetNameLabel])
	} else {
		assert.FailNow(t, "could not completed gs2 allocation")
	}

	// delete the preferred gameserver, and then let's try allocating again, make sure it goes back to the
	// preferred one
	err = gameServers.Delete(ctx, gsa1.Status.GameServerName, metav1.DeleteOptions{})
	if !assert.Nil(t, err) {
		assert.FailNow(t, "could not delete gameserver")
	}

	// wait until the game server is deleted
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		_, err = gameServers.Get(ctx, gsa1.Status.GameServerName, metav1.GetOptions{})

		if err != nil && errors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})
	assert.Nil(t, err)

	// now wait for another one to come along
	framework.AssertFleetCondition(t, preferred, e2e.FleetReadyCount(preferred.Spec.Replicas))

	gsa3, err := framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		assert.Equal(t, allocationv1.GameServerAllocationAllocated, gsa3.Status.State)
		gs, err := gameServers.Get(ctx, gsa3.Status.GameServerName, metav1.GetOptions{})
		assert.Nil(t, err)
		assert.Equal(t, preferred.ObjectMeta.Name, gs.ObjectMeta.Labels[agonesv1.FleetNameLabel])
	}
}

func TestGameServerAllocationReturnLabels(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
	gameServers := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace)
	role := "role"
	label := map[string]string{role: t.Name()}
	annotationKey := "someAnnotation"
	annotationValue := "someValue"
	annotations := map[string]string{annotationKey: annotationValue}

	flt := defaultFleet(framework.Namespace)
	flt.Spec.Replicas = 1
	flt.Spec.Template.ObjectMeta.Labels = label
	flt.Spec.Template.ObjectMeta.Annotations = annotations
	flt, err := fleets.Create(ctx, flt, metav1.CreateOptions{})
	defer fleets.Delete(ctx, flt.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	require.NoError(t, err)

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	gsa := &allocationv1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{
				{LabelSelector: metav1.LabelSelector{MatchLabels: label}},
			},
		}}

	gsa, err = framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	require.NoError(t, err)

	assert.Equal(t, allocationv1.GameServerAllocationAllocated, gsa.Status.State)
	assert.Equal(t, t.Name(), gsa.Status.Metadata.Labels[role])
	assert.Equal(t, flt.ObjectMeta.Name, gsa.Status.Metadata.Labels[agonesv1.FleetNameLabel])
	assert.Equal(t, annotationValue, gsa.Status.Metadata.Annotations[annotationKey])
	gs, err := gameServers.Get(ctx, gsa.Status.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, flt.ObjectMeta.Name, gs.ObjectMeta.Labels[agonesv1.FleetNameLabel])
}

func TestGameServerAllocationDeletionOnUnAllocate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	allocations := framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace)

	gsa := &allocationv1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"never": "goingtohappen"}}}},
		}}

	gsa, err := allocations.Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
	if assert.Nil(t, err) {
		assert.Equal(t, allocationv1.GameServerAllocationUnAllocated, gsa.Status.State)
	}
}

func TestGameServerAllocationDuringMultipleAllocationClients(t *testing.T) {
	t.Parallel()
	log := e2e.TestLogger(t)
	ctx := context.Background()

	fleets := framework.AgonesClient.AgonesV1().Fleets(framework.Namespace)
	label := map[string]string{"role": t.Name()}

	preferred := defaultFleet(framework.Namespace)
	preferred.ObjectMeta.GenerateName = "preferred-"
	preferred.Spec.Replicas = 150
	preferred.Spec.Template.ObjectMeta.Labels = label
	preferred, err := fleets.Create(ctx, preferred, metav1.CreateOptions{})
	if assert.Nil(t, err) {
		defer fleets.Delete(ctx, preferred.ObjectMeta.Name, metav1.DeleteOptions{}) // nolint:errcheck
	} else {
		assert.FailNow(t, "could not create first fleet")
	}

	framework.AssertFleetCondition(t, preferred, e2e.FleetReadyCount(preferred.Spec.Replicas))

	// scale down before starting allocation
	preferred = scaleFleetPatch(ctx, t, preferred, preferred.Spec.Replicas-20)

	gsa := &allocationv1.GameServerAllocation{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"},
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{
				{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: "preferred"}}},
				{LabelSelector: metav1.LabelSelector{MatchLabels: label}},
			},
		}}

	allocatedGS := sync.Map{}

	log.Info("Starting 100 allocation attempts")
	var wg sync.WaitGroup

	// Allocate GS by 10 clients in parallel while the fleet is scaling down
	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				gsa1, err := framework.AgonesClient.AllocationV1().GameServerAllocations(framework.Namespace).Create(ctx, gsa.DeepCopy(), metav1.CreateOptions{})
				if err == nil {
					allocatedGS.LoadOrStore(gsa1.Status.GameServerName, true)
				} else {
					log.Infof("Allocation error: %v", err)
				}
			}
		}()
	}

	// scale down further while allocating
	time.Sleep(1 * time.Second)
	log.Infof("Scaling Fleet down by 10 replicas")
	scaleFleetPatch(ctx, t, preferred, preferred.Spec.Replicas-10)

	wg.Wait()
	log.Infof("Finished allocation attempts")

	// count the number of unique game servers allocated
	// there should not be any duplicate
	uniqueAllocatedGSs := 0
	allocatedGS.Range(func(_, _ interface{}) bool {
		uniqueAllocatedGSs++
		return true
	})

	// TODO: Compromising on the expected allocation count to be between 98 to 100 due to a known allocation issue. Please check: [https://github.com/agones-dev/agones/issues/3553]
	switch {
	case uniqueAllocatedGSs < 98:
		t.Errorf("Test failed: Less than 98 GameServers were allocated. Allocated: %d", uniqueAllocatedGSs)
	case uniqueAllocatedGSs < 100:
		t.Logf("Number of GameServers Allocated: %d. This might be due to a known allocation issue. Please check: [https://github.com/agones-dev/agones/issues/3553]", uniqueAllocatedGSs)
	default:
		t.Logf("Number of GameServers allocated: %d. This matches the expected outcome.", uniqueAllocatedGSs)
	}
}
