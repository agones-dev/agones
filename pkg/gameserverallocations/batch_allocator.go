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
	"slices"
	"time"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
//	│ ListenAndBatchAllocate        │          │ sorted GameServers   │
//	│  1. sortedGameServers         │          └──────────▲───────────┘
//	│  2. findGameServerForBatch... │                     │
//	│  3. plan on a local copy      │                     │ AddGameServer
//	│  4. keep the list sorted      │                     │ (or refresh on conflict)
//	└───────┬───────────────┬───────┘                     │
//	        │ pending.add   │ submit() when idle,         │
//	        │ (key, req)    │ or after N requests         │
//	        ▼               ▼                             │
//	┌──────────────┐  ┌───────────┐  Get   ┌─────────────┴─────────────────┐
//	│ PENDING      │  │ WORKQUEUE │───────▶│ UPDATE WORKERS (N goroutines) │
//	│ key → reqs   │  │ one key   │        │ runUpdateWorker               │
//	│ + projection │◀─│ at a time │        │  take(key) → updateGameServer │
//	└──────────────┘  └───────────┘        │  applyRequests on cached GS   │
//	      take(key)                        │  ONE Update() for all of them │
//	                                       └───────────────┬───────────────┘
//	                                                       │
//	                                                       ▼
//	                                      req.response <- GameServer or error
//	                                      (conflict: error, the caller retries)
//
// Requests planned on the same GameServer while a worker updates it are merged into its next update.

// ListenAndBatchAllocate listens for incoming allocation requests and processes them in batches
func (c *Allocator) ListenAndBatchAllocate(ctx context.Context, updateWorkerCount int) {
	pending := newPendingAllocations()
	queue := c.batchAllocationUpdateWorkers(ctx, updateWorkerCount, pending)

	// candidates are the sorted GameServers the requests are planned onto. They are refreshed from the cache
	// when the loop was idle, after maxBatchBeforeRefresh requests, or when the allocation spec changes
	var candidates []*agonesv1.GameServer
	// candidatesSortKey is used to determine if the candidates need to be refreshed based on the allocation request's sort key
	var candidatesSortKey uint64
	// plannedKeys are the GameServers planned onto since the candidates were refreshed, handed to the update workers
	// all at once, so that each gets a single update for all of them (duplicates are collapsed by the queue)
	var plannedKeys []string

	// submit hands the planned GameServers to the update workers and clears the plannedKeys slice
	submit := func() {
		for _, key := range plannedKeys {
			queue.Add(key)
		}
		plannedKeys = plannedKeys[:0]
	}

	for {
		select {
		case req := <-c.pendingRequests:
			if req.ctx.Err() != nil {
				c.tryRespondWithError(req, ErrTotalTimeoutExceeded)
				continue
			}

			// determine if we need to refresh the candidates based on the sort key, the number of planned requests, or if there are no candidates yet
			reqSortKey := c.batchSortKey(req.gsa)
			if candidates == nil || len(plannedKeys) >= maxBatchBeforeRefresh || reqSortKey != candidatesSortKey {
				submit()
				candidates = c.sortedGameServers(req.gsa, pending)
				candidatesSortKey = reqSortKey
			}

			// find a suitable GameServer for the current allocation request
			gs, index, err := findGameServerForAllocation(req.gsa, candidates)
			if err != nil {
				req.response <- response{request: req, gs: nil, err: err}
				continue
			}

			// make a deep copy of the GameServer to apply the allocation locally
			allocated := gs.DeepCopy()
			applyErr, _, _ := c.applyAllocationToLocalGameServer(req.gsa.Spec.MetaPatch, allocated, req.gsa)
			if applyErr != nil {
				req.response <- response{request: req, gs: nil, err: applyErr}
				continue
			}

			// add the planned GameServer to the pending allocations and record its key for the update workers
			key, _ := cache.MetaNamespaceKeyFunc(gs)
			pending.add(key, req, allocated)
			plannedKeys = append(plannedKeys, key)

			// reorder the candidates after applying the allocation to keep them sorted for the next request
			c.allocationCache.ReorderGameServerAfterAllocation(candidates, index, allocated, req.gsa.Spec.Priorities, req.gsa.Spec.Scheduling)

		case <-ctx.Done():
			return

		default:
			// idle: hand the planned GameServers to the update workers, clear the candidates, and wait for the next batch
			submit()
			candidates = nil
			time.Sleep(c.batchWaitTime)
		}
	}
}

// batchAllocationUpdateWorkers starts the specified number of worker goroutines that process updates for the planned GameServers
func (c *Allocator) batchAllocationUpdateWorkers(ctx context.Context, workerCount int, pending *pendingAllocations) workqueue.TypedInterface[string] {
	queue := workqueue.NewTyped[string]()
	go func() {
		<-ctx.Done()
		queue.ShutDown()
	}()

	for range workerCount {
		// start a new update worker goroutine
		go c.runUpdateWorker(ctx, queue, pending)
	}

	return queue
}

// runUpdateWorker continuously processes keys from the queue, applying the pending requests to the corresponding GameServers until the queue is shut down
func (c *Allocator) runUpdateWorker(ctx context.Context, queue workqueue.TypedInterface[string], pending *pendingAllocations) {
	for {
		// get the next key from the queue, along with a flag indicating if the queue is shutting down
		key, shutdown := queue.Get()
		if shutdown {
			return
		}

		// process the next key from the queue by applying the pending requests to the corresponding GameServer
		c.updateGameServer(ctx, key, pending.take(key))
		// mark the pending requests for this key as finished and signal the queue that processing is done
		pending.finish(key)
		// signal the queue that processing for this key is done
		queue.Done(key)
	}
}

// updateGameServer applies the pending requests to the cached GameServer identified by the key and persists the changes
// If the update fails due to a version conflict, the cache is refreshed with the live GameServer, and the requests receive an error, making them available for retries
func (c *Allocator) updateGameServer(ctx context.Context, key string, reqs []request) {
	// retrieve the cached GameServer by its key from the allocation cache
	gs, ok := c.allocationCache.GetGameServer(key)
	if !ok {
		for _, req := range reqs {
			req.response <- response{request: req, gs: nil, err: ErrNoGameServer}
		}
		return
	}

	// apply the pending requests to a copy of the cached GameServer and collect the results
	toUpdate, applied, counterErrors, listErrors := c.applyRequests(gs, reqs)

	// if no requests were successfully applied, there is nothing to update
	if len(applied) == 0 {
		return
	}

	// attempt to persist the changes to the GameServer in the API server
	updatedGs, err := c.gameServerGetter.GameServers(toUpdate.ObjectMeta.Namespace).Update(ctx, toUpdate, metav1.UpdateOptions{})
	switch {
	// update succeeded without errors
	case err == nil:
		// update the cache with the successfully updated GameServer
		c.allocationCache.AddGameServer(updatedGs)
		c.recordAllocated(updatedGs, counterErrors, listErrors)

	// update failed due to a version conflict
	case k8serrors.IsConflict(err):
		// refresh the cache with the live GameServer if it can be retrieved
		live, getErr := c.gameServerGetter.GameServers(toUpdate.ObjectMeta.Namespace).Get(ctx, toUpdate.ObjectMeta.Name, metav1.GetOptions{})
		if getErr == nil {
			// update the cache with the live GameServer
			c.allocationCache.refreshGameServer(live)
		}
		err = goErrors.Join(ErrGameServerUpdateConflict, err)
	}

	// notify all successfully applied requests of the result, including any errors encountered during the update
	for _, res := range applied {
		res.err = err
		res.request.response <- res
	}
}

// applyRequests applies each allocation request to a copy of the given GameServer
// It returns the updated GameServer, a list of successfully applied responses, and any errors encountered while applying counters or lists
func (c *Allocator) applyRequests(gs *agonesv1.GameServer, reqs []request) (toUpdate *agonesv1.GameServer, applied []response, counterErrors, listErrors error) {
	toUpdate = gs.DeepCopy()
	for _, req := range reqs {
		switch {
		// check if the request context has expired before applying it
		case req.ctx.Err() != nil:
			req.response <- response{request: req, gs: nil, err: ErrTotalTimeoutExceeded}

		// the GameServer is no longer allocatable, so the request is rejected and can be retried
		case toUpdate.IsBeingDeleted() || !readyOrAllocatedGameServerMatcher(toUpdate):
			req.response <- response{request: req, gs: nil, err: ErrConflictInGameServerSelection}

		// apply the request, counter and list action errors do not prevent the allocation
		default:
			applyErr, cErr, lErr := c.applyAllocationToLocalGameServer(req.gsa.Spec.MetaPatch, toUpdate, req.gsa)
			if applyErr != nil {
				req.response <- response{request: req, gs: nil, err: applyErr}
				continue
			}
			counterErrors = goErrors.Join(counterErrors, cErr)
			listErrors = goErrors.Join(listErrors, lErr)
			applied = append(applied, response{request: req, gs: toUpdate.DeepCopy(), err: nil})
		}
	}

	return toUpdate, applied, counterErrors, listErrors
}

// applyAllocationToLocalGameServer patches the GameServer with allocation metadata and sets
// it to Allocated state without persisting to Kubernetes. Counter/List actions are applied
func (c *Allocator) applyAllocationToLocalGameServer(mp allocationv1.MetaPatch, gs *agonesv1.GameServer, gsa *allocationv1.GameServerAllocation) (applyErr, counterErrors, listErrors error) {
	ts, err := time.Now().MarshalText()
	if err != nil {
		return err, nil, nil
	}
	if gs.ObjectMeta.Annotations == nil {
		gs.ObjectMeta.Annotations = make(map[string]string, 1+len(mp.Annotations))
	}
	gs.ObjectMeta.Annotations[LastAllocatedAnnotationKey] = string(ts)
	gs.Status.State = agonesv1.GameServerStateAllocated

	if mp.Labels != nil {
		if gs.ObjectMeta.Labels == nil {
			gs.ObjectMeta.Labels = make(map[string]string, len(mp.Labels))
		}
		maps.Copy(gs.ObjectMeta.Labels, mp.Labels)
	}

	maps.Copy(gs.ObjectMeta.Annotations, mp.Annotations)

	for counter, ca := range gsa.Spec.Counters {
		counterErrors = goErrors.Join(counterErrors, ca.CounterActions(counter, gs))
	}
	for list, la := range gsa.Spec.Lists {
		listErrors = goErrors.Join(listErrors, la.ListActions(list, gs, c.listMaxCapacity))
	}

	return nil, counterErrors, listErrors
}

// recordAllocated records the Allocated event of the GameServer, and any Counter or List action errors as warnings
func (c *Allocator) recordAllocated(gs *agonesv1.GameServer, counterErrors, listErrors error) {
	if counterErrors != nil {
		c.recorder.Event(gs, corev1.EventTypeWarning, "CounterActionError", counterErrors.Error())
	}
	if listErrors != nil {
		c.recorder.Event(gs, corev1.EventTypeWarning, "ListActionError", listErrors.Error())
	}
	c.recorder.Event(gs, corev1.EventTypeNormal, string(gs.Status.State), "Allocated")
}

// batchSortKey returns the sort key for the given GameServerAllocation
func (c *Allocator) batchSortKey(gsa *allocationv1.GameServerAllocation) uint64 {
	// Get the sort key for the GameServerAllocation, which determines the order in which it should be considered for allocation
	sortKey, err := gsa.SortKey()
	if err != nil {
		c.baseLogger.WithError(err).Warn("error getting sortKey for GameServerAllocationSpec")
	}

	return sortKey
}

// sortedGameServers returns the list of GameServers sorted according to the allocation strategy, with pending allocations taken into account
func (c *Allocator) sortedGameServers(gsa *allocationv1.GameServerAllocation, pending *pendingAllocations) []*agonesv1.GameServer {
	var list []*agonesv1.GameServer
	if gsa.Spec.Scheduling == apis.Packed {
		list = c.allocationCache.ListSortedGameServers(gsa)
	} else {
		list = c.allocationCache.ListSortedGameServersPriorities(gsa)
	}

	for _, projected := range pending.projections() {
		// find the index of the projected GameServer in the sorted list
		i := slices.IndexFunc(list, func(gs *agonesv1.GameServer) bool {
			return gs.ObjectMeta.Name == projected.ObjectMeta.Name && gs.ObjectMeta.Namespace == projected.ObjectMeta.Namespace
		})

		// if the projected GameServer is not found in the list, skip it
		if i < 0 {
			continue
		}

		// reorder the GameServer in the list based on the allocation projection
		c.allocationCache.ReorderGameServerAfterAllocation(list, i, projected, gsa.Spec.Priorities, gsa.Spec.Scheduling)
	}

	return list
}
