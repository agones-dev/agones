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
	"context"
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPendingAllocations(t *testing.T) {
	t.Parallel()

	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Namespace: defaultNs, ResourceVersion: "1"}}
	key := defaultNs + "/gs1"
	newRequest := func() request {
		return request{ctx: context.Background(), response: make(chan response, 1)}
	}
	// toApply returns the requests to apply on the cached GameServer
	toApply := func(p *pendingAllocations) []request {
		_, reqs := p.listCachedWithRequestsToApply(func() []*agonesv1.GameServer { return []*agonesv1.GameServer{gs} })
		return reqs[0]
	}

	t.Run("startUpdate returns the requests in planning order and keeps them to apply", func(t *testing.T) {
		t.Parallel()
		p := newPendingAllocations()
		r1, r2 := newRequest(), newRequest()

		p.add(key, r1)
		p.add(key, r2)

		reqs := p.startUpdate(key, gs)
		require.Len(t, reqs, 2)
		assert.Equal(t, r1.response, reqs[0].response)
		assert.Equal(t, r2.response, reqs[1].response)
		assert.Empty(t, p.startUpdate(key, gs))

		// the requests stay visible to the planner while the update is in progress
		assert.Len(t, toApply(p), 2)
	})

	t.Run("finishUpdate drops the requests once nothing is pending", func(t *testing.T) {
		t.Parallel()
		p := newPendingAllocations()
		p.add(key, newRequest())

		p.startUpdate(key, gs)
		p.finishUpdate(key)

		assert.Empty(t, toApply(p))
	})

	t.Run("finishUpdate keeps the requests planned during the update", func(t *testing.T) {
		t.Parallel()
		p := newPendingAllocations()
		p.add(key, newRequest())

		p.startUpdate(key, gs)
		p.add(key, newRequest())
		p.finishUpdate(key)

		assert.Len(t, toApply(p), 1)
		assert.Len(t, p.startUpdate(key, gs), 1)
	})

	t.Run("startUpdate of an unknown key returns nothing", func(t *testing.T) {
		t.Parallel()
		p := newPendingAllocations()

		assert.Empty(t, p.startUpdate(key, gs))
		p.finishUpdate(key)
		assert.Empty(t, toApply(p))
	})
}
