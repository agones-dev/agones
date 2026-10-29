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
	"sync"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
)

// pendingAllocations is keyed by GameServer. The workqueue only carries the keys, and each worker gets the
// requests of its key from here. A key is handed to one worker at a time
//
//	key    reqs(key)   projected(key)        workqueue
//	gs-1   r1 r2 r3    gs-1 after r1 r2 r3   gs-1 -> worker A: take(gs-1) -> r1 r2 r3
//	gs-2   r4          gs-2 after r4         gs-2 -> worker B: take(gs-2) -> r4
//
// add(key, req, projected) appends to reqs[key] and replaces projected[key]
// take(key) returns reqs[key] and clears it
// finish(key) drops projected[key], unless add() was called again since take()
// projections() returns every projected[key] for all GameServers with pending or in progress allocations

// pendingAllocations manages the requests and projected GameServer states for the batch allocator
type pendingAllocations struct {
	reqs      map[string][]request            // pending requests for each GameServer key
	projected map[string]*agonesv1.GameServer // projected GameServer state for each GameServer key
	mu        sync.Mutex
}

// newPendingAllocations creates and returns a new instance of pendingAllocations
func newPendingAllocations() *pendingAllocations {
	return &pendingAllocations{
		reqs:      make(map[string][]request),
		projected: make(map[string]*agonesv1.GameServer),
	}
}

// add records a request planned onto the GameServer, and its projected version
func (p *pendingAllocations) add(key string, req request, projected *agonesv1.GameServer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.reqs[key] = append(p.reqs[key], req)
	p.projected[key] = projected
}

// take returns and clears the requests pending on the GameServer
func (p *pendingAllocations) take(key string) []request {
	p.mu.Lock()
	defer p.mu.Unlock()

	reqs := p.reqs[key]
	delete(p.reqs, key)

	return reqs
}

// finish drops the projected version of the GameServer once it is updated, unless requests were planned onto it in the meantime
func (p *pendingAllocations) finish(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.reqs[key]) == 0 {
		delete(p.projected, key)
	}
}

// projections returns a list of the projected GameServer states for all GameServers with pending or in progress allocations
func (p *pendingAllocations) projections() []*agonesv1.GameServer {
	p.mu.Lock()
	defer p.mu.Unlock()

	list := make([]*agonesv1.GameServer, 0, len(p.projected))
	for _, gs := range p.projected {
		list = append(list, gs)
	}

	return list
}
