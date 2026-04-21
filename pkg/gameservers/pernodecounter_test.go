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

package gameservers

import (
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
)

const (
	defaultNs = "default"
	name1     = "node1"
	name2     = "node2"
)

func TestPerNodeCounterGameServerEvents(t *testing.T) {
	t.Parallel()

	pnc, m := newFakePerNodeCounter()

	fakeWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	_, cancel := agtesting.StartInformers(m)
	defer cancel()

	assert.Empty(t, pnc.Counts())

	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, UID: "uid-gs1", ResourceVersion: "1"},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateStarting, NodeName: name1,
		},
	}

	fakeWatch.Add(gs.DeepCopy())
	require.Eventuallyf(t, func() bool {
		return len(pnc.Counts()) == 0
	}, 5*time.Second, time.Second, "Should be empty, instead has %v elements", len(pnc.Counts()))

	gs.Status.State = agonesv1.GameServerStateReady
	gs.ObjectMeta.ResourceVersion = "2"
	fakeWatch.Add(gs.DeepCopy())

	var counts map[string]NodeCount
	require.Eventuallyf(t, func() bool {
		counts = pnc.Counts()
		return len(counts) == 1
	}, 5*time.Second, time.Second, "len should be 1, instead: %v", len(counts))
	assert.Equal(t, int64(1), counts[name1].Ready)
	assert.Equal(t, int64(0), counts[name1].Allocated)

	gs.Status.State = agonesv1.GameServerStateAllocated
	gs.ObjectMeta.ResourceVersion = "3"
	fakeWatch.Add(gs.DeepCopy())

	require.Eventuallyf(t, func() bool {
		counts = pnc.Counts()
		return len(counts) == 1 && int64(0) == counts[name1].Ready
	}, 5*time.Second, time.Second, "Ready should be 0, but is instead", counts[name1].Ready)
	assert.Equal(t, int64(1), counts[name1].Allocated)

	gs.Status.State = agonesv1.GameServerStateShutdown
	gs.ObjectMeta.ResourceVersion = "4"
	fakeWatch.Add(gs.DeepCopy())
	require.Eventuallyf(t, func() bool {
		counts = pnc.Counts()
		return len(counts) == 1 && int64(0) == counts[name1].Allocated
	}, 5*time.Second, time.Second, "Allocated should be 0, but is instead", counts[name1].Allocated)
	assert.Equal(t, int64(0), counts[name1].Ready)

	gs.ObjectMeta.Name = "gs2"
	gs.ObjectMeta.UID = "uid-gs2"
	gs.ObjectMeta.ResourceVersion = "1"
	gs.Status.State = agonesv1.GameServerStateReady
	gs.Status.NodeName = name2

	fakeWatch.Add(gs.DeepCopy())
	require.Eventuallyf(t, func() bool {
		counts = pnc.Counts()
		return len(counts) == 2
	}, 5*time.Second, time.Second, "len should be 2, instead: %v", len(counts))
	assert.Equal(t, int64(0), counts[name1].Ready)
	assert.Equal(t, int64(0), counts[name1].Allocated)
	assert.Equal(t, int64(1), counts[name2].Ready)
	assert.Equal(t, int64(0), counts[name2].Allocated)

	gs.ObjectMeta.Name = "gs3"
	gs.ObjectMeta.UID = "uid-gs3"
	gs.ObjectMeta.ResourceVersion = "1"
	// not likely, but to test the flow
	gs.Status.State = agonesv1.GameServerStateAllocated
	gs.Status.NodeName = name2

	fakeWatch.Add(gs.DeepCopy())
	require.Eventuallyf(t, func() bool {
		counts = pnc.Counts()
		return len(counts) == 2 && int64(1) == counts[name2].Allocated
	}, 5*time.Second, time.Second, "Allocated should be 1, but is instead", counts[name2].Allocated)
	assert.Equal(t, int64(0), counts[name1].Ready)
	assert.Equal(t, int64(0), counts[name1].Allocated)
	assert.Equal(t, int64(1), counts[name2].Ready)
}

func TestPerNodeCounterNodeEvents(t *testing.T) {
	t.Parallel()

	pnc, m := newFakePerNodeCounter()

	gsWatch := watch.NewFake()
	nodeWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
	m.KubeClient.AddWatchReactor("nodes", k8stesting.DefaultWatchReactor(nodeWatch, nil))

	_, cancel := agtesting.StartInformers(m)
	defer cancel()

	require.Empty(t, pnc.Counts())

	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, UID: "uid-gs1", ResourceVersion: "1"},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateReady, NodeName: name1}}
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Namespace: defaultNs, Name: name1}}

	gsWatch.Add(gs.DeepCopy())
	nodeWatch.Add(node.DeepCopy())
	require.Eventuallyf(t, func() bool {
		return len(pnc.Counts()) == 1
	}, 5*time.Second, time.Second, "Should be 1 element, not %v", len(pnc.Counts()))

	nodeWatch.Delete(node.DeepCopy())
	require.Eventually(t, func() bool {
		return len(pnc.Counts()) == 0
	}, 5*time.Second, time.Second, "pnc.Counts() should be empty, but is instead has %v element", len(pnc.Counts()))
}

func TestPerNodeCounterRun(t *testing.T) {
	t.Parallel()
	pnc, m := newFakePerNodeCounter()

	gs1 := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, UID: "uid-gs1", ResourceVersion: "1"},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateReady, NodeName: name1}}

	gs2 := gs1.DeepCopy()
	gs2.ObjectMeta.Name = "gs2"
	gs2.ObjectMeta.UID = "uid-gs2"
	gs2.ObjectMeta.ResourceVersion = "1"
	gs2.Status.State = agonesv1.GameServerStateAllocated

	gs3 := gs1.DeepCopy()
	gs3.ObjectMeta.Name = "gs3"
	gs3.ObjectMeta.UID = "uid-gs3"
	gs3.ObjectMeta.ResourceVersion = "1"
	gs3.Status.State = agonesv1.GameServerStateStarting
	gs3.Status.NodeName = name2

	gs4 := gs1.DeepCopy()
	gs4.ObjectMeta.Name = "gs4"
	gs4.ObjectMeta.UID = "uid-gs4"
	gs4.ObjectMeta.ResourceVersion = "1"
	gs4.Status.State = agonesv1.GameServerStateAllocated

	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs1, *gs2, *gs3, *gs4}}, nil
	})
	ctx, cancel := agtesting.StartInformers(m, pnc.gameServerSynced)
	defer cancel()

	err := pnc.Run(ctx, 0)
	require.NoError(t, err)

	counts := pnc.Counts()

	require.Len(t, counts, 2)
	assert.Equal(t, int64(1), counts[name1].Ready)
	assert.Equal(t, int64(2), counts[name1].Allocated)
	assert.Equal(t, int64(0), counts[name2].Ready)
	assert.Equal(t, int64(0), counts[name2].Allocated)
}

func TestPerNodeCounterDuplicateEvents(t *testing.T) {
	t.Parallel()

	pnc, m := newFakePerNodeCounter()

	fakeWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	_, cancel := agtesting.StartInformers(m)
	defer cancel()

	assert.Empty(t, pnc.Counts())

	// Create a GameServer in Ready state
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, UID: "uid-gs1", ResourceVersion: "1"},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateReady, NodeName: name1,
		},
	}

	// Send the same Add event twice with the same ResourceVersion
	fakeWatch.Add(gs.DeepCopy())
	fakeWatch.Add(gs.DeepCopy())

	var counts map[string]NodeCount
	require.Eventuallyf(t, func() bool {
		counts = pnc.Counts()
		return len(counts) == 1
	}, 5*time.Second, time.Second, "len should be 1, instead: %v", len(counts))

	// Should only count once, not twice
	assert.Equal(t, int64(1), counts[name1].Ready)
	assert.Equal(t, int64(0), counts[name1].Allocated)

	// Send duplicate Update events with the same ResourceVersion
	gs.Status.State = agonesv1.GameServerStateAllocated
	gs.ObjectMeta.ResourceVersion = "2"

	fakeWatch.Modify(gs.DeepCopy())
	fakeWatch.Modify(gs.DeepCopy())
	fakeWatch.Modify(gs.DeepCopy())

	require.Eventuallyf(t, func() bool {
		counts = pnc.Counts()
		return len(counts) == 1 && int64(0) == counts[name1].Ready && int64(1) == counts[name1].Allocated
	}, 5*time.Second, time.Second, "Should transition to Allocated once")

	// Should only count the transition once despite 3 duplicate events
	assert.Equal(t, int64(0), counts[name1].Ready)
	assert.Equal(t, int64(1), counts[name1].Allocated)

	// Test duplicate events across Add and Modify
	gs2 := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{Name: "gs2", Namespace: defaultNs, UID: "uid-gs2", ResourceVersion: "1"},
		Status: agonesv1.GameServerStatus{
			State: agonesv1.GameServerStateReady, NodeName: name2,
		},
	}

	// Send Add, then send the same event as Modify
	fakeWatch.Add(gs2.DeepCopy())
	fakeWatch.Modify(gs2.DeepCopy())

	require.Eventuallyf(t, func() bool {
		counts = pnc.Counts()
		return len(counts) == 2
	}, 5*time.Second, time.Second, "len should be 2, instead: %v", len(counts))

	// Should only count once despite being sent as both Add and Modify
	assert.Equal(t, int64(1), counts[name2].Ready)
	assert.Equal(t, int64(0), counts[name2].Allocated)
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakePerNodeCounter() (*PerNodeCounter, agtesting.Mocks) {
	m := agtesting.NewMocks()
	c := NewPerNodeCounter(m.KubeInformerFactory, m.AgonesInformerFactory)
	return c, m
}
