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
	"sort"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/gameservers"
	agtesting "agones.dev/agones/pkg/testing"
	utilruntime "agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestSortGameServersByPackedStrategy(t *testing.T) {
	t.Parallel()

	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	require.NoError(t, utilruntime.ParseFeatures(string(utilruntime.FeatureCountsAndLists)+"=true"))

	nc := map[string]gameservers.NodeCount{
		"n1": {Ready: 1, Allocated: 0},
		"n2": {Ready: 0, Allocated: 2},
		"n3": {Ready: 2, Allocated: 2},
		"n4": {Ready: 2, Allocated: 2},
	}

	list := []*agonesv1.GameServer{
		{ObjectMeta: metav1.ObjectMeta{Name: "g1"}, Status: agonesv1.GameServerStatus{NodeName: "n2", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g2"}, Status: agonesv1.GameServerStatus{NodeName: "", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g3"}, Status: agonesv1.GameServerStatus{NodeName: "n1", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g4"}, Status: agonesv1.GameServerStatus{NodeName: "n2", State: agonesv1.GameServerStateCreating}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g5"}, Status: agonesv1.GameServerStatus{NodeName: "n3", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g6"}, Status: agonesv1.GameServerStatus{NodeName: "n4", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g7"}, Status: agonesv1.GameServerStatus{NodeName: "n3", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g8"}, Status: agonesv1.GameServerStatus{NodeName: "n4", State: agonesv1.GameServerStateReady}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g9"}, Status: agonesv1.GameServerStatus{
			NodeName: "n3",
			State:    agonesv1.GameServerStateAllocated,
			Counters: map[string]agonesv1.CounterStatus{
				"foo": {
					Count:    0,
					Capacity: 100,
				}}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g10"}, Status: agonesv1.GameServerStatus{
			NodeName: "n3",
			State:    agonesv1.GameServerStateAllocated,
			Counters: map[string]agonesv1.CounterStatus{
				"foo": {
					Count:    99,
					Capacity: 100,
				}}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g11"}, Status: agonesv1.GameServerStatus{
			NodeName: "n4",
			State:    agonesv1.GameServerStateAllocated,
			Counters: map[string]agonesv1.CounterStatus{
				"foo": {
					Count:    0,
					Capacity: 90,
				}}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "g12"}, Status: agonesv1.GameServerStatus{
			NodeName: "n4",
			State:    agonesv1.GameServerStateAllocated,
			Counters: map[string]agonesv1.CounterStatus{
				"foo": {
					Count:    89,
					Capacity: 90,
				}}}},
	}

	priorities := []agonesv1.Priority{{
		Type:  "Counter",
		Key:   "foo",
		Order: "Descending",
	}}

	result := sortGameServersByPackedStrategy(list, nc, priorities)

	require.Len(t, result, len(list))
	assert.Equal(t, "g2", result[0].ObjectMeta.Name)
	assert.Equal(t, "g3", result[1].ObjectMeta.Name)
	assert.Equal(t, "g4", result[2].ObjectMeta.Name)
	assert.Equal(t, "g1", result[3].ObjectMeta.Name)
	assert.Equal(t, "g9", result[4].ObjectMeta.Name)
	assert.Equal(t, "g10", result[5].ObjectMeta.Name)
	assert.Equal(t, "g5", result[6].ObjectMeta.Name)
	assert.Equal(t, "g7", result[7].ObjectMeta.Name)
	assert.Equal(t, "g11", result[8].ObjectMeta.Name)
	assert.Equal(t, "g12", result[9].ObjectMeta.Name)
	assert.Equal(t, "g6", result[10].ObjectMeta.Name)
	assert.Equal(t, "g8", result[11].ObjectMeta.Name)

	priorities = []agonesv1.Priority{{
		Type:  "Counter",
		Key:   "foo",
		Order: "Ascending",
	}}

	result = sortGameServersByPackedStrategy(list, nc, priorities)

	require.Len(t, result, len(list))
	assert.Equal(t, "g2", result[0].ObjectMeta.Name)
	assert.Equal(t, "g3", result[1].ObjectMeta.Name)
	assert.Equal(t, "g4", result[2].ObjectMeta.Name)
	assert.Equal(t, "g1", result[3].ObjectMeta.Name)
	assert.Equal(t, "g10", result[4].ObjectMeta.Name)
	assert.Equal(t, "g9", result[5].ObjectMeta.Name)
	assert.Equal(t, "g5", result[6].ObjectMeta.Name)
	assert.Equal(t, "g7", result[7].ObjectMeta.Name)
	assert.Equal(t, "g12", result[8].ObjectMeta.Name)
	assert.Equal(t, "g11", result[9].ObjectMeta.Name)
	assert.Equal(t, "g6", result[10].ObjectMeta.Name)
	assert.Equal(t, "g8", result[11].ObjectMeta.Name)
}

func TestSortGameServersByDistributedStrategy(t *testing.T) {
	t.Parallel()

	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()

	require.NoError(t, utilruntime.ParseFeatures(string(utilruntime.FeatureCountsAndLists)+"=true"))

	now := metav1.Now()

	gs1 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "g1", CreationTimestamp: metav1.Time{Time: now.Add(10 * time.Second)}}}
	gs2 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "g2", CreationTimestamp: now}}
	gs3 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "g3", CreationTimestamp: metav1.Time{Time: now.Add(30 * time.Second)}}}
	gs4 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "g4", CreationTimestamp: metav1.Time{Time: now.Add(30 * time.Second)}},
		Status: agonesv1.GameServerStatus{
			Counters: map[string]agonesv1.CounterStatus{
				"bar": {
					Count:    0,
					Capacity: 100,
				}}}}
	gs5 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "g5", CreationTimestamp: now},
		Status: agonesv1.GameServerStatus{
			Counters: map[string]agonesv1.CounterStatus{
				"bar": {
					Count:    0,
					Capacity: 100,
				}}}}
	gs6 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "g6", CreationTimestamp: now},
		Status: agonesv1.GameServerStatus{
			Counters: map[string]agonesv1.CounterStatus{
				"bar": {
					Count:    0,
					Capacity: 1000,
				}}}}

	testScenarios := map[string]struct {
		list       []*agonesv1.GameServer
		priorities []agonesv1.Priority
		want       []*agonesv1.GameServer
	}{
		"No priorities, sort by creation time": {
			list:       []*agonesv1.GameServer{&gs1, &gs2, &gs3},
			priorities: nil,
			want:       []*agonesv1.GameServer{&gs2, &gs1, &gs3},
		},
		"Descending priorities": {
			list: []*agonesv1.GameServer{&gs4, &gs6, &gs5},
			priorities: []agonesv1.Priority{{
				Type:  "Counter",
				Key:   "bar",
				Order: "Descending",
			}},
			want: []*agonesv1.GameServer{&gs6, &gs5, &gs4},
		},
		"Ascending priorities": {
			list: []*agonesv1.GameServer{&gs4, &gs5, &gs6},
			priorities: []agonesv1.Priority{{
				Type:  "Counter",
				Key:   "bar",
				Order: "Ascending",
			}},
			want: []*agonesv1.GameServer{&gs5, &gs4, &gs6},
		},
	}

	for testName, testScenario := range testScenarios {
		t.Run(testName, func(t *testing.T) {

			result := sortGameServersByDistributedStrategy(testScenario.list, testScenario.priorities)
			assert.Equal(t, testScenario.want, result)
		})
	}
}

func TestListGameServersByGameServerSetOwner(t *testing.T) {
	t.Parallel()

	gsSet := &agonesv1.GameServerSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test", UID: "1234"},
		Spec: agonesv1.GameServerSetSpec{
			Replicas: 10,
			Template: agonesv1.GameServerTemplateSpec{},
		},
	}

	gs1 := gsSet.GameServer()
	gs1.ObjectMeta.Name = "test-1"
	gs2 := gsSet.GameServer()
	assert.True(t, metav1.IsControlledBy(gs2, gsSet))

	gs2.ObjectMeta.Name = "test-2"
	gs3 := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "not-included"}}
	gs4 := gsSet.GameServer()
	gs4.ObjectMeta.OwnerReferences = nil

	m := agtesting.NewMocks()
	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs1, *gs2, gs3, *gs4}}, nil
	})

	gameServers := m.AgonesInformerFactory.Agones().V1().GameServers()
	_, cancel := agtesting.StartInformers(m, gameServers.Informer().HasSynced)
	defer cancel()

	list, err := ListGameServersByGameServerSetOwner(gameServers.Lister(), gsSet)
	require.NoError(t, err)

	// sort of stable ordering
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].ObjectMeta.Name < list[j].ObjectMeta.Name
	})
	assert.Equal(t, []*agonesv1.GameServer{gs1, gs2}, list)
}
