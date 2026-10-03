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

package sdkserver

import (
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvert(t *testing.T) {
	t.Parallel()

	fixture := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.Now(),
			Namespace:         "default",
			Name:              "test",
			Labels:            map[string]string{"foo": "bar"},
			Annotations:       map[string]string{"stuff": "things"},
			UID:               "1234",
		},
		Spec: agonesv1.GameServerSpec{
			Health: agonesv1.Health{
				Disabled:            false,
				InitialDelaySeconds: 10,
				FailureThreshold:    15,
				PeriodSeconds:       20,
			},
		},
		Status: agonesv1.GameServerStatus{
			NodeName:  "george",
			Address:   "127.0.0.1",
			Addresses: []corev1.NodeAddress{{Type: "SomeAddressType", Address: "127.0.0.1"}},
			State:     "Ready",
			Ports: []agonesv1.GameServerStatusPort{
				{Name: "default", Port: 12345},
				{Name: "beacon", Port: 123123},
			},
		},
	}

	eq := func(t *testing.T, fixture *agonesv1.GameServer, sdkGs *sdk.GameServer) {
		assert.Equal(t, fixture.ObjectMeta.Name, sdkGs.GetObjectMeta().GetName())
		assert.Equal(t, fixture.ObjectMeta.Namespace, sdkGs.GetObjectMeta().GetNamespace())
		assert.Equal(t, fixture.ObjectMeta.CreationTimestamp.Unix(), sdkGs.GetObjectMeta().GetCreationTimestamp())
		assert.Equal(t, string(fixture.ObjectMeta.UID), sdkGs.GetObjectMeta().GetUid())
		assert.Equal(t, fixture.ObjectMeta.Labels, sdkGs.GetObjectMeta().GetLabels())
		assert.Equal(t, fixture.ObjectMeta.Annotations, sdkGs.GetObjectMeta().GetAnnotations())
		assert.Equal(t, fixture.Spec.Health.Disabled, sdkGs.GetSpec().GetHealth().GetDisabled())
		assert.Equal(t, fixture.Spec.Health.InitialDelaySeconds, sdkGs.GetSpec().GetHealth().GetInitialDelaySeconds())
		assert.Equal(t, fixture.Spec.Health.FailureThreshold, sdkGs.GetSpec().GetHealth().GetFailureThreshold())
		assert.Equal(t, fixture.Spec.Health.PeriodSeconds, sdkGs.GetSpec().GetHealth().GetPeriodSeconds())
		assert.Equal(t, fixture.Status.Address, sdkGs.GetStatus().GetAddress())
		assert.Equal(t, []*sdk.GameServer_Status_Address{{Type: "SomeAddressType", Address: "127.0.0.1"}}, sdkGs.GetStatus().GetAddresses())
		assert.Equal(t, string(fixture.Status.State), sdkGs.GetStatus().GetState())
		assert.Len(t, sdkGs.GetStatus().GetPorts(), len(fixture.Status.Ports))
		for i, fp := range fixture.Status.Ports {
			p := sdkGs.GetStatus().GetPorts()[i]
			assert.Equal(t, fp.Name, p.GetName())
			assert.Equal(t, fp.Port, p.GetPort())
		}
	}

	t.Run(string(runtime.FeatureCountsAndLists)+" disabled", func(t *testing.T) {
		runtime.FeatureTestMutex.Lock()
		defer runtime.FeatureTestMutex.Unlock()
		require.NoError(t, runtime.ParseFeatures(""))

		gs := fixture.DeepCopy()

		sdkGs := convert(gs)
		eq(t, fixture, sdkGs)
		assert.Zero(t, sdkGs.GetObjectMeta().GetDeletionTimestamp())
		assert.Nil(t, sdkGs.GetStatus().GetCounters())
		assert.Nil(t, sdkGs.GetStatus().GetLists())
	})

	t.Run(string(runtime.FeatureCountsAndLists)+" enabled", func(t *testing.T) {
		runtime.FeatureTestMutex.Lock()
		defer runtime.FeatureTestMutex.Unlock()
		require.NoError(t, runtime.ParseFeatures(string(runtime.FeatureCountsAndLists)+"=true"))

		gs := fixture.DeepCopy()
		gs.Status.Counters = map[string]agonesv1.CounterStatus{
			"Games": {
				Count:    1,
				Capacity: 999,
			},
		}
		gs.Status.Lists = map[string]agonesv1.ListStatus{
			"Lobbies": {
				Capacity: 100,
				Values:   []string{"Lobby1", "Lobby2", "Lobby0"},
			},
		}

		sdkGs := convert(gs)
		eq(t, fixture, sdkGs)
		assert.Zero(t, sdkGs.GetObjectMeta().GetDeletionTimestamp())
		assert.Equal(t, gs.Status.Counters["Games"].Count, sdkGs.GetStatus().GetCounters()["Games"].GetCount())
		assert.Equal(t, gs.Status.Counters["Games"].Capacity, sdkGs.GetStatus().GetCounters()["Games"].GetCapacity())
		// Using assert.Equal for List Values here to check for items and item order equal in the List.
		assert.Equal(t, gs.Status.Lists["Lobbies"].Values, sdkGs.GetStatus().GetLists()["Lobbies"].GetValues())
		assert.Equal(t, gs.Status.Lists["Lobbies"].Capacity, sdkGs.GetStatus().GetLists()["Lobbies"].GetCapacity())
	})

	t.Run("DeletionTimestamp", func(t *testing.T) {
		gs := fixture.DeepCopy()

		now := metav1.Now()
		gs.DeletionTimestamp = &now
		sdkGs := convert(gs)
		eq(t, gs, sdkGs)
		assert.Equal(t, gs.ObjectMeta.DeletionTimestamp.Unix(), sdkGs.GetObjectMeta().GetDeletionTimestamp())
	})
}
