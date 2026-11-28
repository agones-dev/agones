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
	"slices"
	"sync"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"k8s.io/client-go/tools/cache"
)

// pendingAllocations keeps track of allocation requests for each GameServer
type pendingAllocations struct {
	// waiting holds the requests that are waiting to be processed for each GameServer key
	waiting map[string][]request
	// inFlight holds the requests a worker took for each GameServer key, while it writes them with one Update
	inFlight map[string][]request
	// inFlightVersion is the resourceVersion of the cached GameServer that the update in progress writes on
	inFlightVersion map[string]string
	mu              sync.Mutex
}

func newPendingAllocations() *pendingAllocations {
	return &pendingAllocations{
		waiting:         make(map[string][]request),
		inFlight:        make(map[string][]request),
		inFlightVersion: make(map[string]string),
	}
}

// add stores a request in the waiting list for the given GameServer key
func (pa *pendingAllocations) add(key string, req request) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	pa.waiting[key] = append(pa.waiting[key], req)
}

// startUpdate moves the requests for the given GameServer key from waiting to inFlight, if there is a cached GameServer
func (pa *pendingAllocations) startUpdate(key string, cached *agonesv1.GameServer) []request {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	reqs := pa.waiting[key]
	delete(pa.waiting, key)
	// bring the requests into inFlight only if there is a cached GameServer
	if len(reqs) > 0 && cached != nil {
		pa.inFlight[key] = reqs
		pa.inFlightVersion[key] = cached.ObjectMeta.ResourceVersion
	}

	return reqs
}

// finishUpdate removes the requests for the given GameServer key from inFlight
func (pa *pendingAllocations) finishUpdate(key string) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	delete(pa.inFlight, key)
	delete(pa.inFlightVersion, key)
}

// listCachedWithRequestsToApply lists the cached GameServers, with the requests to apply on each one, by its index
func (pa *pendingAllocations) listCachedWithRequestsToApply(listCached func() []*agonesv1.GameServer) ([]*agonesv1.GameServer, map[int][]request) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	// list the cached GameServers and prepare the map of requests to apply on each one
	gameServers := listCached()
	toApply := make(map[int][]request, len(pa.inFlight)+len(pa.waiting))
	for index, gs := range gameServers {
		key, _ := cache.MetaNamespaceKeyFunc(gs)
		// include in-flight requests only if they match the current resource version of the cached GameServer
		if reqs, ok := pa.inFlight[key]; ok && pa.inFlightVersion[key] == gs.ObjectMeta.ResourceVersion {
			toApply[index] = slices.Clone(reqs)
		}
		// include waiting requests whose context is still active
		for _, req := range pa.waiting[key] {
			if req.ctx.Err() == nil {
				toApply[index] = append(toApply[index], req)
			}
		}
	}

	return gameServers, toApply
}
