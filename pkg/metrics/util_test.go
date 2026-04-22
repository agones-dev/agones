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

package metrics

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/watch"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

var (
	// counter for unique GameServer names
	counter = 0
)

// newFakeController returns a controller, backed by the fake Clientset
func newFakeController() *fakeController {
	m := agtesting.NewMocks()
	return newFakeControllerWithMock(m)
}

// newFakeControllerWithMock returns a Controller with a pre-populated mock.
// This is useful if you want to populate a data set before the informer starts.
func newFakeControllerWithMock(m agtesting.Mocks) *fakeController {
	c := NewController(m.KubeClient, m.AgonesClient, m.KubeInformerFactory, m.AgonesInformerFactory)
	gsWatch := watch.NewFake()
	fasWatch := watch.NewFake()
	fleetWatch := watch.NewFake()
	nodeWatch := watch.NewFake()
	nsWatch := watch.NewFake()

	m.AgonesClient.AddWatchReactor("gameservers", k8stesting.DefaultWatchReactor(gsWatch, nil))
	m.AgonesClient.AddWatchReactor("fleetautoscalers", k8stesting.DefaultWatchReactor(fasWatch, nil))
	m.AgonesClient.AddWatchReactor("fleets", k8stesting.DefaultWatchReactor(fleetWatch, nil))
	m.KubeClient.AddWatchReactor("nodes", k8stesting.DefaultWatchReactor(nodeWatch, nil))
	m.KubeClient.AddWatchReactor("namespaces", k8stesting.DefaultWatchReactor(nsWatch, nil))

	ctx, cancel := agtesting.StartInformers(m, c.gameServerSynced, c.fleetSynced, c.fasSynced)

	return &fakeController{
		Controller: c,
		Mocks:      m,
		gsWatch:    gsWatch,
		fasWatch:   fasWatch,
		fleetWatch: fleetWatch,
		cancel:     cancel,
		ctx:        ctx,
	}
}

func (c *fakeController) close() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *fakeController) run(t *testing.T) {
	go func() {
		err := c.Controller.Run(c.ctx, 1)
		assert.Nil(t, err)
	}()
	c.sync()
}

func (c *fakeController) sync() bool {
	return cache.WaitForCacheSync(c.ctx.Done(), c.gameServerSynced, c.fleetSynced, c.fasSynced)
}

type fakeController struct {
	*Controller
	agtesting.Mocks
	gsWatch    *watch.FakeWatcher
	fasWatch   *watch.FakeWatcher
	fleetWatch *watch.FakeWatcher
	ctx        context.Context
	cancel     context.CancelFunc
}

func nodeWithName(name string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  uuid.NewUUID(),
		},
	}
}

func gameServerWithNode(nodeName string) *agonesv1.GameServer {
	gs := gameServerWithFleetAndState("fleet", agonesv1.GameServerStateReady)
	gs.Status.NodeName = nodeName
	return gs
}

func gameServerWithFleetStateCreationTimestamp(fleetName string, gsName string, state agonesv1.GameServerState, t metav1.Time) *agonesv1.GameServer {
	gs := gameServerWithFleetAndState(fleetName, state)
	gs.ObjectMeta.CreationTimestamp = t
	gs.ObjectMeta.Name = gsName
	return gs
}

func gameServerWithFleetAndState(fleetName string, state agonesv1.GameServerState) *agonesv1.GameServer {
	lbs := map[string]string{}
	if fleetName != "" {
		lbs[agonesv1.FleetNameLabel] = fleetName
	}
	// ensure the name is unique each time.
	counter++
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.String(10) + strconv.Itoa(counter),
			Namespace: "default",
			UID:       uuid.NewUUID(),
			Labels:    lbs,
		},
		Status: agonesv1.GameServerStatus{
			State: state,
		},
	}
	return gs
}

func generateGsEvents(count int, state agonesv1.GameServerState, fleetName string, fakew *watch.FakeWatcher) {
	for i := 0; i < count; i++ {
		gs := gameServerWithFleetAndState(fleetName, agonesv1.GameServerState(""))
		fakew.Add(gs)
		gsUpdated := gs.DeepCopy()
		gsUpdated.Status.State = state
		fakew.Modify(gsUpdated)
		// make sure we count only one event
		fakew.Modify(gsUpdated)
	}
}

func fleet(fleetName string, total, allocated, ready, desired, reserved int32) *agonesv1.Fleet {
	return &agonesv1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fleetName,
			Namespace: "default",
			UID:       uuid.NewUUID(),
		},
		Spec: agonesv1.FleetSpec{
			Replicas: desired,
		},
		Status: agonesv1.FleetStatus{
			ReservedReplicas:  reserved,
			AllocatedReplicas: allocated,
			ReadyReplicas:     ready,
			Replicas:          total,
		},
	}
}

func fleetAutoScaler(fleetName string, fasName string) *autoscalingv1.FleetAutoscaler {
	return &autoscalingv1.FleetAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fasName,
			Namespace: "default",
			UID:       uuid.NewUUID(),
		},
		Spec: autoscalingv1.FleetAutoscalerSpec{
			FleetName: fleetName,
			Policy: autoscalingv1.FleetAutoscalerPolicy{
				Type: autoscalingv1.BufferPolicyType,
				Buffer: &autoscalingv1.BufferPolicy{
					MaxReplicas: 30,
					MinReplicas: 10,
					BufferSize:  intstr.FromInt(11),
				},
			},
			Sync: &autoscalingv1.FleetAutoscalerSync{
				Type: autoscalingv1.FixedIntervalSyncType,
				FixedInterval: autoscalingv1.FixedIntervalSync{
					Seconds: 30,
				},
			},
		},
		Status: autoscalingv1.FleetAutoscalerStatus{
			AbleToScale:       true,
			ScalingLimited:    false,
			CurrentReplicas:   10,
			DesiredReplicas:   20,
			LastAppliedPolicy: autoscalingv1.BufferPolicyType,
		},
	}
}

func TestParseLabels(t *testing.T) {
	cases := []struct {
		desc     string
		labels   string
		expected map[string]string
		err      string
	}{
		{
			desc:     "Two valid labels, no error",
			labels:   "l1=1,l2=2",
			expected: map[string]string{"l1": "1", "l2": "2"},
			err:      "",
		},
		{
			desc:     "Empty input string, empty struct expected",
			labels:   "",
			expected: map[string]string{},
			err:      "",
		},
		{
			desc:     "Valid labels, invalid separator, error expected",
			labels:   "l1=1|l2=2",
			expected: map[string]string{"l1": "1", "l2": "2"},
			err:      "invalid labels: l1=1|l2=2, expect key=value,key2=value2",
		},
		{
			desc:     "Two valid labels with extra spaces, error expected",
			labels:   "   l1=1,   l2=2   ",
			expected: map[string]string{"l1": "1", "l2": "2"},
			err:      "",
		},
		{
			desc:     "Two invalid labels, error expected",
			labels:   "l1-1,l2-2",
			expected: map[string]string{"l1": "1", "l2": "2"},
			err:      "invalid labels: l1-1,l2-2, expect key=value,key2=value2",
		},
		{
			desc:     "Invalid key utf8 string, error expected",
			labels:   "\xF4=1,l2=2",
			expected: map[string]string{"l1": "1", "l2": "2"},
			err:      "invalid key: \xF4, must be a valid utf-8 string",
		},
		{
			desc:     "Invalid value utf8 string, error expected",
			labels:   "l1=\xF4,l2=2",
			expected: map[string]string{"l1": "1", "l2": "2"},
			err:      "invalid value: \xF4, must be a valid utf-8 string",
		},
		{
			desc:     "Empty key, error expected",
			labels:   " =1,l2=2",
			expected: map[string]string{"l1": "1", "l2": "2"},
			err:      "invalid key: can not be empty",
		},
		{
			desc:     "Empty value, error expected",
			labels:   "l1= ,l2=2",
			expected: map[string]string{"l1": "1", "l2": "2"},
			err:      "invalid value for key l1: can not be empty",
		},
		{
			desc:     "Empty key and value, key err excpected",
			labels:   " = ,l2=2",
			expected: map[string]string{"l1": "1", "l2": "2"},
			err:      "invalid key: can not be empty",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			res, err := parseLabels(tc.labels)

			if tc.err != "" {
				require.Error(t, err)
				assert.Equal(t, tc.err, err.Error())
			} else {
				require.NoError(t, err)
				// retrieve inner map
				m := reflect.ValueOf(res).Elem().FieldByName("m").MapRange()
				for m.Next() {
					val, ok := tc.expected[m.Key().String()]
					require.True(t, ok)
					assert.Equal(t, val, m.Value().FieldByName("val").String())
				}
			}
		})
	}
}
