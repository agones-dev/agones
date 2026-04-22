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

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	agtesting "agones.dev/agones/pkg/testing"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsGameServerPod(t *testing.T) {
	t.Parallel()

	t.Run("it is a game server pod", func(t *testing.T) {
		gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gameserver", UID: "1234"}, Spec: newSingleContainerSpec()}
		gs.ApplyDefaults()
		pod, err := gs.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)
		assert.True(t, isGameServerPod(pod))
	})

	t.Run("it is not a game server pod", func(t *testing.T) {
		pod := &corev1.Pod{}
		assert.False(t, isGameServerPod(pod))
	})
}

func TestAddress(t *testing.T) {
	t.Parallel()

	fixture := map[string]struct {
		node            *corev1.Node
		expectedAddress string
		featureFlags    string
	}{
		"node with external ip": {
			node:            &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName}, Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: "12.12.12.12", Type: corev1.NodeExternalIP}}}},
			expectedAddress: "12.12.12.12",
		},
		"node with an internal ip": {
			node:            &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName}, Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: "11.11.11.11", Type: corev1.NodeInternalIP}}}},
			expectedAddress: "11.11.11.11",
		},
		"node with internal and external ip": {
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName},
				Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{
					{Address: "9.9.9.8", Type: corev1.NodeExternalIP},
					{Address: "12.12.12.12", Type: corev1.NodeInternalIP},
				}},
			},
			expectedAddress: "9.9.9.8",
		},
		"node with external and internal dns": {
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName},
				Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{
					{Address: "external.example.com", Type: corev1.NodeExternalDNS},
					{Address: "internal.example.com", Type: corev1.NodeInternalDNS},
				}},
			},
			expectedAddress: "external.example.com",
		},
		"node with external and internal dns without feature flag": {
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName},
				Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{
					{Address: "external.example.com", Type: corev1.NodeExternalDNS},
					{Address: "internal.example.com", Type: corev1.NodeInternalDNS},
					{Address: "9.9.9.8", Type: corev1.NodeExternalIP},
					{Address: "12.12.12.12", Type: corev1.NodeInternalIP},
				}},
			},
			expectedAddress: "external.example.com",
		},
	}

	dummyGS := &agonesv1.GameServer{}
	dummyGS.Name = "some-gs"

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	for name, fixture := range fixture {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, runtime.ParseFeatures(fixture.featureFlags))

			addr, addrs, err := address(fixture.node)
			require.NoError(t, err)
			assert.Equal(t, fixture.expectedAddress, addr)
			assert.Equal(t, fixture.node.Status.Addresses, addrs)
		})
	}
}

func TestApplyGameServerAddressAndPort(t *testing.T) {
	t.Parallel()

	noopMod := func(*corev1.Pod) {}
	noopSyncer := func(*agonesv1.GameServer, *corev1.Pod) error { return nil }
	for name, tc := range map[string]struct {
		podMod    func(*corev1.Pod)
		podSyncer func(*agonesv1.GameServer, *corev1.Pod) error
		wantPort  int32
	}{
		"normal": {noopMod, noopSyncer, 9999},
		"host ports changed after create": {
			podMod: func(pod *corev1.Pod) {
				pod.Spec.Containers[0].Ports[0].HostPort = 9876
			},
			podSyncer: func(gs *agonesv1.GameServer, pod *corev1.Pod) error {
				gs.Spec.Ports[0].HostPort = pod.Spec.Containers[0].Ports[0].HostPort
				return nil
			},
			wantPort: 9876,
		},
		"container port with PortPolicy None changed after create": {
			podMod: func(pod *corev1.Pod) {
				pod.Spec.Containers[0].Ports[0].ContainerPort = 9876
			},
			podSyncer: func(gs *agonesv1.GameServer, pod *corev1.Pod) error {
				gs.Spec.Ports[0].PortPolicy = agonesv1.None
				gs.Spec.Ports[0].ContainerPort = pod.Spec.Containers[0].Ports[0].ContainerPort
				return nil
			},
			wantPort: 9876,
		},
		"Pod IP changed after create": {
			podMod: func(pod *corev1.Pod) {
				pod.Status.PodIPs = []corev1.PodIP{
					{IP: ipFixture},
				}
			},
			podSyncer: noopSyncer,
			wantPort:  9999,
		},
	} {
		t.Run(name, func(t *testing.T) {
			gsFixture := &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
				Spec:       newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateRequestReady},
			}
			gsFixture.ApplyDefaults()
			node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName}, Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{{Address: ipFixture, Type: corev1.NodeExternalIP}}}}
			pod, err := gsFixture.Pod(agtesting.FakeAPIHooks{})
			require.NoError(t, err)
			pod.Spec.NodeName = node.ObjectMeta.Name
			pod.Status.PodIPs = []corev1.PodIP{{IP: ipFixture}}
			tc.podMod(pod)

			// PortPolicy None is behind a feature flag
			runtime.FeatureTestMutex.Lock()
			defer runtime.FeatureTestMutex.Unlock()
			require.NoError(t, runtime.ParseFeatures(string(runtime.FeaturePortPolicyNone)+"=true"))

			gs, err := applyGameServerAddressAndPort(gsFixture, node, pod, tc.podSyncer)
			require.NoError(t, err)
			if assert.NotEmpty(t, gs.Spec.Ports) {
				assert.Equal(t, tc.wantPort, gs.Status.Ports[0].Port)
			}
			assert.Equal(t, ipFixture, gs.Status.Address)
			assert.Equal(t, node.ObjectMeta.Name, gs.Status.NodeName)
			assert.Equal(t, []corev1.NodeAddress{
				{Address: ipFixture, Type: "ExternalIP"},
				{Address: ipFixture, Type: "PodIP"},
			}, gs.Status.Addresses)
		})
	}

	t.Run("No IP specified, err expected", func(t *testing.T) {
		gsFixture := &agonesv1.GameServer{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec:       newSingleContainerSpec(), Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateRequestReady},
		}
		gsFixture.ApplyDefaults()
		node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeFixtureName}, Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{}}}
		pod, err := gsFixture.Pod(agtesting.FakeAPIHooks{})
		require.NoError(t, err)
		pod.Spec.NodeName = node.ObjectMeta.Name

		_, err = applyGameServerAddressAndPort(gsFixture, node, pod, noopSyncer)
		if assert.Error(t, err) {
			assert.Equal(t, "error getting external address for GameServer test: Could not find an address for Node: node1", err.Error())
		}
	})
}

func TestIsBeforePodCreated(t *testing.T) {
	fixture := map[string]struct {
		state    agonesv1.GameServerState
		expected bool
	}{
		"port":      {state: agonesv1.GameServerStatePortAllocation, expected: true},
		"creating":  {state: agonesv1.GameServerStateCreating, expected: true},
		"starting":  {state: agonesv1.GameServerStateStarting, expected: true},
		"allocated": {state: agonesv1.GameServerStateAllocated, expected: false},
		"ready":     {state: agonesv1.GameServerStateReady, expected: false},
	}

	for k, v := range fixture {
		t.Run(k, func(t *testing.T) {
			gs := &agonesv1.GameServer{Status: agonesv1.GameServerStatus{State: v.state}}

			assert.Equal(t, v.expected, isBeforePodCreated(gs))
		})
	}
}
