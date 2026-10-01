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

package gameserverallocations

import (
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPendingAllocations(t *testing.T) {
	t.Parallel()

	newGs := func(resourceVersion string) *agonesv1.GameServer {
		return &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, ResourceVersion: resourceVersion}}
	}
	key := defaultNs + "/gs1"

	t.Run("take returns the requests in planning order and keeps the last projection", func(t *testing.T) {
		t.Parallel()
		p := newPendingAllocations()
		r1 := request{response: make(chan response, 1)}
		r2 := request{response: make(chan response, 1)}

		p.add(key, r1, newGs("1"))
		p.add(key, r2, newGs("2"))

		projections := p.projections()
		require.Len(t, projections, 1)
		assert.Equal(t, "2", projections[0].ObjectMeta.ResourceVersion)

		reqs := p.take(key)
		require.Len(t, reqs, 2)
		assert.Equal(t, r1.response, reqs[0].response)
		assert.Equal(t, r2.response, reqs[1].response)
		assert.Empty(t, p.take(key))

		// the projection stays visible to the planner while the update is in progress
		assert.Len(t, p.projections(), 1)
	})

	t.Run("finish drops the projection once nothing is pending", func(t *testing.T) {
		t.Parallel()
		p := newPendingAllocations()
		p.add(key, request{}, newGs("1"))

		p.take(key)
		p.finish(key)

		assert.Empty(t, p.projections())
	})

	t.Run("finish keeps the projection of requests planned during the update", func(t *testing.T) {
		t.Parallel()
		p := newPendingAllocations()
		p.add(key, request{}, newGs("1"))

		p.take(key)
		p.add(key, request{}, newGs("2"))
		p.finish(key)

		projections := p.projections()
		require.Len(t, projections, 1)
		assert.Equal(t, "2", projections[0].ObjectMeta.ResourceVersion)
		assert.Len(t, p.take(key), 1)
	})

	t.Run("take of an unknown key returns nothing", func(t *testing.T) {
		t.Parallel()
		p := newPendingAllocations()

		assert.Empty(t, p.take(key))
		p.finish(key)
		assert.Empty(t, p.projections())
	})
}
