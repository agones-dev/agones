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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/cloudproduct/generic"
	"agones.dev/agones/pkg/portallocator"
	agtesting "agones.dev/agones/pkg/testing"
	agruntime "agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/webhooks"
	"github.com/heptiolabs/healthcheck"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/ptr"
)

const (
	ipFixture        = "12.12.12.12"
	ipv6Fixture      = "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
	nodeFixtureName  = "node1"
	sidecarRunAsUser = 1000
)

var GameServerKind = metav1.GroupVersionKind(agonesv1.SchemeGroupVersion.WithKind("GameServer"))
var PodKind = corev1.SchemeGroupVersion.WithKind("Pod")

func TestControllerSyncGameServer(t *testing.T) {
	t.Parallel()

	t.Run("Creating a new GameServer", func(t *testing.T) {
		c, mocks := newFakeController()
		updateCount := 0
		podCreated := false
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{{ContainerPort: 7777}},
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "container", Image: "container/image"}},
				},
				},
			},
		}

		node := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName},
			Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: ipFixture, Type: corev1.NodeExternalIP}}}}

		fixture.ApplyDefaults()

		watchPods := watch.NewFake()
		mocks.KubeClient.AddWatchReactor("pods", k8stesting.DefaultWatchReactor(watchPods, nil))

		mocks.KubeClient.AddReactor("list", "nodes", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.NodeList{Items: []corev1.Node{node}}, nil
		})
		mocks.KubeClient.AddReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.CreateAction)
			pod := ca.GetObject().(*corev1.Pod)
			pod.Spec.NodeName = node.ObjectMeta.Name
			pod.Status.PodIPs = []corev1.PodIP{{IP: ipv6Fixture}}
			podCreated = true
			assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Name)
			watchPods.Add(pod)
			// wait for the change to propagate
			require.Eventually(t, func() bool {
				list, err := c.podLister.List(labels.Everything())
				assert.NoError(t, err)
				return len(list) == 1
			}, 5*time.Second, time.Second)
			return true, pod, nil
		})
		mocks.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			gameServers := &agonesv1.GameServerList{Items: []agonesv1.GameServer{*fixture}}
			return true, gameServers, nil
		})
		mocks.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			updateCount++
			expectedState := agonesv1.GameServerState("notastate")
			switch updateCount {
			case 1:
				expectedState = agonesv1.GameServerStateCreating
			case 2:
				expectedState = agonesv1.GameServerStateStarting
			case 3:
				expectedState = agonesv1.GameServerStateScheduled
			}

			assert.Equal(t, expectedState, gs.Status.State)
			if expectedState == agonesv1.GameServerStateScheduled {
				assert.Equal(t, ipFixture, gs.Status.Address)
				assert.Equal(t, []corev1.NodeAddress{
					{Address: ipFixture, Type: "ExternalIP"},
					{Address: ipv6Fixture, Type: "PodIP"},
				}, gs.Status.Addresses)
				assert.NotEmpty(t, gs.Status.Ports[0].Port)
			}

			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(mocks, c.gameServerSynced)
		defer cancel()

		err := c.portAllocator.Run(ctx)
		assert.Nil(t, err)

		err = c.syncGameServer(ctx, "default/test")
		assert.Nil(t, err)
		assert.Equal(t, 3, updateCount, "update reactor should fire thrice")
		assert.True(t, podCreated, "pod should be created")
	})

	t.Run("When a GameServer has been deleted, the sync operation should be a noop", func(t *testing.T) {
		runReconcileDeleteGameServer(t, &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec:   newSingleContainerSpec(),
			Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}})
	})
}

func TestControllerSyncGameServerWithInitSidecar(t *testing.T) {
	t.Parallel()

	t.Run("Creating a new GameServer", func(t *testing.T) {
		c, mocks := newFakeController()
		updateCount := 0
		podCreated := false
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{
					{ContainerPort: 7777},
					{ContainerPort: 8888, Container: ptr.To("sidecar")},
				},
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "sidecar", Image: "container/image", RestartPolicy: ptr.To(corev1.ContainerRestartPolicyAlways)}},
					Containers:     []corev1.Container{{Name: "container", Image: "container/image"}},
				},
				},
			},
		}

		node := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName},
			Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: ipFixture, Type: corev1.NodeExternalIP}}}}

		fixture.ApplyDefaults()

		watchPods := watch.NewFake()
		mocks.KubeClient.AddWatchReactor("pods", k8stesting.DefaultWatchReactor(watchPods, nil))

		mocks.KubeClient.AddReactor("list", "nodes", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.NodeList{Items: []corev1.Node{node}}, nil
		})
		mocks.KubeClient.AddReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ca := action.(k8stesting.CreateAction)
			pod := ca.GetObject().(*corev1.Pod)
			pod.Spec.NodeName = node.ObjectMeta.Name
			pod.Status.PodIPs = []corev1.PodIP{{IP: ipv6Fixture}}
			podCreated = true
			assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Name)
			watchPods.Add(pod)
			// wait for the change to propagate
			require.Eventually(t, func() bool {
				list, err := c.podLister.List(labels.Everything())
				assert.NoError(t, err)
				return len(list) == 1
			}, 5*time.Second, time.Second)
			return true, pod, nil
		})
		mocks.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			gameServers := &agonesv1.GameServerList{Items: []agonesv1.GameServer{*fixture}}
			return true, gameServers, nil
		})
		mocks.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			updateCount++
			expectedState := agonesv1.GameServerState("notastate")
			switch updateCount {
			case 1:
				expectedState = agonesv1.GameServerStateCreating
			case 2:
				expectedState = agonesv1.GameServerStateStarting
			case 3:
				expectedState = agonesv1.GameServerStateScheduled
			}

			assert.Equal(t, expectedState, gs.Status.State)
			if expectedState == agonesv1.GameServerStateScheduled {
				assert.Equal(t, ipFixture, gs.Status.Address)
				assert.Equal(t, []corev1.NodeAddress{
					{Address: ipFixture, Type: "ExternalIP"},
					{Address: ipv6Fixture, Type: "PodIP"},
				}, gs.Status.Addresses)
				assert.NotEmpty(t, gs.Status.Ports[0].Port)
				assert.NotEmpty(t, gs.Status.Ports[1].Port)
			}

			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(mocks, c.gameServerSynced)
		defer cancel()

		err := c.portAllocator.Run(ctx)
		assert.Nil(t, err)

		err = c.syncGameServer(ctx, "default/test")
		assert.Nil(t, err)
		assert.Equal(t, 3, updateCount, "update reactor should fire thrice")
		assert.True(t, podCreated, "pod should be created")
	})

	t.Run("When a GameServer has been deleted, the sync operation should be a noop", func(t *testing.T) {
		runReconcileDeleteGameServer(t, &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec:   newSingleContainerSpec(),
			Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}})
	})
}

func runReconcileDeleteGameServer(t *testing.T, fixture *agonesv1.GameServer) {
	c, mocks := newFakeController()
	agonesWatch := watch.NewFake()
	podAction := false

	mocks.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(agonesWatch, nil))
	mocks.KubeClient.AddReactor("*", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if action.GetVerb() == "update" || action.GetVerb() == "delete" || action.GetVerb() == "create" || action.GetVerb() == "patch" {
			podAction = true
		}
		return false, nil, nil
	})

	ctx, cancel := agtesting.StartInformers(mocks, c.gameServerSynced)
	defer cancel()

	agonesWatch.Delete(fixture)

	err := c.syncGameServer(ctx, "default/test")
	assert.Nil(t, err, fmt.Sprintf("Shouldn't be an error from syncGameServer: %+v", err))
	assert.False(t, podAction, "Nothing should happen to a Pod")
}

func TestControllerSyncGameServerWithDevIP(t *testing.T) {
	t.Parallel()

	templateDevGs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test",
			Namespace:   "default",
			Annotations: map[string]string{agonesv1.DevAddressAnnotation: ipFixture},
		},
		Spec: agonesv1.GameServerSpec{
			Ports: []agonesv1.GameServerPort{{ContainerPort: 7777, HostPort: 7777, PortPolicy: agonesv1.Static}},
		},
	}

	t.Run("Creating a new GameServer", func(t *testing.T) {
		c, mocks := newFakeController()
		updateCount := 0

		fixture := templateDevGs.DeepCopy()

		fixture.ApplyDefaults()

		mocks.KubeClient.AddReactor("list", "nodes", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return false, nil, k8serrors.NewMethodNotSupported(schema.GroupResource{}, "list nodes should not be called")
		})
		mocks.KubeClient.AddReactor("create", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return false, nil, k8serrors.NewMethodNotSupported(schema.GroupResource{}, "creating a pod with dev mode is not supported")
		})
		mocks.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			gameServers := &agonesv1.GameServerList{Items: []agonesv1.GameServer{*fixture}}
			return true, gameServers, nil
		})
		mocks.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			updateCount++
			expectedState := agonesv1.GameServerStateReady

			assert.Equal(t, expectedState, gs.Status.State)
			assert.Equal(t, ipFixture, gs.Status.Address)
			assert.Equal(t, []corev1.NodeAddress{{Address: ipFixture, Type: "InternalIP"}}, gs.Status.Addresses)
			assert.NotEmpty(t, gs.Status.Ports[0].Port)

			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(mocks, c.gameServerSynced)
		defer cancel()

		err := c.portAllocator.Run(ctx)
		assert.Nil(t, err)

		err = c.syncGameServer(ctx, "default/test")
		assert.Nil(t, err)
		assert.Equal(t, 1, updateCount, "update reactor should fire once")
	})

	t.Run("GameServer with ReadyRequest State", func(t *testing.T) {
		c, mocks := newFakeController()

		updateCount := 0

		gsFixture := templateDevGs.DeepCopy()
		gsFixture.ApplyDefaults()
		gsFixture.Status.State = agonesv1.GameServerStateRequestReady

		mocks.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			gameServers := &agonesv1.GameServerList{Items: []agonesv1.GameServer{*gsFixture}}
			return true, gameServers, nil
		})
		mocks.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			updateCount++

			assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)

			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(mocks, c.gameServerSynced)
		defer cancel()

		err := c.portAllocator.Run(ctx)
		assert.NoError(t, err, "should not error")

		err = c.syncGameServer(ctx, "default/test")
		assert.NoError(t, err, "should not error")
		assert.Equal(t, 1, updateCount, "update reactor should fire once")
	})

	t.Run("Allocated GameServer", func(t *testing.T) {
		c, mocks := newFakeController()

		fixture := templateDevGs.DeepCopy()

		fixture.ApplyDefaults()
		fixture.Status.State = agonesv1.GameServerStateAllocated

		mocks.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			gameServers := &agonesv1.GameServerList{Items: []agonesv1.GameServer{*fixture}}
			return true, gameServers, nil
		})
		mocks.AgonesClient.AddReactor("update", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			require.Fail(t, "should not update")
			return true, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(mocks, c.gameServerSynced)
		defer cancel()

		err := c.portAllocator.Run(ctx)
		require.NoError(t, err)

		err = c.syncGameServer(ctx, "default/test")
		require.NoError(t, err)
	})

	t.Run("When a GameServer has been deleted, the sync operation should be a noop", func(t *testing.T) {
		runReconcileDeleteGameServer(t, &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test",
				Namespace:   "default",
				Annotations: map[string]string{agonesv1.DevAddressAnnotation: ipFixture},
			},
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{{ContainerPort: 7777, HostPort: 7777, PortPolicy: agonesv1.Static}},
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "container", Image: "container/image"}},
				},
				},
			},
		})
	})
}

func TestControllerWatchGameServers(t *testing.T) {
	c, m := newFakeController()
	fixture := agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"}, Spec: newSingleContainerSpec()}
	fixture.ApplyDefaults()
	pod, err := fixture.Pod(agtesting.FakeAPIHooks{})
	assert.Nil(t, err)
	pod.ObjectMeta.Name = pod.ObjectMeta.GenerateName + "-pod"

	gsWatch := watch.NewFake()
	podWatch := watch.NewFake()
	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
	m.KubeClient.AddWatchReactor("pods", k8stesting.DefaultWatchReactor(podWatch, nil))
	m.ExtClient.AddReactor("get", "customresourcedefinitions", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, agtesting.NewEstablishedCRD(), nil
	})

	received := make(chan string)
	defer close(received)

	h := func(_ context.Context, name string) error {
		assert.Equal(t, "default/test", name)
		received <- name
		return nil
	}

	c.workerqueue.SyncHandler = h
	c.creationWorkerQueue.SyncHandler = h
	c.deletionWorkerQueue.SyncHandler = h

	ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced)
	defer cancel()

	noStateChange := func(sync cache.InformerSynced) {
		cache.WaitForCacheSync(ctx.Done(), sync)
		select {
		case <-received:
			assert.Fail(t, "Should not be queued")
		default:
		}
	}

	podSynced := m.KubeInformerFactory.Core().V1().Pods().Informer().HasSynced
	gsSynced := m.AgonesInformerFactory.Agones().V1().GameServers().Informer().HasSynced

	go func() {
		err := c.Run(ctx, 1)
		assert.Nil(t, err, "Run should not error")
	}()

	logrus.Info("Adding first fixture")
	gsWatch.Add(&fixture)
	assert.Equal(t, "default/test", <-received)
	podWatch.Add(pod)
	noStateChange(podSynced)

	// no state change
	gsWatch.Modify(&fixture)
	noStateChange(gsSynced)

	// add a non game pod
	nonGamePod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "default"}}
	podWatch.Add(nonGamePod)
	noStateChange(podSynced)

	// no state change
	gsWatch.Modify(&fixture)
	noStateChange(gsSynced)

	// no state change
	gsWatch.Modify(&fixture)
	noStateChange(gsSynced)

	copyFixture := fixture.DeepCopy()
	copyFixture.Status.State = agonesv1.GameServerStateStarting
	logrus.Info("modify copyFixture")
	gsWatch.Modify(copyFixture)
	assert.Equal(t, "default/test", <-received)

	// modify a gameserver with a deletion timestamp
	now := metav1.Now()
	deleted := copyFixture.DeepCopy()
	deleted.ObjectMeta.DeletionTimestamp = &now
	gsWatch.Modify(deleted)
	assert.Equal(t, "default/test", <-received)

	podWatch.Delete(pod)
	assert.Equal(t, "default/test", <-received)

	// add an unscheduled game pod
	pod, err = fixture.Pod(agtesting.FakeAPIHooks{})
	assert.Nil(t, err)
	pod.ObjectMeta.Name = pod.ObjectMeta.GenerateName + "-pod2"
	podWatch.Add(pod)
	noStateChange(podSynced)

	// schedule it
	podCopy := pod.DeepCopy()
	podCopy.Spec.NodeName = nodeFixtureName

	podWatch.Modify(podCopy)
	assert.Equal(t, "default/test", <-received)
}

func TestControllerCreationMutationHandler(t *testing.T) {
	t.Parallel()

	type expected struct {
		responseAllowed bool
		patches         []jsonpatch.JsonPatchOperation
		nilPatch        bool
	}

	var testCases = []struct {
		description string
		fixture     interface{}
		expected    expected
	}{
		{
			description: "OK",
			fixture: &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
				Spec: newSingleContainerSpec()},
			expected: expected{
				responseAllowed: true,
				patches: []jsonpatch.JsonPatchOperation{
					{Operation: "add", Path: "/metadata/finalizers", Value: []interface{}{"agones.dev/controller"}},
					{Operation: "add", Path: "/spec/ports/0/protocol", Value: "UDP"}},
			},
		},
		{
			description: "Wrong request object, err expected",
			fixture:     "WRONG DATA",
			expected:    expected{nilPatch: true},
		},
	}

	ext := newFakeExtensions()

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			raw, err := json.Marshal(tc.fixture)
			require.NoError(t, err)

			review := admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Kind:      GameServerKind,
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: raw,
					},
				},
				Response: &admissionv1.AdmissionResponse{Allowed: true},
			}

			result, err := ext.creationMutationHandler(review)

			assert.NoError(t, err)
			if tc.expected.nilPatch {
				require.Nil(t, result.Response.PatchType)
			} else {
				assert.True(t, result.Response.Allowed)
				assert.Equal(t, admissionv1.PatchTypeJSONPatch, *result.Response.PatchType)

				patch := &jsonpatch.ByPath{}
				err = json.Unmarshal(result.Response.Patch, patch)
				require.NoError(t, err)

				found := false

				for _, expected := range tc.expected.patches {
					for _, p := range *patch {
						if assert.ObjectsAreEqual(p, expected) {
							found = true
						}
					}
					assert.True(t, found, "Could not find operation %#v in patch %v", expected, *patch)
				}
			}
		})
	}
}

func TestControllerCreationValidationHandler(t *testing.T) {
	t.Parallel()

	ext := newFakeExtensions()

	t.Run("valid gameserver", func(t *testing.T) {
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec()}
		fixture.ApplyDefaults()

		raw, err := json.Marshal(fixture)
		require.NoError(t, err)
		review := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Kind:      GameServerKind,
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw: raw,
				},
			},
			Response: &admissionv1.AdmissionResponse{Allowed: true},
		}

		result, err := ext.creationValidationHandler(review)
		require.NoError(t, err)
		assert.True(t, result.Response.Allowed)
	})

	t.Run("invalid gameserver", func(t *testing.T) {
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: agonesv1.GameServerSpec{
				Container: "NOPE!",
				Ports:     []agonesv1.GameServerPort{{ContainerPort: 7777}},
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "container", Image: "container/image"},
							{Name: "container2", Image: "container/image"},
						},
					},
				},
			},
		}
		raw, err := json.Marshal(fixture)
		require.NoError(t, err)
		review := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Kind:      GameServerKind,
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw: raw,
				},
			},
			Response: &admissionv1.AdmissionResponse{Allowed: true},
		}

		result, err := ext.creationValidationHandler(review)
		require.NoError(t, err)
		assert.False(t, result.Response.Allowed)
		assert.Equal(t, metav1.StatusFailure, review.Response.Result.Status)
		assert.Equal(t, metav1.StatusReasonInvalid, review.Response.Result.Reason)
		assert.Equal(t, review.Request.Kind.Kind, result.Response.Result.Details.Kind)
		assert.Equal(t, review.Request.Kind.Group, result.Response.Result.Details.Group)
		assert.NotEmpty(t, result.Response.Result.Details.Causes)
	})

	t.Run("valid request object, error expected", func(t *testing.T) {
		raw, err := json.Marshal("WRONG DATA")
		require.NoError(t, err)

		review := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Kind:      GameServerKind,
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw: raw,
				},
			},
			Response: &admissionv1.AdmissionResponse{Allowed: true},
		}

		_, err = ext.creationValidationHandler(review)
		if assert.Error(t, err) {
			assert.Equal(t, `error unmarshalling GameServer json after schema validation: "WRONG DATA": json: cannot unmarshal string into Go value of type v1.GameServer`, err.Error())
		}
	})
}

func TestControllerCreationMutationHandlerPod(t *testing.T) {
	t.Parallel()
	ext := newFakeExtensions()

	type expected struct {
		patches []jsonpatch.JsonPatchOperation
	}

	t.Run("valid pod mutation for Passthrough portPolicy, containerPort should be the same as hostPort", func(t *testing.T) {
		gameServerHostPort0 := float64(newPassthroughPortSingleContainerSpec().Spec.Containers[1].Ports[0].HostPort)
		gameServerHostPort2 := float64(newPassthroughPortSingleContainerSpec().Spec.Containers[1].Ports[1].HostPort)
		gameServerHostPort3 := float64(newPassthroughPortSingleContainerSpec().Spec.Containers[2].Ports[0].HostPort)
		fixture := newPassthroughPortSingleContainerSpec()
		raw, err := json.Marshal(fixture)
		require.NoError(t, err)
		review := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Kind:      metav1.GroupVersionKind(PodKind),
				Operation: admissionv1.Create,
				Object: runtime.RawExtension{
					Raw: raw,
				},
			},
			Response: &admissionv1.AdmissionResponse{Allowed: true},
		}
		expected := expected{
			patches: []jsonpatch.JsonPatchOperation{
				{Operation: "replace", Path: "/spec/containers/1/ports/0/containerPort", Value: gameServerHostPort0},
				{Operation: "replace", Path: "/spec/containers/1/ports/1/containerPort", Value: gameServerHostPort2},
				{Operation: "replace", Path: "/spec/containers/2/ports/0/containerPort", Value: gameServerHostPort3}},
		}

		result, err := ext.creationMutationHandlerPod(review)
		assert.NoError(t, err)
		patch := &jsonpatch.ByPath{}
		err = json.Unmarshal(result.Response.Patch, patch)
		found := false

		for _, expected := range expected.patches {
			for _, p := range *patch {
				if assert.ObjectsAreEqual(p, expected) {
					found = true
				}
			}
			assert.True(t, found, "Could not find operation %#v in patch %v", expected, *patch)
		}

		require.NoError(t, err)

	})
}

func TestControllerSyncGameServerDeletionTimestamp(t *testing.T) {
	t.Parallel()

	t.Run("GameServer has a Pod", func(t *testing.T) {
		c, mocks := newFakeController()
		now := metav1.Now()
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", DeletionTimestamp: &now},
			Spec: newSingleContainerSpec()}
		fixture.ApplyDefaults()
		pod, err := fixture.Pod(agtesting.FakeAPIHooks{})
		assert.Nil(t, err)

		deleted := false
		mocks.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		})
		mocks.KubeClient.AddReactor("delete", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			deleted = true
			da := action.(k8stesting.DeleteAction)
			assert.Equal(t, pod.ObjectMeta.Name, da.GetName())
			return true, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(mocks, c.podSynced)
		defer cancel()

		result, err := c.syncGameServerDeletionTimestamp(ctx, fixture)
		assert.NoError(t, err)
		assert.True(t, deleted, "pod should be deleted")
		assert.Equal(t, fixture, result)
		agtesting.AssertEventContains(t, mocks.FakeRecorder.Events, fmt.Sprintf("%s %s %s", corev1.EventTypeNormal,
			fixture.Status.State, "Deleting Pod "+pod.ObjectMeta.Name))
	})

	t.Run("Error on deleting pod", func(t *testing.T) {
		c, mocks := newFakeController()
		now := metav1.Now()
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", DeletionTimestamp: &now},
			Spec: newSingleContainerSpec()}
		fixture.ApplyDefaults()
		pod, err := fixture.Pod(agtesting.FakeAPIHooks{})
		assert.Nil(t, err)

		mocks.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		})
		mocks.KubeClient.AddReactor("delete", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("Delete-err")
		})

		ctx, cancel := agtesting.StartInformers(mocks, c.podSynced)
		defer cancel()

		_, err = c.syncGameServerDeletionTimestamp(ctx, fixture)
		if assert.Error(t, err) {
			assert.Equal(t, `error deleting pod for GameServer. Name: test, Namespace: default: Delete-err`, err.Error())
		}
	})

	t.Run("GameServer's Pods have been deleted", func(t *testing.T) {
		c, mocks := newFakeController()
		now := metav1.Now()
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", DeletionTimestamp: &now},
			Spec: newSingleContainerSpec()}
		fixture.ApplyDefaults()

		updated := false
		mocks.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true

			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, fixture.ObjectMeta.Name, gs.ObjectMeta.Name)
			assert.Empty(t, gs.ObjectMeta.Finalizers)

			return true, gs, nil
		})
		ctx, cancel := agtesting.StartInformers(mocks, c.gameServerSynced)
		defer cancel()

		result, err := c.syncGameServerDeletionTimestamp(ctx, fixture)
		assert.Nil(t, err)
		assert.True(t, updated, "gameserver should be updated, to remove the finaliser")
		assert.Equal(t, fixture.ObjectMeta.Name, result.ObjectMeta.Name)
		assert.Empty(t, result.ObjectMeta.Finalizers)
	})

	t.Run("Local development GameServer", func(t *testing.T) {
		c, mocks := newFakeController()
		now := metav1.Now()
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default",
			Annotations:       map[string]string{agonesv1.DevAddressAnnotation: "1.1.1.1"},
			DeletionTimestamp: &now},
			Spec: newSingleContainerSpec()}
		fixture.ApplyDefaults()

		updated := false
		mocks.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true

			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, fixture.ObjectMeta.Name, gs.ObjectMeta.Name)
			assert.Empty(t, gs.ObjectMeta.Finalizers)

			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(mocks, c.gameServerSynced)
		defer cancel()

		result, err := c.syncGameServerDeletionTimestamp(ctx, fixture)
		assert.Nil(t, err)
		assert.True(t, updated, "gameserver should be updated, to remove the finaliser")
		assert.Equal(t, fixture.ObjectMeta.Name, result.ObjectMeta.Name)
		assert.Empty(t, result.ObjectMeta.Finalizers)
	})
}

func TestControllerSyncGameServerPortAllocationState(t *testing.T) {
	t.Parallel()

	t.Run("Gameserver with port allocation state", func(t *testing.T) {
		t.Parallel()
		c, mocks := newFakeController()
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{{ContainerPort: 7777}},
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "container", Image: "container/image"}},
					},
				},
			},
			Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStatePortAllocation},
		}
		fixture.ApplyDefaults()
		mocks.KubeClient.AddReactor("list", "nodes", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.NodeList{Items: []corev1.Node{{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName}}}}, nil
		})

		updated := false

		mocks.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			updated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, fixture.ObjectMeta.Name, gs.ObjectMeta.Name)
			port := gs.Spec.Ports[0]
			assert.Equal(t, agonesv1.Dynamic, port.PortPolicy)
			assert.NotEqual(t, fixture.Spec.Ports[0].HostPort, port.HostPort)
			assert.True(t, 10 <= port.HostPort && port.HostPort <= 20, "%s not in range", port.HostPort)

			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(mocks, c.gameServerSynced)
		defer cancel()
		err := c.portAllocator.Run(ctx)
		require.NoError(t, err)

		result, err := c.syncGameServerPortAllocationState(ctx, fixture)
		require.NoError(t, err, "sync should not error")
		assert.True(t, updated, "update should occur")
		port := result.Spec.Ports[0]
		assert.Equal(t, agonesv1.Dynamic, port.PortPolicy)
		assert.NotEqual(t, fixture.Spec.Ports[0].HostPort, port.HostPort)
		assert.True(t, 10 <= port.HostPort && port.HostPort <= 20, "%s not in range", port.HostPort)
	})

	t.Run("Error on update", func(t *testing.T) {
		t.Parallel()
		c, mocks := newFakeController()
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{{ContainerPort: 7777}},
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "container", Image: "container/image"}},
					},
				},
			},
			Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStatePortAllocation},
		}
		fixture.ApplyDefaults()
		mocks.KubeClient.AddReactor("list", "nodes", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.NodeList{Items: []corev1.Node{{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName}}}}, nil
		})

		mocks.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			return true, gs, errors.New("update-err")
		})

		ctx, cancel := agtesting.StartInformers(mocks, c.gameServerSynced)
		defer cancel()
		err := c.portAllocator.Run(ctx)
		require.NoError(t, err)

		_, err = c.syncGameServerPortAllocationState(ctx, fixture)
		if assert.Error(t, err) {
			assert.Equal(t, `error updating GameServer test to default values: update-err`, err.Error())
		}
	})

	t.Run("Gameserver with unknown state", func(t *testing.T) {
		testNoChange(t, "Unknown", func(c *Controller, fixture *agonesv1.GameServer) (*agonesv1.GameServer, error) {
			return c.syncGameServerPortAllocationState(context.Background(), fixture)
		})
	})

	t.Run("GameServer with non zero deletion datetime", func(t *testing.T) {
		testWithNonZeroDeletionTimestamp(t, func(c *Controller, fixture *agonesv1.GameServer) (*agonesv1.GameServer, error) {
			return c.syncGameServerPortAllocationState(context.Background(), fixture)
		})
	})
}

func TestControllerSyncGameServerCreatingState(t *testing.T) {
	t.Parallel()

	newFixture := func() *agonesv1.GameServer {
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateCreating}}
		fixture.ApplyDefaults()
		return fixture
	}

	t.Run("Testing TCPUDP protocol of static portpolicy", func(t *testing.T) {
		c, m := newFakeController()
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateCreating}}
		fixture.Spec.Ports[0].Name = "default"
		fixture.Spec.Ports[0].HostPort = 7000
		fixture.Spec.Ports[0].Protocol = agonesv1.ProtocolTCPUDP
		fixture.ApplyDefaults()
		podCreated := false
		gsUpdated := false

		var pod *corev1.Pod
		m.KubeClient.AddReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			podCreated = true
			ca := action.(k8stesting.CreateAction)
			pod = ca.GetObject().(*corev1.Pod)
			assert.True(t, metav1.IsControlledBy(pod, fixture))
			return true, pod, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateStarting, gs.Status.State)
			assert.Len(t, gs.Spec.Ports, 2)
			assert.Equal(t, "default-tcp", gs.Spec.Ports[0].Name)
			assert.Equal(t, corev1.ProtocolTCP, gs.Spec.Ports[0].Protocol)
			assert.Equal(t, "default-udp", gs.Spec.Ports[1].Name)
			assert.Equal(t, corev1.ProtocolUDP, gs.Spec.Ports[1].Protocol)
			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.podSynced)
		defer cancel()

		gs, err := c.syncGameServerCreatingState(ctx, fixture)

		assert.NoError(t, err)
		assert.True(t, podCreated, "Pod should have been created")

		assert.Equal(t, agonesv1.GameServerStateStarting, gs.Status.State)
		assert.True(t, gsUpdated, "GameServer should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Pod")
	})

	t.Run("Testing TCP protocol of static portpolicy", func(t *testing.T) {
		c, m := newFakeController()
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateCreating}}
		fixture.Spec.Ports[0].Name = "tcp-port"
		fixture.Spec.Ports[0].HostPort = 7000
		fixture.Spec.Ports[0].Protocol = corev1.ProtocolTCP
		fixture.ApplyDefaults()
		podCreated := false
		gsUpdated := false

		var pod *corev1.Pod
		m.KubeClient.AddReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			podCreated = true
			ca := action.(k8stesting.CreateAction)
			pod = ca.GetObject().(*corev1.Pod)
			assert.True(t, metav1.IsControlledBy(pod, fixture))
			return true, pod, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateStarting, gs.Status.State)
			assert.Len(t, gs.Spec.Ports, 1)
			assert.Equal(t, "tcp-port", gs.Spec.Ports[0].Name)
			assert.Equal(t, corev1.ProtocolTCP, gs.Spec.Ports[0].Protocol)
			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.podSynced)
		defer cancel()

		gs, err := c.syncGameServerCreatingState(ctx, fixture)

		assert.NoError(t, err)
		assert.True(t, podCreated, "Pod should have been created")

		assert.Equal(t, agonesv1.GameServerStateStarting, gs.Status.State)
		assert.True(t, gsUpdated, "GameServer should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Pod")
	})

	t.Run("Testing default protocol of static portpolicy", func(t *testing.T) {
		c, m := newFakeController()
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateCreating}}
		fixture.Spec.Ports[0].Name = "udp-port"
		fixture.Spec.Ports[0].HostPort = 7000
		fixture.ApplyDefaults()
		podCreated := false
		gsUpdated := false

		var pod *corev1.Pod
		m.KubeClient.AddReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			podCreated = true
			ca := action.(k8stesting.CreateAction)
			pod = ca.GetObject().(*corev1.Pod)
			assert.True(t, metav1.IsControlledBy(pod, fixture))
			return true, pod, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateStarting, gs.Status.State)
			assert.Len(t, gs.Spec.Ports, 1)
			assert.Equal(t, "udp-port", gs.Spec.Ports[0].Name)
			assert.Equal(t, corev1.ProtocolUDP, gs.Spec.Ports[0].Protocol)
			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.podSynced)
		defer cancel()

		gs, err := c.syncGameServerCreatingState(ctx, fixture)

		assert.NoError(t, err)
		assert.True(t, podCreated, "Pod should have been created")

		assert.Equal(t, agonesv1.GameServerStateStarting, gs.Status.State)
		assert.True(t, gsUpdated, "GameServer should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Pod")
	})

	t.Run("Syncing from Created State, with no issues", func(t *testing.T) {
		c, m := newFakeController()
		fixture := newFixture()
		podCreated := false
		gsUpdated := false

		var pod *corev1.Pod
		m.KubeClient.AddReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			podCreated = true
			ca := action.(k8stesting.CreateAction)
			pod = ca.GetObject().(*corev1.Pod)
			assert.True(t, metav1.IsControlledBy(pod, fixture))
			return true, pod, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateStarting, gs.Status.State)
			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.podSynced)
		defer cancel()

		gs, err := c.syncGameServerCreatingState(ctx, fixture)

		assert.NoError(t, err)
		assert.True(t, podCreated, "Pod should have been created")

		assert.Equal(t, agonesv1.GameServerStateStarting, gs.Status.State)
		assert.True(t, gsUpdated, "GameServer should have been updated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Pod")
	})

	t.Run("Error on updating gs", func(t *testing.T) {
		c, m := newFakeController()
		fixture := newFixture()
		podCreated := false

		var pod *corev1.Pod
		m.KubeClient.AddReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			podCreated = true
			ca := action.(k8stesting.CreateAction)
			pod = ca.GetObject().(*corev1.Pod)
			assert.True(t, metav1.IsControlledBy(pod, fixture))
			return true, pod, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateStarting, gs.Status.State)
			return true, gs, errors.New("update-err")
		})

		ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.podSynced)
		defer cancel()

		_, err := c.syncGameServerCreatingState(ctx, fixture)
		require.True(t, podCreated, "Pod should have been created")

		if assert.Error(t, err) {
			assert.Equal(t, `error updating GameServer test to Starting state: update-err`, err.Error())
		}
	})

	t.Run("Previously started sync, created Pod, but didn't move to Starting", func(t *testing.T) {
		c, m := newFakeController()
		fixture := newFixture()
		podCreated := false
		gsUpdated := false
		pod, err := fixture.Pod(agtesting.FakeAPIHooks{})
		assert.Nil(t, err)

		m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		})
		m.KubeClient.AddReactor("create", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			podCreated = true
			return true, nil, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateStarting, gs.Status.State)
			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.podSynced)
		defer cancel()

		gs, err := c.syncGameServerCreatingState(ctx, fixture)
		assert.Nil(t, err)
		assert.Equal(t, agonesv1.GameServerStateStarting, gs.Status.State)
		assert.False(t, podCreated, "Pod should not have been created")
		assert.True(t, gsUpdated, "GameServer should have been updated")
	})

	t.Run("creates an invalid podspec", func(t *testing.T) {
		c, mocks := newFakeController()
		fixture := newFixture()
		podCreated := false
		gsUpdated := false

		mocks.KubeClient.AddReactor("create", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			podCreated = true
			return true, nil, k8serrors.NewInvalid(schema.GroupKind{}, "test", field.ErrorList{})
		})
		mocks.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateError, gs.Status.State)
			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(mocks, c.gameServerSynced, c.podSynced)
		defer cancel()

		gs, err := c.syncGameServerCreatingState(ctx, fixture)
		assert.Nil(t, err)

		assert.True(t, podCreated, "attempt should have been made to create a pod")
		assert.True(t, gsUpdated, "GameServer should be updated")
		assert.Equal(t, agonesv1.GameServerStateError, gs.Status.State)
	})

	t.Run("GameServer with unknown state", func(t *testing.T) {
		testNoChange(t, "Unknown", func(c *Controller, fixture *agonesv1.GameServer) (*agonesv1.GameServer, error) {
			return c.syncGameServerCreatingState(context.Background(), fixture)
		})
	})

	t.Run("GameServer with non zero deletion datetime", func(t *testing.T) {
		testWithNonZeroDeletionTimestamp(t, func(c *Controller, fixture *agonesv1.GameServer) (*agonesv1.GameServer, error) {
			return c.syncGameServerCreatingState(context.Background(), fixture)
		})
	})
}

func TestControllerSyncGameServerStartingState(t *testing.T) {
	t.Parallel()

	newFixture := func() *agonesv1.GameServer {
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateStarting}}
		fixture.ApplyDefaults()
		return fixture
	}

	node := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName}, Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: ipFixture, Type: corev1.NodeExternalIP}}}}

	t.Run("sync from Stating state, with no issues", func(t *testing.T) {
		c, m := newFakeController()
		gsFixture := newFixture()
		gsFixture.ApplyDefaults()
		pod, err := gsFixture.Pod(agtesting.FakeAPIHooks{})
		assert.Nil(t, err)
		pod.Spec.NodeName = nodeFixtureName
		pod.Status.PodIPs = []corev1.PodIP{{IP: ipv6Fixture}}
		gsUpdated := false

		m.KubeClient.AddReactor("list", "nodes", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.NodeList{Items: []corev1.Node{node}}, nil
		})
		m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateScheduled, gs.Status.State)
			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.podSynced, c.nodeSynced)
		defer cancel()

		gs, err := c.syncGameServerStartingState(ctx, gsFixture)
		require.NoError(t, err)

		assert.True(t, gsUpdated)
		assert.Equal(t, gs.Status.NodeName, node.ObjectMeta.Name)
		assert.Equal(t, gs.Status.Address, ipFixture)
		assert.Equal(t, []corev1.NodeAddress{
			{Address: ipFixture, Type: "ExternalIP"},
			{Address: ipv6Fixture, Type: "PodIP"},
		}, gs.Status.Addresses)

		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Address and port populated")
		assert.NotEmpty(t, gs.Status.Ports)
	})

	t.Run("Error on podIPs not populated", func(t *testing.T) {
		c, m := newFakeController()
		gsFixture := newFixture()
		gsFixture.ApplyDefaults()
		pod, err := gsFixture.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)
		pod.Spec.NodeName = nodeFixtureName

		m.KubeClient.AddReactor("list", "nodes", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.NodeList{Items: []corev1.Node{node}}, nil
		})
		m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateScheduled, gs.Status.State)
			return true, gs, errors.New("update-err")
		})
		ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.podSynced, c.nodeSynced)
		defer cancel()

		_, err = c.syncGameServerStartingState(ctx, gsFixture)
		assert.Error(t, err)
	})

	t.Run("Error on update", func(t *testing.T) {
		c, m := newFakeController()
		gsFixture := newFixture()
		gsFixture.ApplyDefaults()
		pod, err := gsFixture.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)
		pod.Spec.NodeName = nodeFixtureName
		pod.Status.PodIPs = []corev1.PodIP{{IP: ipv6Fixture}}

		m.KubeClient.AddReactor("list", "nodes", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.NodeList{Items: []corev1.Node{node}}, nil
		})
		m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateScheduled, gs.Status.State)
			return true, gs, errors.New("update-err")
		})
		ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.podSynced, c.nodeSynced)
		defer cancel()

		_, err = c.syncGameServerStartingState(ctx, gsFixture)
		if assert.Error(t, err) {
			assert.Equal(t, `error updating GameServer test to Scheduled state: update-err`, err.Error())
		}
	})

	t.Run("GameServer with unknown state", func(t *testing.T) {
		testNoChange(t, "Unknown", func(c *Controller, fixture *agonesv1.GameServer) (*agonesv1.GameServer, error) {
			return c.syncGameServerStartingState(context.Background(), fixture)
		})
	})

	t.Run("GameServer with non zero deletion datetime", func(t *testing.T) {
		testWithNonZeroDeletionTimestamp(t, func(c *Controller, fixture *agonesv1.GameServer) (*agonesv1.GameServer, error) {
			return c.syncGameServerStartingState(context.Background(), fixture)
		})
	})
}

func TestControllerCreateGameServerPod(t *testing.T) {
	t.Parallel()

	// TODO: remove mutex when "SidecarContainers" moves to stable.
	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	newFixture := func() *agonesv1.GameServer {
		fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateCreating}}
		fixture.ApplyDefaults()
		return fixture
	}

	t.Run("create pod, with no issues", func(t *testing.T) {
		c, m := newFakeController()
		fixture := newFixture()
		created := false

		m.KubeClient.AddReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			created = true
			ca := action.(k8stesting.CreateAction)
			pod := ca.GetObject().(*corev1.Pod)

			assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Name)
			assert.Equal(t, fixture.ObjectMeta.Namespace, pod.ObjectMeta.Namespace)
			assert.Equal(t, "sdk-service-account", pod.Spec.ServiceAccountName)
			assert.Equal(t, "gameserver", pod.ObjectMeta.Labels[agones.GroupName+"/role"])
			assert.Equal(t, fixture.ObjectMeta.Name, pod.ObjectMeta.Labels[agonesv1.GameServerPodLabel])
			assert.True(t, metav1.IsControlledBy(pod, fixture))

			// gke.MutateGameServerPod assumes that a non-empty NodeSelector / Tolerations are user
			// intent. The generic cloudproduct that we use in unit tests does not manipulate these.
			// So we verify using the generic cloudproduct that NodeSelector/Tolerations are empty
			// as a change detector - if this test fails, gke.MutateGameServerPod will not work.
			assert.Empty(t, pod.Spec.NodeSelector)
			assert.Empty(t, pod.Spec.Tolerations)

			// if sidecar feature enabled, should be 1 container and 1 initContainer, otherwise 2 containers
			var sidecarContainer corev1.Container
			var gsContainer corev1.Container
			if agruntime.FeatureEnabled(agruntime.FeatureSidecarContainers) {
				assert.Len(t, pod.Spec.Containers, 1, "Should have 1 container")
				assert.Len(t, pod.Spec.InitContainers, 1, "Should have init container")
				gsContainer = pod.Spec.Containers[0]
				sidecarContainer = pod.Spec.InitContainers[0]
			} else {
				assert.Len(t, pod.Spec.Containers, 2, "Should have a sidecar container")
				sidecarContainer = pod.Spec.Containers[0]
				gsContainer = pod.Spec.Containers[1]
			}

			assert.Equal(t, sidecarContainer.Image, c.sidecarImage)
			assert.Equal(t, sidecarContainer.Resources.Limits.Cpu(), &c.sidecarCPULimit)
			assert.Equal(t, sidecarContainer.Resources.Requests.Cpu(), &c.sidecarCPURequest)
			assert.Equal(t, sidecarContainer.Resources.Limits.Memory(), &c.sidecarMemoryLimit)
			assert.Equal(t, sidecarContainer.Resources.Requests.Memory(), &c.sidecarMemoryRequest)
			assert.Len(t, sidecarContainer.Env, 5, "5 env vars")
			assert.Equal(t, "GAMESERVER_NAME", sidecarContainer.Env[0].Name)
			assert.Equal(t, fixture.ObjectMeta.Name, sidecarContainer.Env[0].Value)
			assert.Equal(t, "POD_NAMESPACE", sidecarContainer.Env[1].Name)
			assert.Equal(t, "FEATURE_GATES", sidecarContainer.Env[2].Name)
			assert.Equal(t, "LOG_LEVEL", sidecarContainer.Env[3].Name)
			assert.Equal(t, "REQUESTS_RATE_LIMIT", sidecarContainer.Env[4].Name)
			assert.Equal(t, "500ms", sidecarContainer.Env[4].Value)
			assert.Equal(t, string(fixture.Spec.SdkServer.LogLevel), sidecarContainer.Env[3].Value)
			assert.Equal(t, *sidecarContainer.SecurityContext.AllowPrivilegeEscalation, false)
			assert.Equal(t, *sidecarContainer.SecurityContext.RunAsNonRoot, true)
			assert.Equal(t, *sidecarContainer.SecurityContext.RunAsUser, int64(sidecarRunAsUser))

			assert.Equal(t, fixture.Spec.Ports[0].HostPort, gsContainer.Ports[0].HostPort)
			assert.Equal(t, fixture.Spec.Ports[0].ContainerPort, gsContainer.Ports[0].ContainerPort)
			assert.Equal(t, corev1.Protocol("UDP"), gsContainer.Ports[0].Protocol)
			assert.Equal(t, "/gshealthz", gsContainer.LivenessProbe.HTTPGet.Path)
			assert.Equal(t, gsContainer.LivenessProbe.HTTPGet.Port, intstr.FromInt(8080))
			assert.Equal(t, intstr.FromInt(8080), gsContainer.LivenessProbe.HTTPGet.Port)
			assert.Equal(t, fixture.Spec.Health.InitialDelaySeconds, gsContainer.LivenessProbe.InitialDelaySeconds)
			assert.Equal(t, fixture.Spec.Health.PeriodSeconds, gsContainer.LivenessProbe.PeriodSeconds)
			assert.Equal(t, fixture.Spec.Health.FailureThreshold, gsContainer.LivenessProbe.FailureThreshold)
			assert.Len(t, gsContainer.VolumeMounts, 1)
			assert.Equal(t, "/var/run/secrets/kubernetes.io/serviceaccount", gsContainer.VolumeMounts[0].MountPath)

			return true, pod, nil
		})

		gs, err := c.createGameServerPod(context.Background(), fixture)
		require.NoError(t, err)
		assert.Equal(t, fixture.Status.State, gs.Status.State)
		assert.True(t, created)
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Pod")
	})

	t.Run("service account", func(t *testing.T) {
		c, m := newFakeController()
		fixture := newFixture()
		fixture.Spec.Template.Spec.ServiceAccountName = "foobar"

		created := false

		m.KubeClient.AddReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			created = true
			ca := action.(k8stesting.CreateAction)
			pod := ca.GetObject().(*corev1.Pod)
			if agruntime.FeatureEnabled(agruntime.FeatureSidecarContainers) {
				assert.Len(t, pod.Spec.InitContainers, 1, "Should have init container")
			} else {
				assert.Len(t, pod.Spec.Containers, 2, "Should have a sidecar container")
			}
			assert.Empty(t, pod.Spec.Containers[0].VolumeMounts)

			return true, pod, nil
		})

		_, err := c.createGameServerPod(context.Background(), fixture)
		assert.Nil(t, err)
		assert.True(t, created)
	})

	t.Run("invalid podspec", func(t *testing.T) {
		c, mocks := newFakeController()
		fixture := newFixture()
		podCreated := false
		gsUpdated := false

		mocks.KubeClient.AddReactor("create", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			podCreated = true
			return true, nil, k8serrors.NewInvalid(schema.GroupKind{}, "test", field.ErrorList{})
		})
		mocks.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateError, gs.Status.State)
			return true, gs, nil
		})

		gs, err := c.createGameServerPod(context.Background(), fixture)
		require.NoError(t, err)

		assert.True(t, podCreated, "attempt should have been made to create a pod")
		assert.True(t, gsUpdated, "GameServer should be updated")
		if assert.NotEmpty(t, gs.Annotations[agonesv1.GameServerErroredAtAnnotation]) {
			gotTime, err := time.Parse(time.RFC3339, gs.Annotations[agonesv1.GameServerErroredAtAnnotation])
			require.NoError(t, err)
			assert.WithinDuration(t, time.Now(), gotTime, time.Second)
		}
		assert.Equal(t, agonesv1.GameServerStateError, gs.Status.State)
	})

	t.Run("forbidden pods creation", func(t *testing.T) {
		c, mocks := newFakeController()
		fixture := newFixture()
		podCreated := false
		gsUpdated := false

		mocks.KubeClient.AddReactor("create", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			podCreated = true
			return true, nil, k8serrors.NewForbidden(schema.GroupResource{}, "test", errors.New("test"))
		})
		mocks.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateError, gs.Status.State)
			return true, gs, nil
		})

		gs, err := c.createGameServerPod(context.Background(), fixture)
		require.NoError(t, err)

		assert.True(t, podCreated, "attempt should have been made to create a pod")
		assert.True(t, gsUpdated, "GameServer should be updated")
		if assert.NotEmpty(t, gs.Annotations[agonesv1.GameServerErroredAtAnnotation]) {
			gotTime, err := time.Parse(time.RFC3339, gs.Annotations[agonesv1.GameServerErroredAtAnnotation])
			require.NoError(t, err)
			assert.WithinDuration(t, time.Now(), gotTime, time.Second)
		}
		assert.Equal(t, agonesv1.GameServerStateError, gs.Status.State)
	})
}

func TestControllerSyncGameServerRequestReadyState(t *testing.T) {
	t.Parallel()
	nodeName := "node"
	containerID := "1234"

	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()

	t.Run("GameServer with ReadyRequest State", func(t *testing.T) {
		c, m := newFakeController()

		gsFixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateRequestReady}}
		gsFixture.ApplyDefaults()
		gsFixture.Status.NodeName = nodeName
		pod, err := gsFixture.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				Name:        gsFixture.Spec.Container,
				State:       corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				ContainerID: containerID,
			},
		}

		gsUpdated := false
		podUpdated := false

		m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)

			// only set if not using sidecars.
			if !agruntime.FeatureEnabled(agruntime.FeatureSidecarContainers) {
				assert.Equal(t, containerID, gs.Annotations[agonesv1.GameServerReadyContainerIDAnnotation])
			}
			return true, gs, nil
		})
		m.KubeClient.AddReactor("update", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			podUpdated = true
			ua := action.(k8stesting.UpdateAction)
			pod := ua.GetObject().(*corev1.Pod)
			assert.Equal(t, containerID, pod.Annotations[agonesv1.GameServerReadyContainerIDAnnotation])
			return true, pod, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.podSynced)
		defer cancel()

		gs, err := c.syncGameServerRequestReadyState(ctx, gsFixture)
		assert.NoError(t, err, "should not error")
		assert.True(t, gsUpdated, "GameServer wasn't updated")
		if agruntime.FeatureEnabled(agruntime.FeatureSidecarContainers) {
			assert.False(t, podUpdated, "Pod was updated")
		} else {
			assert.True(t, podUpdated, "Pod was not updated")
		}
		assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "SDK.Ready() complete")
	})

	t.Run("Error on GameServer update", func(t *testing.T) {
		require.NoError(t, agruntime.ParseFeatures(string(agruntime.FeatureSidecarContainers)+"=false"))

		c, m := newFakeController()

		gsFixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateRequestReady}}
		gsFixture.ApplyDefaults()
		gsFixture.Status.NodeName = nodeName
		pod, err := gsFixture.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				Name:        gsFixture.Spec.Container,
				State:       corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				ContainerID: containerID,
			},
		}
		podUpdated := false

		m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
			assert.Equal(t, containerID, gs.Annotations[agonesv1.GameServerReadyContainerIDAnnotation])
			return true, gs, errors.New("update-err")
		})
		m.KubeClient.AddReactor("update", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			podUpdated = true
			ua := action.(k8stesting.UpdateAction)
			pod := ua.GetObject().(*corev1.Pod)
			assert.Equal(t, containerID, pod.Annotations[agonesv1.GameServerReadyContainerIDAnnotation])
			return true, pod, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.podSynced)
		defer cancel()

		_, err = c.syncGameServerRequestReadyState(ctx, gsFixture)
		assert.True(t, podUpdated, "pod was not updated")
		require.EqualError(t, err, "error setting Ready, Port and address on GameServer test Status: update-err")
	})

	t.Run("Error on pod update", func(t *testing.T) {
		c, m := newFakeController()

		gsFixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateRequestReady}}
		gsFixture.ApplyDefaults()
		gsFixture.Status.NodeName = nodeName
		pod, err := gsFixture.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				Name:        gsFixture.Spec.Container,
				State:       corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				ContainerID: containerID,
			},
		}
		gsUpdated := false
		podUpdated := false

		m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			return true, nil, nil
		})
		m.KubeClient.AddReactor("update", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			podUpdated = true
			ua := action.(k8stesting.UpdateAction)
			pod := ua.GetObject().(*corev1.Pod)
			assert.Equal(t, containerID, pod.Annotations[agonesv1.GameServerReadyContainerIDAnnotation])
			return true, pod, errors.New("pod-error")
		})

		ctx, cancel := agtesting.StartInformers(m, c.podSynced)
		defer cancel()

		_, err = c.syncGameServerRequestReadyState(ctx, gsFixture)
		assert.True(t, podUpdated, "pod was not updated")
		assert.False(t, gsUpdated, "GameServer was updated")
		require.EqualError(t, err, "error updating ready annotation on Pod: test: pod-error")
	})

	t.Run("Pod annotation already set", func(t *testing.T) {
		c, m := newFakeController()

		gsFixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateRequestReady}}
		gsFixture.ApplyDefaults()
		gsFixture.Status.NodeName = nodeName
		pod, err := gsFixture.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)
		pod.ObjectMeta.Annotations = map[string]string{agonesv1.GameServerReadyContainerIDAnnotation: containerID}
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				Name:        gsFixture.Spec.Container,
				State:       corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				ContainerID: containerID,
			},
		}
		gsUpdated := false
		podUpdated := false

		m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
			assert.Equal(t, containerID, gs.Annotations[agonesv1.GameServerReadyContainerIDAnnotation])
			return true, gs, nil
		})
		m.KubeClient.AddReactor("update", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			podUpdated = true
			return true, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.podSynced)
		defer cancel()

		gs, err := c.syncGameServerRequestReadyState(ctx, gsFixture)
		assert.NoError(t, err, "should not error")
		assert.True(t, gsUpdated, "GameServer wasn't updated")
		assert.False(t, podUpdated, "Pod was updated")
		assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "SDK.Ready() complete")

	})

	t.Run("GameServer without an Address, but RequestReady State", func(t *testing.T) {
		c, m := newFakeController()

		gsFixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateRequestReady}}
		gsFixture.ApplyDefaults()
		pod, err := gsFixture.Pod(agtesting.FakeAPIHooks{})
		assert.Nil(t, err)
		pod.Spec.NodeName = nodeFixtureName
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				Name:        gsFixture.Spec.Container,
				State:       corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				ContainerID: containerID,
			},
		}
		gsUpdated := false
		podUpdated := false

		ipFixture := "12.12.12.12"
		nodeFixture := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName}, Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: ipFixture, Type: corev1.NodeExternalIP}}}}

		m.KubeClient.AddReactor("list", "nodes", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.NodeList{Items: []corev1.Node{nodeFixture}}, nil
		})

		m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
			assert.Equal(t, containerID, gs.Annotations[agonesv1.GameServerReadyContainerIDAnnotation])
			return true, gs, nil
		})
		m.KubeClient.AddReactor("update", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			podUpdated = true
			ua := action.(k8stesting.UpdateAction)
			pod := ua.GetObject().(*corev1.Pod)
			assert.Equal(t, containerID, pod.Annotations[agonesv1.GameServerReadyContainerIDAnnotation])
			return true, pod, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.podSynced, c.nodeSynced)
		defer cancel()

		gs, err := c.syncGameServerRequestReadyState(ctx, gsFixture)
		assert.Nil(t, err, "should not error")
		assert.True(t, gsUpdated, "GameServer wasn't updated")
		assert.True(t, podUpdated, "Pod wasn't updated")
		assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)

		assert.Equal(t, gs.Status.NodeName, nodeFixture.ObjectMeta.Name)
		assert.Equal(t, gs.Status.Address, ipFixture)
		assert.Equal(t, []corev1.NodeAddress{{Address: ipFixture, Type: "ExternalIP"}}, gs.Status.Addresses)

		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "Address and port populated")
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "SDK.Ready() complete")
	})

	t.Run("GameServer with a GameServerReadyContainerIDAnnotation already", func(t *testing.T) {
		c, m := newFakeController()

		gsFixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateRequestReady}}
		gsFixture.ApplyDefaults()
		gsFixture.Status.NodeName = nodeName
		gsFixture.Annotations[agonesv1.GameServerReadyContainerIDAnnotation] = "4321"
		pod, err := gsFixture.Pod(agtesting.FakeAPIHooks{})
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{
			{
				Name:        gsFixture.Spec.Container,
				State:       corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				ContainerID: containerID,
			},
		}
		assert.Nil(t, err)
		gsUpdated := false
		podUpdated := false

		m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
			assert.NotEqual(t, containerID, gs.Annotations[agonesv1.GameServerReadyContainerIDAnnotation])

			return true, gs, nil
		})
		m.KubeClient.AddReactor("update", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
			podUpdated = true
			ua := action.(k8stesting.UpdateAction)
			pod := ua.GetObject().(*corev1.Pod)
			assert.NotEqual(t, containerID, pod.Annotations[agonesv1.GameServerReadyContainerIDAnnotation])
			return true, pod, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.podSynced)
		defer cancel()

		gs, err := c.syncGameServerRequestReadyState(ctx, gsFixture)
		assert.NoError(t, err, "should not error")
		assert.True(t, gsUpdated, "GameServer wasn't updated")
		assert.True(t, podUpdated, "Pod wasn't updated")
		assert.Equal(t, agonesv1.GameServerStateReady, gs.Status.State)
		agtesting.AssertEventContains(t, m.FakeRecorder.Events, "SDK.Ready() complete")
	})

	for _, s := range []agonesv1.GameServerState{"Unknown", agonesv1.GameServerStateUnhealthy} {
		name := fmt.Sprintf("GameServer with %s state", s)
		t.Run(name, func(t *testing.T) {
			testNoChange(t, s, func(c *Controller, fixture *agonesv1.GameServer) (*agonesv1.GameServer, error) {
				return c.syncGameServerRequestReadyState(context.Background(), fixture)
			})
		})
	}

	t.Run("GameServer whose pod is currently not in a running state, so should retry and not update", func(t *testing.T) {
		c, m := newFakeController()

		gsFixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateRequestReady}}
		gsFixture.ApplyDefaults()
		gsFixture.Status.NodeName = nodeName
		pod, err := gsFixture.Pod(agtesting.FakeAPIHooks{})
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: gsFixture.Spec.Container}}
		assert.Nil(t, err)
		gsUpdated := false
		podUpdated := false

		m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			return true, nil, nil
		})
		m.KubeClient.AddReactor("update", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			podUpdated = true
			return true, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.podSynced)
		defer cancel()

		_, err = c.syncGameServerRequestReadyState(ctx, gsFixture)
		assert.EqualError(t, err, "game server container for GameServer test in namespace default is not currently running, try again")
		assert.False(t, gsUpdated, "GameServer was updated")
		assert.False(t, podUpdated, "Pod was updated")
	})

	t.Run("GameServer whose pod is missing ContainerStatuses, so should retry and not update", func(t *testing.T) {
		c, m := newFakeController()

		gsFixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateRequestReady}}
		gsFixture.ApplyDefaults()
		gsFixture.Status.NodeName = nodeName
		pod, err := gsFixture.Pod(agtesting.FakeAPIHooks{})
		assert.Nil(t, err)
		gsUpdated := false
		podUpdated := false

		m.KubeClient.AddReactor("list", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, &corev1.PodList{Items: []corev1.Pod{*pod}}, nil
		})
		m.AgonesClient.AddReactor("update", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			return true, nil, nil
		})
		m.KubeClient.AddReactor("update", "pods", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			podUpdated = true
			return true, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.podSynced)
		defer cancel()

		_, err = c.syncGameServerRequestReadyState(ctx, gsFixture)
		assert.EqualError(t, err, "game server container for GameServer test in namespace default not present in pod status, try again")
		assert.False(t, gsUpdated, "GameServer was updated")
		assert.False(t, podUpdated, "Pod was updated")
	})

	t.Run("GameServer with non zero deletion datetime", func(t *testing.T) {
		testWithNonZeroDeletionTimestamp(t, func(c *Controller, fixture *agonesv1.GameServer) (*agonesv1.GameServer, error) {
			return c.syncGameServerRequestReadyState(context.Background(), fixture)
		})
	})
}

func TestMoveToErrorState(t *testing.T) {
	t.Parallel()

	t.Run("Set GameServer to error state", func(t *testing.T) {
		c, m := newFakeController()

		gsFixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateRequestReady}}
		gsFixture.ApplyDefaults()

		gsUpdated := false

		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsUpdated = true
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateError, gs.Status.State)
			return true, gs, nil
		})

		ctx, cancel := agtesting.StartInformers(m, c.podSynced)
		defer cancel()

		res, err := c.moveToErrorState(ctx, gsFixture, "some-data")
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, gsUpdated)
		assert.Equal(t, agonesv1.GameServerStateError, res.Status.State)
	})

	t.Run("Error on update", func(t *testing.T) {
		c, m := newFakeController()

		gsFixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateRequestReady}}
		gsFixture.ApplyDefaults()

		m.AgonesClient.AddReactor("update", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			ua := action.(k8stesting.UpdateAction)
			gs := ua.GetObject().(*agonesv1.GameServer)
			assert.Equal(t, agonesv1.GameServerStateError, gs.Status.State)
			return true, gs, errors.New("update-err")
		})

		ctx, cancel := agtesting.StartInformers(m, c.podSynced)
		defer cancel()

		_, err := c.moveToErrorState(ctx, gsFixture, "some-data")
		if assert.Error(t, err) {
			assert.Equal(t, `error moving GameServer test to Error State: update-err`, err.Error())
		}
	})
}

func TestControllerSyncGameServerShutdownState(t *testing.T) {
	t.Parallel()

	t.Run("GameServer with a Shutdown state", func(t *testing.T) {
		c, mocks := newFakeController()
		gsFixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateShutdown}}
		gsFixture.ApplyDefaults()
		checkDeleted := false

		mocks.AgonesClient.AddReactor("delete", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			checkDeleted = true
			assert.Equal(t, "default", action.GetNamespace())
			da := action.(k8stesting.DeleteAction)
			assert.Equal(t, "test", da.GetName())

			return true, nil, nil
		})

		ctx, cancel := agtesting.StartInformers(mocks, c.gameServerSynced)
		defer cancel()

		err := c.syncGameServerShutdownState(ctx, gsFixture)
		assert.Nil(t, err)
		assert.True(t, checkDeleted, "GameServer should be deleted")
		assert.Contains(t, <-mocks.FakeRecorder.Events, "Deletion started")
	})

	t.Run("Error on delete", func(t *testing.T) {
		c, mocks := newFakeController()
		gsFixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateShutdown}}
		gsFixture.ApplyDefaults()

		mocks.AgonesClient.AddReactor("delete", "gameservers", func(action k8stesting.Action) (bool, runtime.Object, error) {
			assert.Equal(t, "default", action.GetNamespace())
			da := action.(k8stesting.DeleteAction)
			assert.Equal(t, "test", da.GetName())

			return true, nil, errors.New("delete-err")
		})

		ctx, cancel := agtesting.StartInformers(mocks, c.gameServerSynced)
		defer cancel()

		err := c.syncGameServerShutdownState(ctx, gsFixture)
		if assert.Error(t, err) {
			assert.Equal(t, `error deleting Game Server test: delete-err`, err.Error())
		}
	})

	t.Run("GameServer with unknown state", func(t *testing.T) {
		testNoChange(t, "Unknown", func(c *Controller, fixture *agonesv1.GameServer) (*agonesv1.GameServer, error) {
			return fixture, c.syncGameServerShutdownState(context.Background(), fixture)
		})
	})

	t.Run("GameServer with non zero deletion datetime", func(t *testing.T) {
		testWithNonZeroDeletionTimestamp(t, func(c *Controller, fixture *agonesv1.GameServer) (*agonesv1.GameServer, error) {
			return fixture, c.syncGameServerShutdownState(context.Background(), fixture)
		})
	})
}

func TestControllerGameServerPod(t *testing.T) {
	t.Parallel()

	agruntime.FeatureTestMutex.Lock()
	defer agruntime.FeatureTestMutex.Unlock()
	require.NoError(t, agruntime.ParseFeatures(string(agruntime.FeatureSidecarContainers)+"=false"))

	setup := func() (*Controller, *agonesv1.GameServer, *watch.FakeWatcher, context.Context, context.CancelFunc) {
		c, mocks := newFakeController()
		fakeWatch := watch.NewFake()
		mocks.KubeClient.AddWatchReactor("pods", k8stesting.DefaultWatchReactor(fakeWatch, nil))
		ctx, cancel := agtesting.StartInformers(mocks, c.gameServerSynced)
		gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gameserver",
			Namespace: defaultNs, UID: "1234"}, Spec: newSingleContainerSpec()}
		gs.ApplyDefaults()
		return c, gs, fakeWatch, ctx, cancel
	}

	t.Run("no pod exists", func(t *testing.T) {
		c, gs, _, _, cancel := setup()
		defer cancel()

		require.Never(t, func() bool {
			list, err := c.podLister.List(labels.Everything())
			assert.NoError(t, err)
			return len(list) > 0
		}, time.Second, 100*time.Millisecond)
		_, err := c.gameServerPod(gs)
		assert.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("a pod exists", func(t *testing.T) {
		c, gs, fakeWatch, _, cancel := setup()

		defer cancel()
		pod, err := gs.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)

		fakeWatch.Add(pod.DeepCopy())
		require.Eventually(t, func() bool {
			list, err := c.podLister.List(labels.Everything())
			assert.NoError(t, err)
			return len(list) == 1
		}, 5*time.Second, time.Second)

		pod2, err := c.gameServerPod(gs)
		require.NoError(t, err)
		assert.Equal(t, pod, pod2)

		fakeWatch.Delete(pod.DeepCopy())
		require.Eventually(t, func() bool {
			list, err := c.podLister.List(labels.Everything())
			assert.NoError(t, err)
			return len(list) == 0
		}, 5*time.Second, time.Second)
		_, err = c.gameServerPod(gs)
		assert.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("a pod exists, but isn't owned by the gameserver", func(t *testing.T) {
		c, gs, fakeWatch, ctx, cancel := setup()
		defer cancel()

		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: gs.ObjectMeta.Name, Labels: map[string]string{agonesv1.GameServerPodLabel: gs.ObjectMeta.Name, "owned": "false"}}}
		fakeWatch.Add(pod.DeepCopy())

		// gate
		cache.WaitForCacheSync(ctx.Done(), c.podSynced)
		pod, err := c.podGetter.Pods(defaultNs).Get(ctx, pod.ObjectMeta.Name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.NotNil(t, pod)

		_, err = c.gameServerPod(gs)
		assert.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("dev gameserver pod", func(t *testing.T) {
		c, _ := newFakeController()

		gs := &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: "gameserver", Namespace: defaultNs,
				Annotations: map[string]string{
					agonesv1.DevAddressAnnotation: "1.1.1.1",
				},
				UID: "1234"},

			Spec: newSingleContainerSpec()}

		pod, err := c.gameServerPod(gs)
		require.NoError(t, err)
		assert.Empty(t, pod.ObjectMeta.Name)
	})
}

func TestControllerAddGameServerHealthCheck(t *testing.T) {
	c, _ := newFakeController()
	fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateCreating}}
	fixture.ApplyDefaults()

	assert.False(t, fixture.Spec.Health.Disabled)
	pod, err := fixture.Pod(agtesting.FakeAPIHooks{})
	require.NoError(t, err)
	err = c.addGameServerHealthCheck(fixture, pod)

	assert.NoError(t, err)
	assert.Len(t, pod.Spec.Containers, 1)
	probe := pod.Spec.Containers[0].LivenessProbe
	require.NotNil(t, probe)
	assert.Equal(t, "/gshealthz", probe.HTTPGet.Path)
	assert.Equal(t, intstr.IntOrString{IntVal: 8080}, probe.HTTPGet.Port)
	assert.Equal(t, fixture.Spec.Health.FailureThreshold, probe.FailureThreshold)
	assert.Equal(t, fixture.Spec.Health.InitialDelaySeconds, probe.InitialDelaySeconds)
	assert.Equal(t, fixture.Spec.Health.PeriodSeconds, probe.PeriodSeconds)
}

func TestControllerAddSDKServerEnvVars(t *testing.T) {

	t.Run("legacy game server without ports set", func(t *testing.T) {
		// For backwards compatibility, verify that no variables are set if the ports
		// are not set on the game server.
		c, _ := newFakeController()
		gs := &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: "gameserver", UID: "1234"},
			Spec:       newSingleContainerSpec(),
		}
		gs.ApplyDefaults()
		gs.Spec.SdkServer = agonesv1.SdkServer{}
		pod, err := gs.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)
		before := pod.DeepCopy()
		c.addSDKServerEnvVars(gs, pod)
		assert.Equal(t, before, pod, "Error: pod unexpectedly modified. before = %v, after = %v", before, pod)
	})

	t.Run("game server without any environment", func(t *testing.T) {
		c, _ := newFakeController()
		gs := &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: "gameserver", UID: "2345"},
			Spec:       newSingleContainerSpec(),
		}
		gs.ApplyDefaults()
		pod, err := gs.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)
		c.addSDKServerEnvVars(gs, pod)
		assert.Len(t, pod.Spec.Containers, 1, "Expected 1 container, found %d", len(pod.Spec.Containers))
		assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: grpcPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.GRPCPort))})
		assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: httpPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.HTTPPort))})
	})

	t.Run("game server without any conflicting env vars", func(t *testing.T) {
		c, _ := newFakeController()
		gs := &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: "gameserver", UID: "3456"},
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{{ContainerPort: 7777}},
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container",
								Image: "container/image",
								Env:   []corev1.EnvVar{{Name: "one", Value: "value"}, {Name: "two", Value: "value"}},
							},
						},
					},
				},
			},
		}
		gs.ApplyDefaults()
		pod, err := gs.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)
		c.addSDKServerEnvVars(gs, pod)
		assert.Len(t, pod.Spec.Containers, 1, "Expected 1 container, found %d", len(pod.Spec.Containers))
		assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: grpcPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.GRPCPort))})
		assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: httpPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.HTTPPort))})
	})

	t.Run("game server with conflicting env vars", func(t *testing.T) {
		c, _ := newFakeController()
		gs := &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: "gameserver", UID: "4567"},
			Spec: agonesv1.GameServerSpec{
				Ports: []agonesv1.GameServerPort{{ContainerPort: 7777}},
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container",
								Image: "container/image",
								Env:   []corev1.EnvVar{{Name: grpcPortEnvVar, Value: "value"}, {Name: httpPortEnvVar, Value: "value"}},
							},
						},
					},
				},
			},
		}
		gs.ApplyDefaults()
		pod, err := gs.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)
		c.addSDKServerEnvVars(gs, pod)
		assert.Len(t, pod.Spec.Containers, 1, "Expected 1 container, found %d", len(pod.Spec.Containers))
		assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: grpcPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.GRPCPort))})
		assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: httpPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.HTTPPort))})
	})

	t.Run("game server with multiple containers", func(t *testing.T) {
		c, _ := newFakeController()
		gs := &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: "gameserver", UID: "5678"},
			Spec: agonesv1.GameServerSpec{
				Container: "container1",
				Ports:     []agonesv1.GameServerPort{{ContainerPort: 7777}},
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container1",
								Image: "container/gameserver",
							},
							{
								Name:  "container2",
								Image: "container/image2",
								Env:   []corev1.EnvVar{{Name: "one", Value: "value"}, {Name: "two", Value: "value"}},
							},
							{
								Name:  "container3",
								Image: "container/image2",
								Env:   []corev1.EnvVar{{Name: grpcPortEnvVar, Value: "value"}, {Name: httpPortEnvVar, Value: "value"}},
							},
						},
					},
				},
			},
		}
		gs.ApplyDefaults()
		pod, err := gs.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)
		c.addSDKServerEnvVars(gs, pod)
		for _, c := range pod.Spec.Containers {
			assert.Contains(t, c.Env, corev1.EnvVar{Name: grpcPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.GRPCPort))})
			assert.Contains(t, c.Env, corev1.EnvVar{Name: httpPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.HTTPPort))})
		}
	})

	t.Run("environment variables not applied to the sdkserver container", func(t *testing.T) {
		c, _ := newFakeController()
		gs := &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: "gameserver", UID: "5678"},
			Spec: agonesv1.GameServerSpec{
				Container: "container1",
				Ports:     []agonesv1.GameServerPort{{ContainerPort: 7777}},
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "container1",
								Image: "container/gameserver",
							},
							{
								Name:  "container2",
								Image: "container/image2",
								Env:   []corev1.EnvVar{{Name: "one", Value: "value"}, {Name: "two", Value: "value"}},
							},
							{
								Name:  "container3",
								Image: "container/image2",
								Env:   []corev1.EnvVar{{Name: grpcPortEnvVar, Value: "value"}, {Name: httpPortEnvVar, Value: "value"}},
							},
						},
						InitContainers: []corev1.Container{
							{
								Name:  "init-container1",
								Image: "init-container/image1",
								Env:   []corev1.EnvVar{{Name: "one", Value: "value"}, {Name: "two", Value: "value"}},
							},
							{
								Name:  "init-container1",
								Image: "init-container/image2",
								Env:   []corev1.EnvVar{{Name: grpcPortEnvVar, Value: "value"}, {Name: httpPortEnvVar, Value: "value"}},
							},
						},
					},
				},
			},
		}
		gs.ApplyDefaults()
		sidecar := c.sidecar(gs)
		pod, err := gs.Pod(agtesting.FakeAPIHooks{}, sidecar)
		require.NoError(t, err)
		c.addSDKServerEnvVars(gs, pod)
		for _, c := range pod.Spec.Containers {
			if c.Name == sdkserverSidecarName {
				assert.NotContains(t, c.Env, corev1.EnvVar{Name: grpcPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.GRPCPort))})
				assert.NotContains(t, c.Env, corev1.EnvVar{Name: httpPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.HTTPPort))})
			} else {
				assert.Contains(t, c.Env, corev1.EnvVar{Name: grpcPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.GRPCPort))})
				assert.Contains(t, c.Env, corev1.EnvVar{Name: httpPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.HTTPPort))})
			}
		}
	})

	t.Run("game server with init containers", func(t *testing.T) {
		c, _ := newFakeController()
		gs := &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: "gameserver", UID: "6789"},
			Spec: agonesv1.GameServerSpec{
				Container: "container1",
				Ports:     []agonesv1.GameServerPort{{ContainerPort: 7777}},
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{

						InitContainers: []corev1.Container{
							{
								Name:  "init-container1",
								Image: "initcontainer/image",
							},
							{
								Name:  "init-container2",
								Image: "initcontainer/image2",
							},
						},
						Containers: []corev1.Container{
							{
								Name:  "container1",
								Image: "container/gameserver",
							},
						},
					},
				},
			},
		}
		gs.ApplyDefaults()
		pod, err := gs.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)
		c.addSDKServerEnvVars(gs, pod)
		assert.Contains(t, pod.Spec.InitContainers[0].Env, corev1.EnvVar{Name: grpcPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.GRPCPort))})
		assert.Contains(t, pod.Spec.InitContainers[0].Env, corev1.EnvVar{Name: httpPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.HTTPPort))})
		assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: grpcPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.GRPCPort))})
		assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{Name: httpPortEnvVar, Value: strconv.Itoa(int(gs.Spec.SdkServer.HTTPPort))})
	})
}

// testNoChange runs a test with a state that doesn't exist, to ensure a handler
// doesn't do process anything beyond the state it is meant to handle.
func testNoChange(t *testing.T, state agonesv1.GameServerState, f func(*Controller, *agonesv1.GameServer) (*agonesv1.GameServer, error)) {
	c, mocks := newFakeController()
	fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: state}}
	fixture.ApplyDefaults()
	updated := false
	mocks.AgonesClient.AddReactor("update", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		updated = true
		return true, nil, nil
	})

	result, err := f(c, fixture)
	require.NoError(t, err)
	assert.False(t, updated, "update should occur")
	assert.Equal(t, fixture, result)
}

// testWithNonZeroDeletionTimestamp runs a test with a given state, but
// the DeletionTimestamp set to Now()
func testWithNonZeroDeletionTimestamp(t *testing.T, f func(*Controller, *agonesv1.GameServer) (*agonesv1.GameServer, error)) {
	c, mocks := newFakeController()
	now := metav1.Now()
	fixture := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default", DeletionTimestamp: &now},
		Spec: newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateShutdown}}
	fixture.ApplyDefaults()
	updated := false
	mocks.AgonesClient.AddReactor("update", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		updated = true
		return true, nil, nil
	})

	result, err := f(c, fixture)
	require.NoError(t, err)
	assert.False(t, updated, "update should occur")
	assert.Equal(t, fixture, result)
}

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() (*Controller, agtesting.Mocks) {
	m := agtesting.NewMocks()
	c := NewController(
		generic.New(),
		healthcheck.NewHandler(),
		map[string]portallocator.PortRange{agonesv1.DefaultPortRange: {MinPort: 10, MaxPort: 20}},
		"sidecar:dev", false,
		resource.MustParse("0.05"), resource.MustParse("0.1"),
		resource.MustParse("50Mi"), resource.MustParse("100Mi"), sidecarRunAsUser, 500*time.Millisecond, "sdk-service-account",
		m.KubeClient, m.KubeInformerFactory, m.ExtClient, m.AgonesClient, m.AgonesInformerFactory)
	c.recorder = m.FakeRecorder
	return c, m
}

// newFakeExtensions return a fake extensions struct
func newFakeExtensions() *Extensions {
	return NewExtensions(generic.New(), webhooks.NewWebHook(http.NewServeMux()))
}

func newSingleContainerSpec() agonesv1.GameServerSpec {
	return agonesv1.GameServerSpec{
		Ports: []agonesv1.GameServerPort{{ContainerPort: 7777, HostPort: 9999, PortPolicy: agonesv1.Static}},
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "container", Image: "container/image"}},
			},
		},
	}
}

// Assume container ports 0 and 1 are Passthrough ports for "example-server" container and container port 0 for "example-server-two"
// The annotation would look like autopilot.gke.io/passthrough-port-assignment: '{"example-server":["0","1"], "example-server-two":[0]}'
func newPassthroughPortSingleContainerSpec() corev1.Pod {
	passthroughContainerPortMap := "{\"example-server\":[0,1],\"example-server-two\":[0]}"
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{agonesv1.PassthroughPortAssignmentAnnotation: passthroughContainerPortMap},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "agones-gameserver-sidecar",
					Image: "container/image",
					Env:   []corev1.EnvVar{{Name: passthroughPortEnvVar, Value: "TRUE"}}},
				{Name: "example-server",
					Image: "container2/image",
					Ports: []corev1.ContainerPort{
						{HostPort: 7777, ContainerPort: 5555},
						{HostPort: 7776, ContainerPort: 7797},
						{HostPort: 7775, ContainerPort: 7793}},
					Env: []corev1.EnvVar{{Name: passthroughPortEnvVar, Value: "TRUE"}}},
				{Name: "example-server-two",
					Image: "container3/image",
					Ports: []corev1.ContainerPort{
						{HostPort: 7745, ContainerPort: 7983},
						{HostPort: 7312, ContainerPort: 7364}},
					Env: []corev1.EnvVar{{Name: passthroughPortEnvVar, Value: "TRUE"}}}},
		},
	}
}
