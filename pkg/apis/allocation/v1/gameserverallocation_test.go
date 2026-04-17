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

package v1

import (
	"fmt"
	"sort"
	"testing"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestGameServerAllocationApplyDefaults(t *testing.T) {
	t.Parallel()

	gsa := &GameServerAllocation{}
	gsa.ApplyDefaults()

	assert.Equal(t, apis.Packed, gsa.Spec.Scheduling)

	priorities := []agonesv1.Priority{
		{Type: agonesv1.GameServerPriorityList},
		{Type: agonesv1.GameServerPriorityCounter},
	}
	expectedPrioritiesWithDefault := []agonesv1.Priority{
		{Type: agonesv1.GameServerPriorityList, Order: agonesv1.GameServerPriorityAscending},
		{Type: agonesv1.GameServerPriorityCounter, Order: agonesv1.GameServerPriorityAscending},
	}

	gsa = &GameServerAllocation{Spec: GameServerAllocationSpec{Scheduling: apis.Distributed, Priorities: priorities}}
	gsa.ApplyDefaults()
	assert.Equal(t, apis.Distributed, gsa.Spec.Scheduling)
	assert.Equal(t, expectedPrioritiesWithDefault, gsa.Spec.Priorities)

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true&%s=true", runtime.FeaturePlayerAllocationFilter, runtime.FeatureCountsAndLists)))

	gsa = &GameServerAllocation{}
	gsa.ApplyDefaults()

	assert.Equal(t, agonesv1.GameServerStateReady, *gsa.Spec.Required.GameServerState)
	assert.Equal(t, int64(0), gsa.Spec.Required.Players.MaxAvailable)
	assert.Equal(t, int64(0), gsa.Spec.Required.Players.MinAvailable)
	assert.Equal(t, []agonesv1.Priority(nil), gsa.Spec.Priorities)
	assert.Nil(t, gsa.Spec.Priorities)
}

// nolint // Current lint duplicate threshold will consider this function is a duplication of the function TestGameServerAllocationSpecSelectors
func TestGameServerAllocationSpecPreferredSelectors(t *testing.T) {
	t.Parallel()

	gsas := &GameServerAllocationSpec{
		Preferred: []GameServerSelector{
			{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"check": "blue"}}},
			{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"check": "red"}}},
		},
	}

	require.Len(t, gsas.Preferred, 2)

	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}}}

	for _, s := range gsas.Preferred {
		assert.False(t, s.Matches(gs))
	}

	gs.ObjectMeta.Labels["check"] = "blue"
	assert.True(t, gsas.Preferred[0].Matches(gs))
	assert.False(t, gsas.Preferred[1].Matches(gs))

	gs.ObjectMeta.Labels["check"] = "red"
	assert.False(t, gsas.Preferred[0].Matches(gs))
	assert.True(t, gsas.Preferred[1].Matches(gs))
}

// nolint // Current lint duplicate threshold will consider this function is a duplication of the function TestGameServerAllocationSpecPreferredSelectors
func TestGameServerAllocationSpecSelectors(t *testing.T) {
	t.Parallel()

	gsas := &GameServerAllocationSpec{
		Selectors: []GameServerSelector{
			{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"check": "blue"}}},
			{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"check": "red"}}},
		},
	}

	require.Len(t, gsas.Selectors, 2)

	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{}}}

	for _, s := range gsas.Selectors {
		assert.False(t, s.Matches(gs))
	}

	gs.ObjectMeta.Labels["check"] = "blue"
	assert.True(t, gsas.Selectors[0].Matches(gs))
	assert.False(t, gsas.Selectors[1].Matches(gs))

	gs.ObjectMeta.Labels["check"] = "red"
	assert.False(t, gsas.Selectors[0].Matches(gs))
	assert.True(t, gsas.Selectors[1].Matches(gs))
}

func TestGameServerSelectorApplyDefaults(t *testing.T) {
	t.Parallel()
	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	require.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true&%s=true",
		runtime.FeaturePlayerAllocationFilter,
		runtime.FeatureCountsAndLists)))

	s := &GameServerSelector{}

	// no defaults
	s.ApplyDefaults()
	assert.Equal(t, agonesv1.GameServerStateReady, *s.GameServerState)
	assert.Equal(t, int64(0), s.Players.MinAvailable)
	assert.Equal(t, int64(0), s.Players.MaxAvailable)
	assert.NotNil(t, s.Counters)
	assert.NotNil(t, s.Lists)

	// Test apply defaults is idempotent -- calling ApplyDefaults more than one time does not change the original result.
	s.ApplyDefaults()
	assert.Equal(t, agonesv1.GameServerStateReady, *s.GameServerState)
	assert.Equal(t, int64(0), s.Players.MinAvailable)
	assert.Equal(t, int64(0), s.Players.MaxAvailable)
	assert.NotNil(t, s.Counters)
	assert.NotNil(t, s.Lists)

	state := agonesv1.GameServerStateAllocated
	// set values
	s = &GameServerSelector{
		GameServerState: &state,
		Players:         &PlayerSelector{MinAvailable: 10, MaxAvailable: 20},
		Counters:        map[string]CounterSelector{"foo": {MinAvailable: 1, MaxAvailable: 10}},
		Lists:           map[string]ListSelector{"bar": {MinAvailable: 2}},
	}
	s.ApplyDefaults()
	assert.Equal(t, state, *s.GameServerState)
	assert.Equal(t, int64(10), s.Players.MinAvailable)
	assert.Equal(t, int64(20), s.Players.MaxAvailable)
	assert.Equal(t, int64(0), s.Counters["foo"].MinCount)
	assert.Equal(t, int64(0), s.Counters["foo"].MaxCount)
	assert.Equal(t, int64(1), s.Counters["foo"].MinAvailable)
	assert.Equal(t, int64(10), s.Counters["foo"].MaxAvailable)
	assert.Equal(t, int64(2), s.Lists["bar"].MinAvailable)
	assert.Equal(t, int64(0), s.Lists["bar"].MaxAvailable)
	assert.Equal(t, "", s.Lists["bar"].ContainsValue)

	// Test apply defaults is idempotent -- calling ApplyDefaults more than one time does not change the original result.
	s.ApplyDefaults()
	assert.Equal(t, state, *s.GameServerState)
	assert.Equal(t, int64(10), s.Players.MinAvailable)
	assert.Equal(t, int64(20), s.Players.MaxAvailable)
	assert.Equal(t, int64(0), s.Counters["foo"].MinCount)
	assert.Equal(t, int64(0), s.Counters["foo"].MaxCount)
	assert.Equal(t, int64(1), s.Counters["foo"].MinAvailable)
	assert.Equal(t, int64(10), s.Counters["foo"].MaxAvailable)
	assert.Equal(t, int64(2), s.Lists["bar"].MinAvailable)
	assert.Equal(t, int64(0), s.Lists["bar"].MaxAvailable)
	assert.Equal(t, "", s.Lists["bar"].ContainsValue)
}

func TestGameServerSelectorValidate(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	require.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true&%s=true", runtime.FeaturePlayerAllocationFilter, runtime.FeatureCountsAndLists)))

	allocated := agonesv1.GameServerStateAllocated
	starting := agonesv1.GameServerStateStarting

	fixtures := map[string]struct {
		selector *GameServerSelector
		want     field.ErrorList
	}{
		"valid": {
			selector: &GameServerSelector{GameServerState: &allocated, Players: &PlayerSelector{
				MinAvailable: 0,
				MaxAvailable: 10,
			}},
		},
		"nil values": {
			selector: &GameServerSelector{},
		},
		"invalid state": {
			selector: &GameServerSelector{
				GameServerState: &starting,
			},
			want: field.ErrorList{
				field.Invalid(field.NewPath("fieldName.gameServerState"), starting, "GameServerState must be either Allocated or Ready"),
			},
		},
		"invalid min value": {
			selector: &GameServerSelector{
				Players: &PlayerSelector{
					MinAvailable: -10,
				},
			},
			want: field.ErrorList{
				field.Invalid(field.NewPath("fieldName", "players", "minAvailable"), int64(-10), "must be greater than or equal to 0"),
			},
		},
		"invalid max value": {
			selector: &GameServerSelector{
				Players: &PlayerSelector{
					MinAvailable: -30,
					MaxAvailable: -20,
				},
			},
			want: field.ErrorList{
				field.Invalid(field.NewPath("fieldName", "players", "minAvailable"), int64(-30), "must be greater than or equal to 0"),
				field.Invalid(field.NewPath("fieldName", "players", "maxAvailable"), int64(-20), "must be greater than or equal to 0"),
			},
		},
		"invalid min/max value": {
			selector: &GameServerSelector{
				Players: &PlayerSelector{
					MinAvailable: 10,
					MaxAvailable: 5,
				},
			},
			want: field.ErrorList{
				field.Invalid(field.NewPath("fieldName", "players", "minAvailable"), int64(10), "minAvailable cannot be greater than maxAvailable"),
			},
		},
		"invalid label keys": {
			selector: &GameServerSelector{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"$$$$": "true"},
				},
			},
			want: field.ErrorList{
				field.Invalid(
					field.NewPath("fieldName", "labelSelector"),
					metav1.LabelSelector{MatchLabels: map[string]string{"$$$$": "true"}},
					`Error converting label selector: key: Invalid value: "$$$$": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')`,
				),
			},
		},
		"invalid min/max Counter available value": {
			selector: &GameServerSelector{
				Counters: map[string]CounterSelector{
					"counter": {
						MinAvailable: -1,
						MaxAvailable: -1,
					},
				},
			},
			want: field.ErrorList{
				field.Invalid(field.NewPath("fieldName", "counters[counter]", "minAvailable"), int64(-1), "must be greater than or equal to 0"),
				field.Invalid(field.NewPath("fieldName", "counters[counter]", "maxAvailable"), int64(-1), "must be greater than or equal to 0"),
			},
		},
		"invalid max less than min Counter available value": {
			selector: &GameServerSelector{
				Counters: map[string]CounterSelector{
					"foo": {
						MinAvailable: 10,
						MaxAvailable: 1,
					},
				},
			},
			want: field.ErrorList{
				field.Invalid(field.NewPath("fieldName", "counters[foo]"), int64(1), "maxAvailable must zero or greater than minAvailable 10"),
			},
		},
		"invalid min/max Counter count value": {
			selector: &GameServerSelector{
				Counters: map[string]CounterSelector{
					"counter": {
						MinCount: -1,
						MaxCount: -1,
					},
				},
			},
			want: field.ErrorList{
				field.Invalid(field.NewPath("fieldName", "counters[counter]", "minCount"), int64(-1), "must be greater than or equal to 0"),
				field.Invalid(field.NewPath("fieldName", "counters[counter]", "maxCount"), int64(-1), "must be greater than or equal to 0"),
			},
		},
		"invalid max less than min Counter count value": {
			selector: &GameServerSelector{
				Counters: map[string]CounterSelector{
					"foo": {
						MinCount: 10,
						MaxCount: 1,
					},
				},
			},
			want: field.ErrorList{
				field.Invalid(field.NewPath("fieldName", "counters[foo]"), int64(1), "maxCount must zero or greater than minCount 10"),
			},
		},
		"invalid min/max List value": {
			selector: &GameServerSelector{
				Lists: map[string]ListSelector{
					"list": {
						MinAvailable: -11,
						MaxAvailable: -11,
					},
				},
			},
			want: field.ErrorList{
				field.Invalid(field.NewPath("fieldName", "lists[list]", "minAvailable"), int64(-11), "must be greater than or equal to 0"),
				field.Invalid(field.NewPath("fieldName", "lists[list]", "maxAvailable"), int64(-11), "must be greater than or equal to 0"),
			},
		},
		"invalid max less than min List value": {
			selector: &GameServerSelector{
				Lists: map[string]ListSelector{
					"list": {
						MinAvailable: 11,
						MaxAvailable: 2,
					},
				},
			},
			want: field.ErrorList{
				field.Invalid(field.NewPath("fieldName", "lists[list]"), int64(2), "maxAvailable must zero or greater than minAvailable 11"),
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			v.selector.ApplyDefaults()
			allErrs := v.selector.Validate(field.NewPath("fieldName"))
			assert.ElementsMatch(t, v.want, allErrs)
		})
	}
}

func TestMetaPatchValidate(t *testing.T) {
	t.Parallel()

	// valid
	mp := &MetaPatch{
		Labels:      nil,
		Annotations: nil,
	}
	path := field.NewPath("spec", "metadata")
	allErrs := mp.Validate(path)
	assert.Len(t, allErrs, 0)

	mp.Labels = map[string]string{}
	mp.Annotations = map[string]string{}
	allErrs = mp.Validate(path)
	assert.Len(t, allErrs, 0)

	mp.Labels["foo"] = "bar"
	mp.Annotations["bar"] = "foo"
	allErrs = mp.Validate(path)
	assert.Len(t, allErrs, 0)

	// invalid label
	invalid := mp.DeepCopy()
	invalid.Labels["$$$$"] = "no"
	allErrs = invalid.Validate(path)
	assert.Len(t, allErrs, 1)
	assert.Equal(t, "spec.metadata.labels", allErrs[0].Field)

	// invalid annotation
	invalid = mp.DeepCopy()
	invalid.Annotations["$$$$"] = "no"

	allErrs = invalid.Validate(path)
	require.Len(t, allErrs, 1)
	assert.Equal(t, "spec.metadata.annotations", allErrs[0].Field)

	// invalid both
	invalid.Labels["$$$$"] = "no"
	allErrs = invalid.Validate(path)
	require.Len(t, allErrs, 2)
	assert.Equal(t, "spec.metadata.labels", allErrs[0].Field)
	assert.Equal(t, "spec.metadata.annotations", allErrs[1].Field)
}

func TestGameServerSelectorMatches(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	blueSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{"colour": "blue"},
	}

	allocatedState := agonesv1.GameServerStateAllocated
	fixtures := map[string]struct {
		features   string
		selector   *GameServerSelector
		gameServer *agonesv1.GameServer
		matches    bool
	}{
		"no labels, pass": {
			selector:   &GameServerSelector{},
			gameServer: &agonesv1.GameServer{},
			matches:    true,
		},

		"no labels, fail": {
			selector: &GameServerSelector{
				LabelSelector: blueSelector,
			},
			gameServer: &agonesv1.GameServer{},
			matches:    false,
		},
		"single label, match": {
			selector: &GameServerSelector{
				LabelSelector: blueSelector,
			},
			gameServer: &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"colour": "blue"},
				},
			},
			matches: true,
		},
		"single label, fail": {
			selector: &GameServerSelector{
				LabelSelector: blueSelector,
			},
			gameServer: &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"colour": "purple"},
				},
			},
			matches: false,
		},
		"two labels, pass": {
			selector: &GameServerSelector{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"colour": "blue", "animal": "frog"},
				},
			},
			gameServer: &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"colour": "blue", "animal": "frog"}},
			},
			matches: true,
		},
		"two labels, fail": {
			selector: &GameServerSelector{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"colour": "blue", "animal": "cat"},
				},
			},
			gameServer: &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"colour": "blue", "animal": "frog"}},
			},
			matches: false,
		},
		"state filter, pass": {
			selector: &GameServerSelector{
				GameServerState: &allocatedState,
			},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{State: allocatedState}},
			matches:    true,
		},
		"state filter, fail": {
			selector: &GameServerSelector{
				GameServerState: &allocatedState,
			},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{State: agonesv1.GameServerStateReady}},
			matches:    false,
		},
		"player tracking, between, pass": {
			features: string(runtime.FeaturePlayerAllocationFilter) + "=true",
			selector: &GameServerSelector{Players: &PlayerSelector{
				MinAvailable: 10,
				MaxAvailable: 20,
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Players: &agonesv1.PlayerStatus{
					Count:    20,
					Capacity: 35,
				},
			}},
			matches: true,
		},
		"player tracking, between, fail": {
			features: string(runtime.FeaturePlayerAllocationFilter) + "=true",
			selector: &GameServerSelector{Players: &PlayerSelector{
				MinAvailable: 10,
				MaxAvailable: 20,
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Players: &agonesv1.PlayerStatus{
					Count:    30,
					Capacity: 35,
				},
			}},
			matches: false,
		},
		"player tracking, max, pass": {
			features: string(runtime.FeaturePlayerAllocationFilter) + "=true",
			selector: &GameServerSelector{Players: &PlayerSelector{
				MinAvailable: 10,
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Players: &agonesv1.PlayerStatus{
					Count:    20,
					Capacity: 35,
				},
			}},
			matches: true,
		},
		"player tracking, max, fail": {
			features: string(runtime.FeaturePlayerAllocationFilter) + "=true",
			selector: &GameServerSelector{Players: &PlayerSelector{
				MinAvailable: 10,
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Players: &agonesv1.PlayerStatus{
					Count:    30,
					Capacity: 35,
				},
			}},
			matches: true,
		},
		"combo": {
			features: string(runtime.FeaturePlayerAllocationFilter) + "=true&",
			selector: &GameServerSelector{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"colour": "blue"},
				},
				GameServerState: &allocatedState,
				Players: &PlayerSelector{
					MinAvailable: 10,
					MaxAvailable: 20,
				},
			},
			gameServer: &agonesv1.GameServer{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"colour": "blue"},
				},
				Status: agonesv1.GameServerStatus{
					State: allocatedState,
					Players: &agonesv1.PlayerStatus{
						Count:    5,
						Capacity: 25,
					},
				},
			},
			matches: true,
		},
		"Counter has available capacity": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"sessions": {
					MinAvailable: 1,
					MaxAvailable: 1000,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"sessions": {
						Count:    10,
						Capacity: 1000,
					},
				},
			}},
			matches: true,
		},
		"Counter has below minimum available capacity": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"players": {
					MinAvailable: 100,
					MaxAvailable: 0,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"players": {
						Count:    999,
						Capacity: 1000,
					},
				},
			}},
			matches: false,
		},
		"Counter has above maximum available capacity": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"animals": {
					MinAvailable: 1,
					MaxAvailable: 100,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"animals": {
						Count:    0,
						Capacity: 1000,
					},
				},
			}},
			matches: false,
		},
		"Counter has count in requested range (MaxCount undefined = 0 = unlimited)": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"games": {
					MinCount: 1,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"games": {
						Count:    10,
						Capacity: 1000,
					},
				},
			}},
			matches: true,
		},
		"Counter has count below minimum": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"characters": {
					MinCount: 1,
					MaxCount: 0,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"characters": {
						Count:    0,
						Capacity: 100,
					},
				},
			}},
			matches: false,
		},
		"Counter has count above maximum": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"monsters": {
					MinCount: 0,
					MaxCount: 10,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"monsters": {
						Count:    11,
						Capacity: 100,
					},
				},
			}},
			matches: false,
		},
		"Counter does not exist": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"dragoons": {
					MinCount: 1,
					MaxCount: 10,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"dragons": {
						Count:    1,
						Capacity: 100,
					},
				},
			}},
			matches: false,
		},
		"GameServer does not have Counters": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Counters: map[string]CounterSelector{
				"dragoons": {
					MinCount: 1,
					MaxCount: 10,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"bazzles": {
						Capacity: 3,
						Values:   []string{"baz1", "baz2", "baz3"},
					},
				},
			}},
			matches: false,
		},
		"List has available capacity": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Lists: map[string]ListSelector{
				"lobbies": {
					MinAvailable: 1,
					MaxAvailable: 3,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"lobbies": {
						Capacity: 3,
						Values:   []string{"lobby1", "lobby2"},
					},
				},
			}},
			matches: true,
		},
		"List has below minimum available capacity": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Lists: map[string]ListSelector{
				"avatars": {
					MinAvailable: 1,
					MaxAvailable: 1000,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"avatars": {
						Capacity: 3,
						Values:   []string{"avatar1", "avatar2", "avatar3"},
					},
				},
			}},
			matches: false,
		},
		"List has above maximum available capacity": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Lists: map[string]ListSelector{
				"things": {
					MinAvailable: 1,
					MaxAvailable: 10,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"things": {
						Capacity: 1000,
						Values:   []string{"thing1", "thing2", "thing3"},
					},
				},
			}},
			matches: false,
		},
		"List does not exist": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Lists: map[string]ListSelector{
				"thingamabobs": {
					MinAvailable: 1,
					MaxAvailable: 100,
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"thingamajigs": {
						Capacity: 100,
						Values:   []string{"thingamajig1", "thingamajig2"},
					},
				},
			}},
			matches: false,
		},
		"List contains value": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Lists: map[string]ListSelector{
				"bazzles": {
					ContainsValue: "baz1",
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"bazzles": {
						Capacity: 3,
						Values:   []string{"baz1", "baz2", "baz3"},
					},
				},
			}},
			matches: true,
		},
		"List does not contain value": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Lists: map[string]ListSelector{
				"bazzles": {
					ContainsValue: "BAZ1",
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"bazzles": {
						Capacity: 3,
						Values:   []string{"baz1", "baz2", "baz3"},
					},
				},
			}},
			matches: false,
		},
		"GameServer does not have Lists": {
			features: string(runtime.FeatureCountsAndLists) + "=true",
			selector: &GameServerSelector{Lists: map[string]ListSelector{
				"bazzles": {
					ContainsValue: "BAZ1",
				},
			}},
			gameServer: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"dragons": {
						Count:    1,
						Capacity: 100,
					},
				},
			}},
			matches: false,
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			if v.features != "" {
				require.NoError(t, runtime.ParseFeatures(v.features))
			}

			match := v.selector.Matches(v.gameServer)
			assert.Equal(t, v.matches, match)
		})
	}
}

// Helper function for creating int64 pointers
func int64Pointer(x int64) *int64 {
	return &x
}

func TestGameServerCounterActions(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists)))

	DECREMENT := "Decrement"
	INCREMENT := "Increment"

	testScenarios := map[string]struct {
		ca      CounterAction
		counter string
		gs      *agonesv1.GameServer
		want    *agonesv1.GameServer
		wantErr bool
	}{
		"update counter capacity and count is set to capacity": {
			ca: CounterAction{
				Capacity: int64Pointer(0),
			},
			counter: "mages",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"mages": {
						Count:    1,
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"mages": {
						Count:    0,
						Capacity: 0,
					}}}},
			wantErr: false,
		},
		"fail update counter capacity and truncate update count": {
			ca: CounterAction{
				Action:   &INCREMENT,
				Amount:   int64Pointer(10),
				Capacity: int64Pointer(-1),
			},
			counter: "sages",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"sages": {
						Count:    99,
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"sages": {
						Count:    100,
						Capacity: 100,
					}}}},
			wantErr: true,
		},
		"update counter count": {
			ca: CounterAction{
				Action: &INCREMENT,
				Amount: int64Pointer(10),
			},
			counter: "baddies",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"baddies": {
						Count:    1,
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"baddies": {
						Count:    11,
						Capacity: 100,
					}}}},
			wantErr: false,
		},
		"update counter count and capacity": {
			ca: CounterAction{
				Action:   &DECREMENT,
				Amount:   int64Pointer(10),
				Capacity: int64Pointer(10),
			},
			counter: "heroes",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"heroes": {
						Count:    11,
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Counters: map[string]agonesv1.CounterStatus{
					"heroes": {
						// Note: The Capacity is set first, and Count updated to not be greater than Capacity.
						// Then the Count is decremented. See: gameserver.go/UpdateCounterCapacity
						Count:    0,
						Capacity: 10,
					}}}},
			wantErr: false,
		},
	}

	for test, testScenario := range testScenarios {
		t.Run(test, func(t *testing.T) {
			errs := testScenario.ca.CounterActions(testScenario.counter, testScenario.gs)
			if errs != nil {
				assert.True(t, testScenario.wantErr)
			} else {
				assert.False(t, testScenario.wantErr)
			}
			assert.Equal(t, testScenario.want, testScenario.gs)
		})
	}
}

func TestGameServerListActions(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists)))

	testScenarios := map[string]struct {
		la      ListAction
		list    string
		gs      *agonesv1.GameServer
		want    *agonesv1.GameServer
		wantErr bool
	}{
		"update list capacity truncates list": {
			la: ListAction{
				Capacity: int64Pointer(0),
			},
			list: "pages",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"pages": {
						Values:   []string{"page1", "page2"},
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"pages": {
						Values:   []string{},
						Capacity: 0,
					}}}},
			wantErr: false,
		},
		"update list values": {
			la: ListAction{
				AddValues: []string{"sage1", "sage3"},
			},
			list: "sages",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"sages": {
						Values:   []string{"sage1", "sage2"},
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"sages": {
						Values:   []string{"sage1", "sage2", "sage3"},
						Capacity: 100,
					}}}},
			wantErr: false,
		},
		"update list values and capacity": {
			la: ListAction{
				AddValues:    []string{"magician1", "magician3"},
				Capacity:     int64Pointer(42),
				DeleteValues: []string{"magician2", "magician5", "magician6"},
			},
			list: "magicians",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"magicians": {
						Values:   []string{"magician1", "magician2", "magician4", "magician5"},
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"magicians": {
						Values:   []string{"magician1", "magician4", "magician3"},
						Capacity: 42,
					}}}},
			wantErr: false,
		},
		"update list values and capacity - value add truncates silently": {
			la: ListAction{
				AddValues: []string{"fairy1", "fairy3"},
				Capacity:  int64Pointer(2),
			},
			list: "fairies",
			gs: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"fairies": {
						Values:   []string{"fairy1", "fairy2"},
						Capacity: 100,
					}}}},
			want: &agonesv1.GameServer{Status: agonesv1.GameServerStatus{
				Lists: map[string]agonesv1.ListStatus{
					"fairies": {
						Values:   []string{"fairy1", "fairy2"},
						Capacity: 2,
					}}}},
			wantErr: false,
		},
	}

	for test, testScenario := range testScenarios {
		t.Run(test, func(t *testing.T) {
			errs := testScenario.la.ListActions(testScenario.list, testScenario.gs)
			if errs != nil {
				assert.True(t, testScenario.wantErr)
			} else {
				assert.False(t, testScenario.wantErr)
			}
			assert.Equal(t, testScenario.want, testScenario.gs)
		})
	}
}

func TestValidatePriorities(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists)))

	fieldPath := field.NewPath("spec.Priorities")

	testScenarios := map[string]struct {
		priorities []agonesv1.Priority
		wantErr    bool
	}{
		"Valid priorities": {
			priorities: []agonesv1.Priority{
				{
					Type:  agonesv1.GameServerPriorityList,
					Key:   "test",
					Order: agonesv1.GameServerPriorityAscending,
				},
				{
					Type:  agonesv1.GameServerPriorityCounter,
					Key:   "test",
					Order: agonesv1.GameServerPriorityDescending,
				},
			},
			wantErr: false,
		},
		"No type": {
			priorities: []agonesv1.Priority{
				{
					Key:   "test",
					Order: agonesv1.GameServerPriorityDescending,
				},
			},
			wantErr: true,
		},
		"Invalid type": {
			priorities: []agonesv1.Priority{
				{
					Key:   "test",
					Type:  "invalid",
					Order: agonesv1.GameServerPriorityDescending,
				},
			},
			wantErr: true,
		},
		"No Key": {
			priorities: []agonesv1.Priority{
				{
					Type:  agonesv1.GameServerPriorityCounter,
					Order: agonesv1.GameServerPriorityDescending,
				},
			},
			wantErr: true,
		},
		"No Order": {
			priorities: []agonesv1.Priority{
				{
					Type: agonesv1.GameServerPriorityList,
					Key:  "test",
				},
			},
			wantErr: true,
		},
		"Invalid Order": {
			priorities: []agonesv1.Priority{
				{
					Type:  agonesv1.GameServerPriorityList,
					Key:   "test",
					Order: "invalid",
				},
			},
			wantErr: true,
		},
	}

	for test, testScenario := range testScenarios {
		t.Run(test, func(t *testing.T) {
			allErrs := validatePriorities(testScenario.priorities, fieldPath)
			if testScenario.wantErr {
				assert.NotNil(t, allErrs)
			} else {
				assert.Nil(t, allErrs)
			}
		})
	}
}

func TestValidateCounterActions(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists)))

	fieldPath := field.NewPath("spec.Counters")
	decrement := agonesv1.GameServerPriorityDecrement
	increment := agonesv1.GameServerPriorityIncrement

	testScenarios := map[string]struct {
		counterActions map[string]CounterAction
		wantErr        bool
	}{
		"Valid CounterActions": {
			counterActions: map[string]CounterAction{
				"foo": {
					Action: &increment,
					Amount: int64Pointer(10),
				},
				"bar": {
					Capacity: int64Pointer(100),
				},
				"baz": {
					Action:   &decrement,
					Amount:   int64Pointer(1000),
					Capacity: int64Pointer(0),
				},
			},
			wantErr: false,
		},
		"Negative Amount": {
			counterActions: map[string]CounterAction{
				"foo": {
					Action: &increment,
					Amount: int64Pointer(-1),
				},
			},
			wantErr: true,
		},
		"Negative Capacity": {
			counterActions: map[string]CounterAction{
				"foo": {
					Capacity: int64Pointer(-20),
				},
			},
			wantErr: true,
		},
		"Amount but no Action": {
			counterActions: map[string]CounterAction{
				"foo": {
					Amount: int64Pointer(10),
				},
			},
			wantErr: true,
		},
		"Action but no Amount": {
			counterActions: map[string]CounterAction{
				"foo": {
					Action: &decrement,
				},
			},
			wantErr: true,
		},
	}

	for test, testScenario := range testScenarios {
		t.Run(test, func(t *testing.T) {
			allErrs := validateCounterActions(testScenario.counterActions, fieldPath)
			if testScenario.wantErr {
				assert.NotNil(t, allErrs)
			} else {
				assert.Nil(t, allErrs)
			}
		})
	}
}

func TestValidateListActions(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists)))

	fieldPath := field.NewPath("spec.Lists")

	testScenarios := map[string]struct {
		listActions map[string]ListAction
		wantErr     bool
	}{
		"Valid ListActions": {
			listActions: map[string]ListAction{
				"foo": {
					AddValues: []string{"hello", "world"},
					Capacity:  int64Pointer(10),
				},
				"bar": {
					Capacity: int64Pointer(0),
				},
				"baz": {
					AddValues:    []string{},
					DeleteValues: []string{"good", "bye"},
				},
			},
			wantErr: false,
		},
		"Negative Capacity": {
			listActions: map[string]ListAction{
				"foo": {
					Capacity: int64Pointer(-20),
				},
			},
			wantErr: true,
		},
	}

	for test, testScenario := range testScenarios {
		t.Run(test, func(t *testing.T) {
			allErrs := validateListActions(testScenario.listActions, fieldPath)
			if testScenario.wantErr {
				assert.NotNil(t, allErrs)
			} else {
				assert.Nil(t, allErrs)
			}
		})
	}
}

func TestGameServerAllocationValidate(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true&%s=false",
		runtime.FeaturePlayerAllocationFilter,
		runtime.FeatureCountsAndLists)))

	gsa := &GameServerAllocation{}
	gsa.ApplyDefaults()

	allErrs := gsa.Validate()
	assert.Len(t, allErrs, 0)

	gsa.Spec.Scheduling = "FLERG"

	allErrs = gsa.Validate()
	assert.Len(t, allErrs, 1)

	assert.Equal(t, field.ErrorTypeNotSupported, allErrs[0].Type)
	assert.Equal(t, "spec.scheduling", allErrs[0].Field)

	// invalid player selection
	gsa = &GameServerAllocation{
		Spec: GameServerAllocationSpec{
			Required: GameServerSelector{
				Players: &PlayerSelector{
					MinAvailable: -10,
				},
			},
			Preferred: []GameServerSelector{
				{Players: &PlayerSelector{MaxAvailable: -10}},
			},
			MetaPatch: MetaPatch{
				Labels: map[string]string{"$$$": "foo"},
			},
			Priorities: []agonesv1.Priority{},
			Counters:   map[string]CounterAction{},
			Lists:      map[string]ListAction{},
		},
	}
	gsa.ApplyDefaults()

	allErrs = gsa.Validate()
	sort.Slice(allErrs, func(i, j int) bool {
		return allErrs[i].Field > allErrs[j].Field
	})
	assert.Len(t, allErrs, 7)
	assert.Equal(t, "spec.required.players.minAvailable", allErrs[0].Field)
	assert.Equal(t, "spec.priorities", allErrs[1].Field)
	assert.Equal(t, "spec.preferred[0].players.minAvailable", allErrs[2].Field)
	assert.Equal(t, "spec.preferred[0].players.maxAvailable", allErrs[3].Field)
	assert.Equal(t, "spec.metadata.labels", allErrs[4].Field)
	assert.Equal(t, "spec.lists", allErrs[5].Field)
	assert.Equal(t, "spec.counters", allErrs[6].Field)
}

func TestGameServerAllocationConverter(t *testing.T) {
	t.Parallel()

	gsa := &GameServerAllocation{
		Spec: GameServerAllocationSpec{
			Scheduling: "Packed",
			Required: GameServerSelector{
				Players: &PlayerSelector{
					MinAvailable: 5,
					MaxAvailable: 10,
				},
			},
			Preferred: []GameServerSelector{
				{Players: &PlayerSelector{MinAvailable: 10,
					MaxAvailable: 20}},
			},
		},
	}
	gsaExpected := &GameServerAllocation{
		Spec: GameServerAllocationSpec{
			Scheduling: "Packed",
			Required: GameServerSelector{
				Players: &PlayerSelector{
					MinAvailable: 5,
					MaxAvailable: 10,
				},
			},
			Preferred: []GameServerSelector{
				{Players: &PlayerSelector{MinAvailable: 10,
					MaxAvailable: 20}},
			},
			Selectors: []GameServerSelector{
				{Players: &PlayerSelector{MinAvailable: 10,
					MaxAvailable: 20}},
				{Players: &PlayerSelector{
					MinAvailable: 5,
					MaxAvailable: 10}},
			},
		},
	}

	gsa.Converter()
	assert.Equal(t, gsaExpected, gsa)
}

func TestSortKey(t *testing.T) {
	t.Parallel()

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()
	require.NoError(t, runtime.ParseFeatures(fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists)))

	gameServerAllocation1 := &GameServerAllocation{
		Spec: GameServerAllocationSpec{
			Scheduling: "Packed",
			Priorities: []agonesv1.Priority{
				{
					Type:  "List",
					Key:   "foo",
					Order: "Descending",
				},
			},
		},
	}

	gameServerAllocation2 := &GameServerAllocation{
		Spec: GameServerAllocationSpec{
			Selectors: []GameServerSelector{
				{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}},
			},
			Scheduling: "Packed",
			Priorities: []agonesv1.Priority{
				{
					Type:  "List",
					Key:   "foo",
					Order: "Descending",
				},
			},
		},
	}

	gameServerAllocation3 := &GameServerAllocation{
		Spec: GameServerAllocationSpec{
			Scheduling: "Packed",
			Priorities: []agonesv1.Priority{
				{
					Type:  "Counter",
					Key:   "foo",
					Order: "Descending",
				},
			},
		},
	}

	gameServerAllocation4 := &GameServerAllocation{
		Spec: GameServerAllocationSpec{
			Scheduling: "Distributed",
			Priorities: []agonesv1.Priority{
				{
					Type:  "List",
					Key:   "foo",
					Order: "Descending",
				},
			},
		},
	}

	gameServerAllocation5 := &GameServerAllocation{}

	gameServerAllocation6 := &GameServerAllocation{
		Spec: GameServerAllocationSpec{
			Priorities: []agonesv1.Priority{},
		},
	}

	testScenarios := map[string]struct {
		gsa1      *GameServerAllocation
		gsa2      *GameServerAllocation
		wantEqual bool
	}{
		"equivalent GameServerAllocation": {
			gsa1:      gameServerAllocation1,
			gsa2:      gameServerAllocation2,
			wantEqual: true,
		},
		"different Scheduling GameServerAllocation": {
			gsa1:      gameServerAllocation1,
			gsa2:      gameServerAllocation4,
			wantEqual: false,
		},
		"equivalent empty GameServerAllocation": {
			gsa1:      gameServerAllocation5,
			gsa2:      gameServerAllocation6,
			wantEqual: true,
		},
		"different Priorities GameServerAllocation": {
			gsa1:      gameServerAllocation1,
			gsa2:      gameServerAllocation3,
			wantEqual: false,
		},
	}

	for test, testScenario := range testScenarios {
		t.Run(test, func(t *testing.T) {
			key1, err := testScenario.gsa1.SortKey()
			assert.NoError(t, err)
			key2, err := testScenario.gsa2.SortKey()
			assert.NoError(t, err)

			if testScenario.wantEqual {
				assert.Equal(t, key1, key2)
			} else {
				assert.NotEqual(t, key1, key2)
			}
		})
	}

}
