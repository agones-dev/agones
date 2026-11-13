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
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestControllerRollingUpdateRestMultipleInactiveGameServerSets(t *testing.T) {
	t.Parallel()

	ready := surgeGSSFixture{specReplicas: 50, statusReplicas: 50, readyReplicas: 50}
	fixtures := map[string]struct {
		inactive []surgeGSSFixture
		expected map[string]int32
	}{
		"skips a gameserverset held by allocated gameservers": {
			inactive: []surgeGSSFixture{{specReplicas: 5, statusReplicas: 20, allocatedReplicas: 20}, ready},
			expected: map[string]int32{"inactive-1": 24},
		},
		"skips a gameserverset held by reserved gameservers": {
			inactive: []surgeGSSFixture{{specReplicas: 5, statusReplicas: 20, reservedReplicas: 20}, ready},
			expected: map[string]int32{"inactive-1": 24},
		},
		"skips a gameserverset held by allocated and reserved gameservers": {
			inactive: []surgeGSSFixture{{specReplicas: 5, statusReplicas: 20, allocatedReplicas: 10, reservedReplicas: 10}, ready},
			expected: map[string]int32{"inactive-1": 24},
		},
		"scales down before a gameserverset held by allocated gameservers": {
			inactive: []surgeGSSFixture{ready, {specReplicas: 5, statusReplicas: 20, allocatedReplicas: 20}},
			expected: map[string]int32{"inactive": 24},
		},
		"waits for a gameserverset that is scaling down": {
			inactive: []surgeGSSFixture{{specReplicas: 5, statusReplicas: 20, readyReplicas: 10, allocatedReplicas: 10}, ready},
			expected: map[string]int32{},
		},
	}

	for name, fixture := range fixtures {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := defaultFixture()
			f.Spec.Replicas = 100
			active, rest := surgeGameServerSets(f, surgeGSSFixture{specReplicas: 55, statusReplicas: 55, readyReplicas: 55}, fixture.inactive...)

			c, m := newFakeController()
			updated := map[string]int32{}
			m.AgonesClient.AddReactor("update", "gameserversets", func(action k8stesting.Action) (bool, runtime.Object, error) {
				gsSet := action.(k8stesting.UpdateAction).GetObject().(*agonesv1.GameServerSet)
				updated[gsSet.ObjectMeta.Name] = gsSet.Spec.Replicas
				return true, gsSet, nil
			})

			require.NoError(t, c.rollingUpdateRest(context.Background(), f, active, rest))
			assert.Equal(t, fixture.expected, updated)
		})
	}
}
