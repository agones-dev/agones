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
	"fmt"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/gameservers"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/heptiolabs/healthcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
)

func TestAllocationOverflowControllerWatchGameServers(t *testing.T) {
	t.Parallel()

	gsSet := defaultFixture()
	gsSet.Status.Replicas = gsSet.Spec.Replicas
	gsSet.Status.ReadyReplicas = gsSet.Spec.Replicas
	c, m := newFakeAllocationOverflowController()

	received := make(chan string, 10)
	defer close(received)

	gsSetWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameserversets", k8stesting.DefaultWatchReactor(gsSetWatch, nil))

	c.workerqueue.SyncHandler = func(_ context.Context, name string) error {
		received <- name
		return nil
	}

	ctx, cancel := agtesting.StartInformers(m, c.gameServerSetSynced)
	defer cancel()

	go func() {
		err := c.Run(ctx)
		require.NoError(t, err)
	}()

	change := func() string {
		select {
		case result := <-received:
			return result
		case <-time.After(3 * time.Second):
			require.FailNow(t, "timeout occurred")
		}
		return ""
	}

	nochange := func() {
		select {
		case <-received:
			assert.Fail(t, "Should be no value")
		case <-time.After(time.Second):
		}
	}

	gsSetWatch.Add(gsSet.DeepCopy())
	nochange()

	// update with no allocation overflow
	require.Nil(t, gsSet.Spec.AllocationOverflow)
	gsSet.Spec.Replicas++
	gsSetWatch.Modify(gsSet.DeepCopy())
	nochange()

	// update with no labels or annotations
	gsSet.Spec.AllocationOverflow = &agonesv1.AllocationOverflow{}
	gsSet.Spec.Replicas++
	gsSetWatch.Modify(gsSet.DeepCopy())
	nochange()

	// update with allocation <= replicas (and a label)
	gsSet.Spec.AllocationOverflow.Labels = map[string]string{"colour": "green"}
	gsSet.Status.AllocatedReplicas = 2
	gsSetWatch.Modify(gsSet.DeepCopy())
	nochange()

	// update with allocation > replicas
	gsSet.Status.AllocatedReplicas = 20
	gsSetWatch.Modify(gsSet.DeepCopy())
	require.Equal(t, fmt.Sprintf("%s/%s", gsSet.ObjectMeta.Namespace, gsSet.ObjectMeta.Name), change())

	// delete
	gsSetWatch.Delete(gsSet.DeepCopy())
	nochange()
}

func TestAllocationOverflowSyncGameServerSet(t *testing.T) {
	t.Parallel()

	// setup fictures.
	setup := func(gs func(server *agonesv1.GameServer)) (*agonesv1.GameServerSet, *AllocationOverflowController, agtesting.Mocks) {
		gsSet := defaultFixture()
		gsSet.Status.AllocatedReplicas = 5
		gsSet.Status.Replicas = 3
		gsSet.Spec.Replicas = 3
		gsSet.Spec.AllocationOverflow = &agonesv1.AllocationOverflow{Labels: map[string]string{"colour": "green"}}
		list := createGameServers(gsSet, 5)
		for i := range list {
			list[i].Status.State = agonesv1.GameServerStateAllocated
			gs(&list[i])
		}

		c, m := newFakeAllocationOverflowController()
		m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet}}, nil
		})
		m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &agonesv1.GameServerList{Items: list}, nil
		})
		return gsSet, c, m
	}

	// run the sync process
	run := func(c *AllocationOverflowController, m agtesting.Mocks, gsSet *agonesv1.GameServerSet, update func(action k8stesting.Action) (bool, runtime.Object, error)) func() {
		m.AgonesClient.AddReactor("update", "gameservers", update)
		ctx, cancel := agtesting.StartInformers(m, c.gameServerSetSynced, c.gameServerSynced)
		err := c.syncGameServerSet(ctx, gsSet.ObjectMeta.Namespace+"/"+gsSet.ObjectMeta.Name)
		require.NoError(t, err)
		return cancel
	}

	t.Run("labels are applied", func(t *testing.T) {
		gsSet, c, m := setup(func(_ *agonesv1.GameServer) {})
		count := 0
		cancel := run(c, m, gsSet, func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			require.Equal(t, gs.Status.State, agonesv1.GameServerStateAllocated)
			require.Equal(t, "green", gs.ObjectMeta.Labels["colour"])

			count++
			return true, nil, nil
		})
		defer cancel()
		require.Equal(t, 2, count)
	})

	t.Run("Labels are already set", func(t *testing.T) {
		gsSet, c, m := setup(func(gs *agonesv1.GameServer) {
			gs.ObjectMeta.Labels["colour"] = "green"
		})
		cancel := run(c, m, gsSet, func(_ k8stesting.Action) (bool, runtime.Object, error) {
			require.Fail(t, "should not update")
			return true, nil, nil
		})
		defer cancel()
	})

	t.Run("one label is set", func(t *testing.T) {
		set := false
		gsSet, c, m := setup(func(gs *agonesv1.GameServer) {
			// just make one as already set
			if !set {
				gs.ObjectMeta.Labels["colour"] = "green"
				set = true
			}
		})

		count := 0
		cancel := run(c, m, gsSet, func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			require.Equal(t, gs.Status.State, agonesv1.GameServerStateAllocated)
			require.Equal(t, "green", gs.ObjectMeta.Labels["colour"])

			count++
			return true, nil, nil
		})
		defer cancel()
		require.Equal(t, 1, count)
	})
}

// newFakeAllocationOverflowController returns a controller, backed by the fake Clientset
func newFakeAllocationOverflowController() (*AllocationOverflowController, agtesting.Mocks) {
	m := agtesting.NewMocks()
	counter := gameservers.NewPerNodeCounter(m.KubeInformerFactory, m.AgonesInformerFactory)
	c := NewAllocatorOverflowController(healthcheck.NewHandler(), counter, m.AgonesClient, m.AgonesInformerFactory)
	return c, m
}
