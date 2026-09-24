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
	"math/rand"
	"slices"
	"time"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/util/runtime"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// batchResponses is an async list of responses for matching requests
type batchResponses struct {
	counterErrors error
	listErrors    error
	responses     []response
}

// batchAllocationUpdateWorkers tries to update each newly allocated gs with the last state.
// If the update fails because of a version conflict, all allocations that were applied onto
// a gs will receive an error, thus being available for retries.
func (c *Allocator) batchAllocationUpdateWorkers(ctx context.Context, workerCount int) chan<- batchResponses {
	batchUpdateQueue := make(chan batchResponses)

	for range workerCount {
		go func() {
			for {
				select {
				case batchRes := <-batchUpdateQueue:
					if len(batchRes.responses) > 0 {
						lastGsState := batchRes.responses[len(batchRes.responses)-1].gs

						var propagatedErr error
						updatedGs, updateErr := c.gameServerGetter.GameServers(lastGsState.ObjectMeta.Namespace).Update(ctx, lastGsState, metav1.UpdateOptions{})
						if updateErr != nil {
							if !k8serrors.IsConflict(updateErr) {
								// since we could not allocate, we should put it back
								// but not if it's a conflict, as the cache is no longer up to date, and
								// we should wait for it to get updated with fresh info.
								c.allocationCache.AddGameServer(lastGsState)
								propagatedErr = updateErr
							} else {
								propagatedErr = goErrors.Join(ErrGameServerUpdateConflict, updateErr)
							}
						} else {
							c.allocationCache.AddGameServer(updatedGs)

							if batchRes.counterErrors != nil {
								c.recorder.Event(updatedGs, corev1.EventTypeWarning, "CounterActionError", batchRes.counterErrors.Error())
							}
							if batchRes.listErrors != nil {
								c.recorder.Event(updatedGs, corev1.EventTypeWarning, "ListActionError", batchRes.listErrors.Error())
							}
							c.recorder.Event(updatedGs, corev1.EventTypeNormal, string(updatedGs.Status.State), "Allocated")
						}

						for _, res := range batchRes.responses {
							res.err = propagatedErr
							res.request.response <- res
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	return batchUpdateQueue
}

// ListenAndBatchAllocate is a blocking function that runs in a loop processing allocation
// requests in batches. Unlike ListenAndAllocate, it applies allocations locally to a
// GameServer before batching updates — multiple allocations to the same GameServer within
// a flush window result in a single Kubernetes update, reducing API pressure and improving
// session packing.
func (c *Allocator) ListenAndBatchAllocate(ctx context.Context, updateWorkerCount int) {
	batchUpdateQueue := c.batchAllocationUpdateWorkers(ctx, updateWorkerCount)

	var list []*agonesv1.GameServer
	var sortKey uint64
	requestCount := 0
	gsToReorderIndex := -1
	var gsToReorder *agonesv1.GameServer

	batchResponsesPerGs := make(map[string]batchResponses)

	flush := func() {
		if len(batchResponsesPerGs) > 0 {
			for _, batchRes := range batchResponsesPerGs {
				batchUpdateQueue <- batchRes
			}
			batchResponsesPerGs = make(map[string]batchResponses)
		}

		list = nil
		requestCount = 0
		gsToReorderIndex = -1
		gsToReorder = nil
	}

	checkSortKey := func(gsa *allocationv1.GameServerAllocation) {
		if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
			newSortKey, err := gsa.SortKey()
			if err != nil {
				c.baseLogger.WithError(err).Warn("error getting sortKey for GameServerAllocationSpec")
			}
			if sortKey == 0 {
				sortKey = newSortKey
			}

			if newSortKey != sortKey {
				sortKey = newSortKey
				flush()
			}
		}
	}

	checkRefreshList := func(gsa *allocationv1.GameServerAllocation) {
		if requestCount >= maxBatchBeforeRefresh {
			flush()
		}
		requestCount++

		checkSortKey(gsa)

		if list == nil {
			if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) || gsa.Spec.Scheduling == apis.Packed {
				list = c.allocationCache.ListSortedGameServers(gsa)
			} else {
				list = c.allocationCache.ListSortedGameServersPriorities(gsa)
			}
		} else if gsToReorderIndex >= 0 {
			c.allocationCache.ReorderGameServerAfterAllocation(list, gsToReorderIndex, gsToReorder, gsa.Spec.Priorities, gsa.Spec.Scheduling)
		}
	}

	for {
		select {
		case req := <-c.pendingRequests:
			if req.ctx.Err() != nil {
				c.tryRespondWithError(req, ErrTotalTimeoutExceeded)
				continue
			}

			checkRefreshList(req.gsa)

			foundGs, foundGsIndex, err := c.findGameServerForBatchAllocation(req.gsa, list)
			if err != nil {
				req.response <- response{request: req, gs: nil, err: err}
				continue
			}

			existingBatch, alreadyAllocated := batchResponsesPerGs[string(foundGs.UID)]
			if !alreadyAllocated {
				if removeErr := c.allocationCache.RemoveGameServer(foundGs); removeErr != nil {
					removeErr = c.errs.Wrap(removeErr, "error removing gameserver from cache")
					req.response <- response{request: req, gs: nil, err: removeErr}
					// Setting the entry to nil to mark the gameserver as errored/removed from the list
					list[foundGsIndex] = nil
					continue
				}
			}

			gsToReorder = foundGs.DeepCopy()
			gsToReorderIndex = foundGsIndex
			applyErr, counterErrors, listErrors := c.applyAllocationToLocalGameServer(req.gsa.Spec.MetaPatch, gsToReorder, req.gsa)
			if applyErr == nil {
				if alreadyAllocated {
					existingBatch.responses = append(existingBatch.responses, response{request: req, gs: gsToReorder.DeepCopy(), err: nil})
					existingBatch.counterErrors = goErrors.Join(existingBatch.counterErrors, counterErrors)
					existingBatch.listErrors = goErrors.Join(existingBatch.listErrors, listErrors)
					batchResponsesPerGs[string(gsToReorder.UID)] = existingBatch
				} else {
					batchResponsesPerGs[string(gsToReorder.UID)] = batchResponses{
						responses:     []response{{request: req, gs: gsToReorder.DeepCopy(), err: nil}},
						counterErrors: counterErrors,
						listErrors:    listErrors,
					}
				}
			} else {
				req.response <- response{request: req, gs: nil, err: applyErr}
			}

		case <-ctx.Done():
			return

		default:
			flush()
			time.Sleep(c.batchWaitTime)
		}
	}
}

// applyAllocationToLocalGameServer patches the GameServer with allocation metadata and sets
// it to Allocated state without persisting to Kubernetes. Counter/List actions are applied
// if FeatureCountsAndLists is enabled.
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

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if gsa.Spec.Counters != nil {
			for counter, ca := range gsa.Spec.Counters {
				counterErrors = goErrors.Join(counterErrors, ca.CounterActions(counter, gs))
			}
		}
		if gsa.Spec.Lists != nil {
			for list, la := range gsa.Spec.Lists {
				listErrors = goErrors.Join(listErrors, la.ListActions(list, gs, c.listMaxCapacity))
			}
		}
	}

	return nil, counterErrors, listErrors
}

// findGameServerForBatchAllocation finds an optimal GameServer for the batch allocator
// Returns the first GameServer that matches the selectors and fits the allocation criteria, along with its index in the list
// If no suitable GameServer is found, returns an error
func (c *Allocator) findGameServerForBatchAllocation(gsa *allocationv1.GameServerAllocation, list []*agonesv1.GameServer) (*agonesv1.GameServer, int, error) {
	type result struct {
		gs    *agonesv1.GameServer
		index int
	}

	selectors := make([]*result, len(gsa.Spec.Selectors))

	var loop func(list []*agonesv1.GameServer, f func(i int, gs *agonesv1.GameServer))

	// packed is forward looping, distributed is random looping
	// TODO: this duplicates the scheduling switch in findGameServerForAllocation (find.go)
	// Once FeatureCountsAndLists is stable, batch allocation should replace that function and this
	// block should move to find.go instead of being duplicated here.
	// nolint:dupl // Linter errors on lines are duplicate of findGameServerForAllocation in find.go
	switch gsa.Spec.Scheduling {
	case apis.Packed:
		loop = func(list []*agonesv1.GameServer, f func(i int, gs *agonesv1.GameServer)) {
			for i, gs := range list {
				f(i, gs)
			}
		}
	case apis.Distributed:
		// randomised looping - make a list of indices, and then randomise them
		// as we don't want to change the order of the gameserver slice
		if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) || len(gsa.Spec.Priorities) == 0 {
			l := len(list)
			indices := make([]int, l)
			for i := range l {
				indices[i] = i
			}
			rand.Shuffle(l, func(i, j int) {
				indices[i], indices[j] = indices[j], indices[i]
			})

			loop = func(list []*agonesv1.GameServer, f func(i int, gs *agonesv1.GameServer)) {
				for _, i := range indices {
					f(i, list[i])
				}
			}
		} else {
			// For FeatureCountsAndLists we do not do randomized looping -- instead choose the game
			// server based on the list of Priorities. (The order in which the game servers were sorted
			// in ListSortedGameServersPriorities.)
			loop = func(list []*agonesv1.GameServer, f func(i int, gs *agonesv1.GameServer)) {
				for i, gs := range list {
					f(i, gs)
				}
			}
		}
	default:
		return nil, -1, errs.Errorf("scheduling strategy of '%s' is not supported", gsa.Spec.Scheduling)
	}

	var fits func(*agonesv1.GameServer) bool
	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		fits = counterAndListActionsFit(gsa)
	}

	matchedButFull := false

	loop(list, func(i int, gs *agonesv1.GameServer) {
		if gs == nil {
			return
		}

		// only search the same namespace
		if gs.ObjectMeta.Namespace != gsa.ObjectMeta.Namespace {
			return
		}

		for j, sel := range gsa.Spec.Selectors {
			if selectors[j] != nil || !sel.Matches(gs) {
				continue
			}

			if fits != nil && !fits(gs) {
				matchedButFull = true
				continue
			}

			selectors[j] = &result{gs: gs, index: i}
		}
	})

	for _, r := range selectors {
		if r != nil {
			return r.gs, r.index, nil
		}
	}

	if matchedButFull {
		return nil, 0, ErrConflictInGameServerSelection
	}

	return nil, 0, ErrNoGameServer
}

// counterAndListActionsFit returns a function that checks if a GameServer can
// accommodate all Counter andList actions specified in the GameServerAllocation
func counterAndListActionsFit(gsa *allocationv1.GameServerAllocation) func(*agonesv1.GameServer) bool {
	if len(gsa.Spec.Counters) == 0 && len(gsa.Spec.Lists) == 0 {
		return nil
	}

	return func(gs *agonesv1.GameServer) bool {
		for name, action := range gsa.Spec.Counters {
			status, ok := gs.Status.Counters[name]
			if !ok {
				continue
			}

			capacity := status.Capacity
			count := status.Count
			if action.Capacity != nil {
				capacity = *action.Capacity
				count = min(count, capacity)
			}

			if action.Action == nil || action.Amount == nil {
				continue
			}

			switch *action.Action {
			case agonesv1.GameServerPriorityIncrement:
				if count+*action.Amount > capacity {
					return false
				}
			case agonesv1.GameServerPriorityDecrement:
				if *action.Amount > count {
					return false
				}
			}
		}

		for name, action := range gsa.Spec.Lists {
			status, ok := gs.Status.Lists[name]
			if !ok {
				continue
			}

			if len(action.AddValues) == 0 {
				continue
			}

			capacity := status.Capacity
			if action.Capacity != nil {
				capacity = *action.Capacity
			}

			needed := 0
			seen := make(map[string]bool, len(action.AddValues))
			for _, v := range action.AddValues {
				if seen[v] {
					continue
				}
				seen[v] = true
				if slices.Contains(status.Values, v) {
					continue
				}
				needed++
			}

			if int64(len(status.Values)+needed) > capacity {
				return false
			}
		}

		return true
	}
}
