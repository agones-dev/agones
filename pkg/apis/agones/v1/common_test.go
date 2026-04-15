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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestAllocationOverflowValidate(t *testing.T) {
	// valid
	type expected struct {
		fields []string
	}

	fixtures := map[string]struct {
		ao AllocationOverflow
		expected
	}{
		"empty": {
			ao: AllocationOverflow{},
			expected: expected{
				fields: nil,
			},
		},
		"bad label name": {
			ao: AllocationOverflow{
				Labels:      map[string]string{"$$$foobar": "stuff"},
				Annotations: nil,
			},
			expected: expected{
				fields: []string{"spec.allocationOverflow.labels"},
			},
		},
		"bad label value": {
			ao: AllocationOverflow{
				Labels:      map[string]string{"valid": "$$$NOPE"},
				Annotations: nil,
			},
			expected: expected{
				fields: []string{"spec.allocationOverflow.labels"},
			},
		},
		"bad annotation name": {
			ao: AllocationOverflow{
				Annotations: map[string]string{"$$$foobar": "stuff"},
			},
			expected: expected{
				fields: []string{"spec.allocationOverflow.annotations"},
			},
		},
		"valid full": {
			ao: AllocationOverflow{
				Labels:      map[string]string{"valid": "yes", "still.valid": "check-me-out"},
				Annotations: map[string]string{"icando-this": "yes, I can do all kinds of things here $$$"},
			},
			expected: expected{
				fields: nil,
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			errs := v.ao.Validate(field.NewPath("spec", "allocationOverflow"))
			assert.Len(t, errs, len(v.fields))
			for i, err := range errs {
				assert.Equal(t, field.ErrorTypeInvalid, err.Type)
				assert.Equal(t, v.expected.fields[i], err.Field)
			}
		})
	}
}

func TestAllocationOverflowCountMatches(t *testing.T) {
	type expected struct {
		count int32
		rest  int
	}

	fixtures := map[string]struct {
		list     func([]*GameServer)
		ao       func(*AllocationOverflow)
		expected expected
	}{
		"simple": {
			list: func(_ []*GameServer) {},
			ao:   func(_ *AllocationOverflow) {},
			expected: expected{
				count: 2,
				rest:  0,
			},
		},
		"label selector": {
			list: func(list []*GameServer) {
				list[0].ObjectMeta.Labels = map[string]string{"colour": "blue"}
			},
			ao: func(ao *AllocationOverflow) {
				ao.Labels = map[string]string{"colour": "blue"}
			},
			expected: expected{
				count: 1,
				rest:  1,
			},
		},
		"annotation selector": {
			list: func(list []*GameServer) {
				list[0].ObjectMeta.Annotations = map[string]string{"colour": "green"}
			},
			ao: func(ao *AllocationOverflow) {
				ao.Annotations = map[string]string{"colour": "green"}
			},
			expected: expected{
				count: 1,
				rest:  1,
			},
		},
		"both": {
			list: func(list []*GameServer) {
				list[0].ObjectMeta.Labels = map[string]string{"colour": "blue"}
				list[0].ObjectMeta.Annotations = map[string]string{"colour": "green"}
			},
			ao: func(ao *AllocationOverflow) {
				ao.Labels = map[string]string{"colour": "blue"}
				ao.Annotations = map[string]string{"colour": "green"}
			},
			expected: expected{
				count: 1,
				rest:  1,
			},
		},
	}

	for k, v := range fixtures {
		t.Run(k, func(t *testing.T) {
			list := []*GameServer{
				{ObjectMeta: metav1.ObjectMeta{Name: "g1"}, Status: GameServerStatus{State: GameServerStateAllocated}},
				{ObjectMeta: metav1.ObjectMeta{Name: "g2"}, Status: GameServerStatus{State: GameServerStateAllocated}},
				{ObjectMeta: metav1.ObjectMeta{Name: "g3"}, Status: GameServerStatus{State: GameServerStateReady}},
			}
			v.list(list)
			ao := &AllocationOverflow{
				Labels:      nil,
				Annotations: nil,
			}
			v.ao(ao)

			count, rest := ao.CountMatches(list)
			assert.Equal(t, v.expected.count, count, "count")
			assert.Equal(t, v.expected.rest, len(rest), "rest")
			for _, gs := range rest {
				assert.Equal(t, GameServerStateAllocated, gs.Status.State)
			}
		})
	}
}

func TestAllocationOverflowApply(t *testing.T) {
	// check empty
	gs := &GameServer{}
	ao := AllocationOverflow{Labels: map[string]string{"colour": "green"}, Annotations: map[string]string{"colour": "blue", "map": "ice cream"}}

	ao.Apply(gs)

	require.Equal(t, ao.Annotations, gs.ObjectMeta.Annotations)
	require.Equal(t, ao.Labels, gs.ObjectMeta.Labels)

	// check append
	ao = AllocationOverflow{Labels: map[string]string{"version": "1.0"}, Annotations: map[string]string{"version": "1.0"}}
	ao.Apply(gs)

	require.Equal(t, map[string]string{"colour": "green", "version": "1.0"}, gs.ObjectMeta.Labels)
	require.Equal(t, map[string]string{"colour": "blue", "map": "ice cream", "version": "1.0"}, gs.ObjectMeta.Annotations)

	// check overwrite
	ao = AllocationOverflow{Labels: map[string]string{"colour": "red"}, Annotations: map[string]string{"colour": "green"}}
	ao.Apply(gs)
	require.Equal(t, map[string]string{"colour": "red", "version": "1.0"}, gs.ObjectMeta.Labels)
	require.Equal(t, map[string]string{"colour": "green", "map": "ice cream", "version": "1.0"}, gs.ObjectMeta.Annotations)
}
