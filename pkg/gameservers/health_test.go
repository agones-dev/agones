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
	"context"
	"errors"
	"testing"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	agtesting "agones.dev/agones/pkg/testing"
	agruntime "agones.dev/agones/pkg/util/runtime"
	"github.com/heptiolabs/healthcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
)

func TestHealthControllerFailedContainer(t *testing.T) {
	t.Parallel()

	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()
	require.NoError(t, agruntime.ParseFeatures(string(agruntime.FeatureSidecarContainers)+"=false"))

	m := agtesting.NewMocks()
	hc := NewHealthController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory, false)

	gs := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test"}, Spec: newSingleContainerSpec()}
	gs.ApplyDefaults()

	pod, err := gs.Pod(agtesting.FakeAPIHooks{})
	require.NoError(t, err)
	pod.Status = corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{Name: gs.Spec.Container,
		State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}}}}

	assert.True(t, hc.failedContainer(pod))
	pod2 := pod.DeepCopy()

	pod.Status.ContainerStatuses[0].State.Terminated = nil
	assert.False(t, hc.failedContainer(pod))

	pod.Status.ContainerStatuses[0].LastTerminationState.Terminated = &corev1.ContainerStateTerminated{}
	assert.True(t, hc.failedContainer(pod))

	pod2.Status.ContainerStatuses[0].Name = "Not a matching name"
	assert.False(t, hc.failedContainer(pod2))
}

func TestHealthControllerFailedPod(t *testing.T) {
	t.Parallel()

	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	// set sidecar feature flag
	require.NoError(t, agruntime.ParseFeatures(string(agruntime.FeatureSidecarContainers)+"=true"))

	m := agtesting.NewMocks()
	hc := NewHealthController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory, false)

	gs := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test"}, Spec: newSingleContainerSpec()}
	gs.ApplyDefaults()

	pod, err := gs.Pod(agtesting.FakeAPIHooks{})
	require.NoError(t, err)
	assert.False(t, hc.failedPod(pod))

	// set the pod to failed status
	pod.Status.Phase = corev1.PodFailed
	assert.True(t, hc.failedPod(pod))
}

func TestHealthUnschedulableWithNoFreePorts(t *testing.T) {
	t.Parallel()

	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()
	require.NoError(t, agruntime.ParseFeatures(string(agruntime.FeatureSidecarContainers)+"=false"))

	m := agtesting.NewMocks()
	hc := NewHealthController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory, false)

	gs := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test"}, Spec: newSingleContainerSpec()}
	gs.ApplyDefaults()

	for name, tc := range map[string]struct {
		message           string
		waitOnFreePorts   bool
		wantUnschedulable bool
	}{
		"unschedulable, terminal": {
			message:           "0/4 nodes are available: 4 node(s) didn't have free ports for the requestedpod ports.",
			wantUnschedulable: true,
		},
		"unschedulable, will wait on free ports": {
			message:         "0/4 nodes are available: 4 node(s) didn't have free ports for the requestedpod ports.",
			waitOnFreePorts: true,
		},
		"some other condition": {
			message: "twas brillig and the slithy toves",
		},
	} {
		t.Run(name, func(t *testing.T) {
			pod, err := gs.Pod(agtesting.FakeAPIHooks{})
			require.NoError(t, err)

			pod.Status.Conditions = []corev1.PodCondition{
				{Type: corev1.PodScheduled, Reason: corev1.PodReasonUnschedulable,
					Message: tc.message},
			}
			hc.waitOnFreePorts = tc.waitOnFreePorts
			assert.Equal(t, tc.wantUnschedulable, hc.unschedulableWithNoFreePorts(pod))
		})
	}
}

func TestHealthControllerSkipUnhealthyGameContainer(t *testing.T) {
	t.Parallel()

	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()
	require.NoError(t, agruntime.ParseFeatures(string(agruntime.FeatureSidecarContainers)+"=false"))

	type expected struct {
		result bool
		err    string
	}

	fixtures := map[string]struct {
		setup    func(*agonesv1.GameServer, *corev1.Pod)
		expected expected
	}{
		"scheduled and terminated container": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateScheduled
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
					Name:  gs.Spec.Container,
					State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}},
				}}
			},
			expected: expected{result: true},
		},
		"after ready and terminated container": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateReady
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
					Name:  gs.Spec.Container,
					State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}},
				}}
			},
			expected: expected{result: false},
		},
		"before ready, with no terminated container": {
			setup: func(gs *agonesv1.GameServer, _ *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateScheduled
			},
			expected: expected{result: false},
		},
		"after ready, with no terminated container": {
			setup: func(gs *agonesv1.GameServer, _ *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateAllocated
			},
			expected: expected{result: false},
		},
		"before ready, with a LastTerminated container": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateScheduled
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
					Name:                 gs.Spec.Container,
					LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}},
				}}
			},
			expected: expected{result: true},
		},
		"after ready, with a LastTerminated container, not matching": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateReady
				gs.Annotations[agonesv1.GameServerReadyContainerIDAnnotation] = "4321"
				pod.ObjectMeta.Annotations[agonesv1.GameServerReadyContainerIDAnnotation] = "4321"
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
					ContainerID:          "1234",
					Name:                 gs.Spec.Container,
					LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}},
				}}
			},
			expected: expected{result: false},
		},
		"after ready, with a LastTerminated container, matching": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				gs.Status.State = agonesv1.GameServerStateReserved
				gs.Annotations[agonesv1.GameServerReadyContainerIDAnnotation] = "1234"
				pod.Annotations[agonesv1.GameServerReadyContainerIDAnnotation] = "1234"
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
					ContainerID:          "1234",
					Name:                 gs.Spec.Container,
					LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}},
				}}
			},
			expected: expected{result: true},
		},
		"pod is missing!": {
			setup: func(_ *agonesv1.GameServer, pod *corev1.Pod) {
				pod.ObjectMeta.Name = "missing"
			},
			expected: expected{result: false},
		},
		"annotations do not match": {
			setup: func(gs *agonesv1.GameServer, pod *corev1.Pod) {
				gs.Annotations[agonesv1.GameServerReadyContainerIDAnnotation] = "1234"
				pod.Annotations[agonesv1.GameServerReadyContainerIDAnnotation] = ""
			},
			expected: expected{err: "pod and gameserver test data are out of sync, retrying"},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			m := agtesting.NewMocks()
			hc := NewHealthController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory, false)
			gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: defaultNs}, Spec: newSingleContainerSpec()}
			gs.ApplyDefaults()
			pod, err := gs.Pod(agtesting.FakeAPIHooks{})
			assert.NoError(t, err)

			v.setup(gs, pod)

			m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
			})

			result, err := hc.skipUnhealthyGameContainer(gs, pod)

			if len(v.expected.err) > 0 {
				require.EqualError(t, err, v.expected.err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, v.expected.result, result)
		})
	}
}

func TestHealthControllerSyncGameServer(t *testing.T) {
	t.Parallel()

	type expected struct {
		updated bool
	}
	fixtures := map[string]struct {
		state     agonesv1.GameServerState
		podStatus *corev1.PodStatus
		feature   string
		expected  expected
	}{
		"started": {
			state: agonesv1.GameServerStateStarting,
			expected: expected{
				updated: true,
			},
		},
		"shutdown": {
			state: agonesv1.GameServerStateShutdown,
			expected: expected{
				updated: false,
			},
		},
		"unhealthy": {
			state: agonesv1.GameServerStateUnhealthy,
			expected: expected{
				updated: false,
			},
		},
		"ready": {
			state: agonesv1.GameServerStateReady,
			expected: expected{
				updated: true,
			},
		},
		"allocated": {
			state: agonesv1.GameServerStateAllocated,
			expected: expected{
				updated: true,
			},
		},
		"container failed before ready": {
			state: agonesv1.GameServerStateStarting,
			podStatus: &corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
				{Name: "container", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}}}},
			expected: expected{updated: false},
		},
		"container failed after ready": {
			state:   agonesv1.GameServerStateAllocated,
			feature: string(agruntime.FeatureSidecarContainers) + "=false",
			podStatus: &corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
				{Name: "container", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}}}},
			expected: expected{updated: true},
		},
		"container recovered and starting after queueing": {
			state: agonesv1.GameServerStateStarting,
			podStatus: &corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
				{Name: "container", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{}}}}},
			expected: expected{updated: false},
		},
		"container recovered and ready after queueing": {
			state: agonesv1.GameServerStateReady,
			podStatus: &corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
				{Name: "container", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}},
			expected: expected{updated: false},
		},
		"container recovered and allocated after queueing": {
			state: agonesv1.GameServerStateAllocated,
			podStatus: &corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
				{Name: "container", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}},
			expected: expected{updated: false},
		},
		"pod failed": {
			feature:   string(agruntime.FeatureSidecarContainers) + "=true",
			state:     agonesv1.GameServerStateReady,
			podStatus: &corev1.PodStatus{Phase: corev1.PodFailed},
			expected:  expected{updated: true},
		},
	}

	for name, test := range fixtures {
		t.Run(name, func(t *testing.T) {
			agruntime.FeatureTestMutex.Lock()
			defer agruntime.FeatureTestMutex.Unlock()
			if len(test.feature) > 0 {
				require.NoError(t, agruntime.ParseFeatures(test.feature))
			}

			m := agtesting.NewMocks()
			hc := NewHealthController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory, false)
			hc.recorder = m.FakeRecorder

			gs := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test"}, Spec: newSingleContainerSpec(),
				Status: agonesv1.GameServerStatus{State: test.state}}
			gs.ApplyDefaults()

			got := false
			updated := false
			m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				list := &corev1.PodList{Items: []corev1.Pod{}}
				if test.podStatus != nil {
					pod, err := gs.Pod(agtesting.FakeAPIHooks{})
					assert.NoError(t, err)
					pod.Status = *test.podStatus
					list.Items = append(list.Items, *pod)
				}
				return true, list, nil
			})
			m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
				got = true
				return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
			})
			m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
				updated = true
				ua := action.(k8stesting.UpdateAction)
				gsObj := ua.GetObject().(*agonesv1.GameServer)
				assert.Equal(t, agonesv1.GameServerStateUnhealthy, gsObj.Status.State)
				return true, gsObj, nil
			})

			ctx, cancel := agtesting.StartInformers(m, hc.gameServerSynced, hc.podSynced)
			defer cancel()

			err := hc.syncGameServer(ctx, "default/test")
			assert.Nil(t, err, err)
			assert.True(t, got, "GameServers Should be got!")

			assert.Equal(t, test.expected.updated, updated, "updated test")
		})
	}
}

func TestHealthControllerSyncGameServerUpdateFailed(t *testing.T) {
	t.Parallel()

	m := agtesting.NewMocks()
	hc := NewHealthController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory, false)
	hc.recorder = m.FakeRecorder

	gs := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test"}, Spec: newSingleContainerSpec(),
		Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateAllocated}}
	gs.ApplyDefaults()

	m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		list := &corev1.PodList{Items: []corev1.Pod{}}
		return true, list, nil
	})
	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: []agonesv1.GameServer{gs}}, nil
	})
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		ua := action.(k8stesting.UpdateAction)
		gsObj := ua.GetObject().(*agonesv1.GameServer)
		assert.Equal(t, agonesv1.GameServerStateUnhealthy, gsObj.Status.State)
		return true, gsObj, errors.New("update-err")
	})

	ctx, cancel := agtesting.StartInformers(m, hc.gameServerSynced, hc.podSynced)
	defer cancel()

	err := hc.syncGameServer(ctx, "default/test")

	if assert.Error(t, err) {
		assert.Equal(t, "error updating GameServer test/default to unhealthy: update-err", err.Error())
	}
}

func TestHealthControllerRunNoSideCar(t *testing.T) {
	t.Parallel()

	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()
	require.NoError(t, agruntime.ParseFeatures(string(agruntime.FeatureSidecarContainers)+"=false"))

	m := agtesting.NewMocks()
	hc := NewHealthController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory, false)
	hc.recorder = m.FakeRecorder

	gsWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))

	podWatch := watch.NewFake()
	m.KubeClient.AddWatchReactor("pods", k8stesting.DefaultWatchReactor(podWatch, nil))

	updated := make(chan bool)
	defer close(updated)
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		defer func() {
			updated <- true
		}()
		ua := action.(k8stesting.UpdateAction)
		gsObj := ua.GetObject().(*agonesv1.GameServer)
		assert.Equal(t, agonesv1.GameServerStateUnhealthy, gsObj.Status.State)
		return true, gsObj, nil
	})

	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test"}, Spec: newSingleContainerSpec(),
		Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}}
	gs.ApplyDefaults()
	pod, err := gs.Pod(agtesting.FakeAPIHooks{})
	require.NoError(t, err)

	stop, cancel := agtesting.StartInformers(m)
	defer cancel()

	gsWatch.Add(gs.DeepCopy())
	podWatch.Add(pod.DeepCopy())

	go hc.Run(stop, 1) // nolint: errcheck
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
		return hc.workerqueue.RunCount() == 1, nil
	})
	assert.NoError(t, err)

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: gs.Spec.Container, State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}}}
	// gate
	assert.True(t, hc.failedContainer(pod))
	assert.False(t, hc.unschedulableWithNoFreePorts(pod))

	podWatch.Modify(pod.DeepCopy())

	select {
	case <-updated:
	case <-time.After(10 * time.Second):
		assert.FailNow(t, "timeout on GameServer update")
	}

	agtesting.AssertEventContains(t, m.FakeRecorder.Events, string(agonesv1.GameServerStateUnhealthy))

	pod.Status.ContainerStatuses = nil
	pod.Status.Conditions = []corev1.PodCondition{
		{Type: corev1.PodScheduled, Reason: corev1.PodReasonUnschedulable,
			Message: "0/4 nodes are available: 4 node(s) didn't have free ports for the requestedpod ports."},
	}
	// gate
	assert.True(t, hc.unschedulableWithNoFreePorts(pod))
	assert.False(t, hc.failedContainer(pod))

	podWatch.Modify(pod.DeepCopy())

	select {
	case <-updated:
	case <-time.After(10 * time.Second):
		assert.FailNow(t, "timeout on GameServer update")
	}

	agtesting.AssertEventContains(t, m.FakeRecorder.Events, string(agonesv1.GameServerStateUnhealthy))

	podWatch.Delete(pod.DeepCopy())
	select {
	case <-updated:
	case <-time.After(10 * time.Second):
		assert.FailNow(t, "timeout on GameServer update")
	}

	agtesting.AssertEventContains(t, m.FakeRecorder.Events, string(agonesv1.GameServerStateUnhealthy))
}

func TestHealthControllerRunWithSideCar(t *testing.T) {
	t.Parallel()

	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()
	require.NoError(t, agruntime.ParseFeatures(string(agruntime.FeatureSidecarContainers)+"=true"))

	m := agtesting.NewMocks()
	hc := NewHealthController(healthcheck.NewHandler(), m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory, false)
	hc.recorder = m.FakeRecorder

	gsWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))

	podWatch := watch.NewFake()
	m.KubeClient.AddWatchReactor("pods", k8stesting.DefaultWatchReactor(podWatch, nil))

	updated := make(chan bool)
	defer close(updated)
	m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		defer func() {
			updated <- true
		}()
		ua := action.(k8stesting.UpdateAction)
		gsObj := ua.GetObject().(*agonesv1.GameServer)
		assert.Equal(t, agonesv1.GameServerStateUnhealthy, gsObj.Status.State)
		return true, gsObj, nil
	})

	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test"}, Spec: newSingleContainerSpec(),
		Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}}
	gs.ApplyDefaults()
	pod, err := gs.Pod(agtesting.FakeAPIHooks{})
	require.NoError(t, err)

	stop, cancel := agtesting.StartInformers(m)
	defer cancel()

	gsWatch.Add(gs.DeepCopy())
	podWatch.Add(pod.DeepCopy())

	go hc.Run(stop, 1) // nolint: errcheck
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(_ context.Context) (bool, error) {
		return hc.workerqueue.RunCount() == 1, nil
	})
	assert.NoError(t, err)

	pod.Status = corev1.PodStatus{Phase: corev1.PodFailed}
	// gate
	require.True(t, hc.failedPod(pod))
	require.False(t, hc.evictedPod(pod))

	podWatch.Modify(pod.DeepCopy())

	select {
	case <-updated:
	case <-time.After(10 * time.Second):
		assert.FailNow(t, "timeout on GameServer update")
	}

	agtesting.AssertEventContains(t, m.FakeRecorder.Events, string(agonesv1.GameServerStateUnhealthy))

	podWatch.Delete(pod.DeepCopy())
	select {
	case <-updated:
	case <-time.After(10 * time.Second):
		assert.FailNow(t, "timeout on GameServer update")
	}

	agtesting.AssertEventContains(t, m.FakeRecorder.Events, string(agonesv1.GameServerStateUnhealthy))
}
