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

package fleets

import (
	"context"
	"fmt"
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	utilruntime "agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8stesting "k8s.io/client-go/testing"
)

type surgeGSSFixture struct {
	specReplicas      int32
	statusReplicas    int32
	readyReplicas     int32
	allocatedReplicas int32
	reservedReplicas  int32
}

func surgeValue(v intstr.IntOrString) *intstr.IntOrString {
	return &v
}

func surgeGameServerSets(f *agonesv1.Fleet, active surgeGSSFixture, inactive ...surgeGSSFixture) (*agonesv1.GameServerSet, []*agonesv1.GameServerSet) {
	a := f.GameServerSet()
	a.ObjectMeta.Name = "active"
	a.Spec.Replicas = active.specReplicas
	a.Status.Replicas = active.statusReplicas
	a.Status.ReadyReplicas = active.readyReplicas
	a.Status.AllocatedReplicas = active.allocatedReplicas
	a.Status.ReservedReplicas = active.reservedReplicas

	f.Status.ReadyReplicas = active.readyReplicas
	rest := make([]*agonesv1.GameServerSet, 0, len(inactive))
	for index, fixture := range inactive {
		i := f.GameServerSet()
		i.ObjectMeta.Name = "inactive"
		if index > 0 {
			i.ObjectMeta.Name = fmt.Sprintf("inactive-%d", index)
		}
		i.Spec.Replicas = fixture.specReplicas
		i.Status.Replicas = fixture.statusReplicas
		i.Status.ReadyReplicas = fixture.readyReplicas
		i.Status.AllocatedReplicas = fixture.allocatedReplicas
		i.Status.ReservedReplicas = fixture.reservedReplicas
		f.Status.ReadyReplicas += fixture.readyReplicas
		rest = append(rest, i)
	}
	return a, rest
}

func TestControllerRollingUpdateDeploymentSurge(t *testing.T) {
	t.Cleanup(func() {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()
		require.NoError(t, utilruntime.ParseFeatures(""))
	})

	fixtures := map[string]struct {
		fleetSpecReplicas int32
		maxSurge          *intstr.IntOrString
		active            surgeGSSFixture
		inactive          surgeGSSFixture
		// expected replicas for the active GameServerSet without and with RollingUpdateSurge
		expectedDefault int32
		expectedSurge   int32
		expectedErr     string
	}{
		"allocation heavy inactive at rollout start": {
			// 1000 GameServers, 870 Allocated: without the surge the new GameServerSet starts with
			// only the 130 unallocated ones.
			fleetSpecReplicas: 1000,
			maxSurge:          surgeValue(intstr.FromString("50%")),
			active:            surgeGSSFixture{},
			inactive:          surgeGSSFixture{specReplicas: 1000, statusReplicas: 1000, readyReplicas: 130, allocatedReplicas: 870},
			expectedDefault:   130,
			expectedSurge:     500,
		},
		"allocation heavy inactive mid rollout with an absolute maxSurge": {
			fleetSpecReplicas: 2372,
			maxSurge:          surgeValue(intstr.FromInt32(300)),
			active:            surgeGSSFixture{specReplicas: 1625, statusReplicas: 1625, readyReplicas: 2, allocatedReplicas: 1520},
			inactive:          surgeGSSFixture{specReplicas: 922, statusReplicas: 922, readyReplicas: 151, allocatedReplicas: 747},
			expectedDefault:   1625,
			expectedSurge:     1750,
		},
		"ready heavy inactive keeps the stepwise surge": {
			fleetSpecReplicas: 100,
			maxSurge:          surgeValue(intstr.FromString("10%")),
			active:            surgeGSSFixture{},
			inactive:          surgeGSSFixture{specReplicas: 100, statusReplicas: 100, readyReplicas: 100},
			expectedDefault:   10,
			expectedSurge:     10,
		},
		"keeps the current target above the surge budget": {
			// the unallocated inactive GameServers (50) are more than the surge (10)
			fleetSpecReplicas: 100,
			maxSurge:          surgeValue(intstr.FromString("10%")),
			active:            surgeGSSFixture{specReplicas: 50, statusReplicas: 50, readyReplicas: 50},
			inactive:          surgeGSSFixture{specReplicas: 100, statusReplicas: 100, readyReplicas: 50, allocatedReplicas: 50},
			expectedDefault:   50,
			expectedSurge:     50,
		},
		"fleet scaled down during the rolling update": {
			// 25% of 50 rounds up to 13: 50 + 13 - 20 = 43
			fleetSpecReplicas: 50,
			active:            surgeGSSFixture{specReplicas: 100, statusReplicas: 100, readyReplicas: 100},
			inactive:          surgeGSSFixture{specReplicas: 20, statusReplicas: 20, readyReplicas: 10, allocatedReplicas: 10},
			expectedDefault:   40,
			expectedSurge:     43,
		},
		"fleet scaled down below inactive allocations": {
			fleetSpecReplicas: 5,
			active:            surgeGSSFixture{specReplicas: 10, statusReplicas: 10, readyReplicas: 10},
			inactive:          surgeGSSFixture{specReplicas: 8, statusReplicas: 8, allocatedReplicas: 8},
			expectedDefault:   0,
			expectedSurge:     0,
		},
		"waits while the active gameserverset is scaling": {
			fleetSpecReplicas: 1000,
			maxSurge:          surgeValue(intstr.FromString("50%")),
			active:            surgeGSSFixture{specReplicas: 500, statusReplicas: 300},
			inactive:          surgeGSSFixture{specReplicas: 1000, statusReplicas: 1000, readyReplicas: 130, allocatedReplicas: 870},
			expectedDefault:   500,
			expectedSurge:     500,
		},
		"fleet scaled to zero": {
			fleetSpecReplicas: 0,
			active:            surgeGSSFixture{},
			inactive:          surgeGSSFixture{specReplicas: 10, statusReplicas: 10, readyReplicas: 5, allocatedReplicas: 5},
			expectedDefault:   0,
			expectedSurge:     0,
		},
		"template changed while the inactive gameserverset is scaling up": {
			// the 900 GameServers the inactive GameServerSet is still creating count toward the surge
			fleetSpecReplicas: 1000,
			active:            surgeGSSFixture{},
			inactive:          surgeGSSFixture{specReplicas: 1000, statusReplicas: 100, readyReplicas: 100},
			expectedDefault:   250,
			expectedSurge:     250,
		},
		"value without the surge does not raise the target past the surge budget": {
			// without the surge the active GameServerSet goes up by another 250 while the inactive
			// one is still scaling up to 1000
			fleetSpecReplicas: 1000,
			active:            surgeGSSFixture{specReplicas: 250, statusReplicas: 250},
			inactive:          surgeGSSFixture{specReplicas: 1000, statusReplicas: 100, readyReplicas: 100},
			expectedDefault:   500,
			expectedSurge:     250,
		},
		"active gameserverset held above its target by allocated gameservers": {
			// 25% of 70 rounds up to 18: 70 + 18 - 20 = 68, so 8 Ready GameServers can be created
			fleetSpecReplicas: 70,
			active:            surgeGSSFixture{specReplicas: 48, statusReplicas: 60, allocatedReplicas: 60},
			inactive:          surgeGSSFixture{specReplicas: 20, statusReplicas: 20, allocatedReplicas: 20},
			expectedDefault:   48,
			expectedSurge:     68,
		},
	}

	for k, v := range fixtures {
		for _, enabled := range []bool{false, true} {
			name := k + "/default"
			features := ""
			expected := v.expectedDefault
			if enabled {
				name = k + "/surge"
				features = string(utilruntime.FeatureRollingUpdateSurge) + "=true"
				expected = v.expectedSurge
			}

			t.Run(name, func(t *testing.T) {
				utilruntime.FeatureTestMutex.Lock()
				defer utilruntime.FeatureTestMutex.Unlock()
				require.NoError(t, utilruntime.ParseFeatures(features))

				f := defaultFixture()
				f.Spec.Replicas = v.fleetSpecReplicas
				if v.maxSurge != nil {
					f.Spec.Strategy.RollingUpdate.MaxSurge = v.maxSurge
				}
				active, rest := surgeGameServerSets(f, v.active, v.inactive)

				c, m := newFakeController()
				// triggered inside rollingUpdateRest
				m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
					gsSet := action.(k8stesting.UpdateAction).GetObject().(*agonesv1.GameServerSet)
					return true, gsSet, nil
				})

				replicas, err := c.rollingUpdateDeployment(context.Background(), f, active, rest)
				require.NoError(t, err)
				assert.Equal(t, expected, replicas)

				// all GameServerSets together stay within the surge, unless the current target is
				// already above it.
				if enabled && replicas > v.active.specReplicas {
					surge, err := intstr.GetValueFromIntOrPercent(f.Spec.Strategy.RollingUpdate.MaxSurge, int(f.Spec.Replicas), true)
					require.NoError(t, err)
					inactive := max(v.inactive.specReplicas, v.inactive.statusReplicas)
					assert.LessOrEqual(t, replicas+inactive, f.Spec.Replicas+int32(surge))
				}
			})
		}
	}
}

func TestControllerRollingUpdateSurgeInactiveTargetZero(t *testing.T) {
	t.Cleanup(func() {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()
		require.NoError(t, utilruntime.ParseFeatures(""))
	})
	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()
	require.NoError(t, utilruntime.ParseFeatures(string(utilruntime.FeatureRollingUpdateSurge)+"=true"))

	fixtures := map[string]struct {
		fleetReplicas int32
		active        surgeGSSFixture
		inactive      surgeGSSFixture
		expected      int32
	}{
		"retains the surged target": {
			fleetReplicas: 100,
			active:        surgeGSSFixture{specReplicas: 85, statusReplicas: 85, readyReplicas: 25, allocatedReplicas: 60},
			inactive:      surgeGSSFixture{statusReplicas: 40, allocatedReplicas: 40},
			expected:      85,
		},
		"fleet scale down": {
			fleetReplicas: 70,
			active:        surgeGSSFixture{specReplicas: 85, statusReplicas: 85, readyReplicas: 25, allocatedReplicas: 60},
			inactive:      surgeGSSFixture{statusReplicas: 40, allocatedReplicas: 40},
			expected:      48,
		},
		"fleet scale down below the current active target": {
			fleetReplicas: 70,
			active:        surgeGSSFixture{specReplicas: 65, statusReplicas: 65, readyReplicas: 40, allocatedReplicas: 25},
			inactive:      surgeGSSFixture{statusReplicas: 40, allocatedReplicas: 40},
			expected:      48,
		},
		"fleet zero while creating": {
			active:   surgeGSSFixture{specReplicas: 85, statusReplicas: 70, readyReplicas: 10, allocatedReplicas: 60},
			inactive: surgeGSSFixture{statusReplicas: 40, allocatedReplicas: 40},
		},
		"fleet zero with allocated active gameservers": {
			active:   surgeGSSFixture{specReplicas: 48, statusReplicas: 60, allocatedReplicas: 60},
			inactive: surgeGSSFixture{statusReplicas: 40, allocatedReplicas: 40},
		},
		"allocated active gameservers do not block growth": {
			fleetReplicas: 70,
			active:        surgeGSSFixture{specReplicas: 48, statusReplicas: 60, allocatedReplicas: 60},
			inactive:      surgeGSSFixture{statusReplicas: 20, allocatedReplicas: 20},
			expected:      68,
		},
		"reserved inactive gameservers": {
			fleetReplicas: 100,
			active:        surgeGSSFixture{specReplicas: 70, statusReplicas: 70, readyReplicas: 70},
			inactive:      surgeGSSFixture{statusReplicas: 100, readyReplicas: 50, allocatedReplicas: 10, reservedReplicas: 40},
			expected:      50,
		},
	}

	for name, fixture := range fixtures {
		t.Run(name, func(t *testing.T) {
			f := defaultFixture()
			f.Spec.Replicas = fixture.fleetReplicas
			c, m := newFakeController()
			m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
				return true, action.(k8stesting.UpdateAction).GetObject(), nil
			})
			for _, oldTarget := range []int32{fixture.inactive.statusReplicas, 0} {
				inactive := fixture.inactive
				inactive.specReplicas = oldTarget
				active, rest := surgeGameServerSets(f, fixture.active, inactive)
				replicas, err := c.applyDeploymentStrategy(context.Background(), f, active, rest)
				require.NoError(t, err)
				assert.Equal(t, fixture.expected, replicas, "inactive target %d", oldTarget)
			}
		})
	}
}

func TestControllerRollingUpdateDeploymentSurgeNilMaxSurge(t *testing.T) {
	t.Cleanup(func() {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()
		require.NoError(t, utilruntime.ParseFeatures(""))
	})
	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()
	require.NoError(t, utilruntime.ParseFeatures(string(utilruntime.FeatureRollingUpdateSurge)+"=true"))

	// the active GameServerSet is already at fleet - allocated, so the value without the surge
	// returns before MaxSurge is parsed.
	f := defaultFixture()
	f.Spec.Replicas = 100
	f.Spec.Strategy.RollingUpdate.MaxSurge = nil
	active, rest := surgeGameServerSets(f,
		surgeGSSFixture{specReplicas: 50, statusReplicas: 50, readyReplicas: 50},
		surgeGSSFixture{specReplicas: 60, statusReplicas: 60, readyReplicas: 10, allocatedReplicas: 50})

	c, _ := newFakeController()
	_, err := c.rollingUpdateDeployment(context.Background(), f, active, rest)
	assert.ErrorContains(t, err, "error parsing MaxSurge value: fleet-1: nil value for IntOrString")
}

func TestControllerApplyDeploymentStrategySurge(t *testing.T) {
	t.Cleanup(func() {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()
		require.NoError(t, utilruntime.ParseFeatures(""))
	})

	// The inactive GameServerSet has been scaled to zero and only holds Allocated GameServers.
	fixtures := map[string]struct {
		fleetSpecReplicas int32
		strategy          appsv1.DeploymentStrategyType
		active            surgeGSSFixture
		inactive          surgeGSSFixture
		expectedDefault   int32
		expectedSurge     int32
	}{
		"keeps the surged gameservers when the inactive reaches zero": {
			// without the surge the active GameServerSet drops from 85 to 60 in one step
			fleetSpecReplicas: 100,
			active:            surgeGSSFixture{specReplicas: 85, statusReplicas: 85, readyReplicas: 25, allocatedReplicas: 60},
			inactive:          surgeGSSFixture{statusReplicas: 40, allocatedReplicas: 40},
			expectedDefault:   60,
			expectedSurge:     85,
		},
		"fleet target decrease releases the surged gameservers": {
			// 25% of 70 rounds up to 18: 70 + 18 - 40 = 48
			fleetSpecReplicas: 70,
			active:            surgeGSSFixture{specReplicas: 85, statusReplicas: 85, readyReplicas: 25, allocatedReplicas: 60},
			inactive:          surgeGSSFixture{statusReplicas: 40, allocatedReplicas: 40},
			expectedDefault:   30,
			expectedSurge:     48,
		},
		"draining allocated gameservers make room up to the fleet replicas": {
			fleetSpecReplicas: 100,
			active:            surgeGSSFixture{specReplicas: 85, statusReplicas: 85, readyReplicas: 25, allocatedReplicas: 60},
			inactive:          surgeGSSFixture{statusReplicas: 10, allocatedReplicas: 10},
			expectedDefault:   90,
			expectedSurge:     100,
		},
		"holds while the active gameserverset is creating gameservers": {
			fleetSpecReplicas: 100,
			active:            surgeGSSFixture{specReplicas: 85, statusReplicas: 70, readyReplicas: 10, allocatedReplicas: 60},
			inactive:          surgeGSSFixture{statusReplicas: 40, allocatedReplicas: 40},
			expectedDefault:   60,
			expectedSurge:     85,
		},
		"holds while creating even when the value without the surge is higher": {
			// without the surge the target jumps to 100 - 10 = 90 before the 85 requested exist
			fleetSpecReplicas: 100,
			active:            surgeGSSFixture{specReplicas: 85, statusReplicas: 70, readyReplicas: 10, allocatedReplicas: 60},
			inactive:          surgeGSSFixture{statusReplicas: 10, allocatedReplicas: 10},
			expectedDefault:   90,
			expectedSurge:     85,
		},
		"fleet scale down while the active gameserverset is creating gameservers": {
			// 25% of 70 rounds up to 18: 70 + 18 - 40 = 48, below the 85 requested
			fleetSpecReplicas: 70,
			active:            surgeGSSFixture{specReplicas: 85, statusReplicas: 50, readyReplicas: 10, allocatedReplicas: 40},
			inactive:          surgeGSSFixture{statusReplicas: 40, allocatedReplicas: 40},
			expectedDefault:   30,
			expectedSurge:     48,
		},
		"active gameserverset held above its target by allocated gameservers": {
			// 25% of 70 rounds up to 18: 70 + 18 - 20 = 68, so 8 Ready GameServers can be created
			fleetSpecReplicas: 70,
			active:            surgeGSSFixture{specReplicas: 48, statusReplicas: 60, allocatedReplicas: 60},
			inactive:          surgeGSSFixture{statusReplicas: 20, allocatedReplicas: 20},
			expectedDefault:   50,
			expectedSurge:     68,
		},
		"recreate strategy is unchanged": {
			fleetSpecReplicas: 100,
			strategy:          appsv1.RecreateDeploymentStrategyType,
			active:            surgeGSSFixture{specReplicas: 85, statusReplicas: 85, readyReplicas: 25, allocatedReplicas: 60},
			inactive:          surgeGSSFixture{statusReplicas: 40, allocatedReplicas: 40},
			expectedDefault:   60,
			expectedSurge:     60,
		},
	}

	for k, v := range fixtures {
		for _, enabled := range []bool{false, true} {
			name := k + "/default"
			features := ""
			expected := v.expectedDefault
			if enabled {
				name = k + "/surge"
				features = string(utilruntime.FeatureRollingUpdateSurge) + "=true"
				expected = v.expectedSurge
			}

			t.Run(name, func(t *testing.T) {
				utilruntime.FeatureTestMutex.Lock()
				defer utilruntime.FeatureTestMutex.Unlock()
				require.NoError(t, utilruntime.ParseFeatures(features))

				f := defaultFixture()
				f.Spec.Replicas = v.fleetSpecReplicas
				if v.strategy != "" {
					f.Spec.Strategy.Type = v.strategy
				}
				active, rest := surgeGameServerSets(f, v.active, v.inactive)

				c, _ := newFakeController()
				replicas, err := c.applyDeploymentStrategy(context.Background(), f, active, rest)
				require.NoError(t, err)
				assert.Equal(t, expected, replicas)
			})
		}
	}
}

// TestControllerRollingUpdateSurgeNoReadyCliff walks a rolling update of an allocation heavy Fleet
// through the moment its inactive GameServerSet is scaled to zero, and checks that the active
// GameServerSet target never drops while the Fleet replicas stay the same.
func TestControllerRollingUpdateSurgeNoReadyCliff(t *testing.T) {
	t.Cleanup(func() {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()
		require.NoError(t, utilruntime.ParseFeatures(""))
	})
	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()
	require.NoError(t, utilruntime.ParseFeatures(string(utilruntime.FeatureRollingUpdateSurge)+"=true"))

	f := defaultFixture()
	f.Spec.Replicas = 1000
	f.Spec.Strategy.RollingUpdate.MaxSurge = surgeValue(intstr.FromString("50%"))

	c, m := newFakeController()
	m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gsSet := action.(k8stesting.UpdateAction).GetObject().(*agonesv1.GameServerSet)
		return true, gsSet, nil
	})

	steps := []struct {
		active   surgeGSSFixture
		inactive surgeGSSFixture
	}{
		// rollout starts: 870 of the 1000 old GameServers are Allocated
		{active: surgeGSSFixture{}, inactive: surgeGSSFixture{specReplicas: 1000, statusReplicas: 1000, readyReplicas: 130, allocatedReplicas: 870}},
		// the surged GameServers became Ready, the old Ready ones were scaled down
		{active: surgeGSSFixture{specReplicas: 500, statusReplicas: 500, readyReplicas: 500}, inactive: surgeGSSFixture{specReplicas: 870, statusReplicas: 870, allocatedReplicas: 870}},
		// the inactive GameServerSet only holds Allocated GameServers and is scaled to zero
		{active: surgeGSSFixture{specReplicas: 630, statusReplicas: 630, readyReplicas: 500, allocatedReplicas: 130}, inactive: surgeGSSFixture{statusReplicas: 870, allocatedReplicas: 870}},
	}

	previous := int32(0)
	for i, s := range steps {
		active, rest := surgeGameServerSets(f, s.active, s.inactive)
		replicas, err := c.applyDeploymentStrategy(context.Background(), f, active, rest)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, replicas, previous, "step %d: active GameServerSet target dropped from %d to %d", i, previous, replicas)
		previous = replicas
	}
}

// TestControllerRollingUpdateSurgeAllocationHeavyGrowingFleet replays a production rolling update with
// maxSurge 25% and maxUnavailable 25%: an allocation heavy Fleet growing from about 1000 to 2400
// GameServers, where the inactive GameServerSet stayed at 922 GameServers for almost two hours
// because the GameServers starting for the Fleet growth used up the Ready based scale down budget.
func TestControllerRollingUpdateSurgeAllocationHeavyGrowingFleet(t *testing.T) {
	t.Cleanup(func() {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()
		require.NoError(t, utilruntime.ParseFeatures(""))
	})

	run := func(t *testing.T, features string, fleetReplicas int32, active, inactive surgeGSSFixture) (int32, *int32) {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()
		require.NoError(t, utilruntime.ParseFeatures(features))

		f := defaultFixture()
		f.Spec.Replicas = fleetReplicas
		require.Equal(t, "25%", f.Spec.Strategy.RollingUpdate.MaxSurge.String())
		require.Equal(t, "25%", f.Spec.Strategy.RollingUpdate.MaxUnavailable.String())
		a, rest := surgeGameServerSets(f, active, inactive)

		var inactiveSpec *int32
		c, m := newFakeController()
		m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			gsSet := action.(k8stesting.UpdateAction).GetObject().(*agonesv1.GameServerSet)
			if gsSet.ObjectMeta.Name == "inactive" {
				inactiveSpec = &gsSet.Spec.Replicas
			}
			return true, gsSet, nil
		})

		replicas, err := c.rollingUpdateDeployment(context.Background(), f, a, rest)
		require.NoError(t, err)
		return replicas, inactiveSpec
	}
	surgeOn := string(utilruntime.FeatureRollingUpdateSurge) + "=true"

	t.Run("rollout start", func(t *testing.T) {
		// All 966 GameServers are in the inactive GameServerSet, 838 of them Allocated.
		inactive := surgeGSSFixture{specReplicas: 966, statusReplicas: 966, readyReplicas: 52, allocatedReplicas: 838}

		replicas, _ := run(t, "", 966, surgeGSSFixture{}, inactive)
		assert.Equal(t, int32(128), replicas, "without the surge only the unallocated inactive GameServers are replaced")

		// the new GameServerSet starts with maxSurge: 25% of 966 rounds up to 242
		replicas, _ = run(t, surgeOn, 966, surgeGSSFixture{}, inactive)
		assert.Equal(t, int32(242), replicas)
	})

	t.Run("growing fleet mid rollout", func(t *testing.T) {
		// The Fleet has grown to 2372 and 103 GameServers of the active GameServerSet are starting.
		inactive := surgeGSSFixture{specReplicas: 922, statusReplicas: 922, readyReplicas: 151, allocatedReplicas: 747}

		// Without the surge: Ready 153, budget floor(25% of 153) = 38 is below the 103 starting
		// GameServers, so the inactive GameServerSet is not scaled down.
		replicas, inactiveSpec := run(t, "", 2372,
			surgeGSSFixture{specReplicas: 1625, statusReplicas: 1625, readyReplicas: 2, allocatedReplicas: 1520}, inactive)
		assert.Equal(t, int32(1625), replicas)
		assert.Nil(t, inactiveSpec, "inactive GameServerSet should not be scaled down")

		// With the surge the active target is 2372 + 593 - 922 = 2043. Once the 418 extra GameServers
		// are Ready, Ready is 571 and the budget floor(25% of 571) = 142 exceeds the 103 starting ones.
		active := surgeGSSFixture{specReplicas: 2043, statusReplicas: 2043, readyReplicas: 420, allocatedReplicas: 1520}
		replicas, inactiveSpec = run(t, surgeOn, 2372, active, inactive)
		assert.Equal(t, int32(2043), replicas)
		require.NotNil(t, inactiveSpec, "inactive GameServerSet should be scaled down")
		// first the 24 inactive GameServers that are neither Ready nor Allocated are removed
		assert.Equal(t, int32(898), *inactiveSpec)

		// next sync: 142 of the 151 inactive Ready GameServers are removed, Allocated ones are kept
		inactive = surgeGSSFixture{specReplicas: 898, statusReplicas: 898, readyReplicas: 151, allocatedReplicas: 747}
		_, inactiveSpec = run(t, surgeOn, 2372, active, inactive)
		require.NotNil(t, inactiveSpec, "inactive GameServerSet should be scaled down")
		assert.Equal(t, int32(756), *inactiveSpec)
	})
}

func TestControllerRollingUpdateSurgeMultipleInactiveGameServerSets(t *testing.T) {
	t.Cleanup(func() {
		utilruntime.FeatureTestMutex.Lock()
		defer utilruntime.FeatureTestMutex.Unlock()
		require.NoError(t, utilruntime.ParseFeatures(""))
	})
	utilruntime.FeatureTestMutex.Lock()
	defer utilruntime.FeatureTestMutex.Unlock()
	require.NoError(t, utilruntime.ParseFeatures(string(utilruntime.FeatureRollingUpdateSurge)+"=true"))

	fixtures := map[string]struct {
		fleetReplicas int32
		active        surgeGSSFixture
		inactive      []surgeGSSFixture
		expected      int32
	}{
		"counts each inactive target or status before summing": {
			fleetReplicas: 100,
			inactive: []surgeGSSFixture{
				{specReplicas: 60, statusReplicas: 20, readyReplicas: 20},
				{specReplicas: 10, statusReplicas: 40, readyReplicas: 10, allocatedReplicas: 30},
			},
			expected: 25,
		},
		"bounds retention by all inactive allocations and reservations": {
			fleetReplicas: 70,
			active:        surgeGSSFixture{specReplicas: 85, statusReplicas: 85, readyReplicas: 85},
			inactive: []surgeGSSFixture{
				{statusReplicas: 40, allocatedReplicas: 25, reservedReplicas: 15},
				{statusReplicas: 20, allocatedReplicas: 5, reservedReplicas: 15},
			},
			expected: 28,
		},
		"scales down while the active gameserverset is creating": {
			fleetReplicas: 70,
			active:        surgeGSSFixture{specReplicas: 85, statusReplicas: 50, readyReplicas: 10, allocatedReplicas: 40},
			inactive: []surgeGSSFixture{
				{statusReplicas: 20, allocatedReplicas: 20},
				{statusReplicas: 20, reservedReplicas: 20},
			},
			expected: 48,
		},
		"does not raise the target while the active gameserverset is creating": {
			fleetReplicas: 100,
			active:        surgeGSSFixture{specReplicas: 25, statusReplicas: 10, readyReplicas: 10},
			inactive: []surgeGSSFixture{
				{statusReplicas: 40, allocatedReplicas: 40},
				{statusReplicas: 20, reservedReplicas: 20},
			},
			expected: 25,
		},
		"inactive servers exhaust the fleet and surge budget": {
			fleetReplicas: 30,
			active:        surgeGSSFixture{specReplicas: 10, statusReplicas: 10, readyReplicas: 10},
			inactive: []surgeGSSFixture{
				{statusReplicas: 20, allocatedReplicas: 20},
				{statusReplicas: 20, reservedReplicas: 20},
			},
			expected: 0,
		},
	}

	for name, fixture := range fixtures {
		for _, reverse := range []bool{false, true} {
			order := "forward"
			if reverse {
				order = "reverse"
			}
			t.Run(name+"/"+order, func(t *testing.T) {
				f := defaultFixture()
				f.Spec.Replicas = fixture.fleetReplicas
				active, rest := surgeGameServerSets(f, fixture.active, fixture.inactive...)
				if reverse {
					rest[0], rest[1] = rest[1], rest[0]
				}

				c, _ := newFakeController()
				replicas, err := c.rollingUpdateActive(f, active, rest)
				require.NoError(t, err)
				assert.Equal(t, fixture.expected, replicas)
			})
		}
	}
}
