// Copyright 2020 Google LLC All Rights Reserved.
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
	"context"
	"sync/atomic"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/heptiolabs/healthcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestMissingPodControllerSyncGameServer(t *testing.T) {
	type expected struct {
		updated     bool
		updateTests func(t *testing.T, gs *agonesv1.GameServer)
		postTests   func(t *testing.T, mocks agtesting.Mocks)
	}
	fixtures := map[string]struct {
		setup    func(*agonesv1.GameServer, *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod)
		expected expected
	}{
		"pod exists": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				return gs, pod
			},
			expected: expected{
				updated:     false,
				updateTests: func(_ *testing.T, _ *agonesv1.GameServer) {},
				postTests:   func(_ *testing.T, _ agtesting.Mocks) {},
			},
		},
		"pod doesn't exist: game server is fine": {
			setup: func(gs *agonesv1.GameServer, _ *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				return gs, nil
			},
			expected: expected{
				updated: true,
				updateTests: func(t *testing.T, gs *agonesv1.GameServer) {
					assert.Equal(t, agonesv1.GameServerStateUnhealthy, gs.Status.State)
				},
				postTests: func(t *testing.T, m agtesting.Mocks) {
					agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Warning Unhealthy Pod is missing")
				},
			},
		},
		"pod doesn't exist: game server not found": {
			setup: func(_ *agonesv1.GameServer, _ *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				return nil, nil
			},
			expected: expected{
				updated:     false,
				updateTests: func(_ *testing.T, _ *agonesv1.GameServer) {},
				postTests:   func(_ *testing.T, _ agtesting.Mocks) {},
			},
		},
		"pod doesn't exist: game server is being deleted": {
			setup: func(gs *agonesv1.GameServer, _ *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				now := metav1.Now()
				gs.ObjectMeta.DeletionTimestamp = &now
				return gs, nil
			},
			expected: expected{
				updated:     false,
				updateTests: func(_ *testing.T, _ *agonesv1.GameServer) {},
				postTests:   func(_ *testing.T, _ agtesting.Mocks) {},
			},
		},
		"pod doesn't exist: game server is already Unhealthy": {
			setup: func(gs *agonesv1.GameServer, _ *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateUnhealthy
				return gs, nil
			},
			expected: expected{
				updated:     false,
				updateTests: func(_ *testing.T, _ *agonesv1.GameServer) {},
				postTests:   func(_ *testing.T, _ agtesting.Mocks) {},
			},
		},
		"pod is not a gameserver pod": {
			setup: func(gs *agonesv1.GameServer, _ *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				return gs, &corev1.Pod{ObjectMeta: gs.ObjectMeta}
			},
			expected: expected{
				updated: true,
				updateTests: func(t *testing.T, gs *agonesv1.GameServer) {
					assert.Equal(t, agonesv1.GameServerStateUnhealthy, gs.Status.State)
				},
				postTests: func(t *testing.T, m agtesting.Mocks) {
					agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Warning Unhealthy Pod is missing")
				},
			},
		},
		"pod exists but is in Failed state": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) (*agonesv1.GameServer, *corev1.Pod) {
				pod.Status.Phase = corev1.PodFailed
				return gs, pod
			},
			expected: expected{
				updated: true,
				updateTests: func(t *testing.T, gs *agonesv1.GameServer) {
					assert.Equal(t, agonesv1.GameServerStateUnhealthy, gs.Status.State)
				},
				postTests: func(t *testing.T, m agtesting.Mocks) {
					agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Warning Unhealthy Pod has failed")
				},
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			m := agtesting.NewMocks()
			c := NewMissingPodController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory)
			c.recorder = m.FakeRecorder

			gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
				Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{}}
			gs.ApplyDefaults()

			pod, err := gs.Pod(agtesting.FakeAPIHooks{})
			assert.NoError(t, err)

			gs, pod = v.setup(gs, pod)
			m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				if gs != nil {
					return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gs}}, nil
				}
				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{}}, nil
			})
			m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				if pod != nil {
					return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
				}
				return true, &corev1.PodList{Items: []corev1.Pod{}}, nil
			})

			updated := false
			m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				ua := action.(k8stesting.UpdateAction)
				gs := ua.GetObject().(*agonesv1.GameServer)
				v.expected.updateTests(t, gs)
				return true, gs, nil
			})
			ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.podSynced)
			defer cancel()

			err = c.syncGameServer(ctx, "default/test")
			assert.NoError(t, err)
			assert.Equal(t, v.expected.updated, updated, "updated state")
			v.expected.postTests(t, m)
		})
	}
}

func TestMissingPodControllerRun(t *testing.T) {
	m := agtesting.NewMocks()
	c := NewMissingPodController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory)

	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{}}
	gs.ApplyDefaults()

	received := make(chan string)
	h := func(_ context.Context, name string) error {
		assert.Equal(t, "default/test", name)
		received <- name
		return nil
	}

	gsWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))

	// do we have a pod?
	exist := atomic.Bool{}
	exist.Store(false)

	m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		if exist.Load() {
			pod, err := gs.Pod(agtesting.FakeAPIHooks{})
			require.NoError(t, err)
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		}
		return true, &corev1.PodList{Items: []corev1.Pod{}}, nil
	})

	c.workerqueue.SyncHandler = h

	ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.podSynced)
	defer cancel()

	go func() {
		err := c.Run(ctx, 1)
		assert.Nil(t, err, "Run should not error")
	}()

	noChange := func() {
		assert.True(t, cache.WaitForCacheSync(ctx.Done(), c.gameServerSynced, c.podSynced))
		select {
		case <-received:
			assert.FailNow(t, "should not run sync")
		default:
		}
	}

	result := func() {
		select {
		case res := <-received:
			assert.Equal(t, "default/test", res)
		case <-time.After(2 * time.Second):
			assert.FailNow(t, "did not run sync")
		}
	}

	// initial population
	gsWatch.Add(gs.DeepCopy())
	noChange()

	// gs before pod
	gs.Status.State = agonesv1.GameServerStatePortAllocation
	gsWatch.Modify(gs.DeepCopy())
	noChange()

	// ready gs
	exist.Store(true)
	gs.Status.State = agonesv1.GameServerStateReady
	gsWatch.Modify(gs.DeepCopy())
	result()

	// allocated
	gs.Status.State = agonesv1.GameServerStateAllocated
	gsWatch.Modify(gs.DeepCopy())
	result()

	// unhealthy gs
	gs.Status.State = agonesv1.GameServerStateUnhealthy
	gsWatch.Modify(gs.DeepCopy())
	noChange()

	// shutdown gs
	gs.Status.State = agonesv1.GameServerStateShutdown
	gsWatch.Modify(gs.DeepCopy())
	noChange()

	// dev gameservers
	gs.Status.State = agonesv1.GameServerStateReady
	gs.ObjectMeta.Annotations[agonesv1.DevAddressAnnotation] = ipFixture
	gsWatch.Modify(gs.DeepCopy())
	noChange()
}
