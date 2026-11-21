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
	"sort"
	"time"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/gameservers"
)

// allocationCandidates are the GameServers the listener allocates on, as listed from the cache
type allocationCandidates struct {
	nodeCounts map[string]gameservers.NodeCount
	listedAt   time.Time
	// sorted are the cached GameServers, or their projection
	sorted []*agonesv1.GameServer
	// sortCriteriaHash is the SortKey of the request that last sorted them
	sortCriteriaHash uint64
	// requestCount counts the requests, allocated or not
	requestCount int
}

func newAllocationCandidates(gameServers []*agonesv1.GameServer, nodeCounts map[string]gameservers.NodeCount, gsa *allocationv1.GameServerAllocation) *allocationCandidates {
	candidates := &allocationCandidates{nodeCounts: nodeCounts, sorted: gameServers}
	sortCriteriaHash, _ := gsa.SortKey()
	candidates.sort(gsa, sortCriteriaHash)
	candidates.listedAt = time.Now()
	return candidates
}

// reachedMaxRequests reports whether the candidates have reached the maximum number of requests before a refresh is needed
func (ac *allocationCandidates) reachedMaxRequests() bool {
	return ac.requestCount >= maxBatchBeforeRefresh
}

// olderThan reports whether the candidates were listed at least age ago
func (ac *allocationCandidates) olderThan(age time.Duration) bool {
	return time.Since(ac.listedAt) >= age
}

// find returns the GameServer to allocate on for the given GameServerAllocation, along with its index in the sorted list
// It sorts the candidates if the sort key has changed
func (ac *allocationCandidates) find(gsa *allocationv1.GameServerAllocation) (*agonesv1.GameServer, int, error) {
	sortCriteriaHash, err := gsa.SortKey()
	// without a SortKey, the order of the last request cannot be trusted
	if err != nil || sortCriteriaHash != ac.sortCriteriaHash {
		ac.sort(gsa, sortCriteriaHash)
	}
	return findGameServerForAllocation(gsa, ac.sorted)
}

// reorderAfterAllocation moves the allocated GameServer to its new position in the sorted list, maintaining the order according to the allocation strategy
func (ac *allocationCandidates) reorderAfterAllocation(index int, allocated *agonesv1.GameServer, gsa *allocationv1.GameServerAllocation) {
	sortsBefore := ac.orderFor(gsa)
	replaced := ac.sorted[index]

	// Determine the new index for the allocated GameServer based on the sorting order
	newIndex := index
	switch {
	case sortsBefore(replaced, allocated):
		newIndex += sort.Search(len(ac.sorted)-index-1, func(offset int) bool { return sortsBefore(allocated, ac.sorted[index+1+offset]) })
		copy(ac.sorted[index:newIndex], ac.sorted[index+1:newIndex+1])
	case sortsBefore(allocated, replaced):
		newIndex = sort.Search(index, func(position int) bool { return sortsBefore(allocated, ac.sorted[position]) })
		copy(ac.sorted[newIndex+1:index+1], ac.sorted[newIndex:index])
	}
	ac.sorted[newIndex] = allocated
}

// sort sorts the candidates according to the allocation strategy for the given GameServerAllocation and updates the sort criteria hash
func (ac *allocationCandidates) sort(gsa *allocationv1.GameServerAllocation, sortCriteriaHash uint64) {
	sortsBefore := ac.orderFor(gsa)
	sort.Slice(ac.sorted, func(first, second int) bool { return sortsBefore(ac.sorted[first], ac.sorted[second]) })
	ac.sortCriteriaHash = sortCriteriaHash
}

// orderFor returns a comparison function that determines the order of GameServers for the given GameServerAllocation according to the allocation strategy
func (ac *allocationCandidates) orderFor(gsa *allocationv1.GameServerAllocation) func(first, second *agonesv1.GameServer) bool {
	if gsa.Spec.Scheduling == apis.Packed {
		return func(first, second *agonesv1.GameServer) bool {
			return compareGameServersForPackedStrategy(first, second, gsa.Spec.Priorities, ac.nodeCounts)
		}
	}
	return func(first, second *agonesv1.GameServer) bool {
		return compareGameServersForDistributedStrategy(first, second, gsa.Spec.Priorities)
	}
}
