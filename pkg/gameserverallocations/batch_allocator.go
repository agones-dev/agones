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
	goErrors "errors"
	"maps"
	"sync/atomic"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Batch allocation flow
//
//	 requests (c.pendingRequests)
//	            │
//	            ▼
//	┌───────────────────────────────┐  reads   ┌──────────────────────┐
//	│ LISTENER (single goroutine)   │◀─────────│ ALLOCATION CACHE     │
//	│ ListenAndBatchAllocate        │          └──────────▲───────────┘
//	│  listCandidates: the cache +  │                     │
//	│  the requests to apply on it  │                     │ AddGameServer
//	│  allocateInMemory, for each   │                     │ (or refresh on conflict)
//	│  request:                     │                     │
//	│   1. find a GameServer, or    │                     │
//	│      req.response <- error    │                     │
//	│   2. allocate a copy of it    │                     │
//	│   3. keep candidates sorted   │                     │
//	└───────┬───────────────┬───────┘                     │
//	        │ pending.add   │ queueKeysToUpdate:          │
//	        │ (key, req)    │ updateQueue.Add(key)        │
//	        ▼               ▼                             │
//	┌──────────────┐  ┌───────────┐  Get   ┌──────────────┴────────────────┐
//	│ PENDING      │  │ WORKQUEUE │───────▶│ UPDATE WORKERS (N goroutines) │
//	│ waiting and  │  │ one worker│        │ batchUpdater.runWorker        │
//	│ inFlight reqs│  │ per key   │        │  reads the cached GameServer, │
//	└──────────────┘  └───────────┘        │  pending.startUpdate, then    │
//	                                       │  updateGameServer: allocates  │
//	                                       │  the requests on it, ONE      │
//	                                       │  Update(), then               │
//	                                       │  pending.finishUpdate(key)    │
//	                                       └───────────────┬───────────────┘
//	                                                       ▼
//	                                      req.response <- GameServer or error: on an error, the
//	                                      candidates are marked stale first, for the caller's retry

// batchAllocator is the listener of ListenAndBatchAllocate: it allocates the requests in memory, and hands the
// allocated GameServers to its updater
type batchAllocator struct {
	allocator *Allocator
	// updateQueue is the workqueue used to process updates for allocated GameServers
	updateQueue workqueue.TypedInterface[string]
	// pending is the structure that keeps track of waiting and in-flight allocation requests
	pending *pendingAllocations
	// updater is responsible for processing the allocated GameServers and applying the updates
	updater *batchUpdater
	// candidates holds the current list of allocation candidates
	candidates *allocationCandidates
	// staleCandidates indicates whether the current allocation candidates are stale and need to be refreshed
	staleCandidates *atomic.Bool
	// keysToUpdate are the GameServers allocated on the candidates, not queued yet
	keysToUpdate []string
}

func newBatchAllocator(allocator *Allocator) *batchAllocator {
	batch := &batchAllocator{
		allocator:       allocator,
		updateQueue:     workqueue.NewTyped[string](),
		pending:         newPendingAllocations(),
		candidates:      &allocationCandidates{},
		staleCandidates: &atomic.Bool{},
	}
	batch.staleCandidates.Store(true)
	batch.updater = &batchUpdater{
		allocator:       allocator,
		updateQueue:     batch.updateQueue,
		pending:         batch.pending,
		staleCandidates: batch.staleCandidates,
	}
	return batch
}

// ListenAndBatchAllocate allocates the pending requests in batches, until ctx is done
func (c *Allocator) ListenAndBatchAllocate(ctx context.Context, updateWorkerCount int) {
	batch := newBatchAllocator(c)
	defer batch.updateQueue.ShutDown()
	for range updateWorkerCount {
		go batch.updater.runWorker(ctx)
	}

	for {
		select {
		case req := <-c.pendingRequests:
			if req.ctx.Err() != nil {
				c.tryRespondWithError(req, ErrTotalTimeoutExceeded)
				continue
			}

			// Check if the current allocation candidates are stale or have reached the maximum number of requests
			hadStaleCandidates := batch.staleCandidates.Swap(false) || batch.candidates.reachedMaxRequests()
			if hadStaleCandidates {
				batch.queueKeysToUpdate()
				batch.listCandidates(req.gsa)
			}

			// Try to allocate the request in memory using the current candidates
			err := batch.allocateInMemory(req)
			// If no GameServer was found and the candidates are stale, refresh the candidates and try again
			if goErrors.Is(err, ErrNoGameServer) && !hadStaleCandidates && batch.candidates.olderThan(c.batchWaitTime) {
				batch.queueKeysToUpdate()
				batch.listCandidates(req.gsa)
				err = batch.allocateInMemory(req)
			}

			// If allocation failed, respond with the error
			if err != nil {
				req.response <- response{request: req, err: err}
			}

		case <-ctx.Done():
			return

		default:
			// If idle, mark the candidates as stale and clear them to allow fresh candidates to be listed on the next request
			batch.queueKeysToUpdate()
			batch.candidates = &allocationCandidates{}
			batch.staleCandidates.Store(true)
			time.Sleep(c.batchWaitTime)
		}
	}
}

// listCandidates retrieves the current allocation candidates from the cache and applies the pending requests to them
func (ba *batchAllocator) listCandidates(gsa *allocationv1.GameServerAllocation) {
	nodeCounts := ba.allocator.allocationCache.counter.Counts()
	gameServers, toApply := ba.pending.listCachedWithRequestsToApply(ba.allocator.allocationCache.getGameServers)
	for index, reqs := range toApply {
		gameServers[index] = ba.projectAllocations(gameServers[index], reqs)
	}

	ba.candidates = newAllocationCandidates(gameServers, nodeCounts, gsa)
}

// queueKeysToUpdate adds the keys of the allocated GameServers to the update queue for processing by the updater
func (ba *batchAllocator) queueKeysToUpdate() {
	for _, key := range ba.keysToUpdate {
		ba.updateQueue.Add(key)
	}
	ba.keysToUpdate = nil
}

// projectAllocations returns a copy of the GameServer with the requests allocated on it, for the updater to write and respond to the requests
func (ba *batchAllocator) projectAllocations(gs *agonesv1.GameServer, reqs []request) *agonesv1.GameServer {
	projected := gs.DeepCopy()
	for _, req := range reqs {
		if matchesAllocation(projected, req.gsa) {
			_, _ = applyAllocationInMemory(projected, req.gsa, ba.allocator.listMaxCapacity)
		}
	}
	return projected
}

// allocateInMemory attempts to allocate the request on a candidate in memory, updating the pending requests and keys to update accordingly
func (ba *batchAllocator) allocateInMemory(req request) error {
	ba.candidates.requestCount++
	gs, index, err := ba.candidates.find(req.gsa)
	if err != nil {
		return err
	}

	// make a copy of the GameServer to apply the allocation in memory
	projected := gs.DeepCopy()
	_, _ = applyAllocationInMemory(projected, req.gsa, ba.allocator.listMaxCapacity)
	ba.candidates.reorderAfterAllocation(index, projected, req.gsa)

	key, _ := cache.MetaNamespaceKeyFunc(gs)
	ba.pending.add(key, req)
	ba.keysToUpdate = append(ba.keysToUpdate, key)

	return nil
}

// applyAllocationInMemory applies the allocation's metadata, Counters, and Lists to the GameServer in memory, returning any errors encountered
func applyAllocationInMemory(gs *agonesv1.GameServer, gsa *allocationv1.GameServerAllocation, listMaxCapacity int64) (error, error) {
	metaPatch := gsa.Spec.MetaPatch
	if gs.ObjectMeta.Annotations == nil {
		gs.ObjectMeta.Annotations = make(map[string]string, 1+len(metaPatch.Annotations))
	}
	maps.Copy(gs.ObjectMeta.Annotations, metaPatch.Annotations)
	gs.ObjectMeta.Annotations[LastAllocatedAnnotationKey] = time.Now().Format(time.RFC3339Nano)
	gs.Status.State = agonesv1.GameServerStateAllocated

	if metaPatch.Labels != nil {
		if gs.ObjectMeta.Labels == nil {
			gs.ObjectMeta.Labels = make(map[string]string, len(metaPatch.Labels))
		}
		maps.Copy(gs.ObjectMeta.Labels, metaPatch.Labels)
	}

	var counterErrors, listErrors error
	for counter, counterAction := range gsa.Spec.Counters {
		counterErrors = goErrors.Join(counterErrors, counterAction.CounterActions(counter, gs))
	}
	for list, listAction := range gsa.Spec.Lists {
		listErrors = goErrors.Join(listErrors, listAction.ListActions(list, gs, listMaxCapacity))
	}

	return counterErrors, listErrors
}
